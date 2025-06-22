// /comprehensive-navigation.test.ts

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	addBuildIDListener,
	addLocationListener,
	addRouteChangeListener,
	addStatusListener,
	applyScrollState,
	beginNavigation,
	getBuildID,
	getHistoryInstance,
	getLocation,
	getPrefetchHandlers,
	getRootEl,
	getStatus,
	hmrRunClientLoaders,
	initClient,
	initCustomHistory,
	makeLinkOnClickFn,
	navigate,
	navigationState,
	revalidate,
	type ScrollState,
	submit,
} from "../src/client.ts";
import { internal_RiverClientGlobal } from "../src/river_ctx.ts";

// Mock only what's necessary for testing
const mockSessionStorage = (() => {
	let store: { [key: string]: string } = {};
	return {
		getItem: (key: string) => store[key] || null,
		setItem: (key: string, value: string) => {
			if (key) {
				store[key] = value.toString();
			}
		},
		removeItem: (key: string) => {
			if (key) {
				delete store[key];
			}
		},
		clear: () => {
			store = {};
		},
	};
})();

// Store cleanup functions
const cleanupFns: Array<() => void> = [];

// Helper to setup initial River context
const setupGlobalRiverContext = (initialData = {}) => {
	(globalThis as any)[Symbol.for("__river_internal__")] = {
		buildID: "1",
		matchedPatterns: [],
		importURLs: [],
		exportKeys: [],
		loadersData: [],
		params: {},
		splatValues: [],
		hasRootData: false,
		activeComponents: [],
		clientLoadersData: [],
		patternToWaitFnMap: {},
		viteDevURL: "",
		publicPathPrefix: "",
		...initialData,
	};
};

// Helper to create mock fetch responses
const createMockResponse = (data: any, options: ResponseInit = {}) => {
	return new Response(JSON.stringify(data), {
		status: 200,
		headers: {
			"Content-Type": "application/json",
			"X-River-Build-Id": "1",
			...options.headers,
		},
		...options,
	});
};

