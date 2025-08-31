/// <reference types="vite/client" />

import { createBrowserHistory, type Update } from "history";
import { debounce } from "river.now/kit/debounce";
import { jsonDeepEquals } from "river.now/kit/json";
import type { PatternRegistry } from "river.now/kit/matcher/register";
import {
	getAnchorDetailsFromEvent,
	getHrefDetails,
	getIsGETRequest,
} from "river.now/kit/url";
import type { APIConfig } from "./api_client_helpers.ts";
import { updateHeadEls } from "./head.ts";
import type { historyInstance, historyListener } from "./history_types.ts";
import {
	getBuildIDFromResponse,
	parseFetchResponseForRedirectData,
	type RedirectData,
} from "./redirects.ts";
import {
	type ClientLoaderAwaitedServerData,
	type GetRouteDataOutput,
	internal_RiverClientGlobal,
	type RouteErrorComponent,
} from "./river_ctx.ts";
import { isAbortError, LogError } from "./utils.ts";

// The client loaders matcher system is imported dynamically,
// so it will only increase your bundle size if you actually
// use them. If you do use them, however, this unlocks loading
// discovered client loaders in parallel with the server loaders.
// The first time any client loader is discovered, it will
// necessarily be serial (it wasn't discovered before). But all
// subsequent runs will be parallel. This pattern means we do not
// need to ship a (potentially massive) routes manifest to the client.

let clientPatternRegistry: PatternRegistry | undefined;
let matcherModules:
	| {
			register: typeof import("river.now/kit/matcher/register");
			findNested: typeof import("river.now/kit/matcher/find-nested");
	  }
	| undefined;
let initializationPromise: Promise<void> | undefined;

async function ensureMatcherLoaded(config: APIConfig) {
	if (!initializationPromise) {
		initializationPromise = (async () => {
			if (!matcherModules) {
				const [registerModule, findNestedModule] = await Promise.all([
					import("river.now/kit/matcher/register"),
					import("river.now/kit/matcher/find-nested"),
				]);
				matcherModules = {
					register: registerModule,
					findNested: findNestedModule,
				};
				const { createPatternRegistry } = registerModule;
				clientPatternRegistry = createPatternRegistry({
					dynamicParamPrefixRune: config.loadersDynamicRune,
					splatSegmentRune: config.loadersSplatRune,
					explicitIndexSegment: config.loadersExplicitIndexSegment,
				});
			}
		})();
	}

	await initializationPromise;

	return {
		matcherModules: matcherModules!,
		clientPatternRegistry: clientPatternRegistry!,
	};
}

export async function registerClientLoaderPattern(
	pattern: string,
): Promise<void> {
	// This is called when a client loader is discovered.
	// Load both matcher modules on first use.
	const config = internal_RiverClientGlobal.get("apiConfig");
	const { matcherModules, clientPatternRegistry } =
		await ensureMatcherLoaded(config);
	matcherModules.register.registerPattern(clientPatternRegistry, pattern);
}

// This is needed because the matcher, by definition, will only
// match when you have a full path match. If the path you are
// testing is longer than the registered patterns, you will get
// no match, even if some registered patterns would potentially
// be in the parent segments. This fixes that.
async function findPartialMatchesOnClient(pathname: string) {
	// Only try to match if we have client loaders
	const patternToWaitFnMap =
		internal_RiverClientGlobal.get("patternToWaitFnMap");
	if (Object.keys(patternToWaitFnMap).length === 0) {
		return null;
	}

	// If we have patterns registered, the modules should already be loaded
	if (!matcherModules || !clientPatternRegistry) {
		return null;
	}

	const { findNestedMatches } = matcherModules.findNested;

	// First try the full path
	const fullResult = findNestedMatches(clientPatternRegistry, pathname);
	if (fullResult) {
		// If we get a full match, we have everything we need
		return fullResult;
	}

	// If no full match, try progressively shorter paths to find partial matches
	const segments = pathname.split("/").filter(Boolean);

	// Try from longest to shortest
	for (let i = segments.length; i >= 0; i--) {
		const partialPath =
			i === 0 ? "/" : "/" + segments.slice(0, i).join("/");
		const result = findNestedMatches(clientPatternRegistry, partialPath);
		if (result) {
			return result; // First match is the longest
		}
	}

	return null;
}

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

