import { describe, it } from "vitest";
import { Matcher, type Options, type Params } from "./matcher";

export const NestedPatterns = [
	"/_index", // Index
	"/articles/_index", // Index
	"/articles/test/articles/_index", // Index
	"/bear/_index", // Index
	"/dashboard/_index", // Index
	"/dashboard/customers/_index", // Index
	"/dashboard/customers/:customer_id/_index", // Index
	"/dashboard/customers/:customer_id/orders/_index", // Index
	"/dynamic-index/:pagename/_index", // Index
	"/lion/_index", // Index
	"/tiger/_index", // Index
	"/tiger/:tiger_id/_index", // Index

	// NOTE: This will evaluate to an empty string -- should match to everything
	// __TODO modify this to run tests both with and without this absolute root ("") pattern
	// as well as with and without the catch-all ("/*") pattern. If you don't have the catch-all
	// and you only match the absolute root, it should not be a match at all
	"/",

	"/*",
	"/bear",
	"/bear/:bear_id",
	"/bear/:bear_id/*",
	"/dashboard",
	"/dashboard/*",
	"/dashboard/customers",
	"/dashboard/customers/:customer_id",
	"/dashboard/customers/:customer_id/orders",
	"/dashboard/customers/:customer_id/orders/:order_id",
	"/dynamic-index/index",
	"/lion",
	"/lion/*",
	"/tiger",
	"/tiger/:tiger_id",
	"/tiger/:tiger_id/:tiger_cub_id",
	"/tiger/:tiger_id/*",

	// for when you don't care about dynamic params but still want to match exactly one segment
	"/a/b/:",
	"/c/d/e/:_",
	"/f/g/h/i/:/:",
	"/j/k/l/m/n/:_/:_",
];

interface TestNestedScenario {
	Path: string;
	ExpectedMatches: string[];
	SplatValues: string[] | null;
	Params: Params | null;
}

