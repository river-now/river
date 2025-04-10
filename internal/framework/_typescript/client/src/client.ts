/// <reference types="vite/client" />

import {
	getAnchorDetailsFromEvent,
	getHrefDetails,
	getIsErrorRes,
	getIsGETRequest,
} from "@sjc5/river/kit/url";
import { createBrowserHistory, type Update } from "history";
import { parseFetchResponseForRedirectData, type RedirectData } from "./redirects.ts";
import {
	type GetRouteDataOutput,
	type HeadBlock,
	internal_RiverClientGlobal,
	type RiverClientGlobal,
} from "./river_ctx.ts";
import { isAbortError, LogError, LogInfo, Panic } from "./utils.ts";

if (import.meta.env.MODE === "development") {
	(window as any).__kirunaRevalidate = devRevalidate;
}

/////////////////////////////////////////////////////////////////////
// COMMON
/////////////////////////////////////////////////////////////////////

const RIVER_ROUTE_CHANGE_EVENT_KEY = "river:route-change";

type ScrollState = { x: number; y: number };
type RouteChangeEventDetail = {
	scrollState?: ScrollState;
	index?: number;
};
export type RouteChangeEvent = CustomEvent<RouteChangeEventDetail>;

/////////////////////////////////////////////////////////////////////
// NAVIGATION TYPES AND GLOBAL STATE
/////////////////////////////////////////////////////////////////////

type NavigationResult =
	| {
			response: Response;
			json: GetRouteDataOutput;
			props: NavigateProps;
			cssBundlePromises: Array<any>;
	  }
	| { response: Response; redirectData: RedirectData }
	| undefined;

export type NavigationControl = {
	abortController: AbortController | undefined;
	promise: Promise<NavigationResult>;
};

type NavigationType =
	| "browserHistory"
	| "userNavigation"
	| "revalidation"
	| "dev-revalidation"
	| "redirect"
	| "prefetch";

export type NavigateProps = {
	href: string;
	navigationType: NavigationType;
	scrollStateToRestore?: ScrollState;
	replace?: boolean;
};

export const navigationState = {
	navigations: new Map<
		string,
		{
			control: NavigationControl;
			type: NavigationType;
		}
	>(),
	activeUserNavigation: null as string | null,
	submissions: new Map<
		string,
		{
			controller: AbortController;
			type: "submission";
		}
	>(),
};

/////////////////////////////////////////////////////////////////////
// NAVIGATION UTILS
/////////////////////////////////////////////////////////////////////

export async function __navigate(props: NavigateProps) {
	const x = beginNavigation(props);
	if (!x.promise) {
		return;
	}
	const res = await x.promise;
	if (!res) {
		return;
	}
	await __completeNavigation(res);
}

export function beginNavigation(props: NavigateProps): NavigationControl {
	setLoadingStatus({ type: props.navigationType, value: true });

	// If this is a user navigation, abort any existing user navigation
	if (props.navigationType === "userNavigation") {
		// Abort all other navigations
		abortAllNavigationsExcept(props.href);
		navigationState.activeUserNavigation = props.href;

		// Check if we have an existing prefetch we can upgrade
		const existing = navigationState.navigations.get(props.href);
		if (existing && existing.type === "prefetch") {
			existing.type = "userNavigation";
			return existing.control;
		}
	}

	// For prefetches, check if one already exists
	if (props.navigationType === "prefetch") {
		const existing = navigationState.navigations.get(props.href);
		if (existing) {
			return existing.control;
		}
	}

	const controller = new AbortController();
	const control: NavigationControl = {
		abortController: controller,
		promise: __fetchRouteData(controller, props),
	};

	navigationState.navigations.set(props.href, {
		control,
		type: props.navigationType,
	});

	return control;
}

async function __completeNavigation(x: NavigationResult) {
	if (!x) return;
	if ("redirectData" in x) {
		await effectuateRedirectDataResult(x.redirectData);
		return;
	}
	const oldID = internal_RiverClientGlobal.get("buildID");
	const newID = x.response.headers.get(buildIDHeader) || "";
	if (newID && newID !== oldID) {
		dispatchBuildIDEvent({ newID, oldID, fromGETAction: false });
	}
	try {
		await __reRenderApp({
			json: x.json,
			navigationType: x.props.navigationType,
			runHistoryOptions: x.props,
			cssBundlePromises: x.cssBundlePromises,
		});
		setLoadingStatus({ type: x.props.navigationType, value: false });
	} catch (error) {
		handleNavError(error, x.props);
	}
}