export type StatusEventDetail = {
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
	| "prefetch"
	| "action";

export type NavigateProps = {
	href: string;
	navigationType: NavigationType;
	scrollStateToRestore?: ScrollState;
	replace?: boolean;
	redirectCount?: number;
	scrollToTop?: boolean;
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
	scrollToTop?: boolean;
	replace?: boolean;
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
		scrollToTop?: boolean;
	};
};

/////////////////////////////////////////////////////////////////////
// NAVIGATION STATE MANAGER
/////////////////////////////////////////////////////////////////////

// Navigation phases represent the lifecycle stages
type NavigationPhase =
	| "fetching" // Fetching route data
	| "waiting" // Waiting for assets/loaders
	| "rendering" // Applying changes to DOM
	| "complete"; // Navigation finished

// Navigation intent represents what should happen when complete
type NavigationIntent =
	| "none" // Prefetch -- don't navigate unless upgraded
	| "navigate" // Normal navigation -- update URL and render
	| "revalidate"; // Revalidation -- only update if still on same page

interface NavigationEntry {
	control: NavigationControl;
	type: NavigationType;
	intent: NavigationIntent;
	phase: NavigationPhase;
	startTime: number;
	targetUrl: string; // URL this navigation is targeting
	originUrl: string; // URL when navigation started (for revalidation)
	scrollToTop?: boolean;
	replace?: boolean;
}

interface SubmissionEntry {
	control: NavigationControl;
	startTime: number;
}

class NavigationStateManager {
	private _navigations = new Map<string, NavigationEntry>();
	private _submissions = new Map<string | symbol, SubmissionEntry>();
	private lastDispatchedStatus: StatusEventDetail | null = null;
	private dispatchStatusEventDebounced: () => void;
	private readonly REVALIDATION_COALESCE_MS = 5;

	constructor() {
		this.dispatchStatusEventDebounced = debounce(() => {
			this.dispatchStatusEvent();
		}, 5);
	}

	async navigate(props: NavigateProps): Promise<{ didNavigate: boolean }> {
		const control = this.beginNavigation(props);

		try {
			const result = await control.promise;
			if (!result) {
				return { didNavigate: false };
			}

			// Process based on navigation entry state
			const targetUrl = new URL(props.href, window.location.href).href;
			const entry = this._navigations.get(targetUrl);
			if (!entry) {
				return { didNavigate: false };
			}

			if (entry.intent === "none" && entry.type === "prefetch") {
				// Prefetch complete but not navigating
				this.transitionPhase(targetUrl, "complete");
				return { didNavigate: false };
			}

			if (entry.intent === "navigate" || entry.intent === "revalidate") {
				const now = Date.now();
				lastTriggeredNavOrRevalidateTimestampMS = now;
			}

			await this.processNavigationResult(result, entry);
		} catch (error) {
			const targetUrl = new URL(props.href, window.location.href).href;
			this.deleteNavigation(targetUrl);
			if (!isAbortError(error)) {
				LogError("Navigate error:", error);
			}
			return { didNavigate: false };
		}
		return { didNavigate: true };
	}

	beginNavigation(props: NavigateProps): NavigationControl {
		const existing = this._navigations.get(
			new URL(props.href, window.location.href).href,
		);

		switch (props.navigationType) {
			case "userNavigation":
				return this.beginUserNavigation(props, existing);
			case "prefetch":
				return this.beginPrefetch(props, existing);
			case "revalidation":
				return this.beginRevalidation(props);
			case "browserHistory":
			case "redirect":
			default:
				return this.createNavigation(props, "navigate");
		}
	}

	private beginUserNavigation(
		props: NavigateProps,
		existing: NavigationEntry | undefined,
	): NavigationControl {
		const targetUrl = new URL(props.href, window.location.href).href;

		// Abort all other navigations
		this.abortAllNavigationsExcept(targetUrl);

		if (existing) {
			if (existing.type === "prefetch") {
				// Upgrade prefetch to user navigation
				this.upgradeNavigation(targetUrl, {
					type: "userNavigation",
					intent: "navigate",
					scrollToTop: props.scrollToTop,
					replace: props.replace,
				});
				return existing.control;
			}

			// Already navigating to this URL, return existing
			return existing.control;
		}

		return this.createNavigation(props, "navigate");
	}

