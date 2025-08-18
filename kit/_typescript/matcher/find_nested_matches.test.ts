import { describe, expect, it } from "vitest";
import { findNestedMatches } from "./find_nested_matches";
import {
	createPatternRegistry,
	Params,
	RegisteredPattern,
	registerPattern,
	RegistrationOptions,
} from "./register";

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

const differentOptsToTest: (RegistrationOptions | undefined)[] = [
	undefined,
	{ explicitIndexSegment: "_index" },
	{ dynamicParamPrefixRune: "$" },
	{ splatSegmentRune: "#" },
	{
		explicitIndexSegment: "_______",
		dynamicParamPrefixRune: "<",
		splatSegmentRune: ">",
	},
	{
		explicitIndexSegment: "",
		dynamicParamPrefixRune: "<",
		splatSegmentRune: ">",
	},
];

// Helper functions
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

// Helper function to normalize a pattern for testing
function normalizePatternForTesting(
	pattern: string,
	incomingIndexSegment: string,
): RegisteredPattern {
	// Create a temporary registry just for normalization
	const tempRegistry = createPatternRegistry({
		explicitIndexSegment: incomingIndexSegment,
	});

	// We need to access the normalized pattern, so we'll register it and return
	return registerPattern(tempRegistry, pattern);
}

function modifyPatternsToOpts(
	incomingPatterns: string[],
	incomingIndexSegment: string,
	opts?: RegistrationOptions,
): string[] {
	const rps = incomingPatterns.map((p) =>
		normalizePatternForTesting(p, incomingIndexSegment),
	);
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
						(opts?.dynamicParamPrefixRune || ":") +
						seg.normalizedVal.substring(1);
					break;
				case "splat":
					pattern += opts?.splatSegmentRune || "*";
					break;
				case "index":
					pattern += opts?.explicitIndexSegment || "";
					break;
			}
		}
		newPatterns.push(pattern);
	}

	return newPatterns;
}