function getMaybeNewPreloadLink(x: string) {
	const href = resolvePublicURL(x).href;
	const existing: HTMLLinkElement | null = document.querySelector(
		`link[href^="${href.split("?")[0]}"]`, // without search params
	);
	if (existing) {
		if (existing.href !== href) {
			existing.remove();
		} else {
			return undefined;
		}
	}
	const newLink = document.createElement("link");
	newLink.href = href;
	return newLink;
}

async function __fetchRouteData(
	controller: AbortController,
	props: NavigateProps,
): Promise<NavigationResult | undefined> {
	try {
		const url = new URL(props.href, window.location.href);
		url.searchParams.set("river-json", "1");

		const { redirectData, response } = await handleRedirects({
			abortController: controller,
			url,
			isPrefetch: props.navigationType === "prefetch",
		});

		const redirected = redirectData?.status === "did";
		const responseNotOK = !response?.ok && response?.status !== 304;
		if (redirected || !response || responseNotOK) {
			setLoadingStatus({ type: props.navigationType, value: false });
			return;
		}

		if (redirectData?.status === "should") {
			return { redirectData, response };
		}

		const json = (await response.json()) as GetRouteDataOutput | undefined;
		if (!json) throw new Error("No JSON response");

		// deps are only present in prod because they stem from the rollup metafile
		// (same for CSS bundles -- vite handles them in dev)
		// so in dev, to get similar behavior, we use the importURLs
		// (which is a subset of what the deps would be in prod)
		const depsToPreload = import.meta.env.DEV ? [...new Set(json.importURLs)] : json.deps;

		// Add missing deps modulepreload links
		for (const x of depsToPreload ?? []) {
			const newLink = getMaybeNewPreloadLink(x);
			if (!newLink) continue;
			newLink.rel = "modulepreload";
			document.head.appendChild(newLink);
		}

		// Create an array to store promises for CSS bundle preloads
		const cssBundlePromises = [];

		// Add missing css bundle preload links
		for (const x of json.cssBundles ?? []) {
			const newLink = getMaybeNewPreloadLink(x);
			if (!newLink) continue;
			newLink.rel = "preload";
			newLink.as = "style";
			document.head.appendChild(newLink);

			// Create a promise for this CSS bundle preload
			const preloadPromise = new Promise((resolve, reject) => {
				newLink.onload = resolve;
				newLink.onerror = reject;
			});
			cssBundlePromises.push(preloadPromise);
		}

		return { response, json, props, cssBundlePromises };
	} catch (error) {
		if (!isAbortError(error)) {
			LogError("Navigation failed", error);
			setLoadingStatus({ type: props.navigationType, value: false });
		}
	} finally {
		navigationState.navigations.delete(props.href);
		if (navigationState.activeUserNavigation === props.href) {
			navigationState.activeUserNavigation = null;
		}
	}
}

function abortNavigation(href: string) {
	const nav = navigationState.navigations.get(href);
	if (nav) {
		nav.control.abortController?.abort();
		navigationState.navigations.delete(href);
	}
}

function abortAllNavigationsExcept(excludeHref?: string) {
	for (const [href, nav] of navigationState.navigations.entries()) {
		if (href !== excludeHref) {
			nav.control.abortController?.abort();
			navigationState.navigations.delete(href);
		}
	}
}

function handleNavError(error: unknown, props: NavigateProps) {
	if (!isAbortError(error)) {
		LogError(error);
		setLoadingStatus({ type: props.navigationType, value: false });
	}
}

/////////////////////////////////////////////////////////////////////
// PREFETCH
/////////////////////////////////////////////////////////////////////

type GetPrefetchHandlersInput<E extends Event> = LinkClickListenerCallbacksBase<E> & {
	href: string;
	timeout?: number;
};