	private beginPrefetch(
		props: NavigateProps,
		existing: NavigationEntry | undefined,
	): NavigationControl {
		const targetUrl = new URL(props.href, window.location.href).href;

		if (existing) {
			return existing.control;
		}

		// Don't prefetch current page
		const currentUrl = new URL(window.location.href);
		const targetUrlObj = new URL(targetUrl);
		currentUrl.hash = "";
		targetUrlObj.hash = "";
		if (currentUrl.href === targetUrlObj.href) {
			// Return a no-op control
			return {
				abortController: new AbortController(),
				promise: Promise.resolve(undefined),
			};
		}

		return this.createNavigation(props, "none");
	}

	private beginRevalidation(props: NavigateProps): NavigationControl {
		// Store current URL to validate against later
		const currentUrl = window.location.href;

		// Check for recent revalidation to same URL
		const existing = this._navigations.get(currentUrl);
		if (
			existing?.type === "revalidation" &&
			Date.now() - existing.startTime < this.REVALIDATION_COALESCE_MS
		) {
			return existing.control;
		}

		// Abort other revalidations
		for (const [key, nav] of this._navigations.entries()) {
			if (nav.type === "revalidation") {
				nav.control.abortController?.abort();
				this.deleteNavigation(key);
			}
		}

		// Create revalidation with current URL
		return this.createNavigation(
			{ ...props, href: currentUrl },
			"revalidate",
		);
	}

	private createNavigation(
		props: NavigateProps,
		intent: NavigationIntent,
	): NavigationControl {
		const controller = new AbortController();
		const targetUrl = new URL(props.href, window.location.href).href;

		const entry: NavigationEntry = {
			control: {
				abortController: controller,
				promise: this.fetchRouteData(controller, props).catch(
					(error) => {
						this.deleteNavigation(targetUrl);
						throw error;
					},
				),
			},
			type: props.navigationType,
			intent,
			phase: "fetching",
			startTime: Date.now(),
			targetUrl,
			originUrl: window.location.href,
			scrollToTop: props.scrollToTop,
			replace: props.replace,
		};

		this.setNavigation(targetUrl, entry);
		return entry.control;
	}

	private upgradeNavigation(
		href: string,
		updates: Partial<
			Pick<NavigationEntry, "type" | "intent" | "scrollToTop" | "replace">
		>,
	): void {
		const existing = this._navigations.get(href);
		if (!existing) return;

		this.setNavigation(href, {
			...existing,
			...updates,
		});
	}

	private transitionPhase(href: string, phase: NavigationPhase): void {
		const existing = this._navigations.get(href);
		if (!existing) return;

		this.setNavigation(href, {
			...existing,
			phase,
		});
	}

