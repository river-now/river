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
// TYPES
/////////////////////////////////////////////////////////////////////

export const RIVER_ROUTE_CHANGE_EVENT_KEY = "river:route-change";
export const STATUS_EVENT_KEY = "river:status";
const LOCATION_EVENT_KEY = "river:location";
const BUILD_ID_EVENT_KEY = "river:build-id";
const RIVER_HARD_RELOAD_QUERY_PARAM = "river_reload";

export type ScrollState = { x: number; y: number } | { hash: string };
export type RouteChangeEvent = CustomEvent<RouteChangeEventDetail>;
export type StatusEvent = CustomEvent<StatusEventDetail>;

type RouteChangeEventDetail = {
	scrollState?: ScrollState;
	index?: number;
};

type StatusEventDetail = {
	isNavigating: boolean;
	isSubmitting: boolean;
	isRevalidating: boolean;
};

type BuildIDEvent = {
	oldID: string;
	newID: string;
	fromGETAction: boolean;
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

type NavigationEntry = {
	control: NavigationControl;
	type: NavigationType;
};

type SubmissionEntry = {
	controller: AbortController;
	type: "submission";
};

type LinkOnClickCallback<E extends Event> = (event: E) => void | Promise<void>;

type LinkOnClickCallbacksBase<E extends Event> = {
	beforeBegin?: LinkOnClickCallback<E>;
	beforeRender?: LinkOnClickCallback<E>;
	afterRender?: LinkOnClickCallback<E>;
};

type LinkOnClickCallbacks<E extends Event> = LinkOnClickCallbacksBase<E>;

type GetPrefetchHandlersInput<E extends Event> = LinkOnClickCallbacksBase<E> & {
	href: string;
	delayMs?: number;
};

type PartialWaitFnJSON = Pick<
	GetRouteDataOutput,
	| "matchedPatterns"
	| "splatValues"
	| "params"
	| "hasRootData"
	| "loadersData"
	| "importURLs"
>;

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

/////////////////////////////////////////////////////////////////////
// NAVIGATION STATE MANAGER
/////////////////////////////////////////////////////////////////////

class NavigationStateManager {
	private navigations = new Map<string, NavigationEntry>();
	private activeUserNavigation: string | null = null;
	private submissions = new Map<string, SubmissionEntry>();
	private nonDedupedSubmissions = new Set<AbortController>();
	private lastDispatchedStatus: StatusEventDetail | null = null;
	private dispatchStatusEventDebounced: () => void;

	constructor() {
		this.dispatchStatusEventDebounced = debounce(() => {
			this.dispatchStatusEvent();
		}, 5);
	}

	// Navigation management
	addNavigation(href: string, entry: NavigationEntry): void {
		this.navigations.set(href, entry);
		this.scheduleStatusUpdate();
	}

	removeNavigation(href: string): void {
		this.navigations.delete(href);
		if (this.activeUserNavigation === href) {
			this.activeUserNavigation = null;
		}
		this.scheduleStatusUpdate();
	}

	getNavigation(href: string): NavigationEntry | undefined {
		return this.navigations.get(href);
	}

	hasNavigation(href: string): boolean {
		return this.navigations.has(href);
	}

	getNavigationsSize(): number {
		return this.navigations.size;
	}

	setActiveUserNavigation(href: string | null): void {
		this.activeUserNavigation = href;
	}

	getActiveUserNavigation(): string | null {
		return this.activeUserNavigation;
	}

	abortAllNavigationsExcept(excludeHref?: string): void {
		for (const [href, nav] of this.navigations.entries()) {
			if (href !== excludeHref) {
				nav.control.abortController?.abort();
				this.navigations.delete(href);
			}
		}
		this.scheduleStatusUpdate();
	}

	// Submission management
	addSubmission(key: string, entry: SubmissionEntry): void {
		this.submissions.set(key, entry);
		this.scheduleStatusUpdate();
	}

	removeSubmission(key: string): void {
		this.submissions.delete(key);
		this.scheduleStatusUpdate();
	}

	getSubmission(key: string): SubmissionEntry | undefined {
		return this.submissions.get(key);
	}

	addNonDedupedSubmission(controller: AbortController): void {
		this.nonDedupedSubmissions.add(controller);
		this.scheduleStatusUpdate();
	}

	removeNonDedupedSubmission(controller: AbortController): void {
		this.nonDedupedSubmissions.delete(controller);
		this.scheduleStatusUpdate();
	}

	// Status management
	getStatus(): StatusEventDetail {
		const navigations = Array.from(this.navigations.values());
		const hasActiveSubmissions =
			this.submissions.size > 0 || this.nonDedupedSubmissions.size > 0;

		return {
			isNavigating: navigations.some(
				(nav) => nav.type !== "prefetch" && nav.type !== "revalidation",
			),
			isSubmitting: hasActiveSubmissions,
			isRevalidating: navigations.some(
				(nav) => nav.type === "revalidation",
			),
		};
	}

	private scheduleStatusUpdate(): void {
		this.dispatchStatusEventDebounced();
	}

	private dispatchStatusEvent(): void {
		const newStatus = this.getStatus();

		if (jsonDeepEquals(this.lastDispatchedStatus, newStatus)) {
			return;
		}

		this.lastDispatchedStatus = newStatus;
		window.dispatchEvent(
			new CustomEvent(STATUS_EVENT_KEY, { detail: newStatus }),
		);
	}

	// Getters for backwards compatibility
	getNavigations(): Map<string, NavigationEntry> {
		return this.navigations;
	}

	getSubmissions(): Map<string, SubmissionEntry> {
		return this.submissions;
	}

	clearAll(): void {
		this.navigations.clear();
		this.submissions.clear();
		this.nonDedupedSubmissions.clear();
		this.activeUserNavigation = null;
		this.scheduleStatusUpdate();
	}
}

// Global instance
export const navigationStateManager = new NavigationStateManager();

// Export for backwards compatibility
export const navigationState = {
	get navigations() {
		return navigationStateManager.getNavigations();
	},
	get submissions() {
		return navigationStateManager.getSubmissions();
	},
	get activeUserNavigation() {
		return navigationStateManager.getActiveUserNavigation();
	},
	set activeUserNavigation(value: string | null) {
		navigationStateManager.setActiveUserNavigation(value);
	},
};

/////////////////////////////////////////////////////////////////////
// SCROLL STATE MANAGER
/////////////////////////////////////////////////////////////////////

class ScrollStateManager {
	private readonly STORAGE_KEY = "__river__scrollStateMap";
	private readonly PAGE_REFRESH_KEY = "__river__pageRefreshScrollState";
	private readonly MAX_ENTRIES = 50;

	saveState(key: string, state: ScrollState): void {
		const map = this.getMap();
		map.set(key, state);

		// Enforce size limit
		if (map.size > this.MAX_ENTRIES) {
			const firstKey = map.keys().next().value;
			if (firstKey) map.delete(firstKey);
		}

		this.saveMap(map);
	}

	getState(key: string): ScrollState | undefined {
		return this.getMap().get(key);
	}

	savePageRefreshState(): void {
		const state = {
			x: window.scrollX,
			y: window.scrollY,
			unix: Date.now(),
			href: window.location.href,
		};
		sessionStorage.setItem(this.PAGE_REFRESH_KEY, JSON.stringify(state));
	}

	restorePageRefreshState(): void {
		const stored = sessionStorage.getItem(this.PAGE_REFRESH_KEY);
		if (!stored) return;

		try {
			const state = JSON.parse(stored);
			if (
				state.href === window.location.href &&
				Date.now() - state.unix < 5000
			) {
				sessionStorage.removeItem(this.PAGE_REFRESH_KEY);
				window.requestAnimationFrame(() => {
					applyScrollState({ x: state.x, y: state.y });
				});
			}
		} catch {}
	}

	private getMap(): Map<string, ScrollState> {
		const stored = sessionStorage.getItem(this.STORAGE_KEY);
		if (!stored) return new Map();

		try {
			return new Map(JSON.parse(stored));
		} catch {
			return new Map();
		}
	}

	private saveMap(map: Map<string, ScrollState>): void {
		sessionStorage.setItem(
			this.STORAGE_KEY,
			JSON.stringify(Array.from(map.entries())),
		);
	}
}

const scrollStateManager = new ScrollStateManager();

/////////////////////////////////////////////////////////////////////
// HISTORY MANAGER
/////////////////////////////////////////////////////////////////////

class HistoryManager {
	private static instance: historyInstance;
	private static lastKnownLocation: typeof HistoryManager.instance.location;

	static getInstance(): historyInstance {
		if (!this.instance) {
			this.instance =
				createBrowserHistory() as unknown as historyInstance;
			this.lastKnownLocation = this.instance.location;
		}
		return this.instance;
	}

	static getLastKnownLocation() {
		return this.lastKnownLocation;
	}

	static updateLastKnownLocation(
		location: typeof HistoryManager.instance.location,
	) {
		this.lastKnownLocation = location;
	}

	static init(): void {
		const instance = this.getInstance();
		instance.listen(customHistoryListener as unknown as historyListener);
		this.setManualScrollRestoration();
	}

	private static setManualScrollRestoration(): void {
		if (
			history.scrollRestoration &&
			history.scrollRestoration !== "manual"
		) {
			history.scrollRestoration = "manual";
		}
	}
}

/////////////////////////////////////////////////////////////////////
// ASSET MANAGER
/////////////////////////////////////////////////////////////////////

class AssetManager {
	static preloadModule(url: string): void {
		const href = resolvePublicHref(url);
		if (document.querySelector(`link[href="${CSS.escape(href)}"]`)) {
			return;
		}

		const link = document.createElement("link");
		link.rel = "modulepreload";
		link.href = href;
		document.head.appendChild(link);
	}

	static preloadCSS(url: string): Promise<void> {
		const href = resolvePublicHref(url);

		const link = document.createElement("link");
		link.rel = "preload";
		link.setAttribute("as", "style");
		link.href = href;

		document.head.appendChild(link);

		return new Promise((resolve, reject) => {
			link.onload = () => resolve();
			link.onerror = reject;
		});
	}

	static applyCSS(bundles: string[]): void {
		window.requestAnimationFrame(() => {
			const prefix = internal_RiverClientGlobal.get("publicPathPrefix");

			for (const bundle of bundles) {
				// Check using the data attribute without escaping
				if (
					document.querySelector(
						`link[data-river-css-bundle="${bundle}"]`,
					)
				) {
					continue;
				}

				const link = document.createElement("link");
				link.rel = "stylesheet";
				link.href = prefix + bundle;
				link.setAttribute("data-river-css-bundle", bundle);
				document.head.appendChild(link);
			}
		});
	}
}

/////////////////////////////////////////////////////////////////////
// COMPONENT LOADER
/////////////////////////////////////////////////////////////////////

class ComponentLoader {
	static async loadComponents(
		importURLs: string[],
	): Promise<Map<string, any>> {
		const dedupedURLs = [...new Set(importURLs)];
		const modules = await Promise.all(
			dedupedURLs.map(async (url) => {
				if (!url) return undefined;
				return import(/* @vite-ignore */ resolvePublicHref(url));
			}),
		);

		return new Map(dedupedURLs.map((url, i) => [url, modules[i]]));
	}

	static async handleComponents(importURLs: string[]): Promise<void> {
		const modulesMap = await this.loadComponents(importURLs);
		const originalImportURLs = internal_RiverClientGlobal.get("importURLs");
		const exportKeys = internal_RiverClientGlobal.get("exportKeys") ?? [];

		// Set active components
		const activeComponents = originalImportURLs.map((url, i) => {
			const module = modulesMap.get(url);
			const key = exportKeys[i] ?? "default";
			return module?.[key] ?? null;
		});
		internal_RiverClientGlobal.set("activeComponents", activeComponents);

		// Handle error boundary
		const errorIdx = internal_RiverClientGlobal.get("outermostErrorIdx");
		if (errorIdx != null) {
			const errorModuleURL = originalImportURLs[errorIdx];
			let errorComponent;

			if (errorModuleURL) {
				const errorModule = modulesMap.get(errorModuleURL);
				const errorKey =
					internal_RiverClientGlobal.get("errorExportKey");
				if (errorKey && errorModule) {
					errorComponent = errorModule[errorKey];
				}
			}

			internal_RiverClientGlobal.set(
				"activeErrorBoundary",
				errorComponent ??
					internal_RiverClientGlobal.get("defaultErrorBoundary"),
			);
		}
	}
}

/////////////////////////////////////////////////////////////////////
// NAVIGATION CORE
/////////////////////////////////////////////////////////////////////

export async function __navigate(props: NavigateProps): Promise<void> {
	const control = beginNavigation(props);
	if (!control.promise) return;

	const result = await control.promise;
	if (!result) return;

	await __completeNavigation(result);
}

export function beginNavigation(props: NavigateProps): NavigationControl {
	// Handle user navigation specifics
	if (props.navigationType === "userNavigation") {
		navigationStateManager.abortAllNavigationsExcept(props.href);
		navigationStateManager.setActiveUserNavigation(props.href);

		// Check for existing prefetch to upgrade
		const existing = navigationStateManager.getNavigation(props.href);
		if (existing && existing.type === "prefetch") {
			existing.type = "userNavigation";
			// Update the navigation props to reflect the upgrade
			const originalPromise = existing.control.promise;
			existing.control.promise = originalPromise.then((result) => {
				if (result && !("redirectData" in result)) {
					// Update the navigation type in the result props
					return {
						...result,
						props: {
							...result.props,
							navigationType: "userNavigation" as NavigationType,
						},
					};
				}
				return result;
			});
			return existing.control;
		}
	}

	// Handle prefetch deduplication
	if (props.navigationType === "prefetch") {
		const existing = navigationStateManager.getNavigation(props.href);
		if (existing) return existing.control;
	}

	// Create new navigation
	const controller = new AbortController();
	const control: NavigationControl = {
		abortController: controller,
		promise: __fetchRouteData(controller, props),
	};

	navigationStateManager.addNavigation(props.href, {
		control,
		type: props.navigationType,
	});

	return control;
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
			redirectCount: props.redirectCount,
		});

		const redirected = redirectData?.status === "did";
		const responseNotOK = !response?.ok && response?.status !== 304;

		if (redirected || !response || responseNotOK) {
			return;
		}

		if (redirectData?.status === "should") {
			return { response, redirectData, props };
		}

		const json = (await response.json()) as GetRouteDataOutput | undefined;
		if (!json) {
			throw new Error("No JSON response");
		}

		// Preload assets
		const depsToPreload = import.meta.env.DEV
			? [...new Set(json.importURLs)]
			: json.deps;

		for (const dep of depsToPreload ?? []) {
			if (dep) AssetManager.preloadModule(dep);
		}

		const buildID = getBuildIDFromResponse(response);
		const waitFnPromise = runWaitFns(json, buildID);
		const cssBundlePromises: Array<Promise<any>> = [];

		for (const bundle of json.cssBundles ?? []) {
			cssBundlePromises.push(AssetManager.preloadCSS(bundle));
		}

		return { response, json, props, cssBundlePromises, waitFnPromise };
	} catch (error) {
		if (!isAbortError(error)) {
			LogError("Navigation failed", error);
		}
	} finally {
		navigationStateManager.removeNavigation(props.href);
	}
}