export function getPrefetchHandlers<E extends Event>(input: GetPrefetchHandlersInput<E>) {
	const hrefDetails = getHrefDetails(input.href);
	if (!hrefDetails.isHTTP || !hrefDetails.relativeURL || hrefDetails.isExternal) {
		return;
	}

	let timer: number | undefined;
	let currentNav: NavigationControl | null = null;
	let prerenderResult: NavigationResult | null = null;

	async function finalize(e: E) {
		try {
			if (!prerenderResult && currentNav) {
				prerenderResult = await currentNav.promise;
			}
			if (prerenderResult) {
				await input.beforeRender?.(e);

				if ("redirectData" in prerenderResult) {
					await effectuateRedirectDataResult(prerenderResult.redirectData);
					return;
				}

				if (!("json" in prerenderResult)) {
					throw new Error("No JSON response");
				}

				await __completeNavigation({
					response: prerenderResult.response,
					json: prerenderResult.json,
					props: { ...prerenderResult.props, navigationType: "userNavigation" },
					cssBundlePromises: prerenderResult.cssBundlePromises,
				});

				await input.afterRender?.(e);
			}
		} catch (e) {
			if (!isAbortError(e)) {
				LogError("Error finalizing prefetch", e);
			}
		} finally {
			prerenderResult = null;
			currentNav = null;
		}
	}

	async function prefetch(e: E) {
		if (currentNav || !hrefDetails.isHTTP) {
			return;
		}

		await input.beforeBegin?.(e);

		currentNav = beginNavigation({
			href: hrefDetails.relativeURL,
			navigationType: "prefetch",
		});

		currentNav.promise
			.then((result) => {
				prerenderResult = result;
			})
			.catch((error) => {
				if (!isAbortError(error)) {
					LogError("Prefetch failed", error);
				}
			});
	}

	function start(e: E) {
		if (currentNav) {
			return;
		}
		timer = window.setTimeout(() => prefetch(e), input.timeout);
	}

	function stop() {
		clearTimeout(timer);

		if (!hrefDetails.isHTTP) {
			return;
		}

		// Only abort if it's a prefetch, not a user navigation
		const nav = navigationState.navigations.get(hrefDetails.relativeURL);
		if (nav?.type === "prefetch") {
			abortNavigation(hrefDetails.relativeURL);
		}

		// Ensure future prefetches can occur
		currentNav = null;
		prerenderResult = null;
	}

	async function onClick(e: E) {
		if (e.defaultPrevented || !hrefDetails.isHTTP) {
			return;
		}

		e.preventDefault();
		setLoadingStatus({ type: "userNavigation", value: true });

		if (prerenderResult) {
			await finalize(e); // Use the preloaded result directly
			return;
		}

		await input.beforeBegin?.(e);

		const nav = beginNavigation({
			href: hrefDetails.relativeURL,
			navigationType: "userNavigation",
		});

		currentNav = nav;
		prerenderResult = null;

		try {
			await finalize(e);
		} catch (error) {
			if (!isAbortError(error)) {
				LogError("Error during navigation", error);
			}
		}
	}

	return {
		...hrefDetails,
		start,
		stop,
		onClick,
		addEventListeners(link: HTMLAnchorElement) {
			link.addEventListener("pointerenter", start as any);
			link.addEventListener("focus", start as any);
			link.addEventListener("pointerleave", stop);
			link.addEventListener("blur", stop);
			link.addEventListener("click", onClick as any);
		},
		removeEventListeners(link: HTMLAnchorElement) {
			link.removeEventListener("pointerenter", start as any);
			link.removeEventListener("focus", start as any);
			link.removeEventListener("pointerleave", stop);
			link.removeEventListener("blur", stop);
			link.removeEventListener("click", onClick as any);
		},
	};
}

/////////////////////////////////////////////////////////////////////
// REDIRECTS
/////////////////////////////////////////////////////////////////////

async function effectuateRedirectDataResult(
	redirectData: RedirectData,
): Promise<RedirectData | null> {
	if (redirectData.status === "should") {
		if (redirectData.shouldRedirectStrategy === "hard") {
			window.location.href = redirectData.href;
			return { status: "did", href: redirectData.href };
		}
		if (redirectData.shouldRedirectStrategy === "soft") {
			await __navigate({ href: redirectData.href, navigationType: "redirect" });
			return { status: "did", href: redirectData.href };
		}
	}
	return null;
}