	private async fetchRouteData(
		controller: AbortController,
		props: NavigateProps,
	): Promise<NavigationResult> {
		try {
			const url = new URL(props.href, window.location.href);
			url.searchParams.set(
				"river_json",
				internal_RiverClientGlobal.get("buildID") || "1",
			);

			if (props.navigationType === "revalidation") {
				const deploymentID =
					internal_RiverClientGlobal.get("deploymentID");
				if (deploymentID) {
					url.searchParams.set("dpl", deploymentID);
				}
			}

			// Start server fetch and immediately process the response to JSON
			const serverPromise = handleRedirects({
				abortController: controller,
				url,
				isPrefetch: props.navigationType === "prefetch",
				redirectCount: props.redirectCount,
			}).then(async (result) => {
				// Read the response body once and return both the original result and parsed JSON
				if (result.response && result.response.ok) {
					const json = await result.response.json();
					return { ...result, json };
				}
				return { ...result, json: undefined };
			});

			// Try to match routes on the client and start parallel loaders
			const pathname = url.pathname;
			const matchResult = await findPartialMatchesOnClient(pathname);
			const patternToWaitFnMap =
				internal_RiverClientGlobal.get("patternToWaitFnMap");
			const runningLoaders = new Map<string, Promise<any>>();

			// Start client loaders for already-registered patterns
			if (matchResult) {
				const { params, splatValues, matches } = matchResult;

				for (let i = 0; i < matches.length; i++) {
					const match = matches[i];
					if (!match) continue;

					const pattern = match.registeredPattern.originalPattern;
					const loaderFn = patternToWaitFnMap[pattern];

					if (loaderFn) {
						// Create a promise for this pattern's server data
						const serverDataPromise = serverPromise
							.then(
								({
									response,
									json,
								}): ClientLoaderAwaitedServerData<any, any> => {
									if (!response || !response.ok || !json) {
										return {
											matchedPatterns: [],
											loaderData: undefined,
											rootData: null,
											buildID: "1",
										};
									}
									const serverIdx =
										json.matchedPatterns?.indexOf(pattern);
									const loaderData =
										serverIdx !== -1 &&
										serverIdx !== undefined
											? json.loadersData[serverIdx]
											: undefined;
									const rootData = json.hasRootData
										? json.loadersData[0]
										: null;
									const buildID =
										getBuildIDFromResponse(response) || "1";
									return {
										matchedPatterns:
											json.matchedPatterns || [],
										loaderData,
										rootData,
										buildID,
									};
								},
							)
							.catch(() => ({
								matchedPatterns: [],
								loaderData: undefined,
								rootData: null,
								buildID: "1",
							}));

						const loaderPromise = loaderFn({
							params,
							splatValues,
							serverDataPromise,
							signal: controller.signal,
						}).catch((error: any) => {
							if (!isAbortError(error)) {
								LogError(
									`Client loader error for pattern ${pattern}:`,
									error,
								);
							}
							return undefined;
						});

						runningLoaders.set(pattern, loaderPromise);
					}
				}
			}

			// Wait for server response
			const { redirectData, response, json } = await serverPromise;

			const redirected = redirectData?.status === "did";
			const responseNotOK = !response?.ok && response?.status !== 304;

			if (redirected || !response) {
				// This is a valid end to a navigation attempt (e.g., a redirect occurred
				// or the request was aborted). It's not an error.
				controller.abort();
				return undefined;
			}

			if (responseNotOK) {
				// This is a server error. Throwing an exception allows our .catch()
				// blocks to handle cleanup and reset the loading state.
				controller.abort();
				throw new Error(`Fetch failed with status ${response.status}`);
			}

			if (redirectData?.status === "should") {
				controller.abort();
				return { response, redirectData, props };
			}

			if (!json) {
				controller.abort();
				throw new Error("No JSON response");
			}

			// deps are only present in prod because they stem from the rollup metafile
			// (same for CSS bundles -- vite handles them in dev)
			// so in dev, to get similar behavior, we use the importURLs
			// (which is a subset of what the deps would be in prod)
			const depsToPreload = import.meta.env.DEV
				? [...new Set(json.importURLs)]
				: json.deps;
			for (const dep of depsToPreload ?? []) {
				if (dep) AssetManager.preloadModule(dep);
			}

			const buildID = getBuildIDFromResponse(response);

			// Complete client loader execution
			const waitFnPromise = completeClientLoaders(
				json,
				buildID,
				runningLoaders,
				controller.signal,
			);

			const cssBundlePromises: Array<Promise<any>> = [];
			for (const bundle of json.cssBundles ?? []) {
				cssBundlePromises.push(AssetManager.preloadCSS(bundle));
			}

			return { response, json, props, cssBundlePromises, waitFnPromise };
		} catch (error) {
			if (!isAbortError(error)) {
				LogError("Navigation failed", error);
			}
			throw error;
		}
	}

	private async processNavigationResult(
		result: NavigationResult,
		entry: NavigationEntry,
	): Promise<void> {
		try {
			if (!result) return;

			if ("redirectData" in result) {
				// Clean up before redirect to prevent race conditions
				this.deleteNavigation(entry.targetUrl);

				await effectuateRedirectDataResult(
					result.redirectData,
					result.props.redirectCount || 0,
				);
				return;
			}

			// Type guard to ensure we have the json branch
			if (!("json" in result)) {
				LogError("Invalid navigation result: no JSON or redirect");
				return;
			}

			// Validate revalidation is still applicable
			if (entry.type === "revalidation") {
				const currentUrl = window.location.href;
				if (currentUrl !== entry.originUrl) {
					this.deleteNavigation(entry.targetUrl);
					return;
				}
			}

			// Transition to waiting phase
			this.transitionPhase(entry.targetUrl, "waiting");

			// Complete the navigation
			await this.completeNavigation(result, entry);
		} finally {
			// Always clean up
			this.deleteNavigation(entry.targetUrl);
		}
	}