describe("FindNestedMatches", () => {
	for (const opts of differentOptsToTest) {
		describe(`with options ${JSON.stringify(opts)}`, () => {
			const registry = createPatternRegistry(opts);

			for (const p of modifyPatternsToOpts(
				NestedPatterns,
				"_index",
				opts,
			)) {
				registerPattern(registry, p);
			}

			for (const tc of NestedScenarios) {
				it(tc.Path, () => {
					const results = findNestedMatches(registry, tc.Path);

					if (!results) {
						if (tc.ExpectedMatches.length > 0) {
							throw new Error(`Expected results for ${tc.Path}`);
						}
						return;
					}

					if (!equalParams(tc.Params, results.params)) {
						throw new Error(
							`Expected params ${JSON.stringify(tc.Params)}, got ${JSON.stringify(results.params)}`,
						);
					}
					if (!equalSplat(tc.SplatValues, results.splatValues)) {
						throw new Error(
							`Expected splat values ${JSON.stringify(tc.SplatValues)}, got ${JSON.stringify(results.splatValues)}`,
						);
					}

					const actualMatches = results.matches;
					const errors: string[] = [];

					// Check if there's a failure
					const expectedCount = tc.ExpectedMatches.length;
					const actualCount = actualMatches.length;

					let fail =
						(!results && expectedCount > 0) ||
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

							if (
								expected !==
								actual?.registeredPattern.normalizedPattern
							) {
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
						if (!results && expectedCount > 0) {
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
								`  [${i}] {Pattern: "${actual?.registeredPattern.normalizedPattern}"}`,
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

describe("FindNestedMatchesAdditionalScenarios", () => {
	const testCases = [
		{
			name: "Invalid match with unhandled segment",
			patterns: ["/", "/:slug", "/_index", "/app"],
			path: "/settings/account",
			expectMatch: false,
		},
		{
			name: "Deeper Invalid 'Almost' Match",
			patterns: ["/dashboard/customers"],
			path: "/dashboard/customers/reports",
			expectMatch: false,
		},
		{
			name: "Splat as the Only Full Match",
			patterns: ["/files/*", "/files/images"],
			path: "/files/documents/report.pdf",
			expectMatch: true,
		},
		{
			name: "Index Segment Edge Case with Extra Segment",
			patterns: ["/articles/_index"],
			path: "/articles/some-topic",
			expectMatch: false,
		},
		{
			name: "No Root Fallback for Multi-Segment Path",
			patterns: ["/"],
			path: "/some/random/path",
			expectMatch: false,
		},
	];

	for (const tc of testCases) {
		it(tc.name, () => {
			const registry = createPatternRegistry({
				explicitIndexSegment: "_index",
			});
			for (const p of tc.patterns) {
				registerPattern(registry, p);
			}

			const results = findNestedMatches(registry, tc.path);

			expect(!!results).toBe(tc.expectMatch);

			// If no match was expected, ensure the results are truly empty.
			if (!tc.expectMatch) {
				if (results !== null && results.matches.length !== 0) {
					throw new Error(
						`Expected no matches for path ${tc.path}, but got ${results.matches.length} matches`,
					);
				}
			}

			// If a match was expected, ensure the results are not empty.
			if (tc.expectMatch) {
				if (results === null || results.matches.length === 0) {
					throw new Error(
						`Expected matches for path ${tc.path}, but got none`,
					);
				}
			}
		});
	}
});

describe("TrailingSlashBehavior", () => {
	const patterns = [
		"/",
		"/_index",
		"/about",
		"/about/location",
		"/about/hobbies",
		"/about/:id",
		"/about/*",
	];

	const testCases = [
		{
			name: "about with trailing slash",
			path: "/about/",
			expectedMatches: ["/", "/about"],
			unexpectedMatches: ["/about/:id", "/about/*"],
		},
		{
			name: "about without trailing slash",
			path: "/about",
			expectedMatches: ["/", "/about"],
			unexpectedMatches: ["/about/:id", "/about/*"],
		},
		{
			name: "about with actual id",
			path: "/about/123",
			expectedMatches: ["/", "/about/:id"],
			unexpectedMatches: ["/about/*"],
		},
		{
			name: "about location exact match",
			path: "/about/location",
			expectedMatches: ["/", "/about", "/about/location"],
			unexpectedMatches: [
				"/_index",
				"/about/:id", // exact match should take precedence
				"/about/*", // exact match should take precedence
			],
		},
		{
			name: "about with multiple segments",
			path: "/about/something/else",
			expectedMatches: [
				"/",
				"/about/*", // should catch multiple segments
			],
			unexpectedMatches: [
				"/about/:id", // only handles one segment
				"/about/location", // not an exact match
			],
		},
	];

	const registry = createPatternRegistry({ explicitIndexSegment: "_index" });
	for (const p of patterns) {
		registerPattern(registry, p);
	}

	for (const tc of testCases) {
		it(tc.name, () => {
			const results = findNestedMatches(registry, tc.path);

			if (!results) {
				throw new Error(
					`Path ${tc.path}: expected to find matches, but got none`,
				);
			}

			// Extract the patterns from results
			const actualPatterns = results!.matches;

			// Check for expected patterns
			for (const expected of tc.expectedMatches) {
				const found = actualPatterns.some(
					(actual) =>
						actual.registeredPattern.originalPattern === expected,
				);
				if (!found && expected !== "") {
					// ignore empty string check for now
					throw new Error(
						`Path ${tc.path}: expected pattern ${expected} to match, but it didn't`,
					);
				}
			}

			// Check for unexpected patterns
			for (const unexpected of tc.unexpectedMatches) {
				const found = actualPatterns.some(
					(actual) =>
						actual.registeredPattern.originalPattern === unexpected,
				);
				if (found) {
					throw new Error(
						`Path ${tc.path}: pattern ${unexpected} should NOT match, but it did`,
					);
				}
			}
		});
	}
});

describe("Old bug: loader pattern matching", () => {
	it("should correctly match patterns when ExplicitIndexSegment is set", () => {
		const registry = createPatternRegistry({
			explicitIndexSegment: "_index",
		});

		registerPattern(registry, "/");
		registerPattern(registry, "/*");

		// "/" -- should match "/"
		const rootResults = findNestedMatches(registry, "/");
		expect(rootResults).not.toBeNull();
		expect(rootResults?.matches).toHaveLength(1);
		expect(rootResults?.matches[0]?.registeredPattern.originalPattern).toBe(
			"/",
		);

		// "/docs" -- should match "/" and "/*"
		const docsResults = findNestedMatches(registry, "/docs");
		expect(docsResults).not.toBeNull();
		expect(docsResults?.matches).toHaveLength(2);
		expect(docsResults?.matches[0]?.registeredPattern.originalPattern).toBe(
			"/",
		);
		expect(docsResults?.matches[1]?.registeredPattern.originalPattern).toBe(
			"/*",
		);
	});
});

describe("Partial matching with gaps in registration", () => {
	it("should match parent and deeply nested route without intermediate routes", () => {
		const registry = createPatternRegistry({
			explicitIndexSegment: "_index",
		});

		// Register only the parent and the deeply nested route
		// NOT registering /bob/larry or /bob/larry/susan
		registerPattern(registry, "/bob");
		registerPattern(registry, "/bob/larry/susan/jeff");

		// Try to match the full path
		const results = findNestedMatches(registry, "/bob/larry/susan/jeff");

		expect(results).not.toBeNull();
		expect(results?.matches).toHaveLength(2);

		// Check that we got the right patterns
		const patterns = results!.matches.map(
			(m) => m.registeredPattern.originalPattern,
		);
		expect(patterns).toContain("/bob");
		expect(patterns).toContain("/bob/larry/susan/jeff");
	});

	it("should not match intermediate paths that aren't registered", () => {
		const registry = createPatternRegistry({
			explicitIndexSegment: "_index",
		});

		registerPattern(registry, "/bob");
		registerPattern(registry, "/bob/larry/susan/jeff");

		// Try to match an intermediate path
		const results = findNestedMatches(registry, "/bob/larry");

		// This should NOT find a match because /bob/larry isn't registered
		// and /bob/larry/susan/jeff doesn't match
		expect(results).toBeNull();
	});
});