export async function handleRedirects(props: {
	abortController: AbortController;
	url: URL;
	requestInit?: RequestInit;
	isPrefetch?: boolean;
}): Promise<{
	redirectData: RedirectData | null;
	response?: Response;
}> {
	let res: Response | undefined;
	const bodyParentObj: RequestInit = {};

	const isGET = getIsGETRequest(props.requestInit);

	if (props.requestInit && (props.requestInit.body !== undefined || !isGET)) {
		if (props.requestInit.body instanceof FormData || typeof props.requestInit.body === "string") {
			bodyParentObj.body = props.requestInit.body;
		} else {
			bodyParentObj.body = JSON.stringify(props.requestInit.body);
		}
	}

	const headers = new Headers(props.requestInit?.headers);
	// To temporarily test traditional server redirect behavior,
	// you can set this to "0" instead of "1"
	headers.set("X-Accepts-Client-Redirect", "1");
	bodyParentObj.headers = headers;

	const finalRequestInit = {
		signal: props.abortController.signal,
		...props.requestInit,
		...bodyParentObj,
	};

	try {
		res = await fetch(props.url, finalRequestInit);

		const redirectData = parseFetchResponseForRedirectData(finalRequestInit, res);

		if (props.isPrefetch || !redirectData || redirectData.status === "did") {
			return { redirectData, response: res };
		}

		await effectuateRedirectDataResult(redirectData);
	} catch (error) {
		// If this was an attempted redirect, potentially a CORS error here.
		// Recommend returning a client redirect instruction instead.
		if (!isAbortError(error)) {
			// if a GET and not a prefetch, try just hard reloading
			if (isGET && !props.isPrefetch) {
				window.location.href = props.url.href;
				return { redirectData: { status: "did", href: props.url.href }, response: res };
			}
			LogError(error);
		}
	}

	return { redirectData: null, response: res };
}

/////////////////////////////////////////////////////////////////////
// SUBMISSIONS / MUTATIONS
/////////////////////////////////////////////////////////////////////

function handleSubmissionController(key: string) {
	// Abort existing submission if it exists
	const existing = navigationState.submissions.get(key);
	if (existing) {
		existing.controller.abort();
		navigationState.submissions.delete(key);
	}

	const controller = new AbortController();
	navigationState.submissions.set(key, {
		controller,
		type: "submission",
	});

	return { abortController: controller, didAbort: !!existing };
}

export async function submit<T = any>(
	url: string | URL,
	requestInit?: RequestInit,
): Promise<{ success: true; data: T } | { success: false; error: string }> {
	const submitRes = await submitInner(url, requestInit);

	if (!submitRes.success) {
		LogError(submitRes.error);
		return { success: false, error: submitRes.error };
	}

	try {
		const isGET = getIsGETRequest(requestInit);
		const json = await submitRes.response.json();

		if (!submitRes.alreadyRevalidated && !isGET) {
			await revalidate();
		}

		return {
			success: true,
			data: json as T,
		};
	} catch (e) {
		return {
			success: false,
			error: e instanceof Error ? e.message : "Unknown error",
		};
	}
}

async function submitInner(
	url: string | URL,
	_requestInit_?: RequestInit,
): Promise<
	({ success: true; response: Response } | { success: false; error: string }) & {
		alreadyRevalidated: boolean;
	}
> {
	const requestInit = _requestInit_ || {};

	setLoadingStatus({ type: "submission", value: true });

	const urlStr = typeof url === "string" ? url : url.href;
	const submissionKey = urlStr + (requestInit?.method || "");
	const { abortController, didAbort } = handleSubmissionController(submissionKey);

	const urlToUse = new URL(url, window.location.href);

	const headers = new Headers(requestInit.headers);
	requestInit.headers = headers;

	const isGET = getIsGETRequest(requestInit);

	try {
		const { redirectData, response } = await handleRedirects({
			abortController,
			url: urlToUse,
			requestInit,
		});

		const oldID = internal_RiverClientGlobal.get("buildID");
		const newID = response?.headers.get(buildIDHeader) || "";
		if (newID && newID !== oldID) {
			dispatchBuildIDEvent({ newID, oldID, fromGETAction: isGET });
		}

		const redirected = redirectData?.status === "did";

		navigationState.submissions.delete(submissionKey);

		if (response && getIsErrorRes(response)) {
			setLoadingStatus({ type: "submission", value: false });
			return {
				success: false,
				error: String(response.status),
				alreadyRevalidated: redirected,
			} as const;
		}

		if (didAbort) {
			if (!isGET) {
				// resets status bool
				await revalidate();
			}
			return {
				success: false,
				error: "Aborted",
				alreadyRevalidated: true,
			} as const;
		}

		if (!response?.ok) {
			const msg = String(response?.status || "unknown");
			return {
				success: false,
				error: msg,
				alreadyRevalidated: redirected,
			} as const;
		}

		setLoadingStatus({ type: "submission", value: false });

		return {
			success: true,
			response,
			alreadyRevalidated: redirected,
		} as const;
	} catch (error) {
		if (isAbortError(error)) {
			// eat
			return {
				success: false,
				error: "Aborted",
				alreadyRevalidated: false,
			} as const;
		}

		LogError(error);
		setLoadingStatus({ type: "submission", value: false });

		return {
			success: false,
			error: error instanceof Error ? error.message : "Unknown error",
			alreadyRevalidated: false,
		} as const;
	}
}