async function __completeNavigation(result: NavigationResult): Promise<void> {
	if (!result) return;

	if ("redirectData" in result) {
		await effectuateRedirectDataResult(
			result.redirectData,
			result.props.redirectCount || 0,
		);
		return;
	}

	// Handle build ID change
	const oldID = internal_RiverClientGlobal.get("buildID");
	const newID = getBuildIDFromResponse(result.response);
	if (newID && newID !== oldID) {
		dispatchBuildIDEvent({ newID, oldID, fromGETAction: false });
	}

	// Wait for client loaders
	const clientLoadersData = await result.waitFnPromise;
	internal_RiverClientGlobal.set("clientLoadersData", clientLoadersData);

	// For revalidation, check if we're still on the page we're revalidating
	if (result.props.navigationType === "revalidation") {
		const revalidatingUrl = new URL(
			result.props.href,
			window.location.href,
		);
		const currentUrl = new URL(window.location.href);

		if (
			revalidatingUrl.pathname !== currentUrl.pathname ||
			revalidatingUrl.search !== currentUrl.search
		) {
			// We've navigated away, skip rendering
			return;
		}
	}

	try {
		await __reRenderApp({
			json: result.json,
			navigationType: result.props.navigationType,
			runHistoryOptions: result.props,
			cssBundlePromises: result.cssBundlePromises,
		});
	} catch (error) {
		handleNavError(error, result.props);
	}
}

