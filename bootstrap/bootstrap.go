package bootstrap

import (
	_ "embed"
	"fmt"
	"runtime"
	"strings"

	"github.com/river-now/river"
	"github.com/river-now/river/kit/executil"
	"github.com/river-now/river/kit/fsutil"
)

// __TODO: add "docker" as a deployment target option

type Options struct {
	// e.g., "appname" or "modroot/apps/appname"
	GoImportBase string
	// "react", "preact", or "solid"
	UIVariant string
	// "npm", "pnpm", "yarn", or "bun"
	JSPackageManager string
	// "generic" or "vercel" (defaults to "generic")
	DeploymentTarget string
	IncludeTailwind  bool
	CreatedInDir     string // Empty if not created in a new directory
}

type derivedOptions struct {
	Options
	TSConfigJSXVal             string
	TSConfigJSXImportSourceVal string
	UIVitePlugin               string
	JSPackageManagerBaseCmd    string // "npx", "pnpm", "yarn", or "bunx"
	Call                       string
	VercelPackageJSONExtras    string
	TailwindViteImport         string
	TailwindViteCall           string
	TailwindFileImport         string
	DynamicLinkParamsProp      string
	BackgroundColorKey         string
	StylePropOpen              string // "{{"
	StylePropClose             string // "}}"
}

const tw_vite_import = "import tailwindcss from \"@tailwindcss/vite\";\n"
const tw_vite_call = ", tailwindcss()"
const tw_file_import = "import \"../styles/tailwind.css\";\n"
const dynamic_link_params_prop = `{{ id: "42790214" }}`

func (o Options) derived() derivedOptions {
	if o.UIVariant == "" {
		o.UIVariant = "react"
	}
	if o.JSPackageManager == "" {
		o.JSPackageManager = "npm"
	}

	do := derivedOptions{
		Options: o,
	}

	switch o.JSPackageManager {
	case "npm":
		do.JSPackageManagerBaseCmd = "npx"
	case "pnpm":
		do.JSPackageManagerBaseCmd = "pnpm"
	case "yarn":
		do.JSPackageManagerBaseCmd = "yarn"
	case "bun":
		do.JSPackageManagerBaseCmd = "bunx"
	}

	do.BackgroundColorKey = "backgroundColor"

	switch o.UIVariant {
	case "react":
		do.TSConfigJSXVal = "react-jsx"
		do.TSConfigJSXImportSourceVal = "react"
	case "solid":
		do.TSConfigJSXVal = "preserve"
		do.TSConfigJSXImportSourceVal = "solid-js"
		do.Call = "()"
		do.BackgroundColorKey = `"background-color"`
	case "preact":
		do.TSConfigJSXVal = "react-jsx"
		do.TSConfigJSXImportSourceVal = "preact"
	}

	if o.DeploymentTarget == "" {
		o.DeploymentTarget = "generic"
	}
	if o.DeploymentTarget != "generic" && o.DeploymentTarget != "vercel" {
		panic("unknown DeploymentTarget: " + o.DeploymentTarget)
	}
	goVersion := runtime.Version()
	if o.DeploymentTarget == "vercel" {
		do.VercelPackageJSONExtras = fmt.Sprintf(`,
		"vercel-install-go": "curl -L https://go.dev/dl/%s.linux-amd64.tar.gz | tar -C /tmp -xz",
		"vercel-install": "%s vercel-install-go && %s",
		"vercel-build": "export PATH=/tmp/go/bin:$PATH && go run ./control/cmd/build"`,
			goVersion, do.ResolveJSPackageManagerRunScriptPrefix(), do.ResolveJSPackageManagerInstallCmd(),
		)
	}

	do.UIVitePlugin = resolveUIVitePlugin(do)

	do.DynamicLinkParamsProp = dynamic_link_params_prop

	do.StylePropOpen = "{{"
	do.StylePropClose = "}}"

	if o.IncludeTailwind {
		do.TailwindViteImport = tw_vite_import
		do.TailwindViteCall = tw_vite_call
		do.TailwindFileImport = tw_file_import
	}

	return do
}