/////////////////////////////////////////////////////////////////////
// STATUS
/////////////////////////////////////////////////////////////////////

const STATUS_EVENT_KEY = "river:status";

let isNavigating = false;
let isSubmitting = false;
let isRevalidating = false;

function setLoadingStatus({
	type,
	value,
}: {
	type: NavigationType | "submission";
	value: boolean;
}) {
	if (type === "dev-revalidation" || type === "prefetch") {
		return;
	}

	if (type === "revalidation") {
		isRevalidating = value;
	} else if (type === "submission") {
		isSubmitting = value;
	} else {
		isNavigating = value;
	}

	dispatchStatusEvent();
}

type StatusEventDetail = {
	isNavigating: boolean;
	isSubmitting: boolean;
	isRevalidating: boolean;
};

export type StatusEvent = CustomEvent<StatusEventDetail>;

let dispatchStatusEventDebounceTimer: number | undefined;

function dispatchStatusEvent() {
	clearTimeout(dispatchStatusEventDebounceTimer);

	dispatchStatusEventDebounceTimer = window.setTimeout(() => {
		window.dispatchEvent(
			new CustomEvent(STATUS_EVENT_KEY, {
				detail: {
					isRevalidating,
					isSubmitting,
					isNavigating,
				} satisfies StatusEventDetail,
			}),
		);
	}, 1);
}

export function getStatus(): StatusEventDetail {
	return {
		isNavigating,
		isSubmitting,
		isRevalidating,
	};
}

export const addStatusListener = makeListenerAdder<StatusEventDetail>(STATUS_EVENT_KEY);

/////////////////////////////////////////////////////////////////////
// ROUTE CHANGE LISTENER
/////////////////////////////////////////////////////////////////////

export const addRouteChangeListener = makeListenerAdder<RouteChangeEventDetail>(
	RIVER_ROUTE_CHANGE_EVENT_KEY,
);

/////////////////////////////////////////////////////////////////////
// RE-RENDER APP
/////////////////////////////////////////////////////////////////////

function resolvePublicURL(href: string): URL {
	let baseURL = internal_RiverClientGlobal.get("viteDevURL");
	if (!baseURL) {
		baseURL = window.location.origin + internal_RiverClientGlobal.get("publicPathPrefix");
	}
	if (baseURL.endsWith("/")) {
		baseURL = baseURL.slice(0, -1);
	}
	return new URL(href.startsWith("/") ? baseURL + href : baseURL + "/" + href);
}