async function __reRenderApp(props: RerenderAppProps): Promise<void> {
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
	} else {
		await __reRenderAppInner(props);
	}
}

async function __reRenderAppInner(props: RerenderAppProps): Promise<void> {
	const { json, navigationType, runHistoryOptions, cssBundlePromises } =
		props;

	// Update global state
	const stateKeys = [
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
	] as const;

	for (const key of stateKeys) {
		internal_RiverClientGlobal.set(key, json[key]);
	}

	// Load components
	await ComponentLoader.handleComponents(json.importURLs);

	// Handle history and scroll
	let scrollStateToDispatch: ScrollState | undefined;

	if (runHistoryOptions) {
		const { href, scrollStateToRestore, replace } = runHistoryOptions;
		const hash = href.split("#")[1];
		const history = HistoryManager.getInstance();

		if (
			navigationType === "userNavigation" ||
			navigationType === "redirect"
		) {
			const target = new URL(href, window.location.href).href;
			const current = new URL(window.location.href).href;

			if (target !== current && !replace) {
				history.push(href);
			} else {
				history.replace(href);
			}

			scrollStateToDispatch = hash ? { hash } : { x: 0, y: 0 };
		}

		if (navigationType === "browserHistory") {
			scrollStateToDispatch =
				scrollStateToRestore ?? (hash ? { hash } : undefined);
		}
	}

	// Update title
	if (json.title?.dangerousInnerHTML) {
		const temp = document.createElement("textarea");
		temp.innerHTML = json.title.dangerousInnerHTML;
		if (document.title !== temp.value) {
			document.title = temp.value;
		}
	}

	// Wait for CSS
	if (cssBundlePromises.length > 0) {
		try {
			LogInfo("Waiting for CSS bundle preloads to complete...");
			await Promise.all(cssBundlePromises);
			LogInfo("CSS bundle preloads completed.");
		} catch (error) {
			LogError("Error preloading CSS bundles:", error);
		}
	}

	// Apply CSS
	if (json.cssBundles) {
		AssetManager.applyCSS(json.cssBundles);
	}

	// Dispatch route change event
	window.dispatchEvent(
		new CustomEvent(RIVER_ROUTE_CHANGE_EVENT_KEY, {
			detail: { scrollState: scrollStateToDispatch },
		}),
	);

	// Update head elements
	updateHeadEls("meta", json.metaHeadEls ?? []);
	updateHeadEls("rest", json.restHeadEls ?? []);
}

