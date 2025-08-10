package bootstrap

import (
	_ "embed"
	"fmt"
	"runtime"
	"strings"

	"github.com/river-now/river/kit/executil"
	"github.com/river-now/river/kit/fsutil"
)

// __TODO: add "docker" as a deployment target option

type Options struct {
	// e.g., "appname" or "modroot/apps/appname"
	GoImportBase string
	// "react", "solid", or "preact"
	UIVariant string
	// "npm", "pnpm", "yarn", or "bun"
	JSPackageManager string
	// "generic" or "vercel" (defaults to "generic")
	DeploymentTarget string
}

type derivedOptions struct {
	Options
	TSConfigJSXVal             string
	TSConfigJSXImportSourceVal string
	UIVitePlugin               string
	JSPackageManagerBaseCmd    string // "npx", "pnpm", "yarn", or "bunx"
	Accessor                   string
	VercelPackageJSONExtras    string
}

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

	switch o.UIVariant {
	case "react":
		do.TSConfigJSXVal = "react-jsx"
		do.TSConfigJSXImportSourceVal = "react"
	case "solid":
		do.TSConfigJSXVal = "preserve"
		do.TSConfigJSXImportSourceVal = "solid-js"
		do.Accessor = "()"
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
		"vercel-install": "%s vercel-install-go && export PATH=/tmp/go/bin:$PATH && %s install",
		"vercel-build": "export PATH=/tmp/go/bin:$PATH && go run ./__cmd/build --no-binary"`,
			goVersion, do.ResolveJSPackageManagerRunScriptPrefix(), do.JSPackageManager,
		)
	}

	do.UIVitePlugin = resolveUIVitePlugin(do)

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
	//go:embed tmpls/backend_server_server_go_tmpl.txt
	backend_server_server_go_tmpl_txt string
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
	//go:embed tmpls/frontend_home_tsx_tmpl.txt
	frontend_home_tsx_tmpl_txt string
	//go:embed tmpls/frontend_app_utils_ts_tmpl.txt
	frontend_app_tsx_utils_tmpl_txt string
	//go:embed tmpls/frontend_api_client_ts_str.txt
	frontend_api_client_ts_str_txt string
	//go:embed tmpls/ts_config_json_tmpl.txt
	tsconfig_json_tmpl_txt string
	//go:embed tmpls/vercel_json_tmpl.txt
	vercel_json_tmpl_txt string
	//go:embed tmpls/api_index_go_tmpl.txt
	api_index_go_tmpl_txt string
)

func Init(o Options) {
	if o.GoImportBase == "" {
		panic("GoImportBase must be set")
	}

	do := o.derived()

	fsutil.EnsureDirs(
		"__cmd/app", "__cmd/build",
		"app/__dist/static/internal",
		"backend/__static", "backend/router", "backend/server",
		"frontend/__static", "frontend/css",
	)

	if o.DeploymentTarget == "vercel" {
		fsutil.EnsureDirs("api")
	}

	do.tmplWriteMust("__cmd/app/main.go", cmd_app_main_go_tmpl_txt)
	do.tmplWriteMust("__cmd/build/main.go", cmd_build_main_go_tmpl_txt)
	do.tmplWriteMust("app/__dist/static/.keep", dist_static_keep_tmpl_txt)
	strWriteMust("backend/__static/entry.go.html", backend_static_entry_go_html_str_txt)
	do.tmplWriteMust("backend/router/actions.go", backend_router_actions_go_tmpl_txt)
	do.tmplWriteMust("backend/router/core.go", backend_router_core_go_tmpl_txt)
	do.tmplWriteMust("backend/router/loaders.go", backend_router_loaders_go_tmpl_txt)
	do.tmplWriteMust("backend/server/server.go", backend_server_server_go_tmpl_txt)
	do.tmplWriteMust("app/app.go", app_go_tmpl_txt)
	do.tmplWriteMust("app/wave.config.json", wave_config_json_tmpl_txt)
	do.tmplWriteMust("vite.config.ts", vite_config_ts_tmpl_txt)
	do.tmplWriteMust("package.json", package_json_tmpl_txt)
	strWriteMust(".gitignore", gitignore_str_txt)
	strWriteMust("frontend/css/main.css", main_css_str_txt)
	strWriteMust("frontend/css/main.critical.css", main_critical_css_str_txt)
	strWriteMust("frontend/routes.ts", frontend_routes_ts_str_txt)
	do.tmplWriteMust("frontend/app.tsx", frontend_app_tsx_tmpl_txt)
	do.tmplWriteMust("frontend/home.tsx", frontend_home_tsx_tmpl_txt)
	do.tmplWriteMust("frontend/app_utils.ts", frontend_app_tsx_utils_tmpl_txt)
	strWriteMust("frontend/api_client.ts", frontend_api_client_ts_str_txt)
	if o.DeploymentTarget == "vercel" {
		do.tmplWriteMust("vercel.json", vercel_json_tmpl_txt)
		do.tmplWriteMust("api/index.go", api_index_go_tmpl_txt)
	}

	// last
	do.tmplWriteMust("tsconfig.json", tsconfig_json_tmpl_txt)

	installJSPkg(do, "typescript")
	installJSPkg(do, "vite")
	installJSPkg(do, "river.now")
	installJSPkg(do, resolveUIVitePlugin(do))

	if do.UIVariant == "react" {
		strWriteMust("frontend/entry.tsx", frontend_entry_tsx_react_str_txt)

		installJSPkg(do, "react")
		installJSPkg(do, "react-dom")
		installJSPkg(do, "@types/react")
		installJSPkg(do, "@types/react-dom")
		installJSPkg(do, "jotai")
	}

	if do.UIVariant == "solid" {
		strWriteMust("frontend/entry.tsx", frontend_entry_tsx_solid_str_txt)

		installJSPkg(do, "solid-js")
	}

	if do.UIVariant == "preact" {
		strWriteMust("frontend/entry.tsx", frontend_entry_tsx_preact_str_txt)

		installJSPkg(do, "preact")
		installJSPkg(do, "@preact/signals")
	}

	// install chi (some chi middleware is used in the template)
	if err := executil.RunCmd("go", "get", "github.com/go-chi/chi/v5/middleware"); err != nil {
		panic("failed to install chi middleware: " + err.Error())
	}

	// build once (no binary)
	if err := executil.RunCmd("go", "run", "./__cmd/build", "--no-binary"); err != nil {
		panic("failed to run build command: " + err.Error())
	}

	// tidy go modules (must come after so chi mw stays installed)
	if err := executil.RunCmd("go", "mod", "tidy"); err != nil {
		panic("failed to tidy go modules: " + err.Error())
	}

	fmt.Println()
	fmt.Println("✨ SUCCESS! Your River app is ready.")
	fmt.Println()
	runCmd := do.ResolveJSPackageManagerRunScriptPrefix() + " dev"
	fmt.Printf("💻 Run `%s` to start the development server.\n", runCmd)
	fmt.Println()
}

func (do derivedOptions) ResolveJSPackageManagerRunScriptPrefix() string {
	cmd := "npm run"
	if do.JSPackageManager != "npm" {
		cmd = do.JSPackageManager
	}
	return cmd
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