var (
	//go:embed tmpls/cmd_app_main_go_tmpl.txt
	cmd_app_main_go_tmpl_txt string
	//go:embed tmpls/cmd_build_main_go_tmpl.txt
	cmd_build_main_go_tmpl_txt string
	//go:embed tmpls/dist_static_keep_tmpl.txt
	dist_static_keep_tmpl_txt string
	//go:embed tmpls/backend_static_entry_go_html_str.txt
	backend_static_entry_go_html_str_txt string
	//go:embed tmpls/backend_router_actions_go_tmpl.txt
	backend_router_actions_go_tmpl_txt string
	//go:embed tmpls/backend_router_core_go_tmpl.txt
	backend_router_core_go_tmpl_txt string
	//go:embed tmpls/backend_router_loaders_go_tmpl.txt
	backend_router_loaders_go_tmpl_txt string
	//go:embed tmpls/app_go_tmpl.txt
	app_go_tmpl_txt string
	//go:embed tmpls/wave_config_json_tmpl.txt
	wave_config_json_tmpl_txt string
	//go:embed tmpls/vite_config_ts_tmpl.txt
	vite_config_ts_tmpl_txt string
	//go:embed tmpls/package_json_tmpl.txt
	package_json_tmpl_txt string
	//go:embed tmpls/gitignore_str.txt
	gitignore_str_txt string
	//go:embed tmpls/main_css_str.txt
	main_css_str_txt string
	//go:embed tmpls/main_critical_css_str.txt
	main_critical_css_str_txt string
	//go:embed tmpls/frontend_routes_ts_str.txt
	frontend_routes_ts_str_txt string
	//go:embed tmpls/frontend_entry_tsx_react_str.txt
	frontend_entry_tsx_react_str_txt string
	//go:embed tmpls/frontend_entry_tsx_solid_str.txt
	frontend_entry_tsx_solid_str_txt string
	//go:embed tmpls/frontend_entry_tsx_preact_str.txt
	frontend_entry_tsx_preact_str_txt string
	//go:embed tmpls/frontend_app_tsx_tmpl.txt
	frontend_app_tsx_tmpl_txt string
	//go:embed tmpls/frontend_root_tsx_tmpl.txt
	frontend_root_tsx_tmpl_txt string
	//go:embed tmpls/frontend_home_tsx_tmpl.txt
	frontend_home_tsx_tmpl_txt string
	//go:embed tmpls/frontend_about_tsx_tmpl.txt
	frontend_about_tsx_tmpl_txt string
	//go:embed tmpls/frontend_app_utils_tsx_tmpl.txt
	frontend_app_utils_tsx_tmpl_txt string
	//go:embed tmpls/frontend_api_client_ts_str.txt
	frontend_api_client_ts_str_txt string
	//go:embed tmpls/ts_config_json_tmpl.txt
	tsconfig_json_tmpl_txt string
	//go:embed tmpls/vercel_json_tmpl.txt
	vercel_json_tmpl_txt string
	//go:embed tmpls/api_proxy_ts_str.txt
	api_proxy_ts_str string
	//go:embed tmpls/frontend_css_tailwind_css_str.txt
	frontend_css_tailwind_css_str_txt string
	//go:embed tmpls/frontend_css_nprogress_css_str.txt
	frontend_css_nprogress_css_str_txt string
)