const NestedScenarios: TestNestedScenario[] = [
	{
		Path: "/does-not-exist",
		SplatValues: ["does-not-exist"],
		ExpectedMatches: ["", "/*"],
		Params: null,
	},
	{
		Path: "/this-should-be-ignored",
		SplatValues: ["this-should-be-ignored"],
		ExpectedMatches: ["", "/*"],
		Params: null,
	},
	{
		Path: "/",
		ExpectedMatches: ["", "/"],
		SplatValues: null,
		Params: null,
	},
	{
		Path: "/lion",
		ExpectedMatches: ["", "/lion", "/lion/"],
		SplatValues: null,
		Params: null,
	},
	{
		Path: "/lion/123",
		SplatValues: ["123"],
		ExpectedMatches: ["", "/lion", "/lion/*"],
		Params: null,
	},
	{
		Path: "/lion/123/456",
		SplatValues: ["123", "456"],
		ExpectedMatches: ["", "/lion", "/lion/*"],
		Params: null,
	},
	{
		Path: "/lion/123/456/789",
		SplatValues: ["123", "456", "789"],
		ExpectedMatches: ["", "/lion", "/lion/*"],
		Params: null,
	},
	{
		Path: "/tiger",
		ExpectedMatches: ["", "/tiger", "/tiger/"],
		SplatValues: null,
		Params: null,
	},
	{
		Path: "/tiger/123",
		Params: { tiger_id: "123" },
		ExpectedMatches: [
			"",
			"/tiger",
			"/tiger/:tiger_id",
			"/tiger/:tiger_id/",
		],
		SplatValues: null,
	},
	{
		Path: "/tiger/123/456",
		Params: { tiger_id: "123", tiger_cub_id: "456" },
		ExpectedMatches: [
			"",
			"/tiger",
			"/tiger/:tiger_id",
			"/tiger/:tiger_id/:tiger_cub_id",
		],
		SplatValues: null,
	},
	{
		Path: "/tiger/123/456/789",
		Params: { tiger_id: "123" },
		SplatValues: ["456", "789"],
		ExpectedMatches: [
			"",
			"/tiger",
			"/tiger/:tiger_id",
			"/tiger/:tiger_id/*",
		],
	},
	{
		Path: "/bear",
		ExpectedMatches: ["", "/bear", "/bear/"],
		SplatValues: null,
		Params: null,
	},
	{
		Path: "/bear/123",
		Params: { bear_id: "123" },
		ExpectedMatches: ["", "/bear", "/bear/:bear_id"],
		SplatValues: null,
	},
	{
		Path: "/bear/123/456",
		Params: { bear_id: "123" },
		SplatValues: ["456"],
		ExpectedMatches: ["", "/bear", "/bear/:bear_id", "/bear/:bear_id/*"],
	},
	{
		Path: "/bear/123/456/789",
		Params: { bear_id: "123" },
		SplatValues: ["456", "789"],
		ExpectedMatches: ["", "/bear", "/bear/:bear_id", "/bear/:bear_id/*"],
	},
	{
		Path: "/dashboard",
		ExpectedMatches: ["", "/dashboard", "/dashboard/"],
		SplatValues: null,
		Params: null,
	},
	{
		Path: "/dashboard/asdf",
		SplatValues: ["asdf"],
		ExpectedMatches: ["", "/dashboard", "/dashboard/*"],
		Params: null,
	},
	{
		Path: "/dashboard/customers",
		ExpectedMatches: [
			"",
			"/dashboard",
			"/dashboard/customers",
			"/dashboard/customers/",
		],
		SplatValues: null,
		Params: null,
	},
	{
		Path: "/dashboard/customers/123",
		Params: { customer_id: "123" },
		ExpectedMatches: [
			"",
			"/dashboard",
			"/dashboard/customers",
			"/dashboard/customers/:customer_id",
			"/dashboard/customers/:customer_id/",
		],
		SplatValues: null,
	},
	{
		Path: "/dashboard/customers/123/orders",
		Params: { customer_id: "123" },
		ExpectedMatches: [
			"",
			"/dashboard",
			"/dashboard/customers",
			"/dashboard/customers/:customer_id",
			"/dashboard/customers/:customer_id/orders",
			"/dashboard/customers/:customer_id/orders/",
		],
		SplatValues: null,
	},
	{
		Path: "/dashboard/customers/123/orders/456",
		Params: { customer_id: "123", order_id: "456" },
		ExpectedMatches: [
			"",
			"/dashboard",
			"/dashboard/customers",
			"/dashboard/customers/:customer_id",
			"/dashboard/customers/:customer_id/orders",
			"/dashboard/customers/:customer_id/orders/:order_id",
		],
		SplatValues: null,
	},
	{
		Path: "/articles",
		ExpectedMatches: ["", "/articles/"],
		SplatValues: null,
		Params: null,
	},
	{
		Path: "/articles/bob",
		SplatValues: ["articles", "bob"],
		ExpectedMatches: ["", "/*"],
		Params: null,
	},
	{
		Path: "/articles/test",
		SplatValues: ["articles", "test"],
		ExpectedMatches: ["", "/*"],
		Params: null,
	},
	{
		Path: "/articles/test/articles",
		ExpectedMatches: ["", "/articles/test/articles/"],
		SplatValues: null,
		Params: null,
	},
	{
		Path: "/dynamic-index/index",
		ExpectedMatches: [
			"",
			// no underscore prefix, so not really an index!
			"/dynamic-index/index",
		],
		SplatValues: null,
		Params: null,
	},

	/*
    "/a/b/:",
    "/c/d/e/:_",
    "/f/g/h/i/:/:",
    "/j/k/l/m/n/:_/:_",
  */
	{
		Path: "/a/b/hi",
		ExpectedMatches: ["", "/a/b/:"],
		Params: { "": "hi" },
		SplatValues: null,
	},
	{
		Path: "/c/d/e/hi",
		ExpectedMatches: ["", "/c/d/e/:_"],
		Params: { _: "hi" },
		SplatValues: null,
	},
	{
		Path: "/f/g/h/i/hi/hi2",
		ExpectedMatches: ["", "/f/g/h/i/:/:"],
		Params: { "": "hi2" },
		SplatValues: null,
	},
	{
		Path: "/j/k/l/m/n/hi/hi2",
		ExpectedMatches: ["", "/j/k/l/m/n/:_/:_"],
		Params: { _: "hi2" },
		SplatValues: null,
	},
];

const differentOptsToTest: (Options | undefined)[] = [
	undefined,
	{ ExplicitIndexSegment: "_index" },
	{ DynamicParamPrefixRune: "$" },
	{ SplatSegmentRune: "#" },
	{
		ExplicitIndexSegment: "_______",
		DynamicParamPrefixRune: "<",
		SplatSegmentRune: ">",
	},
	{
		ExplicitIndexSegment: "",
		DynamicParamPrefixRune: "<",
		SplatSegmentRune: ">",
	},
];

