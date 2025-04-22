package configschema

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/river-now/river/kit/jsonschema"
)

func Write(target string) error {
	json, err := json.MarshalIndent(Root_Schema, "", "\t")
	if err != nil {
		return fmt.Errorf("configschema.Write: failed to marshal JSON schema: %w", err)
	}

	json = append(json, []byte("\n")...)

	if err = os.WriteFile(target, json, 0644); err != nil {
		return fmt.Errorf("configschema.Write: failed to write JSON schema: %w", err)
	}

	return nil
}

/////////////////////////////////////////////////////////////////////
/////// ROOT
/////////////////////////////////////////////////////////////////////

var Root_Schema = jsonschema.Entry{
	Schema:      "http://json-schema.org/draft-07/schema#",
	Type:        jsonschema.TypeObject,
	Description: "Wave configuration schema.",
	Required:    []string{"Core"},
	Properties: struct {
		Core  jsonschema.Entry
		Vite  jsonschema.Entry
		Watch jsonschema.Entry
	}{
		Core:  Core_Schema,
		Vite:  Vite_Schema,
		Watch: Watch_Schema,
	},
}

/////////////////////////////////////////////////////////////////////
/////// CORE SETTINGS
/////////////////////////////////////////////////////////////////////

var Core_Schema = jsonschema.RequiredObject(jsonschema.Def{
	Description:      `All paths should be set relative to the directory from which you run commands.`,
	RequiredChildren: []string{"DevBuildHook", "ProdBuildHook", "MainAppEntry", "DistDir"},
	AllOf: []any{jsonschema.IfThen{
		If: map[string]any{
			"properties": map[string]any{
				"ServerOnlyMode": map[string]any{"enum": []bool{false}},
			},
		},
		Then: map[string]any{
			"required": []string{"StaticAssetDirs"},
		},
	}},
	Properties: struct {
		DevBuildHook     jsonschema.Entry
		ProdBuildHook    jsonschema.Entry
		MainAppEntry     jsonschema.Entry
		DistDir          jsonschema.Entry
		StaticAssetDirs  jsonschema.Entry
		CSSEntryFiles    jsonschema.Entry
		PublicPathPrefix jsonschema.Entry
		ServerOnlyMode   jsonschema.Entry
	}{
		DevBuildHook:     DevBuildHook_Schema,
		ProdBuildHook:    ProdBuildHook_Schema,
		MainAppEntry:     MainAppEntry_Schema,
		DistDir:          DistDir_Schema,
		StaticAssetDirs:  StaticAssetDirs_Schema,
		CSSEntryFiles:    CSSEntryFiles_Schema,
		PublicPathPrefix: PublicPathPrefix_Schema,
		ServerOnlyMode:   ServerOnlyMode_Schema,
	},
})

/////////////////////////////////////////////////////////////////////
/////// CORE SETTINGS -- DEV BUILD HOOK
/////////////////////////////////////////////////////////////////////

var DevBuildHook_Schema = jsonschema.OptionalString(jsonschema.Def{
	Description: `Command to run to build your app in dev mode.`,
	Examples:    []string{"go run ./backend/cmd/build -dev"},
})

/////////////////////////////////////////////////////////////////////
/////// CORE SETTINGS -- PROD BUILD HOOK
/////////////////////////////////////////////////////////////////////

var ProdBuildHook_Schema = jsonschema.OptionalString(jsonschema.Def{
	Description: `Command to run to build your app in prod mode.`,
	Examples:    []string{"go run ./backend/cmd/build"},
})

/////////////////////////////////////////////////////////////////////
/////// CORE SETTINGS -- APP ENTRY
/////////////////////////////////////////////////////////////////////

var MainAppEntry_Schema = jsonschema.RequiredString(jsonschema.Def{
	Description: `Path to your app's main.go entry file (or its parent directory).`,
	Examples:    []string{"./cmd/app/main.go", "./cmd/app"},
})

/////////////////////////////////////////////////////////////////////
/////// CORE SETTINGS -- DIST DIR
/////////////////////////////////////////////////////////////////////

var DistDir_Schema = jsonschema.RequiredString(jsonschema.Def{
	Description: jsonschema.UniqueFrom("Core.StaticAssetDirs.Private", "Core.StaticAssetDirs.Public"),
	Examples:    []string{"./dist"},
})

/////////////////////////////////////////////////////////////////////
/////// CORE SETTINGS -- STATIC DIRS
/////////////////////////////////////////////////////////////////////