	private async completeNavigation(
		result: NavigationResult & {
			json: GetRouteDataOutput;
			cssBundlePromises: Array<Promise<any>>;
			waitFnPromise: Promise<any> | undefined;
		},
		entry: NavigationEntry,
	): Promise<void> {
		// Skip if navigation was aborted
		if (!this._navigations.has(entry.targetUrl)) {
			return;
		}

		// Update build ID if needed
		const oldID = internal_RiverClientGlobal.get("buildID");
		const newID = getBuildIDFromResponse(result.response);
		if (newID && newID !== oldID) {
			dispatchBuildIDEvent({ newID, oldID, fromGETAction: false });
		}

		// Wait for client loaders
		const clientLoadersData = await result.waitFnPromise;
		internal_RiverClientGlobal.set("clientLoadersData", clientLoadersData);

		// Wait for CSS
		if (result.cssBundlePromises.length > 0) {
			try {
				await Promise.all(result.cssBundlePromises);
			} catch (error) {
				LogError("Error preloading CSS bundles:", error);
			}
		}

		// Skip rendering for prefetch without intent
		if (entry.intent === "none") {
			return;
		}

		// Skip rendering for revalidation if not on target page
		if (
			entry.type === "revalidation" &&
			window.location.href !== entry.originUrl
		) {
			return;
		}

		// Transition to rendering phase
		this.transitionPhase(entry.targetUrl, "rendering");

		// Render the app
		try {
			await __reRenderApp({
				json: result.json,
				navigationType: entry.type,
				runHistoryOptions:
					entry.intent === "navigate"
						? {
								href: entry.targetUrl,
								scrollStateToRestore:
									result.props.scrollStateToRestore,
								replace: entry.replace || result.props.replace,
								scrollToTop: entry.scrollToTop,
							}
						: undefined,
			});
		} catch (error) {
			if (!isAbortError(error)) {
				LogError("Error completing navigation", error);
			}
			throw error;
		}

		// Mark as complete
		this.transitionPhase(entry.targetUrl, "complete");
	}

	async submit<T = any>(
		url: string | URL,
		requestInit?: RequestInit,
		options?: SubmitOptions,
	): Promise<{ success: true; data: T } | { success: false; error: string }> {
		const abortController = new AbortController();
		const submissionKey = options?.dedupeKey
			? `submission:${options.dedupeKey}`
			: Symbol("submission");

		// Abort duplicate submission
		if (typeof submissionKey === "string") {
			const existing = this._submissions.get(submissionKey);
			if (existing) {
				existing.control.abortController?.abort("deduped");
			}
		}

		const entry: SubmissionEntry = {
			control: {
				abortController,
				promise: Promise.resolve() as any,
			},
			startTime: Date.now(),
		};

		this._submissions.set(submissionKey, entry);
		this.scheduleStatusUpdate();

		try {
			const urlToUse = new URL(url, window.location.href);
			const headers = new Headers(requestInit?.headers);
			const deploymentID = internal_RiverClientGlobal.get("deploymentID");
			if (deploymentID) {
				headers.set("x-deployment-id", deploymentID);
			}
			const finalRequestInit: RequestInit = {
				...requestInit,
				headers,
				signal: abortController.signal,
			};

			const { redirectData, response } = await handleRedirects({
				abortController,
				url: urlToUse,
				isPrefetch: false,
				redirectCount: 0,
				requestInit: finalRequestInit,
			});

			const oldID = internal_RiverClientGlobal.get("buildID");
			const newID = getBuildIDFromResponse(response);
			if (newID && newID !== oldID) {
				const isGET = getIsGETRequest(requestInit);
				dispatchBuildIDEvent({ newID, oldID, fromGETAction: isGET });
			}

			if (!response || !response.ok) {
				return {
					success: false,
					error: String(response?.status || "unknown"),
				};
			}

			if (redirectData?.status === "should") {
				await effectuateRedirectDataResult(redirectData, 0);
				return { success: true, data: undefined as T }; // No data on redirect
			}

			const data = await response.json();

			// Auto-revalidate for mutations
			const isGET = getIsGETRequest(requestInit);
			const redirected = redirectData?.status === "did";
			if (!isGET && !redirected && options?.revalidate !== false) {
				await revalidate();
			}

			return { success: true, data: data as T };
		} catch (error) {
			if (isAbortError(error)) {
				return { success: false, error: "Aborted" };
			}
			LogError(error);
			return {
				success: false,
				error: error instanceof Error ? error.message : "Unknown error",
			};
		} finally {
			this._submissions.delete(submissionKey);
			this.scheduleStatusUpdate();
		}
	}

	private setNavigation(key: string, entry: NavigationEntry): void {
		this._navigations.set(key, entry);
		this.scheduleStatusUpdate();
	}

	private deleteNavigation(key: string): boolean {
		const result = this._navigations.delete(key);
		if (result) {
			this.scheduleStatusUpdate();
		}
		return result;
	}

