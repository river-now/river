/////////////////////////////////////////////////////////////////////
/////// ROUTE COMPONENTS
/////////////////////////////////////////////////////////////////////

import { getPrefetchHandlers, makeLinkOnClickFn, type ScrollState } from "./client.ts";
import { internal_RiverClientGlobal } from "./river_ctx.ts";

export type RiverUntypedLoader = {
	_type: string;
	pattern: string;
	phantomOutputType: any;
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

export type RiverRootOutletPropsGeneric<JSXElement> = {
	idx?: number;
	defaultServerErrorComponent?: () => JSXElement;
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
} & Record<string, any>;

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
	});
}

function linkPropsToOnClickFn<LinkOnClickCallback>(
	props: RiverLinkPropsBase<LinkOnClickCallback>,
) {
	return makeLinkOnClickFn({
		beforeBegin: props.beforeBegin as any,
		beforeRender: props.beforeRender as any,
		afterRender: props.afterRender as any,
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
			if (isFn(props[keys.onPointerEnter])) {
				props[keys.onPointerEnter](e);
			}
		},
		onFocus: (e: any) => {
			prefetchObj?.start(e);
			if (isFn(props[keys.onFocus])) {
				props[keys.onFocus](e);
			}
		},
		onPointerLeave: (e: any) => {
			// we don't want to stop on a touch device, because this triggers
			// even when the user "clicks" on the link for some reason
			if (!internal_RiverClientGlobal.get("isTouchDevice")) {
				prefetchObj?.stop();
			}
			if (isFn(props[keys.onPointerLeave])) {
				props[keys.onPointerLeave](e);
			}
		},
		onBlur: (e: any) => {
			prefetchObj?.stop();
			if (isFn(props[keys.onBlur])) {
				props[keys.onBlur](e);
			}
		},
		onTouchCancel: (e: any) => {
			prefetchObj?.stop();
			if (isFn(props[keys.onTouchCancel])) {
				props[keys.onTouchCancel](e);
			}
		},
		onClick: async (e: any) => {
			if (isFn(props[keys.onClick])) {
				props[keys.onClick](e);
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