var StaticAssetDirs_Schema = jsonschema.ObjectWithOverride(`This object is required unless you are in ServerOnlyMode.`, jsonschema.Def{
	RequiredChildren: []string{"Private", "Public"},
	Properties: struct {
		Private jsonschema.Entry
		Public  jsonschema.Entry
	}{
		Private: Private_Schema,
		Public:  Public_Schema,
	},
})

var Private_Schema = jsonschema.RequiredString(jsonschema.Def{
	Description: jsonschema.UniqueFrom("Core.DistDir", "Core.StaticAssetDirs.Public"),
	Examples:    []string{"./static/private"},
})

var Public_Schema = jsonschema.RequiredString(jsonschema.Def{
	Description: jsonschema.UniqueFrom("Core.DistDir", "Core.StaticAssetDirs.Private"),
	Examples:    []string{"./static/public"},
})

/////////////////////////////////////////////////////////////////////
/////// CORE SETTINGS -- CSS ENTRY FILES
/////////////////////////////////////////////////////////////////////

var CSSEntryFiles_Schema = jsonschema.OptionalObject(jsonschema.Def{
	Description: `Use this if you are using Wave's CSS features.`,
	Properties: struct {
		Critical    jsonschema.Entry
		NonCritical jsonschema.Entry
	}{
		Critical:    Critical_Schema,
		NonCritical: NonCritical_Schema,
	},
})

var Critical_Schema = jsonschema.OptionalString(jsonschema.Def{
	Description: `Path to your critical CSS entry file.`,
	Examples:    []string{"./styles/main.critical.css"},
})

var NonCritical_Schema = jsonschema.OptionalString(jsonschema.Def{
	Description: `Path to your non-critical CSS entry file.`,
	Examples:    []string{"./styles/main.css"},
})

/////////////////////////////////////////////////////////////////////
/////// CORE SETTINGS -- PUBLIC PATH PREFIX
/////////////////////////////////////////////////////////////////////

var PublicPathPrefix_Schema = jsonschema.RequiredString(jsonschema.Def{
	Description: `Path prefix for your public assets. Must both start and end with a "/".`,
	Examples:    []string{"/public/"},
})

/////////////////////////////////////////////////////////////////////
/////// CORE SETTINGS -- SERVER ONLY
/////////////////////////////////////////////////////////////////////

var ServerOnlyMode_Schema = jsonschema.OptionalBoolean(jsonschema.Def{
	Description: `If true, skips static asset processing/serving and browser reloading.`,
	Default:     false,
})

/////////////////////////////////////////////////////////////////////
/////// VITE SETTINGS
/////////////////////////////////////////////////////////////////////

var Vite_Schema = jsonschema.OptionalObject(jsonschema.Def{
	Description: `Vite settings.`,
	Properties: struct {
		JSPackageManagerBaseCmd jsonschema.Entry
		JSPackageManagerCmdDir  jsonschema.Entry
		DefaultPort             jsonschema.Entry
		ViteConfigFile          jsonschema.Entry
	}{
		JSPackageManagerBaseCmd: JSPackageManagerBaseCmd_Schema,
		JSPackageManagerCmdDir:  JSPackageManagerCmdDir_Schema,
		DefaultPort:             DefaultPort_Schema,
		ViteConfigFile:          ViteConfigFile_Schema,
	},
	RequiredChildren: []string{"JSPackageManagerBaseCmd"},
})

/////////////////////////////////////////////////////////////////////
/////// VITE SETTINGS -- JS PACKAGE MANAGER BASE CMD
/////////////////////////////////////////////////////////////////////

var JSPackageManagerBaseCmd_Schema = jsonschema.RequiredString(jsonschema.Def{
	Description: `Base command to run Vite using your preferred package manager. This is not the command to run package.json scripts, but rather the command to run standalone CLIs (e.g., "npx", not "npm run").`,
	Examples:    []string{"npx", "pnpm", "yarn", "bunx"},
})

/////////////////////////////////////////////////////////////////////
/////// VITE SETTINGS -- JS PACKAGE MANAGER CMD DIR
/////////////////////////////////////////////////////////////////////

var JSPackageManagerCmdDir_Schema = jsonschema.OptionalString(jsonschema.Def{
	Description: `Directory to run the package manager command from. For example, if you're running commands from ".", but you want to run Vite from "./frontend", set this to "./frontend".`,
	Examples:    []string{"./frontend"},
	Default:     ".",
})

/////////////////////////////////////////////////////////////////////
/////// VITE SETTINGS -- DEFAULT PORT
/////////////////////////////////////////////////////////////////////

var DefaultPort_Schema = jsonschema.OptionalNumber(jsonschema.Def{
	Description: `Default port to use for Vite. This is used when you run "wave dev" without specifying a port.`,
	Default:     5173,
})