describe("Comprehensive Navigation Test Suite", () => {
	let locationBackup: Location;
	let historyBackup: History;

	beforeEach(() => {
		vi.useFakeTimers({ shouldAdvanceTime: true });

		// Mock CSS.escape if it doesn't exist (not available in jsdom)
		if (!global.CSS) {
			(global as any).CSS = {};
		}
		if (!global.CSS.escape) {
			global.CSS.escape = (str: string) =>
				str.replace(/[!"#$%&'()*+,./:;<=>?@[\\\]^`{|}~]/g, "\\$&");
		}

		vi.doMock("/module1.js", () => ({ default: () => {} }));
		vi.doMock("/module2.js", () => ({ default: () => {} }));

		// Backup original objects
		locationBackup = window.location;
		historyBackup = window.history;

		// Set up a complete window.location mock
		Object.defineProperty(window, "location", {
			value: {
				href: "http://localhost:3000/",
				origin: "http://localhost:3000",
				protocol: "http:",
				host: "localhost:3000",
				hostname: "localhost",
				port: "3000",
				pathname: "/",
				search: "",
				hash: "",
				assign: vi.fn((url) => {
					window.location.href = url;
				}),
				replace: vi.fn((url) => {
					const newUrl = new URL(url, window.location.href);
					window.location.href = newUrl.href;
					window.location.pathname = newUrl.pathname;
					window.location.search = newUrl.search;
					window.location.hash = newUrl.hash;
				}),
				reload: vi.fn(),
				toString: () => window.location.href,
			},
			writable: true,
			configurable: true,
		});

		// Mock Element.prototype.scrollIntoView
		if (!Element.prototype.scrollIntoView) {
			Element.prototype.scrollIntoView = vi.fn();
		}

		// Mock history.scrollRestoration
		Object.defineProperty(window.history, "scrollRestoration", {
			value: "auto",
			writable: true,
			configurable: true,
		});

		// Mock history methods
		window.history.replaceState = vi.fn((state, title, url) => {
			if (url) {
				const newUrl = new URL(url, window.location.href);
				window.location.href = newUrl.href;
				window.location.pathname = newUrl.pathname;
				window.location.search = newUrl.search;
				window.location.hash = newUrl.hash;
			}
		});

		window.history.pushState = vi.fn((state, title, url) => {
			if (url) {
				const newUrl = new URL(url, window.location.href);
				window.location.href = newUrl.href;
				window.location.pathname = newUrl.pathname;
				window.location.search = newUrl.search;
				window.location.hash = newUrl.hash;
			}
		});

		// Mock sessionStorage
		Object.defineProperty(window, "sessionStorage", {
			value: mockSessionStorage,
			writable: true,
			configurable: true,
		});

		// Mock window scroll properties
		Object.defineProperty(window, "scrollTo", { value: vi.fn(), writable: true });
		Object.defineProperty(window, "scrollX", { value: 0, writable: true });
		Object.defineProperty(window, "scrollY", { value: 0, writable: true });

		// Mock startViewTransition
		const mockStartViewTransition = vi.fn((callback) => {
			callback?.();
			return { finished: Promise.resolve() };
		});

		Object.defineProperty(document, "startViewTransition", {
			value: mockStartViewTransition,
			configurable: true,
		});

		// Setup River context
		setupGlobalRiverContext();

		// Setup spies
		vi.spyOn(window, "fetch");
		vi.spyOn(window, "dispatchEvent");
		vi.spyOn(console, "error").mockImplementation(() => {});
		vi.spyOn(console, "info").mockImplementation(() => {});

		// Clear all state
		mockSessionStorage.clear();
		vi.clearAllMocks();
		navigationState.navigations.clear();
		navigationState.submissions.clear();
		navigationState.activeUserNavigation = null;
		document.title = "Initial Page";
		(window as any).scrollX = 0;
		(window as any).scrollY = 0;

		// Clear any existing listeners to prevent memory leaks
		cleanupFns.forEach((fn) => fn());
		cleanupFns.length = 0;

		// Initialize history after location is properly set up
		initCustomHistory();
	});

	afterEach(() => {
		// Run all pending timers
		vi.runOnlyPendingTimers();
		vi.useRealTimers();

		// Clean up all listeners
		cleanupFns.forEach((fn) => fn());
		cleanupFns.length = 0;

		// Clear navigation state
		navigationState.navigations.clear();
		navigationState.submissions.clear();
		navigationState.activeUserNavigation = null;

		// Clear DOM
		document.body.innerHTML = "";
		document.head.innerHTML = "";

		// Clear any pending promises to avoid unhandled rejections
		vi.clearAllMocks();

		// Restore mocks
		vi.restoreAllMocks();

		// Restore original objects
		Object.defineProperty(window, "location", {
			value: locationBackup,
			writable: true,
			configurable: true,
		});
		Object.defineProperty(window, "history", {
			value: historyBackup,
			writable: true,
			configurable: true,
		});

		// Force garbage collection if available
		if (global.gc) {
			global.gc();
		}
	});

	// Add helper to register cleanup functions
	const addCleanup = (fn: () => void) => {
		cleanupFns.push(fn);
		return fn;
	};

	// Update all listener additions to register cleanup
	const addListener = <T>(
		adder: (fn: any) => () => void,
		fn: (e: CustomEvent<T>) => void,
	) => {
		const cleanup = adder(fn);
		addCleanup(cleanup);
		return cleanup;
	};

	describe("1. Core Navigation", () => {
		describe("1.1 Navigation Types", () => {
			it("should handle userNavigation type correctly", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						title: { dangerousInnerHTML: "User Nav Page" },
						importURLs: [],
						cssBundles: [],
					}),
				);

				await navigate("/user-nav");
				await vi.runAllTimersAsync();

				expect(fetch).toHaveBeenCalledWith(
					expect.objectContaining({
						href: "http://localhost:3000/user-nav?river_json=1",
					}),
					expect.any(Object),
				);
			});

			it("should handle browserHistory navigation (back/forward)", async () => {
				// Setup: Navigate to create history
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						title: { dangerousInnerHTML: "Page 2" },
						importURLs: [],
						cssBundles: [],
					}),
				);

				const history = getHistoryInstance();

				// Save the initial location key
				const initialKey = history.location.key;

				// Navigate to create history entries
				history.push("/page1");
				const page1Key = history.location.key;

				history.push("/page2");
				const page2Key = history.location.key;

				// Clear any fetch calls from initialization
				vi.clearAllMocks();

				// Now we need to simulate going back
				// The key insight is that when going back, the location.key changes
				// and the customHistoryListener in the implementation detects this

				// Simulate the browser going back by:
				// 1. Changing the URL back to page1
				window.history.replaceState({}, "", "/page1");

				// 2. Dispatching a popstate event which the history library listens to
				const popstateEvent = new PopStateEvent("popstate", {
					state: { key: page1Key },
				});
				window.dispatchEvent(popstateEvent);

				// Give the async operations time to complete
				await vi.runAllTimersAsync();

				// Should trigger navigation with browserHistory type
				expect(fetch).toHaveBeenCalled();
				expect(fetch).toHaveBeenCalledWith(
					expect.objectContaining({
						href: expect.stringContaining("/page1"),
					}),
					expect.any(Object),
				);
			});

			it("should handle browserHistory navigation (back/forward) -- approach 2", async () => {
				// Setup: Navigate to create history
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						title: { dangerousInnerHTML: "Page 2" },
						importURLs: [],
						cssBundles: [],
					}),
				);

				// Instead of trying to simulate browser back,
				// we can directly test that __navigate handles browserHistory type correctly

				// Clear any existing calls
				vi.clearAllMocks();

				// Directly call navigate with browserHistory type
				// This is what happens internally when the browser back button is pressed
				await beginNavigation({
					href: "/previous-page",
					navigationType: "browserHistory",
				}).promise;

				await vi.runAllTimersAsync();

				// Should trigger navigation with browserHistory type
				expect(fetch).toHaveBeenCalled();
				expect(fetch).toHaveBeenCalledWith(
					expect.objectContaining({
						href: expect.stringContaining("/previous-page"),
					}),
					expect.any(Object),
				);
			});

			it("should handle revalidation type correctly", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						title: { dangerousInnerHTML: "Current Page" },
						importURLs: [],
						cssBundles: [],
					}),
				);

				const history = getHistoryInstance();
				const pushSpy = vi.spyOn(history, "push");
				const replaceSpy = vi.spyOn(history, "replace");

				await revalidate();
				await vi.runAllTimersAsync();

				// Revalidation should not change history
				expect(pushSpy).not.toHaveBeenCalled();
				expect(replaceSpy).not.toHaveBeenCalled();
			});

			it("should handle redirect type from server response", async () => {
				vi.mocked(fetch)
					.mockResolvedValueOnce(
						createMockResponse(null, {
							headers: { "X-Client-Redirect": "/redirected" },
						}),
					)
					.mockResolvedValueOnce(
						createMockResponse({
							title: { dangerousInnerHTML: "Redirected Page" },
							importURLs: [],
							cssBundles: [],
						}),
					);

				await navigate("/original");
				await vi.runAllTimersAsync();

				expect(fetch).toHaveBeenCalledTimes(2);
				expect(window.location.pathname).toBe("/redirected");
			});

			it("should handle prefetch type without affecting loading states", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				const statusListener = vi.fn();
				const cleanup = addListener(addStatusListener, statusListener);

				const handlers = getPrefetchHandlers({ href: "/prefetch-target" });
				handlers?.start({} as Event);

				await vi.advanceTimersByTimeAsync(100);
				await vi.runAllTimersAsync();

				// Prefetch should not trigger loading states
				expect(statusListener).not.toHaveBeenCalledWith(
					expect.objectContaining({
						detail: expect.objectContaining({ isNavigating: true }),
					}),
				);

				cleanup();
			});
		});

		describe("1.2 Navigation State Management", () => {
			it("should enforce single active user navigation", async () => {
				// Create a promise that resolves with AbortError when aborted
				const createAbortablePromise = () => {
					let rejectFn: (reason: any) => void;
					const promise = new Promise((_, reject) => {
						rejectFn = reject;
					});
					return { promise, reject: rejectFn! };
				};

				const { promise: promise1, reject: reject1 } = createAbortablePromise();
				const { promise: promise2, reject: reject2 } = createAbortablePromise();

				let callCount = 0;
				vi.mocked(fetch).mockImplementation((() => {
					callCount++;
					if (callCount === 1) return promise1;
					return promise2;
				}) as any);

				const control1 = beginNavigation({
					href: "/page1",
					navigationType: "userNavigation",
				});

				expect(control1.abortController).toBeDefined();
				const abortSpy1 = vi.spyOn(control1.abortController!, "abort");

				expect(navigationState.activeUserNavigation).toBe("/page1");
				expect(navigationState.navigations.size).toBe(1);

				// Start second navigation
				const control2 = beginNavigation({
					href: "/page2",
					navigationType: "userNavigation",
				});

				expect(abortSpy1).toHaveBeenCalled();
				expect(navigationState.activeUserNavigation).toBe("/page2");

				// When the first navigation is aborted, it's removed from the map
				// So we should only have 1 navigation (the second one)
				expect(navigationState.navigations.size).toBe(1);
				expect(navigationState.navigations.has("/page2")).toBe(true);
				expect(navigationState.navigations.has("/page1")).toBe(false);

				// Clean up by rejecting the promises
				const abortError = new Error("Aborted");
				abortError.name = "AbortError";
				reject1(abortError);
				reject2(abortError);

				await vi.runAllTimersAsync();
			});

			it("should track all navigation types in navigations Map", () => {
				vi.mocked(fetch).mockImplementation(
					() => new Promise(() => {}), // Never resolve
				);

				beginNavigation({ href: "/nav1", navigationType: "userNavigation" });
				beginNavigation({ href: "/nav2", navigationType: "prefetch" });
				beginNavigation({ href: "/nav3", navigationType: "revalidation" });

				expect(navigationState.navigations.size).toBe(3);
				expect(navigationState.navigations.get("/nav1")?.type).toBe("userNavigation");
				expect(navigationState.navigations.get("/nav2")?.type).toBe("prefetch");
				expect(navigationState.navigations.get("/nav3")?.type).toBe("revalidation");

				// Clean up
				for (const [, nav] of navigationState.navigations) {
					nav.control.abortController?.abort();
				}
			});

			it("should clean up navigations from map when complete", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				await navigate("/cleanup-test");
				await vi.runAllTimersAsync();

				// Non-prefetch navigations should be cleaned up
				expect(navigationState.navigations.has("/cleanup-test")).toBe(false);
			});

			it("should clean up prefetch navigations from map when complete", async () => {
				// Mock fetch to return a promise that takes time to resolve
				vi.mocked(fetch).mockImplementation(
					() =>
						new Promise((resolve) =>
							setTimeout(
								() => resolve(createMockResponse({ importURLs: [], cssBundles: [] })),
								100,
							),
						),
				);

				const handlers = getPrefetchHandlers({
					href: "/prefetch-cleanup",
					delayMs: 50,
				});
				handlers?.start({} as Event);

				// Advance past the 50ms delay to start the prefetch
				await vi.advanceTimersByTimeAsync(50);

				// Now it should be in the map
				expect(navigationState.navigations.has("/prefetch-cleanup")).toBe(true);

				// Let it complete
				await vi.runAllTimersAsync();

				// Should be removed from map after completion
				expect(navigationState.navigations.has("/prefetch-cleanup")).toBe(false);

				// Clean up
				handlers?.stop();
			});
		});

		describe("1.3 Link Click Handling", () => {
			it("should prevent default for eligible internal links", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				const event = new MouseEvent("click", { bubbles: true });
				const preventDefault = vi.spyOn(event, "preventDefault");
				const anchor = document.createElement("a");
				anchor.href = "/internal-link";

				Object.defineProperty(event, "target", { value: anchor });

				const onClick = makeLinkOnClickFn({});
				await onClick(event);

				expect(preventDefault).toHaveBeenCalled();

				await vi.runAllTimersAsync();
			});

			it("should ignore external links", async () => {
				const event = new MouseEvent("click", { bubbles: true });
				const preventDefault = vi.spyOn(event, "preventDefault");
				const anchor = document.createElement("a");
				anchor.href = "https://external.com";

				Object.defineProperty(event, "target", { value: anchor });

				const onClick = makeLinkOnClickFn({});
				await onClick(event);

				expect(preventDefault).not.toHaveBeenCalled();
			});

			it("should ignore clicks with modifier keys", async () => {
				const event = new MouseEvent("click", {
					bubbles: true,
					ctrlKey: true,
				});
				const preventDefault = vi.spyOn(event, "preventDefault");
				const anchor = document.createElement("a");
				anchor.href = "/internal";

				Object.defineProperty(event, "target", { value: anchor });

				const onClick = makeLinkOnClickFn({});
				await onClick(event);

				expect(preventDefault).not.toHaveBeenCalled();
			});

			it("should handle hash-only links without navigation", async () => {
				window.history.pushState({}, "", "/current-page");

				const event = new MouseEvent("click", { bubbles: true });
				const anchor = document.createElement("a");
				anchor.href = "/current-page#section";

				Object.defineProperty(event, "target", { value: anchor });

				const onClick = makeLinkOnClickFn({});
				await onClick(event);

				// Should save scroll state but not navigate
				expect(fetch).not.toHaveBeenCalled();

				const scrollState = JSON.parse(
					sessionStorage.getItem("__river__scrollStateMap") || "[]",
				);
				expect(scrollState).toBeDefined();
			});

			it("should use prefetch data immediately on click if available", async () => {
				const prefetchData = {
					title: { dangerousInnerHTML: "Prefetched Content" },
					importURLs: [],
					cssBundles: [],
				};

				vi.mocked(fetch).mockResolvedValue(createMockResponse(prefetchData));

				// Start prefetch
				const handlers = getPrefetchHandlers({ href: "/prefetch-click" });
				handlers?.start({} as Event);
				await vi.advanceTimersByTimeAsync(100);
				await vi.runAllTimersAsync();

				// Create a proper click event with an anchor element
				const anchor = document.createElement("a");
				anchor.href = "/prefetch-click";
				document.body.appendChild(anchor);

				const event = new MouseEvent("click", {
					bubbles: true,
					cancelable: true,
				});
				Object.defineProperty(event, "target", { value: anchor });

				const preventDefault = vi.spyOn(event, "preventDefault");

				// Click while prefetch is complete
				await handlers?.onClick(event);
				await vi.runAllTimersAsync();

				expect(preventDefault).toHaveBeenCalled();
				expect(document.title).toBe("Prefetched Content");

				// Clean up
				document.body.removeChild(anchor);
				handlers?.stop();
			});
		});

		describe("1.4 Programmatic Navigation", () => {
			it("should support navigate() with replace option", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				const history = getHistoryInstance();
				const replaceSpy = vi.spyOn(history, "replace");

				await navigate("/replace-test", { replace: true });
				await vi.runAllTimersAsync();

				expect(replaceSpy).toHaveBeenCalledWith("/replace-test");
			});
		});
	});

	describe("2. Navigation Lifecycle", () => {
		describe("2.1 Begin Navigation Phase", () => {
			it("should set appropriate loading states", () => {
				vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));

				const statusListener = vi.fn();
				const cleanup = addListener(addStatusListener, statusListener);

				const control = beginNavigation({
					href: "/loading",
					navigationType: "userNavigation",
				});
				vi.runAllTimers();

				expect(statusListener).toHaveBeenCalledWith(
					expect.objectContaining({
						detail: expect.objectContaining({ isNavigating: true }),
					}),
				);

				// Cleanup
				control.abortController?.abort();
				cleanup();
			});

			it("should abort all navigations except current for userNavigation", () => {
				vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));

				const control1 = beginNavigation({
					href: "/nav1",
					navigationType: "prefetch",
				});
				const control2 = beginNavigation({
					href: "/nav2",
					navigationType: "revalidation",
				});

				if (!control1.abortController || !control2.abortController) {
					throw new Error("AbortController not set");
				}
				const abort1 = vi.spyOn(control1.abortController, "abort");
				const abort2 = vi.spyOn(control2.abortController, "abort");

				const control3 = beginNavigation({
					href: "/nav3",
					navigationType: "userNavigation",
				});

				expect(abort1).toHaveBeenCalled();
				expect(abort2).toHaveBeenCalled();

				// Cleanup
				control3.abortController?.abort();
			});

			it("should upgrade existing prefetch to userNavigation", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				// Start prefetch
				const prefetchControl = beginNavigation({
					href: "/upgrade",
					navigationType: "prefetch",
				});

				expect(navigationState.navigations.get("/upgrade")?.type).toBe("prefetch");

				// Upgrade to user navigation
				const userControl = beginNavigation({
					href: "/upgrade",
					navigationType: "userNavigation",
				});

				expect(navigationState.navigations.get("/upgrade")?.type).toBe(
					"userNavigation",
				);
				expect(userControl).toBe(prefetchControl);

				// Wait for completion
				await userControl.promise;
				await vi.runAllTimersAsync();
			});

			it("should deduplicate prefetch requests", () => {
				vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));

				const control1 = beginNavigation({
					href: "/prefetch-dedup",
					navigationType: "prefetch",
				});
				const control2 = beginNavigation({
					href: "/prefetch-dedup",
					navigationType: "prefetch",
				});

				expect(control1).toBe(control2);
				expect(navigationState.navigations.size).toBe(1);

				// Cleanup
				control1.abortController?.abort();
			});
		});

		describe("2.2 Fetch Route Data Phase", () => {
			it("should construct URL with river_json and buildID", async () => {
				setupGlobalRiverContext({ buildID: "test-build-123" });
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				await navigate("/test-url");
				await vi.runAllTimersAsync();

				expect(fetch).toHaveBeenCalledWith(
					expect.objectContaining({
						href: "http://localhost:3000/test-url?river_json=test-build-123",
					}),
					expect.any(Object),
				);
			});

			it("should include X-Accepts-Client-Redirect header", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				await navigate("/test-headers");
				await vi.runAllTimersAsync();

				expect(fetch).toHaveBeenCalledWith(
					expect.any(URL),
					expect.objectContaining({
						headers: expect.any(Headers),
					}),
				);

				const headers = vi.mocked(fetch).mock.calls[0]?.[1]?.headers as Headers;
				expect(headers.get("X-Accepts-Client-Redirect")).toBe("1");
			});

			it("should handle redirect responses correctly", async () => {
				vi.mocked(fetch)
					.mockResolvedValueOnce(
						createMockResponse(null, {
							headers: { "X-Client-Redirect": "/new-location" },
						}),
					)
					.mockResolvedValueOnce(
						createMockResponse({
							title: { dangerousInnerHTML: "Redirected" },
							importURLs: [],
							cssBundles: [],
						}),
					);

				await navigate("/original");
				await vi.runAllTimersAsync();

				expect(fetch).toHaveBeenCalledTimes(2);
			});

			it("should handle empty JSON as failure", async () => {
				vi.mocked(fetch).mockResolvedValue(new Response("", { status: 200 }));

				await navigate("/empty-json");

				// The navigation should fail and not complete
				expect(navigationState.navigations.has("/empty-json")).toBe(false);
				expect(document.title).toBe("Initial Page"); // Title shouldn't change
			});

			it("should preload modules in production mode", async () => {
				const originalEnv = import.meta.env.DEV;
				(import.meta.env as any).DEV = false;

				// Spy on appendChild to verify links are created
				const appendChildSpy = vi.spyOn(document.head, "appendChild");

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						importURLs: ["/module1.js", "/module2.js"],
						deps: ["/dep1.js", "/dep2.js", "/module1.js"],
						cssBundles: [],
					}),
				);

				const control = beginNavigation({
					href: "/with-deps",
					navigationType: "userNavigation",
				});

				await control.promise;

				// Verify appendChild was called with modulepreload links
				const modulepreloadCalls = appendChildSpy.mock.calls.filter((call) => {
					const element = call[0] as HTMLElement;
					return (
						element.tagName === "LINK" &&
						element.getAttribute("rel") === "modulepreload"
					);
				});

				// Should create modulepreload for unique deps
				expect(modulepreloadCalls.length).toBe(3);

				(import.meta.env as any).DEV = originalEnv;
			});

			it("should preload CSS bundles", async () => {
				const appendChildSpy = vi.spyOn(document.head, "appendChild");

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						importURLs: [],
						cssBundles: ["/styles1.css", "/styles2.css"],
					}),
				);

				const control = beginNavigation({
					href: "/with-css",
					navigationType: "userNavigation",
				});

				await control.promise;

				// Verify appendChild was called with CSS preload links
				const cssPreloadCalls = appendChildSpy.mock.calls.filter((call) => {
					const element = call[0] as HTMLElement;
					return (
						element.tagName === "LINK" &&
						element.getAttribute("rel") === "preload" &&
						element.getAttribute("as") === "style"
					);
				});

				expect(cssPreloadCalls.length).toBe(2);
			});

			it("should execute client wait functions", async () => {
				const waitFn = vi.fn().mockResolvedValue({ clientData: "test" });
				setupGlobalRiverContext({
					patternToWaitFnMap: {
						"/pattern": waitFn,
					},
				});

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						importURLs: [],
						cssBundles: [],
						matchedPatterns: ["/pattern"],
						loadersData: [{ serverData: "test" }],
						hasRootData: true,
					}),
				);

				await navigate("/pattern/test");
				await vi.runAllTimersAsync();

				expect(waitFn).toHaveBeenCalledWith(
					expect.objectContaining({
						buildID: "1",
						matchedPatterns: ["/pattern"],
						rootData: { serverData: "test" },
						loaderData: { serverData: "test" },
					}),
				);
			});

			it("should cleanup navigation on completion", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				expect(navigationState.navigations.has("/cleanup")).toBe(false);

				const navPromise = navigate("/cleanup");
				expect(navigationState.navigations.has("/cleanup")).toBe(true);

				await navPromise;
				await vi.runAllTimersAsync();

				expect(navigationState.navigations.has("/cleanup")).toBe(false);
				expect(navigationState.activeUserNavigation).toBe(null);
			});
		});

		describe("2.3 Complete Navigation Phase", () => {
			it("should handle redirect data result", async () => {
				vi.mocked(fetch)
					.mockResolvedValueOnce(
						createMockResponse(null, {
							headers: { "X-Client-Redirect": "/redirect-target" },
						}),
					)
					.mockResolvedValueOnce(
						createMockResponse({
							title: { dangerousInnerHTML: "Redirect Target" },
							importURLs: [],
							cssBundles: [],
						}),
					);

				await navigate("/start");
				await vi.runAllTimersAsync();

				expect(document.title).toBe("Redirect Target");
			});

			it("should dispatch build-id event on change", async () => {
				const buildIdListener = vi.fn();
				const cleanup = addListener(addBuildIDListener, buildIdListener);

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse(
						{ importURLs: [], cssBundles: [] },
						{ headers: { "X-River-Build-Id": "new-build-456" } },
					),
				);

				await navigate("/new-build");
				await vi.runAllTimersAsync();

				expect(buildIdListener).toHaveBeenCalledWith(
					expect.objectContaining({
						detail: {
							oldID: "1",
							newID: "new-build-456",
							fromGETAction: false,
						},
					}),
				);

				cleanup();
			});

			it("should wait for client data before rendering", async () => {
				const clientData = { processed: true };
				const waitFn = vi.fn().mockResolvedValue(clientData);

				setupGlobalRiverContext({
					patternToWaitFnMap: { "/": waitFn },
				});

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						importURLs: [],
						cssBundles: [],
						matchedPatterns: ["/"],
						loadersData: [{}],
					}),
				);

				await navigate("/wait-test");
				await vi.runAllTimersAsync();

				expect(internal_RiverClientGlobal.get("clientLoadersData")).toEqual([
					clientData,
				]);
			});
		});

		describe("2.4 Re-render App Phase", () => {
			it("should clear loading state", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				// Check status synchronously
				expect(getStatus().isNavigating).toBe(false);

				const navPromise = navigate("/clear-loading");

				// Should be navigating immediately
				expect(getStatus().isNavigating).toBe(true);

				await navPromise;
				await vi.runAllTimersAsync();

				// Should be done navigating
				expect(getStatus().isNavigating).toBe(false);
			});

			it("should use view transitions when enabled and supported", async () => {
				setupGlobalRiverContext({ useViewTransitions: true });

				const mockStartViewTransition = vi.fn((callback) => {
					callback?.();
					return { finished: Promise.resolve() };
				});
				Object.defineProperty(document, "startViewTransition", {
					value: mockStartViewTransition,
					configurable: true,
				});

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				await navigate("/with-transition");
				await vi.runAllTimersAsync();

				expect(mockStartViewTransition).toHaveBeenCalled();
			});

			it("should skip view transitions for prefetch and revalidation", async () => {
				setupGlobalRiverContext({ useViewTransitions: true });

				const mockStartViewTransition = vi.fn((callback) => {
					callback?.();
					return { finished: Promise.resolve() };
				});
				Object.defineProperty(document, "startViewTransition", {
					value: mockStartViewTransition,
					configurable: true,
				});

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				// Test revalidation
				await revalidate();
				await vi.runAllTimersAsync();

				expect(mockStartViewTransition).not.toHaveBeenCalled();
			});

			it("should update global state with route data", async () => {
				const routeData = {
					matchedPatterns: ["/users/:id"],
					loadersData: [{ user: "data" }],
					importURLs: [], // Empty to avoid import issues
					exportKeys: [],
					hasRootData: true,
					params: { id: "123" },
					splatValues: [],
					cssBundles: [],
				};

				vi.mocked(fetch).mockResolvedValue(createMockResponse(routeData));

				await navigate("/users/123");
				await vi.runAllTimersAsync();

				expect(internal_RiverClientGlobal.get("matchedPatterns")).toEqual(
					routeData.matchedPatterns,
				);
				expect(internal_RiverClientGlobal.get("params")).toEqual(routeData.params);
			});

			it("should handle history management for userNavigation", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				const history = getHistoryInstance();
				const pushSpy = vi.spyOn(history, "push");

				await navigate("/new-page");
				await vi.runAllTimersAsync();

				expect(pushSpy).toHaveBeenCalledWith("/new-page");
			});

			it("should use replace for same URL navigation", async () => {
				window.history.replaceState({}, "", "/same-page");

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				const history = getHistoryInstance();
				const replaceSpy = vi.spyOn(history, "replace");

				await navigate("/same-page");
				await vi.runAllTimersAsync();

				expect(replaceSpy).toHaveBeenCalledWith("/same-page");
			});

			it("should restore scroll state for browserHistory navigation", async () => {
				const scrollState = { x: 100, y: 200 };

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				// Apply scroll state directly
				applyScrollState(scrollState);

				expect(window.scrollTo).toHaveBeenCalledWith(100, 200);
			});

			it("should update document title with HTML entity decoding", async () => {
				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						title: { dangerousInnerHTML: "Title with &amp; entities" },
						importURLs: [],
						cssBundles: [],
					}),
				);

				await navigate("/entity-title");
				await vi.runAllTimersAsync();

				expect(document.title).toBe("Title with & entities");
			});

			it("should wait for CSS bundle preloads", async () => {
				// Store RAF callbacks
				const rafCallbacks: FrameRequestCallback[] = [];
				const rafSpy = vi
					.spyOn(window, "requestAnimationFrame")
					.mockImplementation((cb) => {
						rafCallbacks.push(cb);
						return 1;
					});

				const appendChildSpy = vi.spyOn(document.head, "appendChild");

				// Mock the dynamic imports that will be triggered
				vi.doMock("/static/?river_dev=1", () => ({ default: () => {} }));

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						importURLs: [],
						cssBundles: ["/bundle1.css", "/bundle2.css"],
					}),
				);

				// Navigate and handle the promise properly
				const navPromise = navigate("/css-wait");

				// Wait a bit for the navigation to start
				await vi.advanceTimersByTimeAsync(10);

				// Trigger onload for any preload links that were created
				const preloadLinks = appendChildSpy.mock.calls
					.map((call) => call[0])
					.filter(
						(el) => (el as any).tagName === "LINK" && (el as any).getAttribute("rel") === "preload",
					);

				preloadLinks.forEach((link: any) => {
					if (link.onload) {
						link.onload();
					}
				});

				// Now wait for navigation to complete
				await navPromise;
				await vi.runAllTimersAsync();

				// Execute all RAF callbacks
				rafCallbacks.forEach((cb) => cb(0));

				// Verify stylesheet links were added
				const stylesheetCalls = appendChildSpy.mock.calls.filter((call) => {
					const element = call[0] as HTMLElement;
					return (
						element.tagName === "LINK" && element.getAttribute("rel") === "stylesheet"
					);
				});

				expect(stylesheetCalls.length).toBe(2);

				rafSpy.mockRestore();
			});

			it("should dispatch route-change event", async () => {
				const routeChangeListener = vi.fn();
				const cleanup = addListener(addRouteChangeListener, routeChangeListener);

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({ importURLs: [], cssBundles: [] }),
				);

				await navigate("/route-change-test");
				await vi.runAllTimersAsync();

				expect(routeChangeListener).toHaveBeenCalledWith(
					expect.objectContaining({
						detail: expect.objectContaining({
							scrollState: { x: 0, y: 0 },
						}),
					}),
				);

				cleanup();
			});

			it("should apply CSS bundles avoiding duplicates", async () => {
				setupGlobalRiverContext({ publicPathPrefix: "/static" });

				// Store RAF callbacks
				const rafCallbacks: FrameRequestCallback[] = [];
				vi.spyOn(window, "requestAnimationFrame").mockImplementation((cb) => {
					rafCallbacks.push(cb);
					return 1;
				});

				// Track added bundles and mock querySelector properly
				const addedBundles = new Set<string>();
				const originalQuerySelector = document.querySelector.bind(document);
				const querySelectorSpy = vi
					.spyOn(document, "querySelector")
					.mockImplementation((selector) => {
						if (
							typeof selector === "string" &&
							selector.includes("data-river-css-bundle")
						) {
							const match = selector.match(/data-river-css-bundle="([^"]+)"/);
							if (match && addedBundles.has(match[1]!)) {
								const mockElement = document.createElement("link");
								mockElement.setAttribute("data-river-css-bundle", match[1]!);
								return mockElement;
							}
						}
						return originalQuerySelector(selector);
					});

				const appendChildSpy = vi.spyOn(document.head, "appendChild");

				// Mock the dynamic imports
				vi.doMock("/static/?river_dev=1", () => ({ default: () => {} }));

				vi.mocked(fetch).mockResolvedValue(
					createMockResponse({
						importURLs: [],
						cssBundles: ["/styles.css"],
					}),
				);

				// First navigation
				const nav1Promise = navigate("/css-page");
				await vi.advanceTimersByTimeAsync(10);

				// Trigger onload for preload links
				const preloadLinks1 = appendChildSpy.mock.calls
					.map((call) => call[0])
					.filter(
						(el) => (el as any).tagName === "LINK" && (el as any).getAttribute("rel") === "preload",
					);

				preloadLinks1.forEach((link: any) => {
					if (link.onload) link.onload();
				});

				await nav1Promise;
				await vi.runAllTimersAsync();

				// Execute RAF callbacks for first navigation
				rafCallbacks.forEach((cb) => cb(0));
				addedBundles.add("/styles.css");

				// Clear RAF callbacks for second navigation
				rafCallbacks.length = 0;

				// Second navigation
				const nav2Promise = navigate("/css-page");
				await vi.advanceTimersByTimeAsync(10);

				// Trigger onload for any new preload links
				const preloadLinks2 = appendChildSpy.mock.calls
					.slice(preloadLinks1.length)
					.map((call) => call[0])
					.filter(
						(el) => (el as any).tagName === "LINK" && (el as any).getAttribute("rel") === "preload",
					);

				preloadLinks2.forEach((link: any) => {
					if (link.onload) link.onload();
				});

				await nav2Promise;
				await vi.runAllTimersAsync();

				// Execute RAF callbacks for second navigation
				rafCallbacks.forEach((cb) => cb(0));

				// The implementation should have checked for duplicates
				expect(querySelectorSpy).toHaveBeenCalledWith(
					expect.stringContaining('data-river-css-bundle="/styles.css"'),
				);
			});
		});
	});

	// describe("3. Prefetching", () => {
	// 	describe("3.1 Initialization", () => {
	// 		it("should only create handlers for eligible URLs", () => {
	// 			// HTTP URL - eligible
	// 			const httpHandlers = getPrefetchHandlers({ href: "/internal" });
	// 			expect(httpHandlers).toBeDefined();

	// 			// External URL - not eligible
	// 			const externalHandlers = getPrefetchHandlers({
	// 				href: "https://external.com",
	// 			});
	// 			expect(externalHandlers).toBeUndefined();

	// 			// Non-HTTP URL - not eligible
	// 			const mailtoHandlers = getPrefetchHandlers({ href: "mailto:test@test.com" });
	// 			expect(mailtoHandlers).toBeUndefined();
	// 		});

	// 		it("should not prefetch current page", () => {
	// 			window.history.replaceState({}, "", "/current-page");

	// 			const handlers = getPrefetchHandlers({ href: "/current-page" });
	// 			const startSpy = vi.fn();

	// 			if (handlers?.start) {
	// 				vi.spyOn(handlers, "start").mockImplementation(startSpy);
	// 				handlers.start({} as Event);
	// 			}

	// 			vi.advanceTimersByTime(200);
	// 			expect(fetch).not.toHaveBeenCalled();
	// 		});
	// 	});

	// 	describe("3.2 Prefetch Lifecycle", () => {
	// 		it("should start prefetch after configured delay", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			const handlers = getPrefetchHandlers({
	// 				href: "/delayed-prefetch",
	// 				delayMs: 200,
	// 			});

	// 			handlers?.start({} as Event);

	// 			// Not started yet
	// 			vi.advanceTimersByTime(100);
	// 			expect(fetch).not.toHaveBeenCalled();

	// 			// Started after delay
	// 			vi.advanceTimersByTime(100);
	// 			expect(fetch).toHaveBeenCalled();
	// 		});

	// 		it("should execute beforeBegin callback", async () => {
	// 			const beforeBegin = vi.fn();
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			const handlers = getPrefetchHandlers({
	// 				href: "/callback-test",
	// 				beforeBegin,
	// 			});

	// 			handlers?.start({} as Event);
	// 			await vi.advanceTimersByTimeAsync(100);

	// 			expect(beforeBegin).toHaveBeenCalled();
	// 		});

	// 		it("should store prefetch result for reuse", async () => {
	// 			const responseData = {
	// 				title: { dangerousInnerHTML: "Prefetched" },
	// 				importURLs: [],
	// 				cssBundles: [],
	// 			};

	// 			vi.mocked(fetch).mockResolvedValue(createMockResponse(responseData));

	// 			const handlers = getPrefetchHandlers({ href: "/store-result" });
	// 			handlers?.start({} as Event);

	// 			await vi.advanceTimersByTimeAsync(100);
	// 			vi.runAllTimers();

	// 			// Click should use stored result without fetching again
	// 			vi.clearAllMocks();
	// 			const event = new MouseEvent("click");
	// 			vi.spyOn(event, "preventDefault");

	// 			await handlers?.onClick(event);
	// 			vi.runAllTimers();

	// 			expect(fetch).not.toHaveBeenCalled();
	// 			expect(document.title).toBe("Prefetched");
	// 		});

	// 		it("should cancel timeout on stop", () => {
	// 			const handlers = getPrefetchHandlers({ href: "/cancel-timeout" });

	// 			handlers?.start({} as Event);
	// 			handlers?.stop();

	// 			vi.advanceTimersByTime(200);
	// 			expect(fetch).not.toHaveBeenCalled();
	// 		});

	// 		it("should abort prefetch but not upgraded navigation", async () => {
	// 			vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));

	// 			const handlers = getPrefetchHandlers({ href: "/abort-test" });
	// 			handlers?.start({} as Event);

	// 			await vi.advanceTimersByTimeAsync(100);

	// 			// Upgrade to user navigation
	// 			const nav = navigationState.navigations.get("/abort-test");
	// 			if (nav) {
	// 				nav.type = "userNavigation";
	// 			}

	// 			if (!nav) {
	// 				throw new Error("Navigation not found");
	// 			}
	// 			if (!nav.control.abortController) {
	// 				throw new Error("AbortController not set");
	// 			}
	// 			const abortSpy = vi.spyOn(nav.control.abortController, "abort");
	// 			handlers?.stop();

	// 			// Should not abort upgraded navigation
	// 			expect(abortSpy).not.toHaveBeenCalled();
	// 		});

	// 		it("should handle click during prefetch", async () => {
	// 			vi.mocked(fetch).mockImplementation(
	// 				() =>
	// 					new Promise((resolve) =>
	// 						setTimeout(
	// 							() =>
	// 								resolve(
	// 									createMockResponse({
	// 										title: { dangerousInnerHTML: "Eventual" },
	// 										importURLs: [],
	// 										cssBundles: [],
	// 									}),
	// 								),
	// 							200,
	// 						),
	// 					),
	// 			);

	// 			const beforeRender = vi.fn();
	// 			const afterRender = vi.fn();

	// 			const handlers = getPrefetchHandlers({
	// 				href: "/click-during",
	// 				beforeRender,
	// 				afterRender,
	// 			});

	// 			handlers?.start({} as Event);
	// 			await vi.advanceTimersByTimeAsync(100);

	// 			// Click while prefetch is in progress
	// 			const event = new MouseEvent("click");
	// 			vi.spyOn(event, "preventDefault");

	// 			const clickPromise = handlers?.onClick(event);

	// 			// Complete the fetch
	// 			await vi.advanceTimersByTimeAsync(200);
	// 			await clickPromise;
	// 			vi.runAllTimers();

	// 			expect(beforeRender).toHaveBeenCalled();
	// 			expect(afterRender).toHaveBeenCalled();
	// 			expect(document.title).toBe("Eventual");
	// 		});
	// 	});
	// });

	// describe("4. Scroll Restoration", () => {
	// 	describe("4.1 Storage Mechanism", () => {
	// 		it("should use sessionStorage with correct key", () => {
	// 			const scrollState = { x: 100, y: 200 };
	// 			const key = "test-key";

	// 			sessionStorage.setItem(
	// 				"__river__scrollStateMap",
	// 				JSON.stringify([[key, scrollState]]),
	// 			);

	// 			const stored = JSON.parse(
	// 				sessionStorage.getItem("__river__scrollStateMap") || "[]",
	// 			);
	// 			expect(stored).toEqual([[key, scrollState]]);
	// 		});

	// 		it("should limit to 50 entries with FIFO eviction", async () => {
	// 			// Create 51 entries
	// 			const entries: Array<[string, ScrollState]> = [];
	// 			for (let i = 0; i < 51; i++) {
	// 				entries.push([`key-${i}`, { x: i, y: i }]);
	// 			}

	// 			sessionStorage.setItem(
	// 				"__river__scrollStateMap",
	// 				JSON.stringify(entries.slice(0, 50)),
	// 			);

	// 			// Add one more through navigation
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			const history = getHistoryInstance();
	// 			history.push("/trigger-save");

	// 			(window as any).scrollX = 999;
	// 			(window as any).scrollY = 999;

	// 			await navigate("/new-page");
	// 			vi.runAllTimers();

	// 			const stored = JSON.parse(
	// 				sessionStorage.getItem("__river__scrollStateMap") || "[]",
	// 			);
	// 			expect(stored.length).toBe(50);
	// 			expect(stored[0][0]).toBe("key-1"); // First entry evicted
	// 		});

	// 		it("should set manual scroll restoration on init", () => {
	// 			const scrollRestorationSpy = vi.spyOn(history, "scrollRestoration", "set");
	// 			initCustomHistory();
	// 			expect(scrollRestorationSpy).toHaveBeenCalledWith("manual");
	// 		});
	// 	});

	// 	describe("4.2 Saving Scroll State", () => {
	// 		it("should save before navigation", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			const history = getHistoryInstance();
	// 			history.push("/current");

	// 			(window as any).scrollX = 150;
	// 			(window as any).scrollY = 300;

	// 			await navigate("/next");
	// 			vi.runAllTimers();

	// 			const stored = JSON.parse(
	// 				sessionStorage.getItem("__river__scrollStateMap") || "[]",
	// 			);
	// 			const savedEntry = stored.find(([k]: [string]) => k === history.location.key);
	// 			expect(savedEntry?.[1]).toEqual({ x: 150, y: 300 });
	// 		});

	// 		it("should save on POP to different document", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			const history = getHistoryInstance();
	// 			history.push("/page1");
	// 			history.push("/page2");

	// 			(window as any).scrollX = 50;
	// 			(window as any).scrollY = 100;

	// 			// Simulate browser back
	// 			history.back();
	// 			await vi.runAllTimersAsync();

	// 			const stored = JSON.parse(
	// 				sessionStorage.getItem("__river__scrollStateMap") || "[]",
	// 			);
	// 			expect(stored.length).toBeGreaterThan(0);
	// 		});
	// 	});

	// 	describe("4.3 Restoring Scroll State", () => {
	// 		it("should restore on POP navigation with hash addition", async () => {
	// 			const history = getHistoryInstance();
	// 			history.push("/page");

	// 			// Mock element to scroll to
	// 			const element = document.createElement("div");
	// 			element.id = "section";
	// 			const scrollIntoViewSpy = vi.spyOn(element, "scrollIntoView");
	// 			vi.spyOn(document, "getElementById").mockReturnValue(element);

	// 			history.push("/page#section");
	// 			history.back();
	// 			history.forward();

	// 			await vi.runAllTimersAsync();

	// 			expect(scrollIntoViewSpy).toHaveBeenCalled();
	// 		});

	// 		it("should restore saved position on hash removal", async () => {
	// 			const history = getHistoryInstance();
	// 			history.push("/page");

	// 			const savedState = { x: 75, y: 150 };
	// 			sessionStorage.setItem(
	// 				"__river__scrollStateMap",
	// 				JSON.stringify([[history.location.key, savedState]]),
	// 			);

	// 			history.push("/page#hash");
	// 			history.back();

	// 			await vi.runAllTimersAsync();

	// 			expect(window.scrollTo).toHaveBeenCalledWith(75, 150);
	// 		});

	// 		it("should scroll to top for standard navigation", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			await navigate("/new-page");
	// 			vi.runAllTimers();

	// 			const routeChangeListener = vi.fn();
	// 			addRouteChangeListener(routeChangeListener);

	// 			expect(routeChangeListener).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({
	// 						scrollState: { x: 0, y: 0 },
	// 					}),
	// 				}),
	// 			);
	// 		});

	// 		it("should scroll to element for navigation with hash", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			const element = document.createElement("div");
	// 			element.id = "target";
	// 			document.body.appendChild(element);
	// 			const scrollIntoViewSpy = vi.spyOn(element, "scrollIntoView");

	// 			await navigate("/page#target");
	// 			vi.runAllTimers();

	// 			// Apply scroll state from route change event
	// 			applyScrollState({ hash: "target" });

	// 			expect(scrollIntoViewSpy).toHaveBeenCalled();
	// 		});

	// 		it("should fallback to element scroll without saved state", () => {
	// 			const element = document.createElement("div");
	// 			element.id = "fallback";
	// 			document.body.appendChild(element);
	// 			const scrollIntoViewSpy = vi.spyOn(element, "scrollIntoView");

	// 			applyScrollState({ hash: "fallback" });

	// 			expect(scrollIntoViewSpy).toHaveBeenCalled();
	// 		});
	// 	});

	// 	describe("4.4 Page Refresh Handling", () => {
	// 		it("should save scroll state on unload", () => {
	// 			(window as any).scrollX = 200;
	// 			(window as any).scrollY = 400;

	// 			window.dispatchEvent(new Event("beforeunload"));

	// 			const saved = JSON.parse(
	// 				sessionStorage.getItem("__river__pageRefreshScrollState") || "{}",
	// 			);
	// 			expect(saved).toMatchObject({
	// 				x: 200,
	// 				y: 400,
	// 				href: window.location.href,
	// 			});
	// 			expect(saved.unix).toBeDefined();
	// 		});

	// 		it("should restore scroll state after refresh within 5 seconds", async () => {
	// 			const scrollState = {
	// 				x: 250,
	// 				y: 500,
	// 				unix: Date.now() - 1000, // 1 second ago
	// 				href: window.location.href,
	// 			};

	// 			sessionStorage.setItem(
	// 				"__river__pageRefreshScrollState",
	// 				JSON.stringify(scrollState),
	// 			);

	// 			const requestAnimationFrameSpy = vi.spyOn(window, "requestAnimationFrame");
	// 			requestAnimationFrameSpy.mockImplementation((cb) => {
	// 				cb(0);
	// 				return 0;
	// 			});

	// 			await initClient(() => {});

	// 			expect(window.scrollTo).toHaveBeenCalledWith(250, 500);
	// 			expect(sessionStorage.getItem("__river__pageRefreshScrollState")).toBeNull();
	// 		});

	// 		it("should not restore if different URL", async () => {
	// 			const scrollState = {
	// 				x: 250,
	// 				y: 500,
	// 				unix: Date.now() - 1000,
	// 				href: "/different-page",
	// 			};

	// 			sessionStorage.setItem(
	// 				"__river__pageRefreshScrollState",
	// 				JSON.stringify(scrollState),
	// 			);

	// 			await initClient(() => {});

	// 			expect(window.scrollTo).not.toHaveBeenCalledWith(250, 500);
	// 		});

	// 		it("should not restore if more than 5 seconds", async () => {
	// 			const scrollState = {
	// 				x: 250,
	// 				y: 500,
	// 				unix: Date.now() - 6000, // 6 seconds ago
	// 				href: window.location.href,
	// 			};

	// 			sessionStorage.setItem(
	// 				"__river__pageRefreshScrollState",
	// 				JSON.stringify(scrollState),
	// 			);

	// 			await initClient(() => {});

	// 			expect(window.scrollTo).not.toHaveBeenCalledWith(250, 500);
	// 		});
	// 	});
	// });

	// describe("5. Redirects", () => {
	// 	describe("5.1 Request Configuration", () => {
	// 		it("should include X-Accepts-Client-Redirect header", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			await navigate("/test");

	// 			const headers = vi.mocked(fetch).mock.calls[0]?.[1]?.headers as Headers;
	// 			expect(headers.get("X-Accepts-Client-Redirect")).toBe("1");
	// 		});
	// 	});

	// 	describe("5.2 Response Headers Priority", () => {
	// 		it("should prioritize X-River-Reload over other redirects", async () => {
	// 			Object.defineProperty(window, "location", {
	// 				value: { href: "" },
	// 				writable: true,
	// 			});

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse(null, {
	// 					headers: {
	// 						"X-River-Reload": "/force-reload",
	// 						"X-Client-Redirect": "/ignored",
	// 					},
	// 				}),
	// 			);

	// 			await navigate("/test");
	// 			vi.runAllTimers();

	// 			expect(window.location.href).toContain("river_reload=");
	// 		});

	// 		it("should handle native browser redirect for GET", async () => {
	// 			const redirectedResponse = createMockResponse({
	// 				importURLs: [],
	// 				cssBundles: [],
	// 			});
	// 			Object.defineProperty(redirectedResponse, "redirected", {
	// 				value: true,
	// 				writable: false,
	// 			});
	// 			Object.defineProperty(redirectedResponse, "url", {
	// 				value: "http://localhost:3000/redirected",
	// 				writable: false,
	// 			});

	// 			vi.mocked(fetch).mockResolvedValue(redirectedResponse);

	// 			await navigate("/original");
	// 			vi.runAllTimers();

	// 			// Should complete navigation to redirected URL
	// 			expect(window.location.pathname).toBe("/redirected");
	// 		});

	// 		it("should ignore redirect for non-GET requests", async () => {
	// 			const redirectedResponse = createMockResponse(null);
	// 			Object.defineProperty(redirectedResponse, "redirected", {
	// 				value: true,
	// 				writable: false,
	// 			});

	// 			vi.mocked(fetch).mockResolvedValue(redirectedResponse);

	// 			const result = await submit("/api", { method: "POST" });

	// 			// Should return null for non-GET redirects
	// 			expect(result.success).toBe(false);
	// 		});

	// 		it("should handle X-Client-Redirect as lowest priority", async () => {
	// 			vi.mocked(fetch)
	// 				.mockResolvedValueOnce(
	// 					createMockResponse(null, {
	// 						headers: { "X-Client-Redirect": "/client-redirect" },
	// 					}),
	// 				)
	// 				.mockResolvedValueOnce(
	// 					createMockResponse({
	// 						title: { dangerousInnerHTML: "Client Redirected" },
	// 						importURLs: [],
	// 						cssBundles: [],
	// 					}),
	// 				);

	// 			await navigate("/test");
	// 			vi.runAllTimers();

	// 			expect(document.title).toBe("Client Redirected");
	// 		});
	// 	});

	// 	describe("5.3 Build ID Tracking", () => {
	// 		it("should update build ID from response header", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse(
	// 					{ importURLs: [], cssBundles: [] },
	// 					{ headers: { "X-River-Build-Id": "new-build-789" } },
	// 				),
	// 			);

	// 			await navigate("/new-build");
	// 			vi.runAllTimers();

	// 			expect(getBuildID()).toBe("new-build-789");
	// 		});

	// 		it("should dispatch build-id event before redirect", async () => {
	// 			const buildIdListener = vi.fn();
	// 			addBuildIDListener(buildIdListener);

	// 			vi.mocked(fetch)
	// 				.mockResolvedValueOnce(
	// 					createMockResponse(null, {
	// 						headers: {
	// 							"X-River-Build-Id": "redirect-build",
	// 							"X-Client-Redirect": "/redirect",
	// 						},
	// 					}),
	// 				)
	// 				.mockResolvedValueOnce(
	// 					createMockResponse({ importURLs: [], cssBundles: [] }),
	// 				);

	// 			await navigate("/test");
	// 			vi.runAllTimers();

	// 			expect(buildIdListener).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({
	// 						newID: "redirect-build",
	// 					}),
	// 				}),
	// 			);
	// 		});
	// 	});

	// 	describe("5.4 Redirect Strategies", () => {
	// 		it("should use soft redirect for internal URLs", async () => {
	// 			vi.mocked(fetch)
	// 				.mockResolvedValueOnce(
	// 					createMockResponse(null, {
	// 						headers: { "X-Client-Redirect": "/internal-redirect" },
	// 					}),
	// 				)
	// 				.mockResolvedValueOnce(
	// 					createMockResponse({
	// 						title: { dangerousInnerHTML: "Soft Redirected" },
	// 						importURLs: [],
	// 						cssBundles: [],
	// 					}),
	// 				);

	// 			await navigate("/test");
	// 			vi.runAllTimers();

	// 			// Should navigate without page reload
	// 			expect(document.title).toBe("Soft Redirected");
	// 			expect(fetch).toHaveBeenCalledTimes(2);
	// 		});

	// 		it("should use hard redirect for external URLs", async () => {
	// 			Object.defineProperty(window, "location", {
	// 				value: { href: "" },
	// 				writable: true,
	// 			});

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse(null, {
	// 					headers: { "X-Client-Redirect": "https://external.com" },
	// 				}),
	// 			);

	// 			await navigate("/test");
	// 			vi.runAllTimers();

	// 			expect(window.location.href).toBe("https://external.com");
	// 		});

	// 		it("should add river_reload param for forced internal redirect", async () => {
	// 			Object.defineProperty(window, "location", {
	// 				value: { href: "" },
	// 				writable: true,
	// 			});

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse(null, {
	// 					headers: {
	// 						"X-River-Reload": "/force-internal",
	// 						"X-River-Build-Id": "force-build",
	// 					},
	// 				}),
	// 			);

	// 			await navigate("/test");
	// 			vi.runAllTimers();

	// 			expect(window.location.href).toContain("river_reload=force-build");
	// 		});

	// 		it("should handle max redirect limit", async () => {
	// 			// Create a redirect loop
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse(null, {
	// 					headers: { "X-Client-Redirect": "/loop" },
	// 				}),
	// 			);

	// 			// Start with high redirect count
	// 			const control = beginNavigation({
	// 				href: "/test",
	// 				navigationType: "redirect",
	// 				redirectCount: 10,
	// 			});

	// 			await control.promise;

	// 			// Should stop after max redirects
	// 			expect(fetch).toHaveBeenCalledTimes(1);
	// 		});
	// 	});

	// 	describe("5.5 Error Handling", () => {
	// 		it("should ignore non-HTTP redirect URLs", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse(null, {
	// 					headers: { "X-Client-Redirect": "mailto:test@test.com" },
	// 				}),
	// 			);

	// 			await navigate("/test");
	// 			vi.runAllTimers();

	// 			// Should complete navigation without redirect
	// 			expect(window.location.href).not.toContain("mailto:");
	// 		});

	// 		it("should fallback to hard reload on GET network failure", async () => {
	// 			Object.defineProperty(window, "location", {
	// 				value: { href: "" },
	// 				writable: true,
	// 			});

	// 			vi.mocked(fetch).mockRejectedValue(new Error("Network error"));

	// 			await navigate("/network-fail");
	// 			vi.runAllTimers();

	// 			expect(window.location.href).toContain("/network-fail");
	// 		});

	// 		it("should not fallback for prefetch network failure", async () => {
	// 			vi.mocked(fetch).mockRejectedValue(new Error("Network error"));

	// 			const handlers = getPrefetchHandlers({ href: "/prefetch-fail" });
	// 			handlers?.start({} as Event);

	// 			await vi.advanceTimersByTimeAsync(100);
	// 			vi.runAllTimers();

	// 			// Should not change location
	// 			expect(window.location.href).not.toContain("/prefetch-fail");
	// 		});
	// 	});

	// 	describe("5.6 URL Cleanup", () => {
	// 		it("should remove river_reload param on init", async () => {
	// 			window.history.replaceState({}, "", "/?river_reload=123&other=param");

	// 			const history = getHistoryInstance();
	// 			const replaceSpy = vi.spyOn(history, "replace");

	// 			await initClient(() => {});

	// 			expect(replaceSpy).toHaveBeenCalledWith("/?other=param");
	// 		});
	// 	});
	// });

	// describe("6. Form Submissions", () => {
	// 	describe("6.1 Submit Function", () => {
	// 		it("should deduplicate same URL+method submissions", async () => {
	// 			vi.mocked(fetch).mockImplementation(
	// 				() => new Promise(() => {}), // Never resolve
	// 			);

	// 			const controller1 = (
	// 				navigationState.submissions.get("/api/resourcePOST") as any
	// 			)?.controller;
	// 			const abort1 = controller1 ? vi.spyOn(controller1, "abort") : null;

	// 			submit("/api/resource", { method: "POST" });
	// 			submit("/api/resource", { method: "POST" });

	// 			expect(navigationState.submissions.size).toBe(1);
	// 			if (abort1) {
	// 				expect(abort1).toHaveBeenCalled();
	// 			}
	// 		});

	// 		it("should set isSubmitting loading state", async () => {
	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			vi.mocked(fetch).mockResolvedValue(createMockResponse({ result: "success" }));

	// 			const submitPromise = submit("/api/data", { method: "POST" });
	// 			vi.runAllTimers();

	// 			expect(statusListener).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({ isSubmitting: true }),
	// 				}),
	// 			);

	// 			await submitPromise;
	// 			vi.runAllTimers();

	// 			expect(statusListener).toHaveBeenLastCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({ isSubmitting: false }),
	// 				}),
	// 			);
	// 		});

	// 		it("should send FormData as-is", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(createMockResponse({}));

	// 			const formData = new FormData();
	// 			formData.append("field", "value");

	// 			await submit("/api/form", {
	// 				method: "POST",
	// 				body: formData,
	// 			});

	// 			expect(fetch).toHaveBeenCalledWith(
	// 				expect.any(URL),
	// 				expect.objectContaining({
	// 					body: formData,
	// 				}),
	// 			);
	// 		});

	// 		it("should send string body as-is", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(createMockResponse({}));

	// 			const stringBody = "raw string data";

	// 			await submit("/api/string", {
	// 				method: "POST",
	// 				body: stringBody,
	// 			});

	// 			expect(fetch).toHaveBeenCalledWith(
	// 				expect.any(URL),
	// 				expect.objectContaining({
	// 					body: stringBody,
	// 				}),
	// 			);
	// 		});

	// 		it("should JSON stringify other body types", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(createMockResponse({}));

	// 			const objectBody = { key: "value", nested: { data: true } };

	// 			await submit("/api/json", {
	// 				method: "POST",
	// 				body: objectBody as any,
	// 			});

	// 			expect(fetch).toHaveBeenCalledWith(
	// 				expect.any(URL),
	// 				expect.objectContaining({
	// 					body: JSON.stringify(objectBody),
	// 				}),
	// 			);
	// 		});

	// 		it("should handle redirect responses", async () => {
	// 			vi.mocked(fetch)
	// 				.mockResolvedValueOnce(
	// 					createMockResponse(null, {
	// 						headers: { "X-Client-Redirect": "/after-submit" },
	// 					}),
	// 				)
	// 				.mockResolvedValueOnce(
	// 					createMockResponse({
	// 						title: { dangerousInnerHTML: "After Submit" },
	// 						importURLs: [],
	// 						cssBundles: [],
	// 					}),
	// 				);

	// 			const result = await submit("/api/action", { method: "POST" });

	// 			expect(result.success).toBe(true);
	// 			expect(document.title).toBe("After Submit");
	// 		});

	// 		it("should auto-revalidate after non-GET submission", async () => {
	// 			vi.mocked(fetch)
	// 				.mockResolvedValueOnce(createMockResponse({ submitted: true }))
	// 				.mockResolvedValueOnce(
	// 					createMockResponse({
	// 						title: { dangerousInnerHTML: "Revalidated" },
	// 						importURLs: [],
	// 						cssBundles: [],
	// 					}),
	// 				);

	// 			await submit("/api/mutate", { method: "POST" });
	// 			vi.runAllTimers();

	// 			expect(fetch).toHaveBeenCalledTimes(2);
	// 			expect(document.title).toBe("Revalidated");
	// 		});

	// 		it("should not auto-revalidate after GET submission", async () => {
	// 			vi.mocked(fetch).mockResolvedValueOnce(
	// 				createMockResponse({ data: "search results" }),
	// 			);

	// 			await submit("/api/search", { method: "GET" });
	// 			vi.runAllTimers();

	// 			expect(fetch).toHaveBeenCalledTimes(1);
	// 		});

	// 		it("should manage loading state transition to revalidation", async () => {
	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			vi.mocked(fetch)
	// 				.mockResolvedValueOnce(createMockResponse({}))
	// 				.mockResolvedValueOnce(
	// 					createMockResponse({ importURLs: [], cssBundles: [] }),
	// 				);

	// 			await submit("/api/update", { method: "PUT" });
	// 			vi.runAllTimers();

	// 			// Should transition from submission to revalidation without gap
	// 			const calls = statusListener.mock.calls;
	// 			const submittingOffCall = calls.find(
	// 				(call) => call[0].detail.isSubmitting === false,
	// 			);
	// 			const revalidatingOnCall = calls.find(
	// 				(call) => call[0].detail.isRevalidating === true,
	// 			);

	// 			if (submittingOffCall && revalidatingOnCall) {
	// 				const submittingOffIndex = calls.indexOf(submittingOffCall);
	// 				const revalidatingOnIndex = calls.indexOf(revalidatingOnCall);
	// 				expect(revalidatingOnIndex).toBeLessThan(submittingOffIndex);
	// 			}
	// 		});

	// 		it("should return success with data", async () => {
	// 			const responseData = { id: 123, status: "created" };
	// 			vi.mocked(fetch).mockResolvedValue(createMockResponse(responseData));

	// 			const result = await submit("/api/create", { method: "POST" });

	// 			expect(result).toEqual({
	// 				success: true,
	// 				data: responseData,
	// 			});
	// 		});

	// 		it("should return error on failure", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				new Response(null, { status: 500, statusText: "Internal Server Error" }),
	// 			);

	// 			const result = await submit("/api/fail", { method: "POST" });

	// 			expect(result).toEqual({
	// 				success: false,
	// 				error: "500",
	// 			});
	// 		});

	// 		it("should handle network errors", async () => {
	// 			vi.mocked(fetch).mockRejectedValue(new Error("Network failure"));

	// 			const result = await submit("/api/network-error", { method: "POST" });

	// 			expect(result).toEqual({
	// 				success: false,
	// 				error: "Network failure",
	// 			});
	// 		});

	// 		it("should handle abort errors silently", async () => {
	// 			const abortError = new Error("Aborted");
	// 			abortError.name = "AbortError";
	// 			vi.mocked(fetch).mockRejectedValue(abortError);

	// 			const result = await submit("/api/abort", { method: "POST" });

	// 			expect(result).toEqual({
	// 				success: false,
	// 				error: "Aborted",
	// 			});
	// 		});
	// 	});

	// 	describe("6.2 Revalidate Function", () => {
	// 		it("should debounce revalidation calls", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			// Call revalidate multiple times quickly
	// 			revalidate();
	// 			revalidate();
	// 			revalidate();

	// 			// Should only result in one fetch
	// 			await vi.advanceTimersByTimeAsync(10);
	// 			expect(fetch).toHaveBeenCalledTimes(1);
	// 		});

	// 		it("should use revalidation navigation type", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			await revalidate();
	// 			vi.runAllTimers();

	// 			// Check that navigation was created with correct type
	// 			const navigations = Array.from(navigationState.navigations.values());
	// 			const revalidationNav = navigations.find((n) => n.type === "revalidation");
	// 			expect(revalidationNav).toBeDefined();
	// 		});

	// 		it("should target current window.location.href", async () => {
	// 			window.history.replaceState({}, "", "/current-page?param=value");

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			await revalidate();

	// 			expect(fetch).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					href: "http://localhost:3000/current-page?param=value&river_json=1",
	// 				}),
	// 				expect.any(Object),
	// 			);
	// 		});
	// 	});
	// });

	// describe("7. Events System", () => {
	// 	describe("7.1 Loading States (river:status)", () => {
	// 		it("should track isNavigating state", async () => {
	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			await navigate("/nav-state");
	// 			vi.runAllTimers();

	// 			// Should have dispatched navigating true then false
	// 			expect(statusListener).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({
	// 						isNavigating: true,
	// 						isSubmitting: false,
	// 						isRevalidating: false,
	// 					}),
	// 				}),
	// 			);

	// 			expect(statusListener).toHaveBeenLastCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({
	// 						isNavigating: false,
	// 						isSubmitting: false,
	// 						isRevalidating: false,
	// 					}),
	// 				}),
	// 			);
	// 		});

	// 		it("should track isSubmitting state", async () => {
	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			vi.mocked(fetch).mockResolvedValue(createMockResponse({}));

	// 			await submit("/api/submit", { method: "POST" });
	// 			vi.runAllTimers();

	// 			// Check for isSubmitting true
	// 			expect(statusListener).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({
	// 						isSubmitting: true,
	// 					}),
	// 				}),
	// 			);
	// 		});

	// 		it("should track isRevalidating state", async () => {
	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			await revalidate();
	// 			vi.runAllTimers();

	// 			expect(statusListener).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({
	// 						isRevalidating: true,
	// 					}),
	// 				}),
	// 			);
	// 		});

	// 		it("should debounce status events by 5ms", async () => {
	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));

	// 			// Trigger multiple state changes quickly
	// 			beginNavigation({ href: "/1", navigationType: "userNavigation" });
	// 			beginNavigation({ href: "/2", navigationType: "userNavigation" });

	// 			// No events yet
	// 			expect(statusListener).not.toHaveBeenCalled();

	// 			// After debounce
	// 			vi.advanceTimersByTime(5);
	// 			expect(statusListener).toHaveBeenCalledTimes(1);
	// 		});

	// 		it("should deduplicate identical status events", async () => {
	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			// Set same state multiple times
	// 			const mockFetch = () => new Promise(() => {});
	// 			vi.mocked(fetch).mockImplementation(mockFetch as any);

	// 			beginNavigation({ href: "/same", navigationType: "userNavigation" });
	// 			vi.advanceTimersByTime(5);

	// 			const callCount = statusListener.mock.calls.length;

	// 			// Try to trigger same state again
	// 			beginNavigation({ href: "/same2", navigationType: "userNavigation" });
	// 			vi.advanceTimersByTime(5);

	// 			// Should not dispatch duplicate
	// 			expect(statusListener).toHaveBeenCalledTimes(callCount);
	// 		});

	// 		it("should provide synchronous access via getStatus()", () => {
	// 			const initialStatus = getStatus();
	// 			expect(initialStatus).toEqual({
	// 				isNavigating: false,
	// 				isSubmitting: false,
	// 				isRevalidating: false,
	// 			});

	// 			// Start navigation
	// 			vi.mocked(fetch).mockImplementation(() => new Promise(() => {}));
	// 			beginNavigation({ href: "/sync", navigationType: "userNavigation" });

	// 			const duringNavStatus = getStatus();
	// 			expect(duringNavStatus.isNavigating).toBe(true);
	// 		});
	// 	});

	// 	describe("7.2 Route Changes (river:route-change)", () => {
	// 		it("should fire after navigation completes", async () => {
	// 			const routeChangeListener = vi.fn();
	// 			addRouteChangeListener(routeChangeListener);

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			await navigate("/route-change");
	// 			vi.runAllTimers();

	// 			expect(routeChangeListener).toHaveBeenCalledTimes(1);
	// 		});

	// 		it("should include scroll state in event detail", async () => {
	// 			const routeChangeListener = vi.fn();
	// 			addRouteChangeListener(routeChangeListener);

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			await navigate("/with-hash#section");
	// 			vi.runAllTimers();

	// 			expect(routeChangeListener).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({
	// 						scrollState: { hash: "section" },
	// 					}),
	// 				}),
	// 			);
	// 		});

	// 		it("should fire before UI updates", async () => {
	// 			const routeChangeListener = vi.fn();
	// 			addRouteChangeListener(routeChangeListener);

	// 			const oldTitle = document.title;

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({
	// 					title: { dangerousInnerHTML: "New Title" },
	// 					importURLs: [],
	// 					cssBundles: [],
	// 				}),
	// 			);

	// 			routeChangeListener.mockImplementation(() => {
	// 				// Title should not be updated yet
	// 				expect(document.title).toBe(oldTitle);
	// 			});

	// 			await navigate("/before-ui");
	// 			vi.runAllTimers();

	// 			expect(routeChangeListener).toHaveBeenCalled();
	// 			expect(document.title).toBe("New Title");
	// 		});
	// 	});

	// 	describe("7.3 Location Changes (river:location)", () => {
	// 		it("should fire when location.key changes", async () => {
	// 			const locationListener = vi.fn();
	// 			addLocationListener(locationListener);

	// 			const history = getHistoryInstance();
	// 			history.push("/new-location");

	// 			await vi.runAllTimersAsync();

	// 			expect(locationListener).toHaveBeenCalled();
	// 		});

	// 		it("should provide current location via getLocation()", () => {
	// 			window.history.replaceState({}, "", "/test-path?query=1#hash");

	// 			const location = getLocation();
	// 			expect(location).toEqual({
	// 				pathname: "/test-path",
	// 				search: "?query=1",
	// 				hash: "#hash",
	// 			});
	// 		});
	// 	});

	// 	describe("7.4 Build ID Changes (river:build-id)", () => {
	// 		it("should fire on build ID mismatch", async () => {
	// 			const buildIdListener = vi.fn();
	// 			addBuildIDListener(buildIdListener);

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse(
	// 					{ importURLs: [], cssBundles: [] },
	// 					{ headers: { "X-River-Build-Id": "new-build-456" } },
	// 				),
	// 			);

	// 			await navigate("/new-build");
	// 			vi.runAllTimers();

	// 			expect(buildIdListener).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					detail: {
	// 						oldID: "1",
	// 						newID: "new-build-456",
	// 						fromGETAction: false,
	// 					},
	// 				}),
	// 			);
	// 		});

	// 		it("should update global buildID before dispatching", async () => {
	// 			const buildIdListener = vi.fn();
	// 			addBuildIDListener(buildIdListener);

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse(
	// 					{ importURLs: [], cssBundles: [] },
	// 					{ headers: { "X-River-Build-Id": "updated-build" } },
	// 				),
	// 			);

	// 			buildIdListener.mockImplementation(() => {
	// 				// Build ID should already be updated
	// 				expect(getBuildID()).toBe("updated-build");
	// 			});

	// 			await navigate("/check-update");
	// 			vi.runAllTimers();

	// 			expect(buildIdListener).toHaveBeenCalled();
	// 		});

	// 		it("should indicate fromGETAction for GET submissions", async () => {
	// 			const buildIdListener = vi.fn();
	// 			addBuildIDListener(buildIdListener);

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({}, { headers: { "X-River-Build-Id": "get-build" } }),
	// 			);

	// 			await submit("/api/search", { method: "GET" });
	// 			vi.runAllTimers();

	// 			expect(buildIdListener).toHaveBeenCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({
	// 						fromGETAction: true,
	// 					}),
	// 				}),
	// 			);
	// 		});

	// 		it("should provide current build ID via getBuildID()", () => {
	// 			setupGlobalRiverContext({ buildID: "test-build-999" });
	// 			expect(getBuildID()).toBe("test-build-999");
	// 		});
	// 	});
	// });

	// describe("8. Component & Module Loading", () => {
	// 	describe("8.1 Initial Load", () => {
	// 		it("should dynamically import modules from importURLs", async () => {
	// 			const mockModule = { default: () => "Component" };
	// 			vi.doMock("/static/module.js?river_dev=1", () => mockModule);

	// 			setupGlobalRiverContext({
	// 				importURLs: ["/module.js"],
	// 				publicPathPrefix: "/static",
	// 			});

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({
	// 					importURLs: ["/module.js"],
	// 					exportKeys: ["default"],
	// 					cssBundles: [],
	// 				}),
	// 			);

	// 			await navigate("/with-module");
	// 			vi.runAllTimers();

	// 			expect(internal_RiverClientGlobal.get("activeComponents")).toBeDefined();
	// 		});

	// 		it("should map modules using exportKeys array", async () => {
	// 			const mockModule = {
	// 				default: () => "DefaultExport",
	// 				NamedExport: () => "NamedExport",
	// 			};
	// 			vi.doMock("/static/multi-export.js?river_dev=1", () => mockModule);

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({
	// 					importURLs: ["/multi-export.js", "/multi-export.js"],
	// 					exportKeys: ["default", "NamedExport"],
	// 					cssBundles: [],
	// 				}),
	// 			);

	// 			await navigate("/multi-export");
	// 			vi.runAllTimers();

	// 			const components = internal_RiverClientGlobal.get("activeComponents");
	// 			expect(components).toHaveLength(2);
	// 		});
	// 	});

	// 	describe("8.2 Error Boundaries", () => {
	// 		it("should use outermostErrorIdx to find error component", async () => {
	// 			const errorComponent = () => "Custom Error Component";
	// 			vi.doMock("/static/error.js?river_dev=1", () => ({
	// 				default: errorComponent,
	// 			}));

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({
	// 					importURLs: ["/page.js", "/error.js"],
	// 					exportKeys: ["default", "default"],
	// 					outermostErrorIdx: 1,
	// 					cssBundles: [],
	// 				}),
	// 			);

	// 			await navigate("/with-error");
	// 			vi.runAllTimers();

	// 			expect(internal_RiverClientGlobal.get("activeErrorBoundary")).toBeDefined();
	// 		});

	// 		it("should fallback to defaultErrorBoundary if not found", async () => {
	// 			const defaultError = () => "Default Error";
	// 			setupGlobalRiverContext({ defaultErrorBoundary: defaultError });

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({
	// 					importURLs: [],
	// 					outermostErrorIdx: 0, // No component at this index
	// 					cssBundles: [],
	// 				}),
	// 			);

	// 			await navigate("/missing-error");
	// 			vi.runAllTimers();

	// 			expect(internal_RiverClientGlobal.get("activeErrorBoundary")).toBe(
	// 				defaultError,
	// 			);
	// 		});

	// 		it("should use errorExportKey for non-default exports", async () => {
	// 			const namedErrorComponent = () => "Named Error Export";
	// 			vi.doMock("/static/named-error.js?river_dev=1", () => ({
	// 				ErrorBoundary: namedErrorComponent,
	// 			}));

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({
	// 					importURLs: ["/named-error.js"],
	// 					outermostErrorIdx: 0,
	// 					errorExportKey: "ErrorBoundary",
	// 					cssBundles: [],
	// 				}),
	// 			);

	// 			await navigate("/named-error");
	// 			vi.runAllTimers();

	// 			expect(internal_RiverClientGlobal.get("activeErrorBoundary")).toBeDefined();
	// 		});
	// 	});

	// 	describe("8.3 URL Resolution", () => {
	// 		it("should add ?river_dev=1 in development", async () => {
	// 			const originalEnv = import.meta.env.DEV;
	// 			(import.meta.env as any).DEV = true;

	// 			setupGlobalRiverContext({
	// 				viteDevURL: "http://localhost:5173",
	// 				publicPathPrefix: "/static",
	// 			});

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({
	// 					importURLs: ["/dev-module.js"],
	// 					cssBundles: [],
	// 				}),
	// 			);

	// 			// Mock the import to verify the URL
	// 			let importedUrl = "";
	// 			vi.doMock("http://localhost:5173/dev-module.js?river_dev=1", () => {
	// 				importedUrl = "http://localhost:5173/dev-module.js?river_dev=1";
	// 				return { default: () => {} };
	// 			});

	// 			await navigate("/dev-test");
	// 			vi.runAllTimers();

	// 			// In dev, should use viteDevURL with ?river_dev=1
	// 			expect(importedUrl).toContain("?river_dev=1");

	// 			(import.meta.env as any).DEV = originalEnv;
	// 		});

	// 		it("should use publicPathPrefix in production", async () => {
	// 			const originalEnv = import.meta.env.DEV;
	// 			(import.meta.env as any).DEV = false;

	// 			setupGlobalRiverContext({
	// 				publicPathPrefix: "/assets",
	// 			});

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({
	// 					importURLs: ["/prod-module.js"],
	// 					cssBundles: [],
	// 				}),
	// 			);

	// 			vi.doMock("/assets/prod-module.js", () => ({
	// 				default: () => {},
	// 			}));

	// 			await navigate("/prod-test");
	// 			vi.runAllTimers();

	// 			// Verify it tried to import from the correct path
	// 			expect(internal_RiverClientGlobal.get("activeComponents")).toBeDefined();

	// 			(import.meta.env as any).DEV = originalEnv;
	// 		});

	// 		it("should handle trailing slashes correctly", async () => {
	// 			setupGlobalRiverContext({
	// 				publicPathPrefix: "/static/", // With trailing slash
	// 			});

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({
	// 					importURLs: ["/module.js"],
	// 					cssBundles: ["/styles.css"],
	// 				}),
	// 			);

	// 			await navigate("/slash-test");
	// 			vi.runAllTimers();

	// 			// Should not double-slash
	// 			const cssLinks = document.querySelectorAll('link[rel="stylesheet"]');
	// 			cssLinks.forEach((link) => {
	// 				const href = link.getAttribute("href");
	// 				expect(href).not.toContain("//");
	// 			});
	// 		});
	// 	});
	// });

	// describe("9. Development Features", () => {
	// 	describe("9.1 HMR Support", () => {
	// 		it("should setup HMR listener in development", async () => {
	// 			const originalEnv = import.meta.env.DEV;
	// 			(import.meta.env as any).DEV = true;

	// 			const hotMock = {
	// 				on: vi.fn(),
	// 			};
	// 			(import.meta.hot as any) = hotMock;

	// 			await initClient(() => {});

	// 			expect(hotMock.on).toHaveBeenCalledWith(
	// 				"vite:afterUpdate",
	// 				expect.any(Function),
	// 			);

	// 			(import.meta.env as any).DEV = originalEnv;
	// 		});

	// 		it("should track files with HMR via hmrRunClientLoaders", () => {
	// 			const originalEnv = import.meta.env.DEV;
	// 			(import.meta.env as any).DEV = true;

	// 			const hotMock = {
	// 				on: vi.fn(),
	// 			};
	// 			(import.meta.hot as any) = hotMock;

	// 			const importMeta = {
	// 				url: "http://localhost:5173/routes/home.tsx",
	// 				hot: hotMock,
	// 			};

	// 			hmrRunClientLoaders(importMeta as any);
	// 			hmrRunClientLoaders(importMeta as any); // Call twice

	// 			// Should only register once per file
	// 			expect(hotMock.on).toHaveBeenCalledTimes(1);

	// 			(import.meta.env as any).DEV = originalEnv;
	// 		});

	// 		it("should revalidate on relevant file changes", async () => {
	// 			const originalEnv = import.meta.env.DEV;
	// 			(import.meta.env as any).DEV = true;

	// 			let updateCallback: any;
	// 			const hotMock = {
	// 				on: vi.fn((event, cb) => {
	// 					if (event === "vite:afterUpdate") {
	// 						updateCallback = cb;
	// 					}
	// 				}),
	// 			};
	// 			(import.meta.hot as any) = hotMock;

	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			const importMeta = {
	// 				url: "http://localhost:5173/routes/test.tsx",
	// 				hot: hotMock,
	// 			};

	// 			hmrRunClientLoaders(importMeta as any);

	// 			// Trigger HMR update
	// 			updateCallback({
	// 				updates: [
	// 					{
	// 						type: "js-update",
	// 						path: "/routes/test.tsx",
	// 					},
	// 				],
	// 			});

	// 			// Should trigger revalidation (debounced)
	// 			await vi.advanceTimersByTimeAsync(10);
	// 			expect(fetch).toHaveBeenCalled();

	// 			(import.meta.env as any).DEV = originalEnv;
	// 		});

	// 		it("should update latestHMRTimestamp on updates", async () => {
	// 			const originalEnv = import.meta.env.DEV;
	// 			(import.meta.env as any).DEV = true;

	// 			let updateCallback: any;
	// 			const hotMock = {
	// 				on: vi.fn((event, cb) => {
	// 					if (event === "vite:afterUpdate") {
	// 						updateCallback = cb;
	// 					}
	// 				}),
	// 			};
	// 			(import.meta.hot as any) = hotMock;

	// 			await initClient(() => {});

	// 			const beforeTimestamp = Date.now();
	// 			updateCallback({});

	// 			// Should log HMR update
	// 			expect(console.info).toHaveBeenCalledWith(
	// 				expect.stringContaining("HMR update detected"),
	// 				expect.any(Number),
	// 			);

	// 			(import.meta.env as any).DEV = originalEnv;
	// 		});
	// 	});

	// 	describe("9.2 HMR State", () => {
	// 		it("should expose __waveRevalidate in development", async () => {
	// 			const originalEnv = import.meta.env.DEV;
	// 			(import.meta.env as any).DEV = true;

	// 			await initClient(() => {});

	// 			expect((window as any).__waveRevalidate).toBeDefined();
	// 			expect(typeof (window as any).__waveRevalidate).toBe("function");

	// 			(import.meta.env as any).DEV = originalEnv;
	// 		});

	// 		it("should only register HMR once per file", () => {
	// 			const originalEnv = import.meta.env.DEV;
	// 			(import.meta.env as any).DEV = true;

	// 			const hotMock = {
	// 				on: vi.fn(),
	// 			};

	// 			const importMeta1 = {
	// 				url: "http://localhost:5173/routes/page1.tsx",
	// 				hot: hotMock,
	// 			};

	// 			const importMeta2 = {
	// 				url: "http://localhost:5173/routes/page2.tsx",
	// 				hot: hotMock,
	// 			};

	// 			hmrRunClientLoaders(importMeta1 as any);
	// 			hmrRunClientLoaders(importMeta1 as any); // Same file
	// 			hmrRunClientLoaders(importMeta2 as any); // Different file

	// 			// Should register twice (once per unique file)
	// 			expect(hotMock.on).toHaveBeenCalledTimes(2);

	// 			(import.meta.env as any).DEV = originalEnv;
	// 		});
	// 	});
	// });

	// describe("10. Initialization", () => {
	// 	it("should configure options correctly", async () => {
	// 		const customErrorBoundary = () => "Custom Error";

	// 		await initClient(() => {}, {
	// 			defaultErrorBoundary: customErrorBoundary,
	// 			useViewTransitions: true,
	// 		});

	// 		expect(internal_RiverClientGlobal.get("defaultErrorBoundary")).toBe(
	// 			customErrorBoundary,
	// 		);
	// 		expect(internal_RiverClientGlobal.get("useViewTransitions")).toBe(true);
	// 	});

	// 	it("should initialize history with POP listener", async () => {
	// 		const listenSpy = vi.spyOn(getHistoryInstance(), "listen");

	// 		await initClient(() => {});

	// 		expect(listenSpy).toHaveBeenCalled();
	// 	});

	// 	it("should set scrollRestoration to manual", async () => {
	// 		const scrollRestorationSpy = vi.spyOn(history, "scrollRestoration", "set");

	// 		await initClient(() => {});

	// 		expect(scrollRestorationSpy).toHaveBeenCalledWith("manual");
	// 	});

	// 	it("should clean river_reload param from URL", async () => {
	// 		window.history.replaceState({}, "", "/?river_reload=old-build&keep=this");

	// 		const replaceSpy = vi.spyOn(getHistoryInstance(), "replace");

	// 		await initClient(() => {});

	// 		expect(replaceSpy).toHaveBeenCalledWith("/?keep=this");
	// 	});

	// 	it("should load initial components", async () => {
	// 		setupGlobalRiverContext({
	// 			importURLs: ["/initial.js"],
	// 			publicPathPrefix: "/",
	// 		});

	// 		vi.doMock("/initial.js?river_dev=1", () => ({
	// 			default: () => "Initial Component",
	// 		}));

	// 		await initClient(() => {});

	// 		expect(internal_RiverClientGlobal.get("activeComponents")).toHaveLength(1);
	// 	});

	// 	it("should run initial client wait functions", async () => {
	// 		const waitFn = vi.fn().mockResolvedValue({ initialized: true });

	// 		setupGlobalRiverContext({
	// 			patternToWaitFnMap: { "/": waitFn },
	// 			matchedPatterns: ["/"],
	// 			loadersData: [{ initial: "data" }],
	// 		});

	// 		await initClient(() => {});

	// 		expect(waitFn).toHaveBeenCalled();
	// 		expect(internal_RiverClientGlobal.get("clientLoadersData")).toEqual([
	// 			{ initialized: true },
	// 		]);
	// 	});

	// 	it("should execute user render function", async () => {
	// 		const renderFn = vi.fn();

	// 		await initClient(renderFn);

	// 		expect(renderFn).toHaveBeenCalled();
	// 	});

	// 	it("should restore scroll after refresh", async () => {
	// 		const scrollState = {
	// 			x: 300,
	// 			y: 600,
	// 			unix: Date.now() - 1000,
	// 			href: window.location.href,
	// 		};

	// 		sessionStorage.setItem(
	// 			"__river__pageRefreshScrollState",
	// 			JSON.stringify(scrollState),
	// 		);

	// 		const rafSpy = vi
	// 			.spyOn(window, "requestAnimationFrame")
	// 			.mockImplementation((cb) => {
	// 				cb(0);
	// 				return 0;
	// 			});

	// 		await initClient(() => {});

	// 		expect(window.scrollTo).toHaveBeenCalledWith(300, 600);
	// 		expect(sessionStorage.getItem("__river__pageRefreshScrollState")).toBeNull();

	// 		rafSpy.mockRestore();
	// 	});

	// 	it("should detect touch devices on first touch", async () => {
	// 		await initClient(() => {});

	// 		expect(internal_RiverClientGlobal.get("isTouchDevice")).toBeUndefined();

	// 		window.dispatchEvent(new Event("touchstart"));

	// 		expect(internal_RiverClientGlobal.get("isTouchDevice")).toBe(true);
	// 	});
	// });

	// describe("11. History Management", () => {
	// 	describe("11.1 Custom History", () => {
	// 		it("should create browser history instance", () => {
	// 			const history = getHistoryInstance();
	// 			expect(history).toBeDefined();
	// 			expect(history.location).toBeDefined();
	// 			expect(history.push).toBeDefined();
	// 			expect(history.replace).toBeDefined();
	// 		});

	// 		it("should maintain lastKnownCustomLocation", async () => {
	// 			const history = getHistoryInstance();
	// 			const initialLocation = history.location;

	// 			history.push("/new-location");
	// 			await vi.runAllTimersAsync();

	// 			// Location should be updated after push
	// 			expect(history.location.pathname).toBe("/new-location");
	// 		});
	// 	});

	// 	describe("11.2 POP Event Handling", () => {
	// 		it("should dispatch location event on key change", async () => {
	// 			const locationListener = vi.fn();
	// 			addLocationListener(locationListener);

	// 			const history = getHistoryInstance();
	// 			history.push("/trigger-key-change");

	// 			await vi.runAllTimersAsync();

	// 			expect(locationListener).toHaveBeenCalled();
	// 		});

	// 		it("should handle hash-only changes within same document", async () => {
	// 			const history = getHistoryInstance();
	// 			history.push("/same-doc");

	// 			// Add hash
	// 			history.push("/same-doc#new-hash");

	// 			// Mock scrollIntoView
	// 			const element = document.createElement("div");
	// 			element.id = "new-hash";
	// 			document.body.appendChild(element);
	// 			const scrollSpy = vi.spyOn(element, "scrollIntoView");

	// 			// Trigger POP
	// 			history.back();
	// 			history.forward();
	// 			await vi.runAllTimersAsync();

	// 			applyScrollState({ hash: "new-hash" });
	// 			expect(scrollSpy).toHaveBeenCalled();
	// 		});

	// 		it("should trigger full navigation for different documents", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(
	// 				createMockResponse({ importURLs: [], cssBundles: [] }),
	// 			);

	// 			const history = getHistoryInstance();
	// 			history.push("/page1");
	// 			history.push("/page2");

	// 			// Clear any existing calls
	// 			vi.clearAllMocks();

	// 			// Go back to page1
	// 			history.back();
	// 			await vi.runAllTimersAsync();

	// 			expect(fetch).toHaveBeenCalled();
	// 		});

	// 		it("should save scroll before navigating away", async () => {
	// 			const history = getHistoryInstance();
	// 			history.push("/current");

	// 			(window as any).scrollX = 123;
	// 			(window as any).scrollY = 456;

	// 			// Navigate to different page
	// 			history.push("/different");
	// 			await vi.runAllTimersAsync();

	// 			const saved = JSON.parse(
	// 				sessionStorage.getItem("__river__scrollStateMap") || "[]",
	// 			);
	// 			expect(saved).toContainEqual([expect.any(String), { x: 123, y: 456 }]);
	// 		});
	// 	});
	// });

	// describe("12. Error Handling", () => {
	// 	describe("12.1 Abort Errors", () => {
	// 		it("should identify abort errors correctly", async () => {
	// 			const abortError = new Error("The operation was aborted");
	// 			abortError.name = "AbortError";

	// 			vi.mocked(fetch).mockRejectedValue(abortError);

	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			await navigate("/will-abort");
	// 			vi.runAllTimers();

	// 			// Should not log abort errors
	// 			expect(console.error).not.toHaveBeenCalledWith(
	// 				expect.stringContaining("abort"),
	// 			);
	// 		});

	// 		it("should not affect loading states on abort", async () => {
	// 			const abortError = new Error("Aborted");
	// 			abortError.name = "AbortError";

	// 			vi.mocked(fetch).mockRejectedValue(abortError);

	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			await navigate("/abort-loading");
	// 			vi.runAllTimers();

	// 			// Loading state should still be cleared
	// 			const lastCall =
	// 				statusListener.mock.calls[statusListener.mock.calls.length - 1];
	// 			expect(lastCall?.[0].detail.isNavigating).toBe(false);
	// 		});
	// 	});

	// 	describe("12.2 Navigation Failures", () => {
	// 		it("should log non-abort errors", async () => {
	// 			const error = new Error("Network failure");
	// 			vi.mocked(fetch).mockRejectedValue(error);

	// 			const consoleErrorSpy = vi
	// 				.spyOn(console, "error")
	// 				.mockImplementation(() => {});

	// 			await navigate("/fail");
	// 			vi.runAllTimers();

	// 			expect(consoleErrorSpy).toHaveBeenCalledWith(
	// 				expect.stringContaining("Navigation failed"),
	// 				error,
	// 			);

	// 			consoleErrorSpy.mockRestore();
	// 		});

	// 		it("should clear loading state on failure", async () => {
	// 			vi.mocked(fetch).mockRejectedValue(new Error("Failed"));

	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			await navigate("/clear-on-fail");
	// 			vi.runAllTimers();

	// 			expect(statusListener).toHaveBeenLastCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({ isNavigating: false }),
	// 				}),
	// 			);
	// 		});

	// 		it("should keep user on current page after failure", async () => {
	// 			const currentPath = window.location.pathname;
	// 			vi.mocked(fetch).mockRejectedValue(new Error("Navigation error"));

	// 			await navigate("/unreachable");
	// 			vi.runAllTimers();

	// 			expect(window.location.pathname).toBe(currentPath);
	// 		});

	// 		it("should not update any state on failure", async () => {
	// 			const initialState = {
	// 				title: document.title,
	// 				components: internal_RiverClientGlobal.get("activeComponents"),
	// 				params: internal_RiverClientGlobal.get("params"),
	// 			};

	// 			vi.mocked(fetch).mockRejectedValue(new Error("State test error"));

	// 			await navigate("/state-fail");
	// 			vi.runAllTimers();

	// 			expect(document.title).toBe(initialState.title);
	// 			expect(internal_RiverClientGlobal.get("activeComponents")).toBe(
	// 				initialState.components,
	// 			);
	// 			expect(internal_RiverClientGlobal.get("params")).toBe(initialState.params);
	// 		});
	// 	});

	// 	describe("12.3 Special Cases", () => {
	// 		it("should treat empty JSON response as failure", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(new Response("", { status: 200 }));

	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			await navigate("/empty");
	// 			vi.runAllTimers();

	// 			expect(statusListener).toHaveBeenLastCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({ isNavigating: false }),
	// 				}),
	// 			);
	// 		});

	// 		it("should handle network errors", async () => {
	// 			vi.mocked(fetch).mockRejectedValue(new TypeError("Failed to fetch"));

	// 			const consoleErrorSpy = vi
	// 				.spyOn(console, "error")
	// 				.mockImplementation(() => {});

	// 			await navigate("/network-error");
	// 			vi.runAllTimers();

	// 			expect(consoleErrorSpy).toHaveBeenCalled();
	// 			consoleErrorSpy.mockRestore();
	// 		});

	// 		it("should handle 404/500 responses", async () => {
	// 			vi.mocked(fetch).mockResolvedValue(new Response("Not Found", { status: 404 }));

	// 			const statusListener = vi.fn();
	// 			addListener(addStatusListener, statusListener);

	// 			await navigate("/not-found");
	// 			vi.runAllTimers();

	// 			// Should clear loading state
	// 			expect(statusListener).toHaveBeenLastCalledWith(
	// 				expect.objectContaining({
	// 					detail: expect.objectContaining({ isNavigating: false }),
	// 				}),
	// 			);
	// 		});
	// 	});
	// });

	// describe("13. Utility Functions", () => {
	// 	describe("13.1 Listener Management", () => {
	// 		it("should return cleanup function for removing listeners", () => {
	// 			const listener = vi.fn();
	// 			const cleanup = addStatusListener(listener);

	// 			// Trigger event
	// 			window.dispatchEvent(
	// 				new CustomEvent("river:status", {
	// 					detail: {
	// 						isNavigating: false,
	// 						isSubmitting: false,
	// 						isRevalidating: false,
	// 					},
	// 				}),
	// 			);

	// 			expect(listener).toHaveBeenCalledTimes(1);

	// 			// Clean up
	// 			cleanup();

	// 			// Trigger again
	// 			window.dispatchEvent(
	// 				new CustomEvent("river:status", {
	// 					detail: { isNavigating: true, isSubmitting: false, isRevalidating: false },
	// 				}),
	// 			);

	// 			// Should not be called again
	// 			expect(listener).toHaveBeenCalledTimes(1);
	// 		});

	// 		it("should use window as event target for all listeners", () => {
	// 			const addEventListenerSpy = vi.spyOn(window, "addEventListener");

	// 			addStatusListener(() => {});
	// 			addRouteChangeListener(() => {});
	// 			addLocationListener(() => {});
	// 			addBuildIDListener(() => {});

	// 			expect(addEventListenerSpy).toHaveBeenCalledWith(
	// 				"river:status",
	// 				expect.any(Function),
	// 			);
	// 			expect(addEventListenerSpy).toHaveBeenCalledWith(
	// 				"river:route-change",
	// 				expect.any(Function),
	// 			);
	// 			expect(addEventListenerSpy).toHaveBeenCalledWith(
	// 				"river:location",
	// 				expect.any(Function),
	// 			);
	// 			expect(addEventListenerSpy).toHaveBeenCalledWith(
	// 				"river:build-id",
	// 				expect.any(Function),
	// 			);
	// 		});
	// 	});

	// 	describe("13.2 Public Utilities", () => {
	// 		it("should return root element via getRootEl()", () => {
	// 			const root = document.createElement("div");
	// 			root.id = "river-root";
	// 			document.body.appendChild(root);

	// 			expect(getRootEl()).toBe(root);
	// 		});

	// 		it("should apply scroll state correctly", () => {
	// 			// Test coordinate scroll
	// 			applyScrollState({ x: 100, y: 200 });
	// 			expect(window.scrollTo).toHaveBeenCalledWith(100, 200);

	// 			// Test hash scroll
	// 			const element = document.createElement("div");
	// 			element.id = "test-hash";
	// 			document.body.appendChild(element);
	// 			const scrollSpy = vi.spyOn(element, "scrollIntoView");

	// 			applyScrollState({ hash: "test-hash" });
	// 			expect(scrollSpy).toHaveBeenCalled();

	// 			// Test no state with hash in URL
	// 			window.location.hash = "#url-hash";
	// 			const urlElement = document.createElement("div");
	// 			urlElement.id = "url-hash";
	// 			document.body.appendChild(urlElement);
	// 			const urlScrollSpy = vi.spyOn(urlElement, "scrollIntoView");

	// 			applyScrollState(undefined);
	// 			expect(urlScrollSpy).toHaveBeenCalled();
	// 		});

	// 		it("should return current location parts", () => {
	// 			window.history.replaceState({}, "", "/test/path?query=value#section");

	// 			const location = getLocation();
	// 			expect(location).toEqual({
	// 				pathname: "/test/path",
	// 				search: "?query=value",
	// 				hash: "#section",
	// 			});
	// 		});

	// 		it("should return current build ID", () => {
	// 			setupGlobalRiverContext({ buildID: "test-build-12345" });
	// 			expect(getBuildID()).toBe("test-build-12345");
	// 		});
	// 	});
	// });
});