	removeNavigation(key: string): void {
		this.deleteNavigation(key);
	}

	getNavigation(key: string): NavigationEntry | undefined {
		return this._navigations.get(key);
	}

	hasNavigation(key: string): boolean {
		return this._navigations.has(key);
	}

	getNavigationsSize(): number {
		return this._navigations.size;
	}

	getNavigations(): Map<string, NavigationEntry> {
		return this._navigations;
	}

	private abortAllNavigationsExcept(excludeHref?: string): void {
		for (const [href, nav] of this._navigations.entries()) {
			if (href !== excludeHref) {
				nav.control.abortController?.abort();
				this.deleteNavigation(href);
			}
		}
	}

	getStatus(): StatusEventDetail {
		const navigations = Array.from(this._navigations.values());
		const submissions = Array.from(this._submissions.values());

		const isNavigating = navigations.some(
			(nav) => nav.intent === "navigate" && nav.phase !== "complete",
		);

		const isRevalidating = navigations.some(
			(nav) => nav.type === "revalidation" && nav.phase !== "complete",
		);

		const isSubmitting = submissions.length > 0;

		return { isNavigating, isSubmitting, isRevalidating };
	}

	clearAll(): void {
		for (const nav of this._navigations.values()) {
			nav.control.abortController?.abort();
		}
		this._navigations.clear();
		for (const sub of this._submissions.values()) {
			sub.control.abortController?.abort();
		}
		this._submissions.clear();
		this.scheduleStatusUpdate();
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
}

// Global instance
export const navigationStateManager = new NavigationStateManager();

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

		// Build new components array
		const newActiveComponents = originalImportURLs.map(
			(url: string, i: number) => {
				const module = modulesMap.get(url);
				const key = exportKeys[i] ?? "default";
				return module?.[key] ?? null;
			},
		);

		// Only update if components actually changed
		if (
			!jsonDeepEquals(
				newActiveComponents,
				internal_RiverClientGlobal.get("activeComponents"),
			)
		) {
			internal_RiverClientGlobal.set(
				"activeComponents",
				newActiveComponents,
			);
		}

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

			const newErrorBoundary =
				errorComponent ??
				internal_RiverClientGlobal.get("defaultErrorBoundary");

			// Only update if changed
			const currentErrorBoundary = internal_RiverClientGlobal.get(
				"activeErrorBoundary",
			);
			if (currentErrorBoundary !== newErrorBoundary) {
				internal_RiverClientGlobal.set(
					"activeErrorBoundary",
					newErrorBoundary,
				);
			}
		}
	}
}

/////////////////////////////////////////////////////////////////////
// PUBLIC API
/////////////////////////////////////////////////////////////////////

export async function navigate(
	href: string,
	options?: { replace?: boolean; scrollToTop?: boolean },
): Promise<void> {
	await navigationStateManager.navigate({
		href,
		navigationType: "userNavigation",
		replace: options?.replace,
		scrollToTop: options?.scrollToTop,
	});
}

let lastTriggeredNavOrRevalidateTimestampMS = Date.now();

export function getLastTriggeredNavOrRevalidateTimestampMS(): number {
	return lastTriggeredNavOrRevalidateTimestampMS;
}

export async function revalidate() {
	await navigationStateManager.navigate({
		href: window.location.href,
		navigationType: "revalidation",
	});
}

export type SubmitOptions = {
	dedupeKey?: string;
	revalidate?: boolean;
};

export async function submit<T = any>(
	url: string | URL,
	requestInit?: RequestInit,
	options?: SubmitOptions,
): Promise<{ success: true; data: T } | { success: false; error: string }> {
	return navigationStateManager.submit(url, requestInit, options);
}

export function beginNavigation(props: NavigateProps): NavigationControl {
	return navigationStateManager.beginNavigation(props);
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

export function getHistoryInstance(): historyInstance {
	return HistoryManager.getInstance();
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
	callbacks: LinkOnClickCallbacks<E> & {
		scrollToTop?: boolean;
		replace?: boolean;
	},
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

			const control = navigationStateManager.beginNavigation({
				href: anchor.href,
				navigationType: "userNavigation",
				scrollToTop: callbacks.scrollToTop,
				replace: callbacks.replace,
			});

			if (!control.promise) return;

			const res = await control.promise;

			if (!res) {
				// If not here, loading indicator can get stuck on
				// following redirects
				const targetUrl = new URL(anchor.href, window.location.href)
					.href;
				navigationStateManager.removeNavigation(targetUrl);
				return;
			}

			await callbacks.beforeRender?.(e);

			const targetUrl = new URL(anchor.href, window.location.href).href;
			const entry = navigationStateManager.getNavigation(targetUrl);
			if (entry) {
				await navigationStateManager["processNavigationResult"](
					res,
					entry,
				);
			}

			await callbacks.afterRender?.(e);
		}
	};
}

