export {
	addBuildIDListener,
	addLocationListener,
	addRouteChangeListener,
	addStatusListener,
	applyScrollState,
	getBuildID,
	getHistoryInstance,
	getLocation,
	getPrefetchHandlers,
	getRootEl,
	getStatus,
	hmrRunClientLoaders,
	initClient,
	makeLinkOnClickFn,
	navigate,
	type RouteChangeEvent,
	revalidate,
	type StatusEvent,
	submit,
} from "./src/client.ts";
export {
	makeFinalLinkProps,
	type RiverLinkPropsBase,
	type RiverRootOutletPropsGeneric,
	type RiverRouteGeneric,
	type RiverRoutePropsGeneric,
	type RiverUntypedLoader,
	type UseRouterDataFunction,
} from "./src/impl_helpers.ts";
export { getRouterData, internal_RiverClientGlobal } from "./src/river_ctx.ts";
export type { RiverRoutes } from "./src/route_def_helpers.ts";