/////////////////////////////////////////////////////////////////////
/////// VITE SETTINGS -- CONFIG FILE
/////////////////////////////////////////////////////////////////////

var ViteConfigFile_Schema = jsonschema.OptionalString(jsonschema.Def{
	Description: `Path to your Vite config file if it is in a non-standard location. Should be set relative to the JSPackageManagerCmdDir, if set, otherwise your current working directory.`,
	Examples:    []string{"./configs/vite.ts"},
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS
/////////////////////////////////////////////////////////////////////

var Watch_Schema = jsonschema.OptionalObject(jsonschema.Def{
	Description: `WatchRoot is the outermost directory to watch for changes in, and your dev config watched files will be set relative to this directory.`,
	Properties: struct {
		WatchRoot           jsonschema.Entry
		HealthcheckEndpoint jsonschema.Entry
		Include             jsonschema.Entry
		Exclude             jsonschema.Entry
	}{
		WatchRoot:           WatchRoot_Schema,
		HealthcheckEndpoint: HealthcheckEndpoint_Schema,
		Include:             Include_Schema,
		Exclude:             Exclude_Schema,
	},
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- ROOT DIR
/////////////////////////////////////////////////////////////////////

var WatchRoot_Schema = jsonschema.OptionalString(jsonschema.Def{
	Description: `The directory against which all watch settings paths are relative. If not set, all paths are relative to the directory from which you run commands.`,
	Default:     ".",
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- HEALTHCHECK PATH
/////////////////////////////////////////////////////////////////////

var HealthcheckEndpoint_Schema = jsonschema.OptionalString(jsonschema.Def{
	Description: `Path to your app's healthcheck endpoint. Must return 200 OK if healthy. During dev-time rebuilds and restarts, this endpoint will be polled to determine when your app is ready to begin serving normal requests.`,
	Examples:    []string{"/healthz"},
	Default:     "/",
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- INCLUDE
/////////////////////////////////////////////////////////////////////

var Include_Schema = jsonschema.OptionalArray(jsonschema.Def{
	Description: `This is where you tell the dev server what to watch for, and what to do when it detects a change.`,
	Items:       IncludeItems_Schema,
})

var IncludeItems_Schema = jsonschema.OptionalObject(jsonschema.Def{
	RequiredChildren: []string{"Pattern"},
	Properties: struct {
		Pattern                            jsonschema.Entry
		OnChangeHooks                      jsonschema.Entry
		RecompileGoBinary                  jsonschema.Entry
		RestartApp                         jsonschema.Entry
		OnlyRunClientDefinedRevalidateFunc jsonschema.Entry
		RunOnChangeOnly                    jsonschema.Entry
		SkipRebuildingNotification         jsonschema.Entry
		TreatAsNonGo                       jsonschema.Entry
	}{
		Pattern:                            Pattern_Schema,
		OnChangeHooks:                      OnChangeHooks_Schema,
		RecompileGoBinary:                  RecompileGoBinary_Schema,
		RestartApp:                         RestartApp_Schema,
		OnlyRunClientDefinedRevalidateFunc: OnlyRunClientDefinedRevalidateFunc_Schema,
		RunOnChangeOnly:                    RunOnChangeOnly_Schema,
		SkipRebuildingNotification:         SkipRebuildingNotification_Schema,
		TreatAsNonGo:                       TreatAsNonGo_Schema,
	},
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- INCLUDE -- PATTERN
/////////////////////////////////////////////////////////////////////

var Pattern_Schema = jsonschema.RequiredString(jsonschema.Def{
	Description: `Glob pattern for matching files (set relative to WatchRoot).`,
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- INCLUDE -- ON CHANGE
/////////////////////////////////////////////////////////////////////

var OnChangeHooks_Schema = jsonschema.OptionalArray(jsonschema.Def{
	Description: `Commands to run when a file matching the pattern changes. By default, onchange commands will run before any Wave processing. As long as "SkipRebuildingNotification" is false (default), Wave will send a signal to the browser to show the "Rebuilding..." status message first. The default Timing is "pre". You can also change the Timing to "post", "concurrent", or "concurrent-no-wait" if desired.`,
	Items:       OnChangeHooksItems_Schema,
})

var OnChangeHooksItems_Schema = jsonschema.OptionalObject(jsonschema.Def{
	RequiredChildren: []string{"Cmd"},
	Properties: struct {
		Cmd     jsonschema.Entry
		Timing  jsonschema.Entry
		Exclude jsonschema.Entry
	}{
		Cmd:     Cmd_Schema,
		Timing:  Timing_Schema,
		Exclude: OnChangeHooksExclude_Schema,
	},
})

var Cmd_Schema = jsonschema.RequiredString(jsonschema.Def{
	Description: `Command to run when a file matching the pattern changes.`,
	Examples:    []string{"echo 'File changed!'"},
})

var Timing_Schema = jsonschema.OptionalString(jsonschema.Def{
	Description: `Timing of the given command relative to rebuilding the Wave file system.`,
	Enum:        []string{"pre", "post", "concurrent", "concurrent-no-wait"},
	Default:     "pre",
})

var OnChangeHooksExclude_Schema = jsonschema.OptionalArray(jsonschema.Def{
	Description: `Glob patterns for files to exclude from the onchange command (set relative to WatchRoot).`,
	Items:       jsonschema.OptionalString(jsonschema.Def{}),
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- INCLUDE -- RECOMPILE BINARY
/////////////////////////////////////////////////////////////////////

var RecompileGoBinary_Schema = jsonschema.OptionalBoolean(jsonschema.Def{
	Description: `Use this if you need the Go binary recompiled before the browser is reloaded.`,
	Default:     false,
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- INCLUDE -- RESTART APP
/////////////////////////////////////////////////////////////////////

var RestartApp_Schema = jsonschema.OptionalBoolean(jsonschema.Def{
	Description: `Use this if you explicitly need the app to be restarted before reloading the browser. Example: You might need this if you memory cache template files on first hit, in which case you would want to restart the app to reset the cache.`,
	Default:     false,
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- INCLUDE -- RUN CLIENT DEFINED REVALIDATE FUNC
/////////////////////////////////////////////////////////////////////

var OnlyRunClientDefinedRevalidateFunc_Schema = jsonschema.OptionalBoolean(jsonschema.Def{
	Description: `If set to true, Wave will send a message to the browser to run a method called "__waveRevalidate" (if it exists on the window object). For example, your framework might provide you with a client-side revalidate function in order to maintain client state, in which case you'd set "window.__waveRevalidate" to that function, and set this field to true.`,
	Default:     false,
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- INCLUDE -- RUN ON CHANGE ONLY
/////////////////////////////////////////////////////////////////////

var RunOnChangeOnly_Schema = jsonschema.OptionalBoolean(jsonschema.Def{
	Description: `Use this if your onChange saves a file that will trigger another reload process, or if you need this behavior for any other reason. Will not reload the browser. Note that if you use this setting, you should not set an explicit Timing on the OnChangeHooks (or set them explicitly to "pre"). If you set them to "post", "concurrent", or "concurrent-no-wait" while using RunOnChangeOnly, the onchange commands will not run.`,
	Default:     false,
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- INCLUDE -- SKIP REBUILDING NOTIFICATION
/////////////////////////////////////////////////////////////////////

var SkipRebuildingNotification_Schema = jsonschema.OptionalBoolean(jsonschema.Def{
	Description: `Use this if you are using RunOnChangeOnly, but your onchange won't actually trigger another reload process (so you dont get stuck with "Rebuilding..." showing in the browser)`,
	Default:     false,
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- INCLUDE -- TREAT AS NON GO
/////////////////////////////////////////////////////////////////////

var TreatAsNonGo_Schema = jsonschema.OptionalBoolean(jsonschema.Def{
	Description: `This may come into play if you have a .go file that is totally independent from your app, such as a wasm file that you are building with a separate build process and serving from your app. If you set this to true, processing on any captured .go file will be as though it were an arbitrary non-Go file extension. Only relevant for Go files (for non-Go files, this is a no-op).`,
	Default:     false,
})

/////////////////////////////////////////////////////////////////////
/////// WATCH SETTINGS -- EXCLUDE
/////////////////////////////////////////////////////////////////////

var Exclude_Schema = jsonschema.OptionalObject(jsonschema.Def{
	Description: `Glob patterns for files and directories to exclude from the watcher (set relative to WatchRoot). This is for carving out exceptions from your Include list.`,
	Properties: struct {
		Dirs  jsonschema.Entry
		Files jsonschema.Entry
	}{
		Dirs:  ExcludeDirs_Schema,
		Files: ExcludeFiles_Schema,
	},
})

var ExcludeDirs_Schema = jsonschema.OptionalArray(jsonschema.Def{
	Description: `Glob patterns for directories to exclude from the watcher (set relative to WatchRoot).`,
	Items:       jsonschema.OptionalString(jsonschema.Def{}),
})

var ExcludeFiles_Schema = jsonschema.OptionalArray(jsonschema.Def{
	Description: `Glob patterns for files to exclude from the watcher (set relative to WatchRoot).`,
	Items:       jsonschema.OptionalString(jsonschema.Def{}),
})
