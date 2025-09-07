import {
	getPrefetchHandlers,
	makeLinkOnClickFn,
	riverNavigate,
} from "./client.ts";
import {
	resolvePath,
	type ExtractApp,
	type PatternBasedProps,
	type RiverAppBase,
	type RiverAppConfig,
	type RiverLoaderPattern,
	type RiverRouteParams,
} from "./river_app_helpers.ts";
import { internal_RiverClientGlobal, type getRouterData } from "./river_ctx.ts";

/////////////////////////////////////////////////////////////////////
/////// ROUTE COMPONENTS
/////////////////////////////////////////////////////////////////////

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

/////////////////////////////////////////////////////////////////////
/////// LINK COMPONENTS
/////////////////////////////////////////////////////////////////////

export type RiverLinkPropsBase<LinkOnClickCallback> = {
	href?: string;
	prefetch?: "intent";
	prefetchDelayMs?: number;
	beforeBegin?: LinkOnClickCallback;
	beforeRender?: LinkOnClickCallback;
	afterRender?: LinkOnClickCallback;
	scrollToTop?: boolean;
	replace?: boolean;
};

function linkPropsToPrefetchObj<LinkOnClickCallback>(
	props: RiverLinkPropsBase<LinkOnClickCallback>,
) {
	if (!props.href || props.prefetch !== "intent") {
		return undefined;
	}

	return getPrefetchHandlers({
		href: props.href,
		delayMs: props.prefetchDelayMs,
		beforeBegin: props.beforeBegin as any,
		beforeRender: props.beforeRender as any,
		afterRender: props.afterRender as any,
		scrollToTop: props.scrollToTop,
		replace: props.replace,
	});
}

function linkPropsToOnClickFn<LinkOnClickCallback>(
	props: RiverLinkPropsBase<LinkOnClickCallback>,
) {
	return makeLinkOnClickFn({
		beforeBegin: props.beforeBegin as any,
		beforeRender: props.beforeRender as any,
		afterRender: props.afterRender as any,
		scrollToTop: props.scrollToTop,
		replace: props.replace,
	});
}

type handlerKeys = {
	onPointerEnter: string;
	onFocus: string;
	onPointerLeave: string;
	onBlur: string;
	onTouchCancel: string;
	onClick: string;
};

const standardCamelHandlerKeys = {
	onPointerEnter: "onPointerEnter",
	onFocus: "onFocus",
	onPointerLeave: "onPointerLeave",
	onBlur: "onBlur",
	onTouchCancel: "onTouchCancel",
	onClick: "onClick",
} satisfies handlerKeys;

export function makeFinalLinkProps<LinkOnClickCallback>(
	props: RiverLinkPropsBase<LinkOnClickCallback>,
	keys: {
		onPointerEnter: string;
		onFocus: string;
		onPointerLeave: string;
		onBlur: string;
		onTouchCancel: string;
		onClick: string;
	} = standardCamelHandlerKeys,
) {
	const prefetchObj = linkPropsToPrefetchObj(props);

	return {
		dataExternal: prefetchObj?.isExternal || undefined,
		onPointerEnter: (e: any) => {
			prefetchObj?.start(e);
			if (isFn((props as any)[keys.onPointerEnter])) {
				(props as any)[keys.onPointerEnter](e);
			}
		},
		onFocus: (e: any) => {
			prefetchObj?.start(e);
			if (isFn((props as any)[keys.onFocus])) {
				(props as any)[keys.onFocus](e);
			}
		},
		onPointerLeave: (e: any) => {
			// we don't want to stop on a touch device, because this triggers
			// even when the user "clicks" on the link for some reason
			if (!internal_RiverClientGlobal.get("isTouchDevice")) {
				prefetchObj?.stop();
			}
			if (isFn((props as any)[keys.onPointerLeave])) {
				(props as any)[keys.onPointerLeave](e);
			}
		},
		onBlur: (e: any) => {
			prefetchObj?.stop();
			if (isFn((props as any)[keys.onBlur])) {
				(props as any)[keys.onBlur](e);
			}
		},
		onTouchCancel: (e: any) => {
			prefetchObj?.stop();
			if (isFn((props as any)[keys.onTouchCancel])) {
				(props as any)[keys.onTouchCancel](e);
			}
		},
		onClick: async (e: any) => {
			if (isFn((props as any)[keys.onClick])) {
				(props as any)[keys.onClick](e);
			}
			if (prefetchObj) {
				await prefetchObj.onClick(e);
			} else {
				await linkPropsToOnClickFn(props)(e);
			}
		},
	};
}

function isFn(fn: any): fn is (...args: Array<any>) => any {
	return typeof fn === "function";
}

/////////////////////////////////////////////////////////////////////
/////// NAVIGATION
/////////////////////////////////////////////////////////////////////

type TypedNavigateOptions<
	App extends RiverAppBase,
	Pattern extends RiverLoaderPattern<App>,
> = PatternBasedProps<App, Pattern> & {
	replace?: boolean;
	scrollToTop?: boolean;
};

export function makeTypedNavigate<C extends RiverAppConfig>(riverAppConfig: C) {
	type App = ExtractApp<C>;

	return async function typedNavigate<
		Pattern extends RiverLoaderPattern<App>,
	>(options: TypedNavigateOptions<App, Pattern>): Promise<void> {
		const { pattern, params, splatValues, replace, scrollToTop } =
			options as any;

		const href = resolvePath({
			riverAppConfig,
			type: "loader",
			props: {
				pattern,
				...(params && { params }),
				...(splatValues && { splatValues }),
			},
		});

		return riverNavigate(href, { replace, scrollToTop });
	};
}