/////////////////////////////////////////////////////////////////////
// REDIRECTS
/////////////////////////////////////////////////////////////////////

async function effectuateRedirectDataResult(
	redirectData: RedirectData,
	redirectCount: number,
): Promise<RedirectData | null> {
	if (redirectData.status !== "should") return null;

	if (redirectData.shouldRedirectStrategy === "hard") {
		if (!redirectData.hrefDetails.isHTTP) return null;

		if (redirectData.hrefDetails.isExternal) {
			window.location.href = redirectData.href;
		} else {
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

	return null;
}

export async function handleRedirects(props: {
	abortController: AbortController;
	url: URL;
	requestInit?: RequestInit;
	isPrefetch?: boolean;
	redirectCount?: number;
}): Promise<{ redirectData: RedirectData | null; response?: Response }> {
	const MAX_REDIRECTS = 10;
	const redirectCount = props.redirectCount || 0;

	if (redirectCount >= MAX_REDIRECTS) {
		LogError("Too many redirects");
		return { redirectData: null, response: undefined };
	}

	// Prepare request
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
	headers.set("X-Accepts-Client-Redirect", "1");
	bodyParentObj.headers = headers;

	const finalRequestInit = {
		signal: props.abortController.signal,
		...props.requestInit,
		...bodyParentObj,
	};

	// Execute request
	const res = await fetch(props.url, finalRequestInit);
	let redirectData = parseFetchResponseForRedirectData(finalRequestInit, res);

	if (props.isPrefetch || !redirectData || redirectData.status === "did") {
		return { redirectData, response: res };
	}

	redirectData = await effectuateRedirectDataResult(
		redirectData,
		redirectCount,
	);
	return { redirectData, response: res };
}

/////////////////////////////////////////////////////////////////////
// SUBMISSIONS
/////////////////////////////////////////////////////////////////////

export async function submit<T = any>(
	url: string | URL,
	requestInit?: RequestInit,
	options?: { dedupeKey?: string },
): Promise<{ success: true; data: T } | { success: false; error: string }> {
	const submitRes = await submitInner(url, requestInit, options);
	const isGET = getIsGETRequest(requestInit);
	const needsRevalidation = !submitRes.alreadyRevalidated && !isGET;

	if (needsRevalidation) {
		await revalidate();
	}

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
	_requestInit?: RequestInit,
	options?: { dedupeKey?: string },
): Promise<
	(
		| { success: true; response: Response }
		| { success: false; error: string }
	) & {
		alreadyRevalidated: boolean;
	}
> {
	const requestInit = _requestInit || {};
	let abortController: AbortController;
	let submissionKey: string | undefined;
	let didAbort = false;

	// Handle deduplication
	if (options?.dedupeKey) {
		const urlStr = typeof url === "string" ? url : url.href;
		submissionKey = `${urlStr}${requestInit?.method || ""}${options.dedupeKey}`;

		const existing = navigationStateManager.getSubmission(submissionKey);
		if (existing) {
			existing.controller.abort();
			navigationStateManager.removeSubmission(submissionKey);
			didAbort = true;
		}

		abortController = new AbortController();
		navigationStateManager.addSubmission(submissionKey, {
			controller: abortController,
			type: "submission",
		});
	} else {
		abortController = new AbortController();
		navigationStateManager.addNonDedupedSubmission(abortController);
	}

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

		// Handle build ID
		const oldID = internal_RiverClientGlobal.get("buildID");
		const newID = getBuildIDFromResponse(response);
		if (newID && newID !== oldID) {
			dispatchBuildIDEvent({ newID, oldID, fromGETAction: isGET });
		}

		const redirected = redirectData?.status === "did";

		// Clean up
		if (submissionKey !== undefined) {
			navigationStateManager.removeSubmission(submissionKey);
		} else {
			navigationStateManager.removeNonDedupedSubmission(abortController);
		}

		// Handle response
		if (response && getIsErrorRes(response)) {
			return {
				success: false,
				error: String(response.status),
				alreadyRevalidated: redirected,
			};
		}

		if (didAbort) {
			return {
				success: false,
				error: "Aborted",
				alreadyRevalidated: false,
			};
		}

		if (!response?.ok) {
			return {
				success: false,
				error: String(response?.status || "unknown"),
				alreadyRevalidated: redirected,
			};
		}

		return {
			success: true,
			response,
			alreadyRevalidated: redirected,
		};
	} catch (error) {
		// Clean up
		if (submissionKey !== undefined) {
			navigationStateManager.removeSubmission(submissionKey);
		} else {
			navigationStateManager.removeNonDedupedSubmission(abortController);
		}

		if (isAbortError(error)) {
			return {
				success: false,
				error: "Aborted",
				alreadyRevalidated: false,
			};
		}

		LogError(error);
		return {
			success: false,
			error: error instanceof Error ? error.message : "Unknown error",
			alreadyRevalidated: false,
		};
	}
}

/////////////////////////////////////////////////////////////////////
// PREFETCH
/////////////////////////////////////////////////////////////////////

export function getPrefetchHandlers<E extends Event>(
	input: GetPrefetchHandlersInput<E>,
) {
	const hrefDetails = getHrefDetails(input.href);
	if (
		!hrefDetails.isHTTP ||
		!hrefDetails.relativeURL ||
		hrefDetails.isExternal
	) {
		return;
	}

	let timer: number | undefined;
	let currentNav: NavigationControl | null = null;
	let prerenderResult: NavigationResult | null = null;
	const delayMs = input.delayMs ?? 100;

	async function prefetch(e: E): Promise<void> {
		if (currentNav || !hrefDetails.isHTTP) return;

		// Don't prefetch current page
		const currentUrl = new URL(window.location.href);
		const targetUrl = hrefDetails.url;
		currentUrl.hash = "";
		targetUrl.hash = "";
		if (currentUrl.href === targetUrl.href) return;

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
		if (currentNav) return;
		timer = window.setTimeout(() => prefetch(e), delayMs);
	}

	function stop(): void {
		if (timer) {
			clearTimeout(timer);
		}

		if (!hrefDetails.isHTTP) return;

		const nav = navigationStateManager.getNavigation(
			hrefDetails.relativeURL,
		);
		if (nav?.type === "prefetch") {
			nav.control.abortController?.abort();
			navigationStateManager.removeNavigation(hrefDetails.relativeURL);
		}

		currentNav = null;
		prerenderResult = null;
	}

	async function onClick(e: E): Promise<void> {
		if (e.defaultPrevented || !hrefDetails.isHTTP) return;

		const anchorDetails = getAnchorDetailsFromEvent(
			e as unknown as MouseEvent,
		);
		if (!anchorDetails) return;

		const { isEligibleForDefaultPrevention, isInternal } = anchorDetails;
		if (!isEligibleForDefaultPrevention || !isInternal) return;

		if (isJustAHashChange(anchorDetails)) {
			saveScrollState();
			return;
		}

		e.preventDefault();

		if (prerenderResult) {
			await finalize(e);
			return;
		}

		if (timer) clearTimeout(timer);

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
					throw new Error(
						"Invalid navigation result: no JSON response.",
					);
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

	return {
		...hrefDetails,
		start,
		stop,
		onClick,
	};
}

/////////////////////////////////////////////////////////////////////
// PUBLIC API
/////////////////////////////////////////////////////////////////////

export async function navigate(
	href: string,
	options?: { replace?: boolean },
): Promise<void> {
	await __navigate({
		href,
		navigationType: "userNavigation",
		replace: options?.replace,
	});
}

export const revalidate = debounce(async () => {
	await __navigate({
		href: window.location.href,
		navigationType: "revalidation",
	});
}, 10);

export function getHistoryInstance(): historyInstance {
	return HistoryManager.getInstance();
}

export function getStatus(): StatusEventDetail {
	return navigationStateManager.getStatus();
}

export function getLocation() {
	return {
		pathname: window.location.pathname,
		search: window.location.search,
		hash: window.location.hash,
	};
}

export function getBuildID(): string {
	return internal_RiverClientGlobal.get("buildID");
}

export function getRootEl(): HTMLDivElement {
	return document.getElementById("river-root") as HTMLDivElement;
}

export function applyScrollState(state?: ScrollState): void {
	if (!state) {
		const id = window.location.hash.slice(1);
		if (id) {
			document.getElementById(id)?.scrollIntoView();
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

export function makeLinkOnClickFn<E extends Event>(
	callbacks: LinkOnClickCallbacks<E>,
) {
	return async (e: E) => {
		if (e.defaultPrevented) return;

		const anchorDetails = getAnchorDetailsFromEvent(
			e as unknown as MouseEvent,
		);
		if (!anchorDetails) return;

		const { anchor, isEligibleForDefaultPrevention, isInternal } =
			anchorDetails;
		if (!anchor) return;

		if (isJustAHashChange(anchorDetails)) {
			saveScrollState();
			return;
		}

		if (isEligibleForDefaultPrevention && isInternal) {
			e.preventDefault();

			await callbacks.beforeBegin?.(e);

			const control = beginNavigation({
				href: anchor.href,
				navigationType: "userNavigation",
			});

			if (!control.promise) return;

			const res = await control.promise;
			if (!res) return;

			await callbacks.beforeRender?.(e);
			await __completeNavigation(res);
			await callbacks.afterRender?.(e);
		}
	};
}

/////////////////////////////////////////////////////////////////////
// EVENT LISTENERS
/////////////////////////////////////////////////////////////////////

export const addStatusListener =
	makeListenerAdder<StatusEventDetail>(STATUS_EVENT_KEY);
export const addRouteChangeListener = makeListenerAdder<RouteChangeEventDetail>(
	RIVER_ROUTE_CHANGE_EVENT_KEY,
);
export const addLocationListener = makeListenerAdder<void>(LOCATION_EVENT_KEY);
export const addBuildIDListener =
	makeListenerAdder<BuildIDEvent>(BUILD_ID_EVENT_KEY);

function makeListenerAdder<T>(key: string) {
	return function addListener(
		listener: (event: CustomEvent<T>) => void,
	): () => void {
		window.addEventListener(key, listener as any);
		return () => window.removeEventListener(key, listener as any);
	};
}

/////////////////////////////////////////////////////////////////////
// INITIALIZATION
/////////////////////////////////////////////////////////////////////

export async function initClient(
	renderFn: () => void,
	options?: {
		defaultErrorBoundary?: RouteErrorComponent;
		useViewTransitions?: boolean;
	},
): Promise<void> {
	// Set options
	if (options?.defaultErrorBoundary) {
		internal_RiverClientGlobal.set(
			"defaultErrorBoundary",
			options.defaultErrorBoundary,
		);
	} else {
		internal_RiverClientGlobal.set(
			"defaultErrorBoundary",
			defaultErrorBoundary,
		);
	}

	if (options?.useViewTransitions) {
		internal_RiverClientGlobal.set("useViewTransitions", true);
	}

	// Initialize history
	HistoryManager.init();

	// Clean URL
	const url = new URL(window.location.href);
	if (url.searchParams.has(RIVER_HARD_RELOAD_QUERY_PARAM)) {
		url.searchParams.delete(RIVER_HARD_RELOAD_QUERY_PARAM);
		HistoryManager.getInstance().replace(url.href);
	}

	// Load initial components
	await ComponentLoader.handleComponents(
		internal_RiverClientGlobal.get("importURLs"),
	);

	// Setup client loaders
	await setupClientLoaders();

	// Render
	renderFn();

	// Restore scroll
	scrollStateManager.restorePageRefreshState();

	// Touch detection
	window.addEventListener(
		"touchstart",
		() => {
			LogInfo("Touch device detected");
			internal_RiverClientGlobal.set("isTouchDevice", true);
		},
		{ once: true },
	);
}

export function initCustomHistory(): void {
	HistoryManager.init();
}

export async function customHistoryListener({
	action,
	location,
}: Update): Promise<void> {
	const lastKnownLocation = HistoryManager.getLastKnownLocation();

	if (location.key !== lastKnownLocation.key) {
		dispatchLocationEvent();
	}

	const popWithinSameDoc =
		action === "POP" &&
		location.pathname === lastKnownLocation.pathname &&
		location.search === lastKnownLocation.search;

	const removingHash =
		popWithinSameDoc && lastKnownLocation.hash && !location.hash;
	const addingHash =
		popWithinSameDoc && !lastKnownLocation.hash && location.hash;
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
			const stored = scrollStateManager.getState(location.key);
			applyScrollState(stored ?? { x: 0, y: 0 });
		}

		if (!popWithinSameDoc) {
			await __navigate({
				href: window.location.href,
				navigationType: "browserHistory",
				scrollStateToRestore: scrollStateManager.getState(location.key),
			});
		}
	}

	HistoryManager.updateLastKnownLocation(location);
}

/////////////////////////////////////////////////////////////////////
// HELPER FUNCTIONS
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

export async function setupClientLoaders(): Promise<void> {
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

async function runWaitFns(
	json: PartialWaitFnJSON,
	buildID: string,
): Promise<Array<any>> {
	await ComponentLoader.loadComponents(json.importURLs);

	const matchedPatterns = json.matchedPatterns ?? [];
	const patternToWaitFnMap =
		internal_RiverClientGlobal.get("patternToWaitFnMap");
	const waitFnPromises: Array<Promise<any>> = [];

	let i = 0;
	for (const pattern of matchedPatterns) {
		if (patternToWaitFnMap[pattern]) {
			waitFnPromises.push(
				patternToWaitFnMap[pattern](
					resolveWaitFnPropsFromJSON(json, buildID, i),
				),
			);
		} else {
			waitFnPromises.push(Promise.resolve());
		}
		i++;
	}

	return Promise.all(waitFnPromises);
}

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

function isJustAHashChange(
	anchorDetails: ReturnType<typeof getAnchorDetailsFromEvent>,
): boolean {
	if (!anchorDetails) return false;

	const { pathname, search, hash } = new URL(
		anchorDetails.anchor.href,
		window.location.href,
	);

	return !!(
		hash &&
		pathname === window.location.pathname &&
		search === window.location.search
	);
}

function saveScrollState(): void {
	const lastKnownLocation = HistoryManager.getLastKnownLocation();
	scrollStateManager.saveState(lastKnownLocation.key, {
		x: window.scrollX,
		y: window.scrollY,
	});
}

function handleNavError(error: unknown, props: NavigateProps): void {
	if (!isAbortError(error)) {
		LogError(error);
	}
}

function dispatchLocationEvent(): void {
	window.dispatchEvent(new CustomEvent(LOCATION_EVENT_KEY));
}

function dispatchBuildIDEvent(detail: BuildIDEvent): void {
	internal_RiverClientGlobal.set("buildID", detail.newID);
	window.dispatchEvent(new CustomEvent(BUILD_ID_EVENT_KEY, { detail }));
}

const defaultErrorBoundary: RouteErrorComponent = (props: {
	error: string;
}) => {
	return "Route Error: " + props.error;
};

// Legacy function for backwards compatibility
export function setLoadingStatus({
	type,
	value,
}: {
	type: NavigationType | "submission";
	value: boolean;
}) {
	// This function is no longer used internally
	// Status is now derived from navigationStateManager
}

// Setup beforeunload handler for scroll restoration
window.addEventListener("beforeunload", () => {
	scrollStateManager.savePageRefreshState();
});
