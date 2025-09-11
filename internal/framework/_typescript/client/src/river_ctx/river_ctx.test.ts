import { JSDOM } from "jsdom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { __getRiverClientGlobal, RIVER_SYMBOL } from "./river_ctx.ts";

let dom: JSDOM;
let mockGlobal: any;

function setup() {
	dom = new JSDOM("<!DOCTYPE html><body></body>", {
		url: "https://example.com",
	});
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