async function __reRenderApp({
	json,
	navigationType,
	runHistoryOptions,
	cssBundlePromises,
}: {
	json: GetRouteDataOutput;
	navigationType: NavigationType;
	runHistoryOptions?: {
		href: string;
		scrollStateToRestore?: ScrollState;
		replace?: boolean;
	};
	cssBundlePromises: Array<any>;
}) {
	// Changing the title instantly makes it feel faster
	// The temp textarea trick is to decode any HTML entities in the title
	const tempTxt = document.createElement("textarea");
	tempTxt.innerHTML = json.title ?? "";
	if (document.title !== tempTxt.value) document.title = tempTxt.value;

	// NOW ACTUALLY SET EVERYTHING
	const identicalKeysToSet = [
		"loadersData",
		"importURLs",
		"exportKeys",
		"outermostErrorIndex",
		"splatValues",
		"params",
		"hasRootData",
	] as const satisfies ReadonlyArray<keyof RiverClientGlobal>;

	for (const key of identicalKeysToSet) {
		internal_RiverClientGlobal.set(key, json[key]);
	}

	await handleComponents();

	let scrollStateToDispatch: ScrollState | undefined;

	if (runHistoryOptions) {
		// __TODO
		// - scroll to top on link clicks, but provide an opt-out
		// - scroll to top on form responses, but provide an opt-out

		const { href, scrollStateToRestore, replace } = runHistoryOptions;

		if (navigationType === "userNavigation" || navigationType === "redirect") {
			if (href !== location.href && !replace) {
				customHistory.push(href);
			} else {
				customHistory.replace(href);
			}
			scrollStateToDispatch = { x: 0, y: 0 };
		}

		if (navigationType === "browserHistory" && scrollStateToRestore) {
			scrollStateToDispatch = scrollStateToRestore;
		}

		// if revalidation, do nothing
	}

	// dispatch event
	const detail: RouteChangeEventDetail = { scrollState: scrollStateToDispatch } as const;

	// Wait for all CSS bundle preloads to complete
	if (cssBundlePromises.length > 0) {
		try {
			LogInfo("Waiting for CSS bundle preloads to complete...");
			await Promise.all(cssBundlePromises);
			LogInfo("CSS bundle preloads completed.");
		} catch (error) {
			LogError("Error preloading CSS bundles:", error);
		}
	}

	// Now that CSS is preloaded, update the DOM with any unseen CSS bundles
	window.requestAnimationFrame(() => {
		for (const x of json.cssBundles ?? []) {
			if (document.querySelector(`link[${cssBundleDataAttr}="${x}"]`)) {
				return;
			}
			const newLink = document.createElement("link");
			newLink.rel = "stylesheet";
			newLink.href = internal_RiverClientGlobal.get("publicPathPrefix") + x;
			newLink.setAttribute(cssBundleDataAttr, x);
			document.head.appendChild(newLink);
		}
	});

	window.dispatchEvent(new CustomEvent(RIVER_ROUTE_CHANGE_EVENT_KEY, { detail }));

	head.removeAllBetween("meta");
	head.addBlocks("meta", json.metaHeadBlocks ?? []);
	head.removeAllBetween("rest");
	head.addBlocks("rest", json.restHeadBlocks ?? []);
}

const cssBundleDataAttr = "data-river-css-bundle";

/////////////////////////////////////////////////////////////////////
// SIMPLE WRAPPERS
/////////////////////////////////////////////////////////////////////

export async function navigate(href: string, options?: { replace?: boolean }) {
	await __navigate({
		href,
		navigationType: "userNavigation",
		replace: options?.replace,
	});
}

export async function revalidate() {
	await __navigate({ href: window.location.href, navigationType: "revalidation" });
}

export async function devRevalidate() {
	await __navigate({ href: window.location.href, navigationType: "dev-revalidation" });
}

/////////////////////////////////////////////////////////////////////
// HEAD
/////////////////////////////////////////////////////////////////////

const head = {
	addBlocks,
	removeAllBetween,
} as const;

const markerCache: Record<string, { startComment: Comment | null; endComment: Comment | null }> =
	{};

function removeAllBetween(type: "meta" | "rest") {
	const { startComment, endComment } = getStartAndEndComments(type);
	if (!startComment || !endComment) {
		return;
	}

	let currentNode = startComment.nextSibling as Node | null;

	while (currentNode && currentNode !== endComment) {
		const nextNode = currentNode.nextSibling;
		currentNode.parentNode?.removeChild(currentNode);
		currentNode = nextNode;
	}
}

function addBlocks(type: "meta" | "rest", blocks: Array<HeadBlock>) {
	const { startComment, endComment } = getStartAndEndComments(type);
	if (!startComment || !endComment) {
		return;
	}

	const fragment = document.createDocumentFragment();

	for (const block of blocks) {
		if (!block.tag) {
			continue;
		}

		const newEl = document.createElement(block.tag);

		if (block.safeAttributes) {
			for (const key of Object.keys(block.safeAttributes)) {
				newEl.setAttribute(key, block.safeAttributes[key] ?? Panic());
			}
		}

		if (block.booleanAttributes) {
			for (const key of block.booleanAttributes) {
				newEl.setAttribute(key, "");
			}
		}

		if (block.innerHTML) {
			newEl.innerHTML = block.innerHTML;
		}

		fragment.appendChild(newEl);
	}

	endComment.parentNode?.insertBefore(fragment, endComment);
}

function getStartAndEndComments(type: "meta" | "rest") {
	if (!markerCache[type]) {
		markerCache[type] = {
			startComment: findComment(`data-river="${type}-start"`),
			endComment: findComment(`data-river="${type}-end"`),
		};
	}
	return markerCache[type];
}

