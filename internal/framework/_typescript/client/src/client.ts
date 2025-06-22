// ./src/client.ts
/// <reference types="vite/client" />

import { createBrowserHistory, type Update } from "history";
import { debounce } from "river.now/kit/debounce";
import { jsonDeepEquals } from "river.now/kit/json";
import {
	getAnchorDetailsFromEvent,
	getHrefDetails,
	getIsErrorRes,
	getIsGETRequest,
} from "river.now/kit/url";
import { updateHeadEls } from "./head.ts";
import type { historyInstance, historyListener } from "./history_types.ts";
import {
	getBuildIDFromResponse,
	parseFetchResponseForRedirectData,
	type RedirectData,
} from "./redirects.ts";
import {
	type GetRouteDataOutput,
	internal_RiverClientGlobal,
	type RiverClientGlobal,
	type RouteErrorComponent,
} from "./river_ctx.ts";
import { isAbortError, LogError, LogInfo, Panic } from "./utils.ts";

/////////////////////////////////////////////////////////////////////
// COMMON
/////////////////////////////////////////////////////////////////////

const RIVER_ROUTE_CHANGE_EVENT_KEY = "river:route-change";

export type ScrollState = { x: number; y: number } | { hash: string };
type RouteChangeEventDetail = {
	scrollState?: ScrollState;
	index?: number;
};
export type RouteChangeEvent = CustomEvent<RouteChangeEventDetail>;

/////////////////////////////////////////////////////////////////////
// NAVIGATION TYPES AND GLOBAL STATE
/////////////////////////////////////////////////////////////////////

type NavigationResult =
	| ({
			response: Response;
			props: NavigateProps;
	  } & (
			| {
					json: GetRouteDataOutput;
					cssBundlePromises: Array<Promise<any>>;
					waitFnPromise: Promise<any> | undefined;
			  }
			| { redirectData: RedirectData }
	  ))
	| undefined;

export type NavigationControl = {
	abortController: AbortController | undefined;
	promise: Promise<NavigationResult>;
};

type NavigationType =
	| "browserHistory"
	| "userNavigation"
	| "revalidation"
	| "redirect"
	| "prefetch";

