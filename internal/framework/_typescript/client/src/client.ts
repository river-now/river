/// <reference types="vite/client" />

import { debounce } from "river.now/kit/debounce";
import { jsonDeepEquals } from "river.now/kit/json";
import { getIsGETRequest } from "river.now/kit/url";
import { AssetManager } from "./asset_manager.ts";
import {
	completeClientLoaders,
	findPartialMatchesOnClient,
} from "./client_loaders.ts";
import {
	dispatchBuildIDEvent,
	dispatchStatusEvent,
	type StatusEventDetail,
} from "./events.ts";
import { HistoryManager } from "./history/history.ts";
import type { historyInstance } from "./history/npm_history_types.ts";
import {
	effectuateRedirectDataResult,
	getBuildIDFromResponse,
	handleRedirects,
	type RedirectData,
} from "./redirects/redirects.ts";
import { __reRenderApp } from "./rendering.ts";
import {
	__riverClientGlobal,
	type ClientLoaderAwaitedServerData,
	type GetRouteDataOutput,
} from "./river_ctx/river_ctx.ts";
import {
	__applyScrollState,
	type ScrollState,
} from "./scroll_state_manager.ts";
import { isAbortError } from "./utils/errors.ts";
import { logError } from "./utils/logging.ts";

/////////////////////////////////////////////////////////////////////
// TYPES
/////////////////////////////////////////////////////////////////////

export type RiverNavigationType =
	| "browserHistory"
	| "userNavigation"
	| "revalidation"
	| "redirect"
	| "prefetch"
	| "action";

export type NavigateProps = {
	href: string;
	state?: unknown;
	navigationType: RiverNavigationType;
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
	type: RiverNavigationType;
	intent: NavigationIntent;
	phase: NavigationPhase;
	startTime: number;
	targetUrl: string; // URL this navigation is targeting
	originUrl: string; // URL when navigation started (for revalidation)
	scrollToTop?: boolean;
	replace?: boolean;
	state?: unknown;
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
				logError("Navigate error:", error);
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
					state: props.state,
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
			state: props.state,
		};

		this.setNavigation(targetUrl, entry);
		return entry.control;
	}

	private upgradeNavigation(
		href: string,
		updates: Partial<
			Pick<
				NavigationEntry,
				"type" | "intent" | "scrollToTop" | "replace" | "state"
			>
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
				__riverClientGlobal.get("buildID") || "1",
			);

			if (props.navigationType === "revalidation") {
				const deploymentID = __riverClientGlobal.get("deploymentID");
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
				if (
					result.response &&
					result.response.ok &&
					!result.redirectData?.status
				) {
					const json = await result.response.json();
					return { ...result, json };
				}
				return { ...result, json: undefined };
			});

			// Try to match routes on the client and start parallel loaders
			const pathname = url.pathname;
			const matchResult = await findPartialMatchesOnClient(pathname);
			const patternToWaitFnMap =
				__riverClientGlobal.get("patternToWaitFnMap");
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
								logError(
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
				logError("Navigation failed", error);
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
					result.props,
				);
				return;
			}

			// Type guard to ensure we have the json branch
			if (!("json" in result)) {
				logError("Invalid navigation result: no JSON or redirect");
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
		const oldID = __riverClientGlobal.get("buildID");
		const newID = getBuildIDFromResponse(result.response);
		if (newID && newID !== oldID) {
			dispatchBuildIDEvent({ newID, oldID });
		}

		// Wait for client loaders
		const clientLoadersData = await result.waitFnPromise;
		__riverClientGlobal.set("clientLoadersData", clientLoadersData);

		// Wait for CSS
		if (result.cssBundlePromises.length > 0) {
			try {
				await Promise.all(result.cssBundlePromises);
			} catch (error) {
				logError("Error preloading CSS bundles:", error);
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
								state: entry.state,
							}
						: undefined,
				onFinish: () => {
					this.transitionPhase(entry.targetUrl, "complete");
				},
			});
		} catch (error) {
			this.transitionPhase(entry.targetUrl, "complete");
			if (!isAbortError(error)) {
				logError("Error completing navigation", error);
			}
			throw error;
		}
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
			const deploymentID = __riverClientGlobal.get("deploymentID");
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

			const oldID = __riverClientGlobal.get("buildID");
			const newID = getBuildIDFromResponse(response);
			if (newID && newID !== oldID) {
				dispatchBuildIDEvent({ newID, oldID });
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
			logError(error);
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
		dispatchStatusEvent(newStatus);
	}
}

// Global instance
export const navigationStateManager = new NavigationStateManager();

/////////////////////////////////////////////////////////////////////
// PUBLIC API
/////////////////////////////////////////////////////////////////////

export async function riverNavigate(
	href: string,
	options?: {
		replace?: boolean;
		scrollToTop?: boolean;
		search?: string;
		hash?: string;
		state?: unknown;
	},
): Promise<void> {
	const url = new URL(href, window.location.href);

	if (options?.search !== undefined) {
		url.search = options.search;
	}
	if (options?.hash !== undefined) {
		url.hash = options.hash;
	}

	await navigationStateManager.navigate({
		href: url.href,
		navigationType: "userNavigation",
		replace: options?.replace,
		scrollToTop: options?.scrollToTop,
		state: options?.state,
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
		state: HistoryManager.getInstance().location.state,
	};
}

export function getBuildID(): string {
	return __riverClientGlobal.get("buildID");
}

export function getRootEl(): HTMLDivElement {
	return document.getElementById("river-root") as HTMLDivElement;
}

export function getHistoryInstance(): historyInstance {
	return HistoryManager.getInstance();
}