// Helper functions to treat nil maps/slices as empty, avoiding false mismatches
function equalParams(a: Params | null, b: Params): boolean {
	// Consider nil and empty as the same
	if (
		(a === null || Object.keys(a).length === 0) &&
		Object.keys(b).length === 0
	) {
		return true;
	}
	if (a === null) return false;
	return JSON.stringify(a) === JSON.stringify(b);
}

function equalSplat(a: string[] | null, b: string[]): boolean {
	// Consider nil and empty slice as the same
	if ((a === null || a.length === 0) && b.length === 0) {
		return true;
	}
	if (a === null) return false;
	return JSON.stringify(a) === JSON.stringify(b);
}

function modifyPatternsToOpts(
	incomingPatterns: string[],
	incomingIndexSegment: string,
	opts?: Options,
): string[] {
	const m = new Matcher({
		ExplicitIndexSegment: incomingIndexSegment,
		Quiet: true,
	});

	const rps = incomingPatterns.map((p) => m.NormalizePattern(p));
	const newPatterns: string[] = [];

	for (const rp of rps) {
		let pattern = "";
		for (const seg of rp.normalizedSegments) {
			pattern += "/";
			switch (seg.segType) {
				case "static":
					pattern += seg.normalizedVal;
					break;
				case "dynamic":
					pattern +=
						(opts?.DynamicParamPrefixRune || ":") +
						seg.normalizedVal.substring(1);
					break;
				case "splat":
					pattern += opts?.SplatSegmentRune || "*";
					break;
				case "index":
					pattern += opts?.ExplicitIndexSegment || "";
					break;
			}
		}
		newPatterns.push(pattern);
	}

	return newPatterns;
}

describe("FindAllMatches", () => {
	for (const opts of differentOptsToTest) {
		describe(`with options ${JSON.stringify(opts)}`, () => {
			const m = new Matcher(opts);

			for (const p of modifyPatternsToOpts(
				NestedPatterns,
				"_index",
				opts,
			)) {
				m.RegisterPattern(p);
			}

			for (const tc of NestedScenarios) {
				it(tc.Path, () => {
					const [results, ok] = m.FindNestedMatches(tc.Path);

					if (!results) {
						throw new Error(`Expected results for ${tc.Path}`);
					}

					if (!equalParams(tc.Params, results.Params)) {
						throw new Error(
							`Expected params ${JSON.stringify(tc.Params)}, got ${JSON.stringify(results.Params)}`,
						);
					}
					if (!equalSplat(tc.SplatValues, results.SplatValues)) {
						throw new Error(
							`Expected splat values ${JSON.stringify(tc.SplatValues)}, got ${JSON.stringify(results.SplatValues)}`,
						);
					}

					const actualMatches = results.Matches;
					const errors: string[] = [];

					// Check if there's a failure
					const expectedCount = tc.ExpectedMatches.length;
					const actualCount = actualMatches.length;

					let fail =
						(!ok && expectedCount > 0) ||
						expectedCount !== actualCount;

					// Compare each matched pattern
					for (
						let i = 0;
						i < Math.max(expectedCount, actualCount);
						i++
					) {
						if (i < expectedCount && i < actualCount) {
							const expected = tc.ExpectedMatches[i];
							const actual = actualMatches[i];

							if (expected !== actual?.normalizedPattern) {
								fail = true;
								break;
							}
						} else {
							fail = true;
							break;
						}
					}

					// Only output errors if a failure occurred
					if (fail) {
						errors.push(`\n===== Path: "${tc.Path}" =====`);

						// Expected matches exist but got none
						if (!ok && expectedCount > 0) {
							errors.push("Expected matches but got none.");
						}

						// Length mismatch
						if (expectedCount !== actualCount) {
							errors.push(
								`Expected ${expectedCount} matches, got ${actualCount}`,
							);
						}

						// Always output all expected and actual matches for debugging
						errors.push("Expected Matches:");
						for (let i = 0; i < tc.ExpectedMatches.length; i++) {
							const expected = tc.ExpectedMatches[i];
							errors.push(`  [${i}] {Pattern: "${expected}"}`);
						}

						errors.push("Actual Matches:");
						for (let i = 0; i < actualMatches.length; i++) {
							const actual = actualMatches[i];
							errors.push(
								`  [${i}] {Pattern: "${actual?.normalizedPattern}"}`,
							);
						}

						// Print only if something went wrong
						throw new Error(errors.join("\n"));
					}
				});
			}
		});
	}
});