export type NavigateProps = {
	href: string;
	navigationType: NavigationType;
	scrollStateToRestore?: ScrollState;
	replace?: boolean;
	redirectCount?: number;
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

type PartialWaitFnJSON = Pick<
	GetRouteDataOutput,
	| "matchedPatterns"
	| "splatValues"
	| "params"
	| "hasRootData"
	| "loadersData"
	| "importURLs"
>;

function resolveWaitFnPropsFromJSON(
	json: PartialWaitFnJSON,
	buildID: string,
	idx: number,
) {
	return {
		buildID: buildID,
		matchedPatterns: json.matchedPatterns || [],
		splatValues: json.splatValues || [],
		params: json.params || {},
		rootData: json.hasRootData ? json.loadersData[0] : null,
		loaderData: json.loadersData[idx],
	};
}

async function __completeNavigation(x: NavigationResult) {
	if (!x) {
		return;
	}
	if ("redirectData" in x) {
		await effectuateRedirectDataResult(x.redirectData, x.props.redirectCount || 0);
		return;
	}
	const oldID = internal_RiverClientGlobal.get("buildID");
	const newID = getBuildIDFromResponse(x.response);
	if (newID && newID !== oldID) {
		dispatchBuildIDEvent({ newID, oldID, fromGETAction: false });
	}
	const clientLoadersData = await x.waitFnPromise;
	internal_RiverClientGlobal.set("clientLoadersData", clientLoadersData);
	try {
		await __reRenderApp({
			json: x.json,
			navigationType: x.props.navigationType,
			runHistoryOptions: x.props,
			cssBundlePromises: x.cssBundlePromises,
		});
	} catch (error) {
		handleNavError(error, x.props);
	}
}

function getMaybeNewPreloadLink(x: string) {
	const href = resolvePublicHref(x);
	const existing = document.querySelector(`link[href="${CSS.escape(href)}"]`);
	if (existing) {
		return;
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
		url.searchParams.set(
			"river_json",
			internal_RiverClientGlobal.get("buildID") || "1",
		);

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
			return { response, redirectData, props };
		}

		const json = (await response.json()) as GetRouteDataOutput | undefined;
		if (!json) {
			throw new Error("No JSON response");
		}

		// deps are only present in prod because they stem from the rollup metafile
		// (same for CSS bundles -- vite handles them in dev)
		// so in dev, to get similar behavior, we use the importURLs
		// (which is a subset of what the deps would be in prod)
		const depsToPreload = import.meta.env.DEV
			? [...new Set(json.importURLs)]
			: json.deps;

		// Add missing deps modulepreload links
		for (const x of depsToPreload ?? []) {
			if (x === "") {
				continue;
			}
			const newLink = getMaybeNewPreloadLink(x);
			if (!newLink) {
				continue;
			}
			newLink.rel = "modulepreload";
			document.head.appendChild(newLink);
		}

		const buildID = getBuildIDFromResponse(response);

		const waitFnPromise = runWaitFns(json, buildID);

		// Create an array to store promises for CSS bundle preloads
		const cssBundlePromises: Array<Promise<any>> = [];

		// Add missing css bundle preload links
		for (const x of json.cssBundles ?? []) {
			const newLink = getMaybeNewPreloadLink(x);
			if (!newLink) {
				continue;
			}
			newLink.rel = "preload";
			newLink.setAttribute("as", "style");
			document.head.appendChild(newLink);

			// Create a promise for this CSS bundle preload
			const preloadPromise = new Promise((resolve, reject) => {
				newLink.onload = resolve;
				newLink.onerror = reject;
			});
			cssBundlePromises.push(preloadPromise);
		}

		return { response, json, props, cssBundlePromises, waitFnPromise };
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

function isJustAHashChange(
	anchorDetails: ReturnType<typeof getAnchorDetailsFromEvent>,
): boolean {
	if (!anchorDetails) {
		return false;
	}
	const { pathname, search, hash } = new URL(
		anchorDetails.anchor.href,
		window.location.href,
	);
	if (
		hash &&
		pathname === window.location.pathname &&
		search === window.location.search
	) {
		return true;
	}
	return false;
}

/////////////////////////////////////////////////////////////////////
// PREFETCH
/////////////////////////////////////////////////////////////////////

type GetPrefetchHandlersInput<E extends Event> = LinkOnClickCallbacksBase<E> & {
	href: string;
	delayMs?: number;
};

export function getPrefetchHandlers<E extends Event>(
	input: GetPrefetchHandlersInput<E>,
) {
	const hrefDetails = getHrefDetails(input.href);
	if (!hrefDetails.isHTTP || !hrefDetails.relativeURL || hrefDetails.isExternal) {
		return;
	}

	let timer: number | undefined;
	let currentNav: NavigationControl | null = null;
	let prerenderResult: NavigationResult | null = null;

	// by default, wait 100ms before prefetching
	const delayMsToUse = input.delayMs ?? 100;

	async function finalize(e: E): Promise<void> {
		try {
			if (!prerenderResult && currentNav) {
				prerenderResult = await currentNav.promise;
			}
			if (prerenderResult) {
				await input.beforeRender?.(e);

				if ("redirectData" in prerenderResult) {
					await effectuateRedirectDataResult(
						prerenderResult.redirectData,
						prerenderResult.props.redirectCount || 0,
					);
					return;
				}

				if (!("json" in prerenderResult)) {
					throw new Error("Invalid navigation result: no JSON response.");
				}

				await __completeNavigation({
					...prerenderResult,
					props: {
						...prerenderResult.props,
						navigationType: "userNavigation" as const,
					},
				});

				await input.afterRender?.(e);
			}
		} catch (e) {
			if (!isAbortError(e)) {
				LogError("Error finalizing prefetch:", e);
			}
		} finally {
			prerenderResult = null;
			currentNav = null;
		}
	}

	async function prefetch(e: E): Promise<void> {
		if (currentNav || !hrefDetails.isHTTP) {
			return;
		}

		// We don't really want to prefetch if the user is already on the page.
		// In those cases, wait for an actual click.
		const currentUrl = new URL(window.location.href);
		const targetUrl = hrefDetails.url;
		currentUrl.hash = "";
		targetUrl.hash = "";
		if (currentUrl.href === targetUrl.href) {
			return;
		}

		if (input.beforeBegin) {
			await input.beforeBegin(e);
		}

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
					LogError("Prefetch failed:", error);
				}
			});
	}

	function start(e: E): void {
		if (currentNav) {
			return;
		}
		timer = window.setTimeout(() => prefetch(e), delayMsToUse);
	}

	function stop(): void {
		if (timer) {
			clearTimeout(timer);
		}

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

	async function onClick(e: E): Promise<void> {
		if (e.defaultPrevented || !hrefDetails.isHTTP) {
			return;
		}

		const anchorDetails = getAnchorDetailsFromEvent(e as unknown as MouseEvent);
		if (!anchorDetails) {
			return;
		}

		const { isEligibleForDefaultPrevention, isInternal } = anchorDetails;

		if (!isEligibleForDefaultPrevention || !isInternal) {
			return;
		}

		if (isJustAHashChange(anchorDetails)) {
			saveScrollState();
			return;
		}

		e.preventDefault();
		setLoadingStatus({ type: "userNavigation", value: true });

		if (prerenderResult) {
			await finalize(e); // Use the preloaded result directly
			return;
		}

		if (timer) {
			clearTimeout(timer);
		}

		if (input.beforeBegin) {
			await input.beforeBegin(e);
		}

		currentNav = beginNavigation({
			href: hrefDetails.relativeURL,
			navigationType: "userNavigation",
		});

		prerenderResult = null;

		try {
			await finalize(e);
		} catch (error) {
			if (!isAbortError(error)) {
				LogError("Error during click navigation:", error);
			}
		}
	}

	return {
		...hrefDetails,
		start,
		stop,
		onClick,
	};
}

