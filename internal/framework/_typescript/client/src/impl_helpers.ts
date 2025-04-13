/////////////////////////////////////////////////////////////////////
/////// ROUTE COMPONENTS
/////////////////////////////////////////////////////////////////////

import { getPrefetchHandlers, makeLinkClickListenerFn } from "./client.ts";

export type RiverUntypedLoader = { _type: string; pattern: string; phantomOutputType: any };

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

export type RiverRootOutletPropsGeneric<JSXElement> = {
	idx?: number;
	defaultServerErrorComponent?: () => JSXElement;
};

/////////////////////////////////////////////////////////////////////
/////// LINK COMPONENTS
/////////////////////////////////////////////////////////////////////

// __TODO add prefetch = "render" and prefetch = "viewport" options, a la Remix
// __TODO don't link prefetch where you already are -- eg make an exception for window.location.pathname

export type RiverLinkPropsBase<LinkClickCallback> = {
	href?: string;
	prefetch?: "intent";
	prefetchTimeout?: number;
	beforeBegin?: LinkClickCallback;
	beforeRender?: LinkClickCallback;
	afterRender?: LinkClickCallback;
} & Record<string, any>;

function linkPropsToPrefetchObj<LinkClickCallback>(props: RiverLinkPropsBase<LinkClickCallback>) {
	if (!props.href || props.prefetch !== "intent") {
		return undefined;
	}

	return getPrefetchHandlers({
		href: props.href,
		timeout: props.prefetchTimeout,
		beforeBegin: props.beforeBegin as any,
		beforeRender: props.beforeRender as any,
		afterRender: props.afterRender as any,
	});
}

function linkPropsToClickListenerFn<LinkClickCallback>(
	props: RiverLinkPropsBase<LinkClickCallback>,
) {
	return makeLinkClickListenerFn({
		beforeBegin: props.beforeBegin as any,
		beforeRender: props.beforeRender as any,
		afterRender: props.afterRender as any,
		requireDataBoostAttribute: false,
	});
}

type handlerKeys = {
	onPointerEnter: string;
	onFocus: string;
	onTouchStart: string;
	onPointerLeave: string;
	onBlur: string;
	onTouchCancel: string;
	onClick: string;
};

const standardCamelHandlerKeys = {
	onPointerEnter: "onPointerEnter",
	onFocus: "onFocus",
	onTouchStart: "onTouchStart",
	onPointerLeave: "onPointerLeave",
	onBlur: "onBlur",
	onTouchCancel: "onTouchCancel",
	onClick: "onClick",
} satisfies handlerKeys;

export function makeFinalLinkProps<LinkClickCallback>(
	props: RiverLinkPropsBase<LinkClickCallback>,
	keys: {
		onPointerEnter: string;
		onFocus: string;
		onTouchStart: string;
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
			if (isFn(props[keys.onPointerEnter])) props[keys.onPointerEnter](e);
		},
		onFocus: (e: any) => {
			prefetchObj?.start(e);
			if (isFn(props[keys.onFocus])) props[keys.onFocus](e);
		},
		onTouchStart: (e: any) => {
			prefetchObj?.start(e);
			if (isFn(props[keys.onTouchStart])) props[keys.onTouchStart](e);
		},
		onPointerLeave: (e: any) => {
			prefetchObj?.stop();
			if (isFn(props[keys.onPointerLeave])) props[keys.onPointerLeave](e);
		},
		onBlur: (e: any) => {
			prefetchObj?.stop();
			if (isFn(props[keys.onBlur])) props[keys.onBlur](e);
		},
		onTouchCancel: (e: any) => {
			prefetchObj?.stop();
			if (isFn(props[keys.onTouchCancel])) props[keys.onTouchCancel](e);
		},
		onClick: async (e: any) => {
			if (isFn(props[keys.onClick])) props[keys.onClick](e);
			if (prefetchObj) {
				await prefetchObj.onClick(e);
			} else {
				await linkPropsToClickListenerFn(props)(e);
			}
		},
	};
}

function isFn(fn: any): fn is (...args: any[]) => any {
	return typeof fn === "function";
}
