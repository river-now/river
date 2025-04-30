package framework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/river-now/river/kit/esbuildutil"
	"github.com/river-now/river/kit/id"
	"github.com/river-now/river/kit/matcher"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/stringsutil"
	"github.com/river-now/river/kit/tsgen"
	"github.com/river-now/river/kit/viteutil"
)

const (
	riverPrehashedFilePrefix       = "river_out_"
	RiverPathsStageOneJSONFileName = "river_paths_stage_1.json"
	RiverPathsStageTwoJSONFileName = "river_paths_stage_2.json"
)

type PathsFile struct {
	// both stages one and two
	Stage          string           `json:"stage"`
	BuildID        string           `json:"buildID"`
	ClientEntrySrc string           `json:"clientEntrySrc"`
	Paths          map[string]*Path `json:"paths"`

	// stage two only
	ClientEntryOut    string            `json:"clientEntryOut,omitempty"`
	ClientEntryDeps   []string          `json:"clientEntryDeps,omitempty"`
	DepToCSSBundleMap map[string]string `json:"depToCSSBundleMap,omitempty"`
}

func (h *River) writePathsToDisk_StageOne() error {
	pathsJSONOut_StageOne := filepath.Join(
		h.Wave.GetStaticPrivateOutDir(),
		"river_out",
		RiverPathsStageOneJSONFileName,
	)
	err := os.MkdirAll(filepath.Dir(pathsJSONOut_StageOne), os.ModePerm)
	if err != nil {
		return err
	}

	pathsAsJSON, err := json.MarshalIndent(PathsFile{
		Stage:          "one",
		Paths:          h._paths,
		ClientEntrySrc: h.Wave.GetRiverClientEntry(),
		BuildID:        h._buildID,
	}, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(pathsJSONOut_StageOne, pathsAsJSON, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

var (
	reactDedupeList  = []string{"react", "react-dom"}
	preactDedupeList = []string{"preact", "preact/hooks"}
	solidDedupeList  = []string{"solid-js", "solid-js/web"}
)

// 0 = func name, 1,2 = backtick literal, 3 = public dir, 4 = backtick literal
const vitePluginTemplateStr = `
export function riverVitePlugin(): Plugin {
	return {
		name: "river-vite-plugin",
		config(c, { command }) {
			const mp = c.build?.modulePreload;
			const roi = c.build?.rollupOptions?.input;
			const ign = c.server?.watch?.ignored;
			const dedupe = c.resolve?.dedupe;

			const isDev = command === "serve";

			return {
				...c,
				base: isDev ? "/" : "{{.PublicPathPrefix}}",
				build: {
					target: "es2022",
					...c.build,
					emptyOutDir: false,
					modulePreload: { 
						polyfill: false,
						...(typeof mp === "object" ? mp : {}),
					},
					rollupOptions: {
						...c.build?.rollupOptions,
						...rollupOptions,
						input: [
							...rollupOptions.input,
							...(Array.isArray(roi) ? roi : []),
						],
					},
				},
				server: {
					...c.server,
					headers: {
						...c.server?.headers,
						// ensure versions of dynamic imports without the latest
						// hmr updates are not cached by the browser during dev
						"cache-control": "no-store",
					},
					watch: {
						...c.server?.watch,
						ignored: [
							...(Array.isArray(ign) ? ign : []),
							...{{.IgnoredList}},
						],
					},
				},
				resolve: {
					...c.resolve,
					dedupe: [
						...(Array.isArray(dedupe) ? dedupe : []),
						...{{.DedupeList}}
					],
				},
			};
		},
		transform(code, id) {
			const isNodeModules = /node_modules/.test(id);
			if (isNodeModules) return null;
			const assetRegex = /{{.FuncName}}\s*\(\s*(["'{{.Tick}}])(.*?)\1\s*\)/g;
			const needsReplacement = assetRegex.test(code);
			if (!needsReplacement) return null;
			const replacedCode = code.replace(
				assetRegex,
				(_, __, assetPath) => {
					const hashed = (staticPublicAssetMap as Record<string, string>)[assetPath];
					if (!hashed) return '\"' + assetPath + '\"';
					return {{.Tick}}"/{{.PublicDir}}/${hashed}"{{.Tick}};
				},
			);
			if (replacedCode === code) return null;
			return replacedCode;
		},
	};
}
`

var vitePluginTemplate = template.Must(template.New("vitePlugin").Parse(vitePluginTemplateStr))

func (h *River) toRollupOptions(entrypoints []string, fileMap map[string]string) (string, error) {
	var sb stringsutil.Builder

	sb.Return()
	sb.Write(tsgen.Comment("River Vite Plugin:"))
	sb.Return().Return()

	sb.Line("import type { Plugin } from \"vite\";")
	sb.Return()

	sb.Line("const rollupOptions = {")
	sb.Tab().Line("input: [")
	for i, entrypoint := range entrypoints {
		if i > 0 {
			sb.Write(",").Return()
		}
		sb.Tab().Tab().Writef(`"%s"`, entrypoint)
	}
	sb.Line(",")
	sb.Tab().Line("] as string[],")
	sb.Tab().Line(`preserveEntrySignatures: "exports-only",`)
	sb.Tab().Line("output: {")
	sb.Tab().Tab().Line(`assetFileNames: "` + riverPrehashedFilePrefix + `[name]-[hash][extname]",`)
	sb.Tab().Tab().Line(`chunkFileNames: "` + riverPrehashedFilePrefix + `[name]-[hash].js",`)
	sb.Tab().Tab().Line(`entryFileNames: "` + riverPrehashedFilePrefix + `[name]-[hash].js",`)
	sb.Tab().Line("},")
	sb.Line("} as const;")

	sb.Return()
	sb.Write("export const staticPublicAssetMap = ")
	mapAsJSON, err := json.MarshalIndent(fileMap, "", "\t")
	if err != nil {
		return "", fmt.Errorf("error marshalling map to JSON: %w", err)
	}
	sb.Line(string(mapAsJSON) + " as const;")
	sb.Return()
	sb.Line("export type StaticPublicAsset = keyof typeof staticPublicAssetMap;")

	sb.Return()

	sb.Line(fmt.Sprintf(
		"declare global {\n\tfunction %s(staticPublicAsset: StaticPublicAsset): string;\n}",
		h.Wave.GetRiverBuildtimePublicURLFuncName(),
	))

	sb.Return()

	sb.Line(fmt.Sprintf(
		"export const publicPathPrefix = \"%s\";",
		h.Wave.GetPublicPathPrefix(),
	))

	sb.Return()

	sb.Line(`export function waveRuntimeURL(originalPublicURL: keyof typeof staticPublicAssetMap) {
	const url = staticPublicAssetMap[originalPublicURL] ?? originalPublicURL;
	return publicPathPrefix + url;
}`)

	publicPrefixToUse := path.Clean(h.Wave.GetPublicPathPrefix())
	publicPrefixToUse = matcher.StripLeadingSlash(publicPrefixToUse)
	publicPrefixToUse = matcher.StripTrailingSlash(publicPrefixToUse)
	tick := "`"

	var buf bytes.Buffer

	var dedupeList []string
	switch UIVariant(h.Wave.GetRiverUIVariant()) {
	case UIVariants.React:
		dedupeList = reactDedupeList
	case UIVariants.Preact:
		dedupeList = preactDedupeList
	case UIVariants.Solid:
		dedupeList = solidDedupeList
	}

	ignoredList := []string{
		"**/*.go",
		path.Join("**", h.Wave.GetDistDir()+"/**/*"),
		path.Join("**", h.Wave.GetPrivateStaticDir()+"/**/*"),
		path.Join("**", h.Wave.GetConfigFile()),
		path.Join("**", h.Wave.GetRiverTSGenOutPath()),
		path.Join("**", h.Wave.GetRiverClientRouteDefsFile()),
	}

	ignoreTabs := strings.Repeat("\t", 7)
	stringifiedIgnoreBytes, err := json.MarshalIndent(ignoredList, "", ignoreTabs+"\t")
	if err != nil {
		return "", fmt.Errorf("error marshalling ignored list to JSON: %w", err)
	}

	stringifiedDedupeBytes, err := json.Marshal(dedupeList)
	if err != nil {
		return "", fmt.Errorf("error marshalling dedupe list to JSON: %w", err)
	}

	stringifiedIgnore := string(stringifiedIgnoreBytes)
	stringifiedIgnore = strings.TrimSuffix(stringifiedIgnore, "]")
	stringifiedIgnore += ignoreTabs + "]"

	err = vitePluginTemplate.Execute(&buf, map[string]any{
		"FuncName":         h.Wave.GetRiverBuildtimePublicURLFuncName(),
		"PublicDir":        publicPrefixToUse,
		"PublicPathPrefix": h.Wave.GetPublicPathPrefix(),
		"Tick":             tick,
		"IgnoredList":      template.HTML(stringifiedIgnore),
		"DedupeList":       template.HTML(stringifiedDedupeBytes),
	})
	if err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	sb.Write(buf.String())

	return sb.String(), nil
}

func (h *River) handleViteConfigHelper(extraTS string) error {
	entrypoints := h.getEntrypoints()

	publicFileMap, err := h.Wave.GetSimplePublicFileMapBuildtime()
	if err != nil {
		Log.Error(fmt.Sprintf("HandleEntrypoints: error getting public file map: %s", err))
		return err
	}

	rollupOptions, err := h.toRollupOptions(entrypoints, publicFileMap)
	if err != nil {
		Log.Error(fmt.Sprintf("HandleEntrypoints: error converting entrypoints to rollup options: %s", err))
		return err
	}

	rollupOptions = extraTS + rollupOptions

	target := filepath.Join(".", h.Wave.GetRiverTSGenOutPath())

	err = os.MkdirAll(filepath.Dir(target), os.ModePerm)
	if err != nil {
		Log.Error(fmt.Sprintf("HandleEntrypoints: error creating directory: %s", err))
		return err
	}

	if err = os.WriteFile(target, []byte(rollupOptions), os.ModePerm); err != nil {
		Log.Error(fmt.Sprintf("HandleEntrypoints: error writing entrypoints to disk: %s", err))
		return err
	}

	return nil
}

const nodeScript = `
const path = await import("node:path");
const { pathToFileURL } = await import("node:url");
const importPath = path.resolve(".", process.argv.slice(1)[0]);
const importPathClean = pathToFileURL(importPath);
const routesFile = await import(importPathClean);
const final = routesFile.default.__all_routes();
console.log(JSON.stringify(final));
`

const routesBuilderSnippet = `
function __river_routes_builder() {
	const routes = [];
	function Add(pattern, ip, ck, ebk) {
		const routeObj = { p: pattern, m: ip, k: ck };
		if (ebk) {
			routeObj.ek = ebk;
		}
		routes.push(routeObj);
	}
	return { Add, __all_routes: () => routes };
}
const routes = __river_routes_builder();
`

type NodeScriptResultItem struct {
	Pattern  string `json:"p"`
	Module   string `json:"m"`
	Key      string `json:"k"`
	ErrorKey string `json:"ek,omitempty"`
}

type NodeScriptResult []NodeScriptResultItem

func (h *River) Build(opts *BuildOptions) error {
	a := time.Now()

	h.mu.Lock()
	defer h.mu.Unlock()

	h._isDev = opts.IsDev

	buildID, err := id.New(16)
	if err != nil {
		Log.Error(fmt.Sprintf("error generating random ID: %s", err))
		return err
	}
	h._buildID = buildID

	Log.Info("START building River", "buildID", h._buildID)

	esbuildResult := esbuild.Build(esbuild.BuildOptions{
		EntryPoints: []string{h.Wave.GetRiverClientRouteDefsFile()},
		Bundle:      false,
		Write:       false,
		Format:      esbuild.FormatESModule,
		Platform:    esbuild.PlatformNode,
		Metafile:    true,
	})
	if err := esbuildutil.CollectErrors(esbuildResult); err != nil {
		Log.Error(fmt.Sprintf("esbuild errors: %s", err))
		return err
	}

	metafile, err := esbuildutil.UnmarshalOutput(esbuildResult)
	if err != nil {
		Log.Error(fmt.Sprintf("error unmarshalling esbuild output: %s", err))
		return err
	}

	importsUnfiltered := metafile.Outputs["routes.js"].Imports
	var imports []string
	for _, imp := range importsUnfiltered {
		if imp.Kind != esbuildutil.KindDymanicImport {
			continue
		}
		imports = append(imports, imp.Path)
	}

	tempDirName, err := os.MkdirTemp(".", "river-build")
	if err != nil {
		Log.Error(fmt.Sprintf("error creating temp dir: %s", err))
		return err
	}
	defer os.RemoveAll(tempDirName)

	code := string(esbuildResult.OutputFiles[0].Contents)

	code = routesBuilderSnippet + code

	routesSrcFile := path.Join(".", h.Wave.GetRiverClientRouteDefsFile())
	routesDir := path.Dir(routesSrcFile)

	for _, imp := range imports {
		doubleQuotes := fmt.Sprintf(`import("%s")`, imp)
		singleQuotes := fmt.Sprintf("import('%s')", imp)
		backticks := fmt.Sprintf("import(`%s`)", imp)
		replacement := fmt.Sprintf(`"%s"`, path.Join(routesDir, imp))
		code = strings.ReplaceAll(code, doubleQuotes, replacement)
		code = strings.ReplaceAll(code, singleQuotes, replacement)
		code = strings.ReplaceAll(code, backticks, replacement)
	}

	location := filepath.Join(".", tempDirName, "routes.js")
	err = os.MkdirAll(filepath.Dir(location), os.ModePerm)
	if err != nil {
		Log.Error(fmt.Sprintf("error creating directory: %s", err))
		return err
	}
	err = os.WriteFile(location, []byte(code), os.ModePerm)
	if err != nil {
		Log.Error(fmt.Sprintf("error writing file to disk: %s", err))
		return err
	}

	cmd := exec.Command("node", "--input-type=module", "-e", nodeScript)
	cmd.Args = append(cmd.Args, filepath.ToSlash(location))

	output, err := cmd.Output()
	if err != nil {
		Log.Error(fmt.Sprintf("error running node script: %s | output: %s", err, string(output)))
		return err
	}

	var nodeScriptResult NodeScriptResult
	if err := json.Unmarshal(output, &nodeScriptResult); err != nil {
		Log.Error(fmt.Sprintf("error unmarshalling node script output: %s", err))
	}

	h._paths = make(map[string]*Path)

	for _, item := range nodeScriptResult {
		h._paths[item.Pattern] = &Path{
			OriginalPattern: item.Pattern,
			SrcPath:         item.Module,
			ExportKey:       item.Key,
			ErrorExportKey:  item.ErrorKey,
		}
	}

	if err = h.writePathsToDisk_StageOne(); err != nil {
		Log.Error(fmt.Sprintf("error writing paths to disk: %s", err))
		return err
	}

	// Remove all files in StaticPublicOutDir starting with riverChunkPrefix or riverEntryPrefix.
	err = cleanStaticPublicOutDir(h.Wave.GetStaticPublicOutDir())
	if err != nil {
		Log.Error(fmt.Sprintf("error cleaning static public out dir: %s", err))
		return err
	}

	tsgenOutput, err := h.GenerateTypeScript(&TSGenOptions{
		UIRouter:      opts.UIRouter,
		ActionsRouter: opts.ActionsRouter,
		AdHocTypes:    opts.AdHocTypes,
		ExtraTSCode:   opts.ExtraTSCode,
	})
	if err != nil {
		Log.Error(fmt.Sprintf("error generating TypeScript: %s", err))
		return err
	}

	if err = h.handleViteConfigHelper(tsgenOutput); err != nil {
		// already logged internally in handleViteConfigHelper
		return err
	}

	if !h._isDev {
		if err := h.Wave.ViteProdBuild(); err != nil {
			Log.Error(fmt.Sprintf("error running vite prod build: %s", err))
			return err
		}

		if err := h.PostViteProdBuild(); err != nil {
			Log.Error(fmt.Sprintf("error running post vite prod build: %s", err))
			return err
		}
	}

	Log.Info("DONE building River",
		"buildID", h._buildID,
		"routes found", len(nodeScriptResult),
		"duration", time.Since(a),
	)

	return nil
}

func (h *River) getViteDevURL() string {
	if !h._isDev {
		return ""
	}
	return fmt.Sprintf("http://localhost:%s", viteutil.GetVitePortStr())
}

/////////////////////////////////////////////////////////////////////
/////// CLEAN STATIC PUBLIC OUT DIR
/////////////////////////////////////////////////////////////////////

func cleanStaticPublicOutDir(staticPublicOutDir string) error {
	fileInfo, err := os.Stat(staticPublicOutDir)
	if err != nil {
		if os.IsNotExist(err) {
			Log.Warn(fmt.Sprintf("static public out dir does not exist: %s", staticPublicOutDir))
			return nil
		}
		return err
	}

	if !fileInfo.IsDir() {
		wrapped := fmt.Errorf("%s is not a directory", staticPublicOutDir)
		Log.Error(wrapped.Error())
		return wrapped
	}

	// delete the ".vite" directory
	err = os.RemoveAll(filepath.Join(staticPublicOutDir, ".vite"))
	if err != nil {
		wrapped := fmt.Errorf("error removing .vite directory: %s", err)
		Log.Error(wrapped.Error())
		return wrapped
	}

	// delete all files starting with riverPrehashedFilePrefix
	err = filepath.Walk(staticPublicOutDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(filepath.Base(path), riverPrehashedFilePrefix) {
			err = os.Remove(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

/////////////////////////////////////////////////////////////////////
/////// GET ENTRYPOINTS
/////////////////////////////////////////////////////////////////////

func (h *River) getEntrypoints() []string {
	entryPoints := make(map[string]struct{}, len(h._paths)+1)
	entryPoints[path.Clean(h.Wave.GetRiverClientEntry())] = struct{}{}
	for _, path := range h._paths {
		if path.SrcPath != "" {
			entryPoints[path.SrcPath] = struct{}{}
		}
	}
	keys := make([]string, 0, len(entryPoints))
	for key := range entryPoints {
		keys = append(keys, key)
	}
	slices.SortStableFunc(keys, strings.Compare)
	return keys
}

/////////////////////////////////////////////////////////////////////
/////// TO PATHS FILE -- STAGE TWO
/////////////////////////////////////////////////////////////////////

func (h *River) toPathsFile_StageTwo() (*PathsFile, error) {
	riverClientEntryOut := ""
	riverClientEntryDeps := []string{}
	depToCSSBundleMap := make(map[string]string)

	viteManifest, err := viteutil.ReadManifest(h.Wave.GetViteManifestLocation())
	if err != nil {
		Log.Error(fmt.Sprintf("error reading vite manifest: %s", err))
		return nil, err
	}

	cleanClientEntry := filepath.Clean(h.Wave.GetRiverClientEntry())

	// Assuming manifestJSON is your Vite manifest
	for key, chunk := range viteManifest {
		cleanKey := filepath.Base(chunk.File)

		// Handle CSS bundles
		// In Vite, CSS is handled through the CSS array
		if len(chunk.CSS) > 0 {
			for _, cssFile := range chunk.CSS {
				depToCSSBundleMap[cleanKey] = filepath.Base(cssFile)
			}
		}

		// Get dependencies
		deps := viteutil.FindAllDependencies(viteManifest, key)

		// Handle client entry
		if chunk.IsEntry && cleanClientEntry == chunk.Src {
			riverClientEntryOut = cleanKey
			depsWithoutClientEntry := make([]string, 0, len(deps)-1)
			for _, dep := range deps {
				if dep != riverClientEntryOut {
					depsWithoutClientEntry = append(depsWithoutClientEntry, dep)
				}
			}
			riverClientEntryDeps = depsWithoutClientEntry
		} else {
			// Handle other paths
			for i, path := range h._paths {
				// Compare with source path instead of entryPoint
				if path.SrcPath == chunk.Src {
					h._paths[i].OutPath = cleanKey
					h._paths[i].Deps = deps
				}
			}
		}
	}

	return &PathsFile{
		Stage:             "two",
		DepToCSSBundleMap: depToCSSBundleMap,
		Paths:             h._paths,
		ClientEntrySrc:    h.Wave.GetRiverClientEntry(),
		ClientEntryOut:    riverClientEntryOut,
		ClientEntryDeps:   riverClientEntryDeps,
		BuildID:           h._buildID,
	}, nil
}

type BuildOptions struct {
	IsDev         bool
	UIRouter      *mux.NestedRouter
	ActionsRouter *mux.Router
	AdHocTypes    []*AdHocType
	ExtraTSCode   string
}