export function getPrefetchHandlers<E extends Event>(
	input: GetPrefetchHandlersInput<E>,
) {
	const hrefDetails = getHrefDetails(input.href);
	if (!hrefDetails.isHTTP) {
		return;
	}

	// TypeScript type guard -- after this check, we know relativeURL exists
	const { relativeURL } = hrefDetails;
	if (!relativeURL || hrefDetails.isExternal) {
		return;
	}

	let timer: number | undefined;
	let prefetchStarted = false;
	const delayMs = input.delayMs ?? 100;

	async function prefetch(e: E): Promise<void> {
		if (prefetchStarted) return;
		prefetchStarted = true;

		if (input.beforeBegin) {
			await input.beforeBegin(e);
		}

		// Use the navigation system
		await navigationStateManager.navigate({
			href: relativeURL,
			navigationType: "prefetch",
		});
	}

	function start(e: E): void {
		if (prefetchStarted) return;
		timer = window.setTimeout(() => prefetch(e), delayMs);
	}

	function stop(): void {
		if (timer) {
			clearTimeout(timer);
			timer = undefined;
		}

		// Abort prefetch if it exists and hasn't been upgraded
		const targetUrl = new URL(relativeURL, window.location.href).href;
		const nav = navigationStateManager.getNavigation(targetUrl);
		if (nav && nav.type === "prefetch" && nav.intent === "none") {
			nav.control.abortController?.abort();
			navigationStateManager.removeNavigation(targetUrl);
		}

		prefetchStarted = false;
	}

	async function onClick(e: E): Promise<void> {
		if (e.defaultPrevented) return;

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

		if (timer) {
			clearTimeout(timer);
			timer = undefined;
		}

		// Execute callbacks
		if (input.beforeBegin && !prefetchStarted) {
			await input.beforeBegin(e);
		}

		if (input.beforeRender) {
			await input.beforeRender(e);
		}

		// Use standard navigation -- it will upgrade the prefetch if it exists
		await navigate(relativeURL, {
			scrollToTop: input.scrollToTop,
			replace: input.replace,
		});

		if (input.afterRender) {
			await input.afterRender(e);
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
	options: {
		apiConfig: APIConfig;
		defaultErrorBoundary?: RouteErrorComponent;
		useViewTransitions?: boolean;
	},
): Promise<void> {
	internal_RiverClientGlobal.set("apiConfig", options.apiConfig);

	// Set options
	if (options.defaultErrorBoundary) {
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

	if (options.useViewTransitions) {
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

	let navigationSucceeded = true;

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
			const result = await navigationStateManager.navigate({
				href: window.location.href,
				navigationType: "browserHistory",
				scrollStateToRestore: scrollStateManager.getState(location.key),
			});

			if (!result.didNavigate) {
				navigationSucceeded = false;
				LogError(
					"Browser POP navigation failed, attempting hard reload of the destination.",
				);

				// This just reloads the current (failed) URL.
				// It preserves the history stack and ensures no UI/URL mismatch,
				// which could otherwise happen if a browser forward/back navigation fails
				window.location.reload();
			}
		}
	}

	if (navigationSucceeded) {
		HistoryManager.updateLastKnownLocation(location);
	}
}

/////////////////////////////////////////////////////////////////////
// INTERNAL FUNCTIONS
/////////////////////////////////////////////////////////////////////

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
	const { json, navigationType, runHistoryOptions } = props;

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
		const { href, scrollStateToRestore, replace, scrollToTop } =
			runHistoryOptions;
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

			scrollStateToDispatch = hash
				? { hash }
				: scrollToTop !== false
					? { x: 0, y: 0 }
					: undefined;
		}

		if (navigationType === "browserHistory") {
			scrollStateToDispatch =
				scrollStateToRestore ?? (hash ? { hash } : undefined);
		}
	}

	// Changing the title instantly makes it feel faster
	// The temp textarea trick is to decode any HTML entities in the title.
	// This should come after pushing to history though, so that the title is
	// correct in the history entry.
	const tempTxt = document.createElement("textarea");
	tempTxt.innerHTML = json.title?.dangerousInnerHTML || "";
	if (document.title !== tempTxt.value) {
		document.title = tempTxt.value;
	}

	// Apply CSS
	if (json.cssBundles) {
		AssetManager.applyCSS(json.cssBundles);
	}

	// Dispatch route change event -- this triggers the actual UI update
	window.dispatchEvent(
		new CustomEvent(RIVER_ROUTE_CHANGE_EVENT_KEY, {
			detail: { scrollState: scrollStateToDispatch },
		}),
	);

	// Update head elements
	updateHeadEls("meta", json.metaHeadEls ?? []);
	updateHeadEls("rest", json.restHeadEls ?? []);
}