function saveScrollState() {
	scrollStateMapSubKey.set(lastKnownCustomLocation.key, {
		x: window.scrollX,
		y: window.scrollY,
	});
}

/////////////////////////////////////////////////////////////////////
// REDIRECTS
/////////////////////////////////////////////////////////////////////

const RIVER_HARD_RELOAD_QUERY_PARAM = "river_reload";

async function effectuateRedirectDataResult(
	redirectData: RedirectData,
	redirectCount: number,
): Promise<RedirectData | null> {
	if (redirectData.status === "should") {
		if (redirectData.shouldRedirectStrategy === "hard") {
			if (!redirectData.hrefDetails.isHTTP) {
				return null;
			}
			if (redirectData.hrefDetails.isExternal) {
				window.location.href = redirectData.href;
			}
			if (redirectData.hrefDetails.isInternal) {
				const url = new URL(redirectData.href, window.location.href);
				url.searchParams.set(
					RIVER_HARD_RELOAD_QUERY_PARAM,
					redirectData.latestBuildID,
				);
				window.location.href = url.href;
			}
			return {
				hrefDetails: redirectData.hrefDetails,
				status: "did",
				href: redirectData.href,
			};
		}
		if (redirectData.shouldRedirectStrategy === "soft") {
			await __navigate({
				href: redirectData.href,
				navigationType: "redirect",
				redirectCount: redirectCount + 1,
			});
			return {
				hrefDetails: redirectData.hrefDetails,
				status: "did",
				href: redirectData.href,
			};
		}
	}
	return null;
}