func Init(o Options) {
	if o.GoImportBase == "" {
		panic("GoImportBase must be set")
	}

	do := o.derived()

	fsutil.EnsureDirs(
		"assets/private",
		"assets/public",
		"app/server/router",
		"control/cmd/serve",
		"control/cmd/build",
		"control/dist/static/internal",
		"app/client/components",
		"app/client/styles",
	)

	if o.DeploymentTarget == "vercel" {
		fsutil.EnsureDirs("api")
	}

	do.tmplWriteMust("control/cmd/serve/main.go", cmd_app_main_go_tmpl_txt)
	do.tmplWriteMust("control/cmd/build/main.go", cmd_build_main_go_tmpl_txt)
	do.tmplWriteMust("control/dist/static/.keep", dist_static_keep_tmpl_txt)
	strWriteMust("assets/private/entry.go.html", backend_static_entry_go_html_str_txt)
	do.tmplWriteMust("app/server/router/actions.go", backend_router_actions_go_tmpl_txt)
	do.tmplWriteMust("app/server/router/core.go", backend_router_core_go_tmpl_txt)
	do.tmplWriteMust("app/server/router/loaders.go", backend_router_loaders_go_tmpl_txt)
	do.tmplWriteMust("control/river.config.go", app_go_tmpl_txt)
	do.tmplWriteMust("control/wave.config.json", wave_config_json_tmpl_txt)
	do.tmplWriteMust("vite.config.ts", vite_config_ts_tmpl_txt)
	do.tmplWriteMust("package.json", package_json_tmpl_txt)
	strWriteMust(".gitignore", gitignore_str_txt)
	strWriteMust("app/client/styles/main.css", main_css_str_txt)
	strWriteMust("app/client/styles/main.critical.css", main_critical_css_str_txt)
	strWriteMust("app/client/routes.ts", frontend_routes_ts_str_txt)
	do.tmplWriteMust("app/client/components/app.tsx", frontend_app_tsx_tmpl_txt)
	do.tmplWriteMust("app/client/components/root.tsx", frontend_root_tsx_tmpl_txt)
	do.tmplWriteMust("app/client/components/home.tsx", frontend_home_tsx_tmpl_txt)
	do.tmplWriteMust("app/client/components/about.tsx", frontend_about_tsx_tmpl_txt)
	do.tmplWriteMust("app/client/app_utils.tsx", frontend_app_utils_tsx_tmpl_txt)
	strWriteMust("app/client/api_client.ts", frontend_api_client_ts_str_txt)
	strWriteMust("app/client/styles/nprogress.css", frontend_css_nprogress_css_str_txt)
	if o.DeploymentTarget == "vercel" {
		do.tmplWriteMust("vercel.json", vercel_json_tmpl_txt)
		do.tmplWriteMust("api/proxy.ts", api_proxy_ts_str)
	}

	// last
	do.tmplWriteMust("tsconfig.json", tsconfig_json_tmpl_txt)

	installJSPkg(do, "typescript")
	installJSPkg(do, "vite")
	installJSPkg(do, fmt.Sprintf("river.now@%s", river.Internal__GetCurrentNPMVersion()))
	installJSPkg(do, resolveUIVitePlugin(do))
	installJSPkg(do, "nprogress")
	installJSPkg(do, "@types/nprogress")

	if do.UIVariant == "react" {
		strWriteMust("app/client/entry.tsx", frontend_entry_tsx_react_str_txt)

		installJSPkg(do, "react")
		installJSPkg(do, "react-dom")
		installJSPkg(do, "@types/react")
		installJSPkg(do, "@types/react-dom")
		installJSPkg(do, "jotai")
	}

	if do.UIVariant == "solid" {
		strWriteMust("app/client/entry.tsx", frontend_entry_tsx_solid_str_txt)

		installJSPkg(do, "solid-js")
	}

	if do.UIVariant == "preact" {
		strWriteMust("app/client/entry.tsx", frontend_entry_tsx_preact_str_txt)

		installJSPkg(do, "preact")
		installJSPkg(do, "@preact/signals")
	}

	if do.DeploymentTarget == "vercel" {
		installJSPkg(do, "@vercel/node")
	}

	if do.IncludeTailwind {
		installJSPkg(do, "@tailwindcss/vite")
		installJSPkg(do, "tailwindcss")
		strWriteMust("app/client/styles/tailwind.css", frontend_css_tailwind_css_str_txt)
	}

	// install chi (some chi middleware is used in the template)
	if err := executil.RunCmd("go", "get", "github.com/go-chi/chi/v5/middleware"); err != nil {
		panic("failed to install chi middleware: " + err.Error())
	}

	// build once (no binary)
	if err := executil.RunCmd("go", "run", "./control/cmd/build", "--no-binary"); err != nil {
		panic("failed to run build command: " + err.Error())
	}

	// tidy go modules (must come after so chi mw stays installed)
	if err := executil.RunCmd("go", "mod", "tidy"); err != nil {
		panic("failed to tidy go modules: " + err.Error())
	}

	fmt.Println()
	fmt.Println("✨ SUCCESS! Your River app is ready.")
	fmt.Println()
	if o.CreatedInDir != "" {
		fmt.Printf("💻 Run `cd %s && %s dev` to start the development server.\n",
			o.CreatedInDir,
			do.ResolveJSPackageManagerRunScriptPrefix(),
		)
	} else {
		fmt.Printf("💻 Run `%s dev` to start the development server.\n",
			do.ResolveJSPackageManagerRunScriptPrefix(),
		)
	}
	fmt.Println()
}

func (do derivedOptions) ResolveJSPackageManagerRunScriptPrefix() string {
	cmd := "npm run"
	if do.JSPackageManager != "npm" {
		cmd = do.JSPackageManager
	}
	return cmd
}

func (do derivedOptions) ResolveJSPackageManagerInstallCmd() string {
	pm := do.JSPackageManager
	switch pm {
	case "npm":
		return "npm i"
	case "pnpm":
		return "pnpm i"
	case "yarn":
		return "yarn"
	case "bun":
		return "bun i"
	}
	panic("unknown JSPackageManager: " + pm)
}

func installJSPkg(do derivedOptions, pkg string) {
	var cmd string

	switch do.JSPackageManager {
	case "npm":
		cmd = "npm i -D"
	case "pnpm":
		cmd = "pnpm add -D"
	case "yarn":
		cmd = "yarn add -D"
	case "bun":
		cmd = "bun add -d"
	}

	cmd += " " + pkg

	split := strings.Split(cmd, " ")
	err := executil.RunCmd(split...)
	if err != nil {
		panic("failed to install JS package: " + pkg + ": " + err.Error())
	}
}

func resolveUIVitePlugin(do derivedOptions) string {
	switch do.UIVariant {
	case "":
		panic("UIVariant must be set")
	case "react":
		return "@vitejs/plugin-react-swc"
	case "solid":
		return "vite-plugin-solid"
	case "preact":
		return "@preact/preset-vite"
	}
	panic("unknown UI variant: " + do.UIVariant)
}
