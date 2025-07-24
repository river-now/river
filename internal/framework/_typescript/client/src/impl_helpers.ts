/////////////////////////////////////////////////////////////////////
/////// ROUTE COMPONENTS
/////////////////////////////////////////////////////////////////////

import { getPrefetchHandlers, makeLinkOnClickFn } from "./client.ts";
import { type getRouterData, internal_RiverClientGlobal } from "./river_ctx.ts";

export type RiverUntypedLoader = {
	_type: string;
	pattern: string;
	phantomOutputType: any;
	params: ReadonlyArray<string>;
};

export type RiverRoutePropsGeneric<
	JSXElement,
	T extends RiverUntypedLoader,
	Pattern extends T["pattern"] = T["pattern"],
> = {
	idx: number;
	Outlet: (props: Record<string, any>) => JSXElement;
	__phantom_pattern: Pattern;
} & Record<string, any>;

export type RiverRouteGeneric<
	JSXElement,
	T extends RiverUntypedLoader,
	Pattern extends T["pattern"] = T["pattern"],
> = (props: RiverRoutePropsGeneric<JSXElement, T, Pattern>) => JSXElement;

type ParamsForPattern<
	Loader extends RiverUntypedLoader,
	Pattern extends Loader["pattern"],
> = Extract<Loader, { pattern: Pattern }>["params"][number];

type BaseRouterData<RootData, Params extends string> = ReturnType<
	typeof getRouterData<RootData, Record<Params, string>>
>;

type Wrapper<UseAccessor extends boolean, T> = UseAccessor extends false
	? T
	: () => T;

export type UseRouterDataFunction<
	OuterLoader extends RiverUntypedLoader,
	RootData,
	UseAccessor extends boolean = false,
> = {
	<Pattern extends OuterLoader["pattern"]>(
		props: RiverRoutePropsGeneric<any, OuterLoader, Pattern>,
	): Wrapper<
		UseAccessor,
		BaseRouterData<RootData, ParamsForPattern<OuterLoader, Pattern>>
	>;
	<Pattern extends OuterLoader["pattern"]>(): Wrapper<
		UseAccessor,
		BaseRouterData<RootData, ParamsForPattern<OuterLoader, Pattern>>
	>;
	(): Wrapper<UseAccessor, BaseRouterData<RootData, string>>;
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