export async function handleRedirects(props: {
	abortController: AbortController;
	url: URL;
	requestInit?: RequestInit;
	isPrefetch?: boolean;
	redirectCount?: number;
}): Promise<{
	redirectData: RedirectData | null;
	response?: Response;
}> {
	const MAX_REDIRECTS = 10;
	const redirectCount = props.redirectCount || 0;

	if (redirectCount >= MAX_REDIRECTS) {
		LogError("Too many redirects");
		return { redirectData: null, response: undefined };
	}

	let res: Response | undefined;
	const bodyParentObj: RequestInit = {};

	const isGET = getIsGETRequest(props.requestInit);

	if (props.requestInit && (props.requestInit.body !== undefined || !isGET)) {
		if (
			props.requestInit.body instanceof FormData ||
			typeof props.requestInit.body === "string"
		) {
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

	let redirectData: RedirectData | null = null;

	try {
		res = await fetch(props.url, finalRequestInit);

		redirectData = parseFetchResponseForRedirectData(finalRequestInit, res);

		if (props.isPrefetch || !redirectData || redirectData.status === "did") {
			return { redirectData, response: res };
		}

		redirectData = await effectuateRedirectDataResult(redirectData, redirectCount);
	} catch (error) {
		// If this was an attempted redirect, potentially a CORS error here.
		// Recommend returning a client redirect instruction instead.
		if (!isAbortError(error)) {
			// if a GET and not a prefetch, try just hard reloading
			if (isGET && !props.isPrefetch) {
				window.location.href = props.url.href;
				return {
					redirectData: {
						// satisfy TypeScript (does not matter, we are hard reloading)
						hrefDetails: null as any,
						status: "did",
						href: props.url.href,
					},
					response: res,
				};
			}
			LogError(error);
		}
	}

	return { redirectData, response: res };
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
	const isGET = getIsGETRequest(requestInit);
	const needsRevalidation = !submitRes.alreadyRevalidated && !isGET;

	async function handleReval() {
		if (needsRevalidation) {
			// We want to set revalidation status to true before
			// turning off submission status so there is no flicker
			// of "all-off" loading state between submission and
			// revalidation.
			setLoadingStatus({ type: "revalidation", value: true });
			setLoadingStatus({ type: "submission", value: false });
			await revalidate();
		} else {
			setLoadingStatus({ type: "submission", value: false });
		}
	}

	await handleReval();

	if (!submitRes.success) {
		LogError(submitRes.error);
		return { success: false, error: submitRes.error };
	}

	try {
		const json = await submitRes.response.json();
		return { success: true, data: json as T };
	} catch (e) {
		LogError(e);
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
		const newID = getBuildIDFromResponse(response);
		if (newID && newID !== oldID) {
			dispatchBuildIDEvent({ newID, oldID, fromGETAction: isGET });
		}

		const redirected = redirectData?.status === "did";

		navigationState.submissions.delete(submissionKey);

		if (response && getIsErrorRes(response)) {
			return {
				success: false,
				error: String(response.status),
				alreadyRevalidated: redirected,
			} as const;
		}
		if (didAbort) {
			return {
				success: false,
				error: "Aborted",
				alreadyRevalidated: false,
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
	if (type === "prefetch") {
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
let lastStatusEvent: StatusEventDetail | null = null;

const STATUS_EVENT_DEBOUNCE_MS = 5;

function dispatchStatusEvent() {
	clearTimeout(dispatchStatusEventDebounceTimer);
	dispatchStatusEventDebounceTimer = window.setTimeout(() => {
		const newStatusEvent: StatusEventDetail = {
			isNavigating,
			isSubmitting,
			isRevalidating,
		};
		if (jsonDeepEquals(lastStatusEvent, newStatusEvent)) {
			return;
		}
		lastStatusEvent = newStatusEvent;
		window.dispatchEvent(
			new CustomEvent(STATUS_EVENT_KEY, { detail: newStatusEvent }),
		);
	}, STATUS_EVENT_DEBOUNCE_MS);
}

export function getStatus(): StatusEventDetail {
	return {
		isNavigating,
		isSubmitting,
		isRevalidating,
	};
}

export const addStatusListener =
	makeListenerAdder<StatusEventDetail>(STATUS_EVENT_KEY);

/////////////////////////////////////////////////////////////////////
// ROUTE CHANGE LISTENER
/////////////////////////////////////////////////////////////////////

export const addRouteChangeListener = makeListenerAdder<RouteChangeEventDetail>(
	RIVER_ROUTE_CHANGE_EVENT_KEY,
);

/////////////////////////////////////////////////////////////////////
// RE-RENDER APP
/////////////////////////////////////////////////////////////////////

function resolvePublicHref(relativeHref: string): string {
	let baseURL = internal_RiverClientGlobal.get("viteDevURL");
	if (!baseURL) {
		baseURL = internal_RiverClientGlobal.get("publicPathPrefix");
	}
	if (baseURL.endsWith("/")) {
		baseURL = baseURL.slice(0, -1);
	}
	let final = relativeHref.startsWith("/")
		? baseURL + relativeHref
		: baseURL + "/" + relativeHref;
	if (import.meta.env.DEV) {
		final += "?river_dev=1";
	}
	return final;
}

type RerenderAppProps = {
	json: GetRouteDataOutput;
	navigationType: NavigationType;
	runHistoryOptions?: {
		href: string;
		scrollStateToRestore?: ScrollState;
		replace?: boolean;
	};
	cssBundlePromises: Array<any>;
};

async function __reRenderApp(props: RerenderAppProps) {
	setLoadingStatus({ type: props.navigationType, value: false });
	const shouldUseViewTransitions =
		internal_RiverClientGlobal.get("useViewTransitions") &&
		!!document.startViewTransition &&
		props.navigationType !== "prefetch" &&
		props.navigationType !== "revalidation";
	if (shouldUseViewTransitions) {
		const transition = document.startViewTransition(async () => {
			await __reRenderAppInner(props);
		});
		await transition.finished;
		return;
	}
	await __reRenderAppInner(props);
}

async function __reRenderAppInner({
	json,
	navigationType,
	runHistoryOptions,
	cssBundlePromises,
}: RerenderAppProps) {
	// NOW ACTUALLY SET EVERYTHING
	const identicalKeysToSet = [
		"outermostError",
		"outermostErrorIdx",
		"errorExportKey",

		"matchedPatterns",
		"loadersData",
		"importURLs",
		"exportKeys",
		"hasRootData",

		"params",
		"splatValues",
	] as const satisfies ReadonlyArray<keyof RiverClientGlobal>;

	for (const key of identicalKeysToSet) {
		internal_RiverClientGlobal.set(key, json[key]);
	}

	await handleComponents(json.importURLs);

	let scrollStateToDispatch: ScrollState | undefined;

	if (runHistoryOptions) {
		const { href, scrollStateToRestore, replace } = runHistoryOptions;

		const hash = href.split("#")[1];

		if (navigationType === "userNavigation" || navigationType === "redirect") {
			const target = new URL(href, window.location.href).href;
			const current = new URL(window.location.href).href;
			if (target !== current && !replace) {
				getHistoryInstance().push(href);
			} else {
				getHistoryInstance().replace(href);
			}

			scrollStateToDispatch = hash ? { hash } : { x: 0, y: 0 };
		}

		if (navigationType === "browserHistory") {
			if (scrollStateToRestore) {
				scrollStateToDispatch = scrollStateToRestore;
			} else if (hash) {
				scrollStateToDispatch = { hash };
			}
		}

		// if revalidation, do nothing
	}

	// Changing the title instantly makes it feel faster
	// The temp textarea trick is to decode any HTML entities in the title.
	// This should come after pushing to history though, so that the title is
	// correct in the history entry.
	const tempTxt = document.createElement("textarea");
	tempTxt.innerHTML = json.title?.dangerousInnerHTML ?? "";
	if (document.title !== tempTxt.value) {
		document.title = tempTxt.value;
	}

	// dispatch event
	const detail: RouteChangeEventDetail = {
		scrollState: scrollStateToDispatch,
	} as const;

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

	updateHeadEls("meta", json.metaHeadEls ?? []);
	updateHeadEls("rest", json.restHeadEls ?? []);
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

async function revalidateNonDebounced() {
	await __navigate({
		href: window.location.href,
		navigationType: "revalidation",
	});
}

export const revalidate = debounce(revalidateNonDebounced, 10);

let devTimeSetupClientLoadersDebounced: () => Promise<void> = () => Promise.resolve();

if (import.meta.env.DEV) {
	(window as any).__waveRevalidate = revalidate;
	devTimeSetupClientLoadersDebounced = debounce(async () => {
		setLoadingStatus({ type: "revalidation", value: true });
		await setupClientLoaders();
		setLoadingStatus({ type: "revalidation", value: false });
		window.dispatchEvent(
			new CustomEvent(RIVER_ROUTE_CHANGE_EVENT_KEY, { detail: {} }),
		);
	}, 10);
}

/////////////////////////////////////////////////////////////////////
// SCROLL RESTORATION
/////////////////////////////////////////////////////////////////////

const scrollStateMapKey = "__river__scrollStateMap";
type ScrollStateMap = Map<string, ScrollState>;

function getScrollStateMapFromSessionStorage() {
	const scrollStateMapString = sessionStorage.getItem(scrollStateMapKey);
	let scrollStateMap: ScrollStateMap;
	if (scrollStateMapString) {
		scrollStateMap = new Map(JSON.parse(scrollStateMapString));
	} else {
		scrollStateMap = new Map();
	}
	return scrollStateMap;
}

function setScrollStateMapToSessionStorage(newScrollStateMap: ScrollStateMap) {
	sessionStorage.setItem(
		scrollStateMapKey,
		JSON.stringify(Array.from(newScrollStateMap.entries())),
	);
}

function setScrollStateMapSubKey(key: string, value: ScrollState) {
	const scrollStateMap = getScrollStateMapFromSessionStorage();
	scrollStateMap.set(key, value);

	// if new item would brought it over 50 entries, delete the oldest one
	if (scrollStateMap.size > 50) {
		const oldestKey = Array.from(scrollStateMap.keys())[0];
		scrollStateMap.delete(oldestKey ?? Panic());
	}

	setScrollStateMapToSessionStorage(scrollStateMap);
}

function readScrollStateMapSubKey(key: string) {
	const scrollStateMap = getScrollStateMapFromSessionStorage();
	return scrollStateMap.get(key);
}

const scrollStateMapSubKey = {
	read: readScrollStateMapSubKey,
	set: setScrollStateMapSubKey,
};

/////////////////////////////////////////////////////////////////////
// CUSTOM HISTORY
/////////////////////////////////////////////////////////////////////

let __customHistory: historyInstance;
let lastKnownCustomLocation: (typeof __customHistory)["location"];

export function getHistoryInstance(): historyInstance {
	if (!__customHistory) {
		__customHistory = createBrowserHistory() as unknown as historyInstance;
	}
	return __customHistory;
}

export function initCustomHistory() {
	lastKnownCustomLocation = getHistoryInstance().location;
	getHistoryInstance().listen(customHistoryListener as unknown as historyListener);
	setNativeScrollRestorationToManual();
}

function setNativeScrollRestorationToManual() {
	if (history.scrollRestoration && history.scrollRestoration !== "manual") {
		history.scrollRestoration = "manual";
	}
}

async function customHistoryListener({ action, location }: Update) {
	if (location.key !== lastKnownCustomLocation.key) {
		dispatchLocationEvent();
	}

	const popWithinSameDoc =
		action === "POP" &&
		location.pathname === lastKnownCustomLocation.pathname &&
		location.search === lastKnownCustomLocation.search;

	const removingHash =
		popWithinSameDoc && lastKnownCustomLocation.hash && !location.hash;
	const addingHash =
		popWithinSameDoc && !lastKnownCustomLocation.hash && location.hash;
	const updatingHash = popWithinSameDoc && location.hash;

	if (!popWithinSameDoc) {
		saveScrollState();
	}

	if (action === "POP") {
		const newHash = location.hash.slice(1);

		if (addingHash || updatingHash) {
			applyScrollState({ hash: newHash });
		}

		if (removingHash) {
			const stored = scrollStateMapSubKey.read(location.key);
			applyScrollState(stored ?? { x: 0, y: 0 });
		}

		if (!popWithinSameDoc) {
			await __navigate({
				href: window.location.href,
				navigationType: "browserHistory",
				scrollStateToRestore: scrollStateMapSubKey.read(location.key),
			});
		}
	}

	lastKnownCustomLocation = location;
}

/////////////////////////////////////////////////////////////////////
// INIT CLIENT
/////////////////////////////////////////////////////////////////////

export function getRootEl() {
	return document.getElementById("river-root") as HTMLDivElement;
}

let latestHMRTimestamp = Date.now();

let hmrRevalidateSet: Set<string>;

export let hmrRunClientLoaders: (importMeta: ImportMeta) => void = () => {};

if (import.meta.env.DEV) {
	hmrRunClientLoaders = (importMeta: ImportMeta) => {
		if (hmrRevalidateSet === undefined) {
			hmrRevalidateSet = new Set();
		}
		if (import.meta.env.DEV && import.meta.hot) {
			const thisURL = new URL(importMeta.url, location.href);
			thisURL.search = "";
			const thisPathname = thisURL.pathname;
			const alreadyRegistered = hmrRevalidateSet.has(thisPathname);
			if (alreadyRegistered) {
				return;
			}
			hmrRevalidateSet.add(thisPathname);
			import.meta.hot.on("vite:afterUpdate", (props) => {
				for (const update of props.updates) {
					if (update.type === "js-update") {
						const updateURL = new URL(update.path, location.href);
						updateURL.search = "";
						if (updateURL.pathname === thisURL.pathname) {
							devTimeSetupClientLoadersDebounced();
						}
					}
				}
			});
		}
	};
}

async function setupClientLoaders() {
	const clientLoadersData = await runWaitFns(
		{
			hasRootData: internal_RiverClientGlobal.get("hasRootData"),
			importURLs: internal_RiverClientGlobal.get("importURLs"),
			loadersData: internal_RiverClientGlobal.get("loadersData"),
			matchedPatterns: internal_RiverClientGlobal.get("matchedPatterns"),
			params: internal_RiverClientGlobal.get("params"),
			splatValues: internal_RiverClientGlobal.get("splatValues"),
		},
		internal_RiverClientGlobal.get("buildID"),
	);

	internal_RiverClientGlobal.set("clientLoadersData", clientLoadersData);
}

const pageRefreshScrollStateKey = "__river__pageRefreshScrollState";

type pageRefreshScrollState = {
	x: number;
	y: number;
	unix: number;
	href: string;
};

window.addEventListener("beforeunload", () => {
	const scrollState: pageRefreshScrollState = {
		x: window.scrollX,
		y: window.scrollY,
		unix: Date.now(),
		href: window.location.href,
	};
	sessionStorage.setItem(pageRefreshScrollStateKey, JSON.stringify(scrollState));
});

function checkIfShouldScrollPostRefresh() {
	const scrollStateString = sessionStorage.getItem(pageRefreshScrollStateKey);
	if (scrollStateString) {
		const scrollState: pageRefreshScrollState = JSON.parse(scrollStateString);
		if (
			scrollState.href === window.location.href &&
			Date.now() - scrollState.unix < 5_000
		) {
			sessionStorage.removeItem(pageRefreshScrollStateKey);
			window.requestAnimationFrame(() => {
				applyScrollState(scrollState);
			});
		}
	}
}

export async function initClient(
	renderFn: () => void,
	options?: {
		defaultErrorBoundary?: RouteErrorComponent;
		useViewTransitions?: boolean;
	},
) {
	if (import.meta.env.DEV && import.meta.hot) {
		import.meta.hot.on("vite:afterUpdate", () => {
			latestHMRTimestamp = Date.now();
			LogInfo("HMR update detected", latestHMRTimestamp);
		});
	}

	if (options?.defaultErrorBoundary) {
		internal_RiverClientGlobal.set(
			"defaultErrorBoundary",
			options.defaultErrorBoundary,
		);
	} else {
		internal_RiverClientGlobal.set("defaultErrorBoundary", defaultErrorBoundary);
	}

	if (options?.useViewTransitions) {
		internal_RiverClientGlobal.set("useViewTransitions", true);
	}

	// HANDLE HISTORY STUFF
	initCustomHistory();

	const url = new URL(window.location.href);
	if (url.searchParams.has(RIVER_HARD_RELOAD_QUERY_PARAM)) {
		url.searchParams.delete(RIVER_HARD_RELOAD_QUERY_PARAM);
		__customHistory.replace(url.href);
	}

	// HANDLE COMPONENTS
	await handleComponents(internal_RiverClientGlobal.get("importURLs"));

	// SETUP CLIENT LOADERS
	await setupClientLoaders();

	// RUN THE RENDER FUNCTION
	renderFn();

	checkIfShouldScrollPostRefresh();

	window.addEventListener(
		"touchstart",
		() => {
			LogInfo("Touch device detected");
			internal_RiverClientGlobal.set("isTouchDevice", true);
		},
		{ once: true },
	);
}

async function importNewComponentsAndGetModulesMap(
	importURLs: Array<string>,
): Promise<Map<string, any>> {
	const dedupedImportURLs = [...new Set(importURLs ?? [])];
	const dedupedModules = await Promise.all(
		dedupedImportURLs.map(async (x) => {
			if (x === "") {
				return;
			}
			return await import(/* @vite-ignore */ resolvePublicHref(x));
		}),
	);
	return new Map(dedupedImportURLs.map((url, index) => [url, dedupedModules[index]]));
}

async function handleComponents(importURLs: Array<string>) {
	const modulesMap = await importNewComponentsAndGetModulesMap(importURLs);
	const originalImportURLs = internal_RiverClientGlobal.get("importURLs");
	const exportKeys = internal_RiverClientGlobal.get("exportKeys") ?? [];

	internal_RiverClientGlobal.set(
		"activeComponents",
		originalImportURLs.map(
			(x, i) => modulesMap.get(x)?.[exportKeys[i] ?? "default"] ?? null,
		),
	);

	const outermostErrorIdx = internal_RiverClientGlobal.get("outermostErrorIdx");

	if (outermostErrorIdx != null) {
		let errorComp: any;

		const errorModuleImportURL = originalImportURLs[outermostErrorIdx];

		if (errorModuleImportURL) {
			const errorModule = modulesMap.get(errorModuleImportURL);

			const errorExportKey = internal_RiverClientGlobal.get("errorExportKey");
			if (errorExportKey) {
				errorComp = errorModule[errorExportKey];
			}
		}

		internal_RiverClientGlobal.set(
			"activeErrorBoundary",
			errorComp ?? internal_RiverClientGlobal.get("defaultErrorBoundary"),
		);
	}
}

const defaultErrorBoundary: RouteErrorComponent = (props: { error: string }) => {
	return "Route Error: " + props.error;
};

async function runWaitFns(
	json: PartialWaitFnJSON,
	buildID: string,
): Promise<Array<any>> {
	await importNewComponentsAndGetModulesMap(json.importURLs);

	const matchedPatterns = json.matchedPatterns ?? [];
	const patternToWaitFnMap = internal_RiverClientGlobal.get("patternToWaitFnMap");
	const waitFnPromises: Array<Promise<any>> = [];

	let i = 0;
	for (const pattern of matchedPatterns) {
		if (patternToWaitFnMap[pattern]) {
			waitFnPromises.push(
				patternToWaitFnMap[pattern](resolveWaitFnPropsFromJSON(json, buildID, i)),
			);
		} else {
			waitFnPromises.push(Promise.resolve());
		}
		i++;
	}

	return Promise.all(waitFnPromises);
}

/////////////////////////////////////////////////////////////////////
// LOCATION EVENTS
/////////////////////////////////////////////////////////////////////

const LOCATION_EVENT_KEY = "river:location";

function dispatchLocationEvent() {
	window.dispatchEvent(new CustomEvent(LOCATION_EVENT_KEY));
}

export const addLocationListener = makeListenerAdder<BuildIDEvent>(LOCATION_EVENT_KEY);

export function getLocation() {
	return {
		pathname: window.location.pathname,
		search: window.location.search,
		hash: window.location.hash,
	};
}

/////////////////////////////////////////////////////////////////////
// BUILD ID
/////////////////////////////////////////////////////////////////////

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
	return function addListener(
		listener: (event: CustomEvent<T>) => void,
	): CleanupFunction {
		window.addEventListener(key, listener as any);
		return () => {
			window.removeEventListener(key, listener as any);
		};
	};
}

/////////////////////////////////////////////////////////////////////
// LINK ONCLICK HANDLERS
/////////////////////////////////////////////////////////////////////

type LinkOnClickCallback<E extends Event> = (event: E) => void | Promise<void>;

type LinkOnClickCallbacksBase<E extends Event> = {
	beforeBegin?: LinkOnClickCallback<E>;
	beforeRender?: LinkOnClickCallback<E>;
	afterRender?: LinkOnClickCallback<E>;
};

type LinkOnClickCallbacks<E extends Event> = LinkOnClickCallbacksBase<E>;

export function makeLinkOnClickFn<E extends Event>(
	callbacks: LinkOnClickCallbacks<E>,
) {
	return async (e: E) => {
		if (e.defaultPrevented) {
			return;
		}

		const anchorDetails = getAnchorDetailsFromEvent(e as unknown as MouseEvent);
		if (!anchorDetails) {
			return;
		}

		const { anchor, isEligibleForDefaultPrevention, isInternal } = anchorDetails;

		if (!anchor) {
			return;
		}

		if (isJustAHashChange(anchorDetails)) {
			saveScrollState();
			return;
		}

		if (isEligibleForDefaultPrevention && isInternal) {
			e.preventDefault();

			await callbacks.beforeBegin?.(e);

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

			await callbacks.beforeRender?.(e);

			await __completeNavigation(res);

			await callbacks.afterRender?.(e);
		}
	};
}

/////////////////////////////////////////////////////////////////////
/////// SCROLL STATE
/////////////////////////////////////////////////////////////////////

export function applyScrollState(state?: ScrollState) {
	if (!state) {
		const id = window.location.hash.slice(1);
		if (id) {
			window.document.getElementById(id)?.scrollIntoView();
		}
		return;
	}
	if ("hash" in state) {
		if (state.hash) {
			document.getElementById(state.hash)?.scrollIntoView();
		}
	} else {
		window.scrollTo(state.x, state.y);
	}
}