function findComment(matchingText: string) {
	const iterator = document.createNodeIterator(document.head, NodeFilter.SHOW_COMMENT, {
		acceptNode(node: Comment) {
			return node.nodeValue?.trim() === matchingText
				? NodeFilter.FILTER_ACCEPT
				: NodeFilter.FILTER_REJECT;
		},
	});
	return iterator.nextNode() as Comment | null;
}

/////////////////////////////////////////////////////////////////////
// SCROLL RESTORATION
/////////////////////////////////////////////////////////////////////

const scrollStateMapKey = "__river__scrollStateMap";
type ScrollStateMap = Map<string, ScrollState>;

function getScrollStateMapFromLocalStorage() {
	const scrollStateMapString = localStorage.getItem(scrollStateMapKey);
	let scrollStateMap: ScrollStateMap;
	if (scrollStateMapString) {
		scrollStateMap = new Map(JSON.parse(scrollStateMapString));
	} else {
		scrollStateMap = new Map();
	}
	return scrollStateMap;
}

function setScrollStateMapToLocalStorage(newScrollStateMap: ScrollStateMap) {
	localStorage.setItem(scrollStateMapKey, JSON.stringify(Array.from(newScrollStateMap.entries())));
}

function setScrollStateMapSubKey(key: string, value: ScrollState) {
	const scrollStateMap = getScrollStateMapFromLocalStorage();
	scrollStateMap.set(key, value);

	// if new item would brought it over 50 entries, delete the oldest one
	if (scrollStateMap.size > 50) {
		const oldestKey = Array.from(scrollStateMap.keys())[0];
		scrollStateMap.delete(oldestKey ?? Panic());
	}

	setScrollStateMapToLocalStorage(scrollStateMap);
}

function readScrollStateMapSubKey(key: string) {
	const scrollStateMap = getScrollStateMapFromLocalStorage();
	return scrollStateMap.get(key);
}

const scrollStateMapSubKey = {
	read: readScrollStateMapSubKey,
	set: setScrollStateMapSubKey,
};

/////////////////////////////////////////////////////////////////////
// CUSTOM HISTORY
/////////////////////////////////////////////////////////////////////

let customHistory: ReturnType<typeof createBrowserHistory>;
let lastKnownCustomLocation: (typeof customHistory)["location"];

function initCustomHistory() {
	customHistory = createBrowserHistory();
	lastKnownCustomLocation = customHistory.location;
	customHistory.listen(customHistoryListener);
	setNativeScrollRestorationToManual();
}

function setNativeScrollRestorationToManual() {
	if (history.scrollRestoration && history.scrollRestoration !== "manual") {
		history.scrollRestoration = "manual";
	}
}

async function customHistoryListener({ action, location }: Update) {
	// save current scroll state to map
	scrollStateMapSubKey.set(lastKnownCustomLocation.key, {
		x: window.scrollX,
		y: window.scrollY,
	});

	if (action === "POP") {
		if (
			location.key !== lastKnownCustomLocation.key &&
			(location.pathname !== lastKnownCustomLocation.pathname ||
				location.search !== lastKnownCustomLocation.search)
		) {
			await __navigate({
				href: window.location.href,
				navigationType: "browserHistory",
				scrollStateToRestore: scrollStateMapSubKey.read(location.key),
			});
		}
	}

	// now set lastKnownCustomLocation to new location
	lastKnownCustomLocation = location;
}

export function getHistoryInstance() {
	return customHistory;
}

/////////////////////////////////////////////////////////////////////
// INIT CLIENT
/////////////////////////////////////////////////////////////////////

export function getRootEl() {
	return document.getElementById("river-root") as HTMLDivElement;
}

let latestHMRTimestamp = Date.now();

export async function initClient(renderFn: () => void) {
	if (import.meta.hot) {
		import.meta.hot.on("vite:afterUpdate", () => {
			latestHMRTimestamp = Date.now();
			LogInfo("HMR update detected", latestHMRTimestamp);
		});
	}

	// HANDLE HISTORY STUFF
	initCustomHistory();

	// HANDLE COMPONENTS
	await handleComponents();

	// RUN THE RENDER FUNCTION
	renderFn();

	// INSTANTIATE GLOBAL EVENT LISTENERS
	__addAnchorClickListener();
}

