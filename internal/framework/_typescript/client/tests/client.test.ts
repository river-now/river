import { JSDOM } from "jsdom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	beginNavigation,
	type NavigateProps,
	type NavigationControl,
	navigationState,
} from "../src/client.ts";
import { __getRiverClientGlobal, RIVER_SYMBOL } from "../src/river_ctx.ts";

let dom: JSDOM;
let mockGlobal: any;

function setup() {
	dom = new JSDOM("<!DOCTYPE html><body></body>", { url: "https://example.com" });
	global.window = dom.window as unknown as Window & typeof globalThis;
	global.document = dom.window.document;
	mockGlobal = {};
	(globalThis as any)[RIVER_SYMBOL] = mockGlobal;
}

function teardown() {
	delete (globalThis as any)[RIVER_SYMBOL];
	dom.window.close();
	global.window = undefined as any;
	global.document = undefined as any;
}

describe("__getRiverClientGlobal", () => {
	beforeEach(setup);

	afterEach(teardown);

	it("should get a value from the global state", () => {
		mockGlobal.params = { key: "value" };
		const { get } = __getRiverClientGlobal();
		expect(get("params")).toEqual({ key: "value" });
	});

	it("should set a value in the global state", () => {
		const { set, get } = __getRiverClientGlobal();
		set("buildID", "123");
		expect(get("buildID")).toBe("123");
	});

	it("should update existing global values correctly", () => {
		mockGlobal.activeComponents = [];
		const { set, get } = __getRiverClientGlobal();
		set("activeComponents", ["Component1"]);
		expect(get("activeComponents")).toEqual(["Component1"]);
	});
});

describe("beginNavigation", () => {
	let mockSetStatus: any;

	beforeEach(() => {
		// Reset navigation state and mock any necessary functions
		navigationState.navigations.clear();
		navigationState.activeUserNavigation = null;
		mockSetStatus = vi.fn();
		setup();
	});

	afterEach(() => {
		vi.restoreAllMocks(); // Restore any mocked functions
		teardown();
	});

	it("should start a new user navigation", () => {
		const props: NavigateProps = { href: "/test", navigationType: "userNavigation" };
		const _ = beginNavigation(props);

		expect(navigationState.activeUserNavigation).toBe("/test");
		expect(navigationState.navigations.has("/test")).toBe(true);
	});

	it("should upgrade prefetch to user navigation", () => {
		const prefetchControl: NavigationControl = {
			abortController: undefined,
			promise: Promise.resolve() as any,
		};
		navigationState.navigations.set("/test", {
			control: prefetchControl,
			type: "prefetch",
		});

		const props: NavigateProps = { href: "/test", navigationType: "userNavigation" };
		const control = beginNavigation(props);

		expect(control).toBe(prefetchControl); // Prefetch upgraded
		expect(navigationState.navigations.get("/test")?.type).toBe("userNavigation");
	});

	it("should not start duplicate prefetch", () => {
		const existingControl: NavigationControl = {
			abortController: undefined,
			promise: Promise.resolve() as any,
		};
		navigationState.navigations.set("/test", {
			control: existingControl,
			type: "prefetch",
		});

		const props: NavigateProps = { href: "/test", navigationType: "prefetch" };
		const control = beginNavigation(props);

		expect(control).toBe(existingControl); // Uses existing prefetch
	});
});