async function effectuateRedirectDataResult(
	redirectData: RedirectData,
	redirectCount: number,
): Promise<RedirectData | null> {
	if (redirectData.status !== "should") {
		return null;
	}

	// Clean up any active redirect or revalidations when redirecting.
	// Otherwise loading state will get stuck.
	const navEntries = navigationStateManager.getNavigations().entries();
	for (const [key, nav] of navEntries) {
		if (nav.type === "redirect" || nav.type === "revalidation") {
			nav.control.abortController?.abort();
			navigationStateManager.removeNavigation(key);
		}
	}

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
		await navigationStateManager.navigate({
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

async function handleRedirects(props: {
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
	// To temporarily test traditional server redirect behavior,
	// you can set this to "0" instead of "1"
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

	return { redirectData, response: res };
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
		new AbortController().signal,
	);

	internal_RiverClientGlobal.set("clientLoadersData", clientLoadersData);
}

async function runWaitFns(
	json: PartialWaitFnJSON,
	buildID: string,
	signal: AbortSignal,
): Promise<Array<any>> {
	await ComponentLoader.loadComponents(json.importURLs);

	const matchedPatterns = json.matchedPatterns ?? [];
	const patternToWaitFnMap =
		internal_RiverClientGlobal.get("patternToWaitFnMap");
	const waitFnPromises: Array<Promise<any>> = [];

	let i = 0;
	for (const pattern of matchedPatterns) {
		if (patternToWaitFnMap[pattern]) {
			const serverDataPromise = Promise.resolve({
				matchedPatterns: json.matchedPatterns,
				loaderData: json.loadersData[i],
				rootData: json.hasRootData ? json.loadersData[0] : null,
				buildID: buildID,
			});

			waitFnPromises.push(
				patternToWaitFnMap[pattern]({
					params: json.params || {},
					splatValues: json.splatValues || [],
					serverDataPromise,
					signal,
				}),
			);
		} else {
			waitFnPromises.push(Promise.resolve());
		}
		i++;
	}

	return Promise.all(waitFnPromises);
}

async function completeClientLoaders(
	json: PartialWaitFnJSON,
	buildID: string,
	runningLoaders: Map<string, Promise<any>>,
	signal: AbortSignal,
): Promise<Array<any>> {
	await ComponentLoader.loadComponents(json.importURLs);

	const matchedPatterns = json.matchedPatterns ?? [];
	const patternToWaitFnMap =
		internal_RiverClientGlobal.get("patternToWaitFnMap");
	const finalPromises: Array<Promise<any>> = [];

	let i = 0;
	for (const pattern of matchedPatterns) {
		if (runningLoaders.has(pattern)) {
			finalPromises.push(runningLoaders.get(pattern)!);
		} else if (patternToWaitFnMap[pattern]) {
			const serverDataPromise = Promise.resolve({
				matchedPatterns: json.matchedPatterns,
				loaderData: json.loadersData[i],
				rootData: json.hasRootData ? json.loadersData[0] : null,
				buildID: buildID,
			});

			const loaderPromise = patternToWaitFnMap[pattern]({
				splatValues: json.splatValues || [],
				params: json.params || {},
				serverDataPromise,
				signal,
			}).catch((error: any) => {
				if (!isAbortError(error)) {
					LogError(
						`Client loader error for pattern ${pattern}:`,
						error,
					);
				}
				return undefined;
			});
			finalPromises.push(loaderPromise);
		} else {
			finalPromises.push(Promise.resolve());
		}
		i++;
	}

	return Promise.all(finalPromises);
}

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
	return final;
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

// Setup beforeunload handler for scroll restoration
window.addEventListener("beforeunload", () => {
	scrollStateManager.savePageRefreshState();
});
