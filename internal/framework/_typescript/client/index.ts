export {
	__applyScrollState,
	__getPrefetchHandlers,
	__makeLinkOnClickFn,
	__registerClientLoaderPattern,
	addBuildIDListener,
	addLocationListener,
	addRouteChangeListener,
	addStatusListener,
	getBuildID,
	getHistoryInstance,
	getLocation,
	getRootEl,
	getStatus,
	initClient,
	revalidate,
	riverNavigate,
	submit,
	type RouteChangeEvent,
	type StatusEvent,
	type SubmitOptions,
} from "./src/client.ts";
export {
	revalidateOnWindowFocus,
	setupGlobalLoadingIndicator,
} from "./src/global_loading_indicator.ts";
export { hmrRunClientLoaders } from "./src/hmr.ts";
export {
	__resolvePath,
	buildMutationURL,
	buildQueryURL,
	type ExtractApp,
	type PermissivePatternBasedProps,
	type RiverAppBase,
	type RiverAppConfig,
	type RiverLoaderOutput,
	type RiverLoaderPattern,
	type RiverMutationInput,
	type RiverMutationOutput,
	type RiverMutationPattern,
	type RiverMutationProps,
	type RiverQueryInput,
	type RiverQueryOutput,
	type RiverQueryPattern,
	type RiverQueryProps,
	type RiverRoutePropsGeneric,
} from "./src/river_app_helpers.ts";
export {
	__riverClientGlobal,
	getRouterData,
	type ClientLoaderAwaitedServerData,
} from "./src/river_ctx.ts";
export type { RiverRoutes } from "./src/route_def_helpers.ts";
export {
	__makeFinalLinkProps,
	makeTypedNavigate,
	type ParamsForPattern,
	type RiverLinkPropsBase,
	type RiverRouteGeneric,
	type UseRouterDataFunction,
} from "./src/ui_lib_impl_helpers.ts";