async function handleComponents() {
	const originalImportURLs = internal_RiverClientGlobal.get("importURLs");
	const dedupedImportURLs = [...new Set(originalImportURLs)];

	const dedupedModules = await Promise.all(
		dedupedImportURLs.map((x) => {
			return import(/* @vite-ignore */ resolvePublicURL(x).href);
		}),
	);
	const modulesMap = new Map(dedupedImportURLs.map((url, index) => [url, dedupedModules[index]]));

	const exportKeys = internal_RiverClientGlobal.get("exportKeys") ?? [];
	internal_RiverClientGlobal.set(
		"activeComponents",
		originalImportURLs.map((x, i) => modulesMap.get(x)?.[exportKeys[i] ?? "default"] ?? null),
	);
	internal_RiverClientGlobal.set(
		"activeErrorBoundaries",
		originalImportURLs.map((x, i) => modulesMap.get(x)?.ErrorBoundary ?? null),
	);
}

export function getCurrentRiverData<T = any>() {
	let rootData: T | null = null;
	if (internal_RiverClientGlobal.get("hasRootData")) {
		rootData = internal_RiverClientGlobal.get("loadersData")[0];
	}
	return {
		buildID: internal_RiverClientGlobal.get("buildID") || "",
		splatValues: internal_RiverClientGlobal.get("splatValues") || [],
		params: internal_RiverClientGlobal.get("params") || {},
		rootData,
	};
}

/////////////////////////////////////////////////////////////////////
// BUILD ID
/////////////////////////////////////////////////////////////////////

const buildIDHeader = "X-River-Build-Id";

const BUILD_ID_EVENT_KEY = "river:build-id";

type BuildIDEvent = { oldID: string; newID: string; fromGETAction: boolean };

function dispatchBuildIDEvent(detail: BuildIDEvent) {
	internal_RiverClientGlobal.set("buildID", detail.newID);
	window.dispatchEvent(new CustomEvent(BUILD_ID_EVENT_KEY, { detail }));
}

export const addBuildIDListener = makeListenerAdder<BuildIDEvent>(BUILD_ID_EVENT_KEY);

export function getBuildID() {
	return internal_RiverClientGlobal.get("buildID");
}

/////////////////////////////////////////////////////////////////////
// LISTENER UTILS
/////////////////////////////////////////////////////////////////////

type CleanupFunction = () => void;

function makeListenerAdder<T>(key: string) {
	return function addListener(listener: (event: CustomEvent<T>) => void): CleanupFunction {
		window.addEventListener(key, listener as any);
		return () => {
			window.removeEventListener(key, listener as any);
		};
	};
}

/////////////////////////////////////////////////////////////////////
// DATA BOOST LISTENERS
/////////////////////////////////////////////////////////////////////

type LinkClickListenerCallback<E extends Event> = (event: E) => void | Promise<void>;

type LinkClickListenerCallbacksBase<E extends Event> = {
	beforeBegin?: LinkClickListenerCallback<E>;
	beforeRender?: LinkClickListenerCallback<E>;
	afterRender?: LinkClickListenerCallback<E>;
};

type LinkClickListenerCallbacks<E extends Event> = LinkClickListenerCallbacksBase<E> & {
	requireDataBoostAttribute: boolean;
};

export function makeLinkClickListenerFn<E extends Event>(callbacks: LinkClickListenerCallbacks<E>) {
	return async (event: E) => {
		if (event.defaultPrevented) {
			return;
		}

		const anchorDetails = getAnchorDetailsFromEvent(event as unknown as MouseEvent);
		if (!anchorDetails) {
			return;
		}

		const { anchor, isEligibleForDefaultPrevention, isInternal } = anchorDetails;

		if (!anchor || (callbacks.requireDataBoostAttribute && !anchor.dataset.boost)) {
			return;
		}

		if (isEligibleForDefaultPrevention && isInternal) {
			event.preventDefault();

			await callbacks.beforeBegin?.(event);

			const x = beginNavigation({
				href: anchor.href,
				navigationType: "userNavigation",
			});
			if (!x.promise) {
				return;
			}

			const res = await x.promise;
			if (!res) {
				return;
			}

			await callbacks.beforeRender?.(event);

			await __completeNavigation(res);

			await callbacks.afterRender?.(event);
		}
	};
}

const baseDataBoostListenerFn = makeLinkClickListenerFn({ requireDataBoostAttribute: true });

function __addAnchorClickListener() {
	document.body.addEventListener("click", baseDataBoostListenerFn);
}
