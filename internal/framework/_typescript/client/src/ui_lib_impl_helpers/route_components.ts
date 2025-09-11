import {
	__resolvePath,
	type RiverAppBase,
	type RiverLoaderPattern,
	type RiverRouteParams,
} from "../river_app_helpers/river_app_helpers.ts";
import {
	__riverClientGlobal,
	type getRouterData,
} from "../river_ctx/river_ctx.ts";

export type RiverRoutePropsGeneric<
	JSXElement,
	App extends RiverAppBase,
	Pattern extends RiverLoaderPattern<App> = RiverLoaderPattern<App>,
> = {
	idx: number;
	Outlet: (props: Record<string, any>) => JSXElement;
	__phantom_pattern: Pattern;
} & Record<string, any>;

export type RiverRouteGeneric<
	JSXElement,
	App extends RiverAppBase,
	Pattern extends RiverLoaderPattern<App> = RiverLoaderPattern<App>,
> = (props: RiverRoutePropsGeneric<JSXElement, App, Pattern>) => JSXElement;

export type ParamsForPattern<
	App extends RiverAppBase,
	Pattern extends RiverLoaderPattern<App>,
> = RiverRouteParams<App, Pattern>;

type BaseRouterData<RootData, Params extends string> = ReturnType<
	typeof getRouterData<RootData, Record<Params, string>>
>;

type Wrapper<UseAccessor extends boolean, T> = UseAccessor extends false
	? T
	: () => T;

export type UseRouterDataFunction<
	App extends RiverAppBase,
	UseAccessor extends boolean = false,
> = {
	<Pattern extends RiverLoaderPattern<App>>(
		props: RiverRoutePropsGeneric<any, App, Pattern>,
	): Wrapper<
		UseAccessor,
		BaseRouterData<App["rootData"], ParamsForPattern<App, Pattern>>
	>;
	<Pattern extends RiverLoaderPattern<App>>(): Wrapper<
		UseAccessor,
		BaseRouterData<App["rootData"], ParamsForPattern<App, Pattern>>
	>;
	(): Wrapper<UseAccessor, BaseRouterData<App["rootData"], string>>;
};
