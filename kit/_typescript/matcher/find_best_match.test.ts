import { describe, expect, it } from "vitest";
import { findBestMatch } from "./find_best_match.ts";
import {
	createPatternRegistry,
	type Params,
	type RegisteredPattern,
	registerPattern,
	type RegistrationOptions,
} from "./register.ts";

const NOT_FOUND = "NOT FOUND";

interface TestCase {
	name: string;
	patterns: string[];
	path: string;
	wantPattern: string;
	wantParams: Params | null;
	wantSplatSegments: string[] | null;
}

function getTestCases(): TestCase[] {
	return [
		// empty-str cases
		{
			name: "home route -- should match empty-str",
			patterns: [""],
			path: "/",
			wantPattern: "",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "home route -- idx should beat empty-str",
			patterns: ["", "/"],
			path: "/",
			wantPattern: "/",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "home route -- empty-str should beat root-splat",
			patterns: ["", "/*"],
			path: "/",
			wantPattern: "",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "home route -- idx should beat root-splat",
			patterns: ["/", "/*"],
			path: "/",
			wantPattern: "/",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "home route -- idx should win (empty-str, idx, root-splat registered)",
			patterns: ["", "/", "/*"],
			path: "/",
			wantPattern: "/",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "home route -- should match root-splat if no idx or empty-str",
			patterns: ["/*"],
			path: "/",
			wantPattern: "/*",
			wantParams: null,
			wantSplatSegments: [""],
		},

		// TRAILING SLASH SHOULD NOT MATCH DYNAMIC ROUTE
		{
			name: "trailing slash should not match following dynamic route",
			patterns: ["/users/:user"],
			path: "/users/",
			wantPattern: NOT_FOUND,
			wantParams: null,
			wantSplatSegments: null,
		},

		// TEST TRAILING SLASH BEHAVIOR RELATED TO STATIC MATCHES
		{
			name: "exact match should win over following splat",
			patterns: ["/", "/users", "/users/*", "/posts"],
			path: "/users",
			wantPattern: "/users",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "exact match, with trailing slash, should win over catch-all",
			patterns: ["/", "/users", "/users/*", "/posts"],
			path: "/users/",
			wantPattern: "/users",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "with no trailing slash, should NOT match following catch-all",
			patterns: ["/", "/users/*", "/posts"],
			path: "/users",
			wantPattern: NOT_FOUND,
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "with trailing slash, should match following catch-all",
			patterns: ["/", "/users/*", "/posts"],
			path: "/users/",
			wantPattern: "/users/*",
			wantParams: null,
			wantSplatSegments: [""],
		},

		// SAME AS ABOVE, BUT WITH A TRAILING SLASH AS AN ACTUAL REGISTERED PATTERN
		{
			name: "with registered trailing slash -- exact match without trailing should win over catch-all and should win over registered pattern with trailing slash",
			patterns: ["/", "/users/", "/users", "/users/*", "/posts"],
			path: "/users",
			wantPattern: "/users",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "with registered trailing slash -- exact match, with trailing slash, should win over catch-all and should match pattern with trailing slash, not pattern without trailing slash",
			patterns: ["/", "/users/", "/users", "/users/*", "/posts"],
			path: "/users/",
			wantPattern: "/users/",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "with registered trailing slash -- with no trailing slash, should NOT match following catch-all, nor should it match a registered pattern with a trailing slash",
			patterns: ["/", "/users/", "/users/*", "/posts"],
			path: "/users",
			wantPattern: NOT_FOUND,
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "with registered trailing slash -- with trailing slash, should match the registered pattern with a trailing slash, not the following catch-all",
			patterns: ["/", "/users/", "/users/*", "/posts"],
			path: "/users/",
			wantPattern: "/users/",
			wantParams: null,
			wantSplatSegments: null,
		},

		// TEST TRAILING SLASH BEHAVIOR RELATED TO DYNAMIC MATCHES
		{
			name: "dynamic match should win over catch-all",
			patterns: ["/", "/:user", "/:user/*", "/posts"],
			path: "/bob",
			wantPattern: "/:user",
			wantParams: { user: "bob" },
			wantSplatSegments: null,
		},
		{
			name: "dynamic match, with trailing slash, should win over catch-all",
			patterns: ["/", "/:user", "/:user/*", "/posts"],
			path: "/bob/",
			wantPattern: "/:user",
			wantParams: { user: "bob" },
			wantSplatSegments: null,
		},
		{
			name: "dynamic - with no trailing slash, should NOT match following catch-all",
			patterns: ["/", "/:user/*", "/posts"],
			path: "/bob",
			wantPattern: NOT_FOUND,
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "dynamic - with trailing slash, should match following catch-all",
			patterns: ["/", "/:user/*", "/posts"],
			path: "/bob/",
			wantPattern: "/:user/*",
			wantParams: { user: "bob" },
			wantSplatSegments: [""],
		},

		// SAME AS ABOVE, BUT WITH A TRAILING SLASH AS AN ACTUAL REGISTERED PATTERN
		{
			name: "with registered trailing slash -- dynamic match without trailing should win over catch-all and should win over registered pattern with trailing slash",
			patterns: ["/", "/:user/", "/:user", "/:user/*", "/posts"],
			path: "/bob",
			wantPattern: "/:user",
			wantParams: { user: "bob" },
			wantSplatSegments: null,
		},
		{
			name: "with registered trailing slash -- dynamic match, with trailing slash, should win over catch-all and should match pattern with trailing slash, not pattern without trailing slash",
			patterns: ["/", "/:user/", "/:user", "/:user/*", "/posts"],
			path: "/bob/",
			wantPattern: "/:user/",
			wantParams: { user: "bob" },
			wantSplatSegments: null,
		},
		{
			name: "with registered trailing slash -- dynamic with no trailing slash, should NOT match following catch-all, nor should it match a registered pattern with a trailing slash",
			patterns: ["/", "/:user/", "/:user/*", "/posts"],
			path: "/bob",
			wantPattern: NOT_FOUND,
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "with registered trailing slash -- dynamic with trailing slash, should match the registered pattern with a trailing slash, not the following catch-all",
			patterns: ["/", "/:user/", "/:user/*", "/posts"],
			path: "/bob/",
			wantPattern: "/:user/",
			wantParams: { user: "bob" },
			wantSplatSegments: null,
		},

		// MORE TESTS
		{
			name: "parameter match",
			patterns: ["/users", "/users/:id", "/users/profile"],
			path: "/users/123",
			wantPattern: "/users/:id",
			wantParams: { id: "123" },
			wantSplatSegments: null,
		},
		{
			name: "multiple matches",
			patterns: ["/", "/api", "/api/:version", "/api/v1"],
			path: "/api/v1",
			wantPattern: "/api/v1",
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "splat match",
			patterns: ["/files", "/files/*"],
			path: "/files/documents/report.pdf",
			wantPattern: "/files/*",
			wantParams: null,
			wantSplatSegments: ["documents", "report.pdf"],
		},
		{
			name: "no match",
			patterns: ["/users", "/posts", "/settings"],
			path: "/profile",
			wantPattern: NOT_FOUND,
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "complex nested paths",
			patterns: [
				"/api/v1/users",
				"/api/:version/users",
				"/api/v1/users/:id",
				"/api/:version/users/:id",
				"/api/v1/users/:id/posts",
				"/api/:version/users/:id/posts",
			],
			path: "/api/v2/users/123/posts",
			wantPattern: "/api/:version/users/:id/posts",
			wantParams: { version: "v2", id: "123" },
			wantSplatSegments: null,
		},
		{
			name: "no patterns",
			patterns: [],
			path: "/users",
			wantPattern: NOT_FOUND,
			wantParams: null,
			wantSplatSegments: null,
		},
		{
			name: "many params",
			patterns: ["/api/:p1/:p2/:p3/:p4/:p5"],
			path: "/api/a/b/c/d/e",
			wantPattern: "/api/:p1/:p2/:p3/:p4/:p5",
			wantParams: { p1: "a", p2: "b", p3: "c", p4: "d", p5: "e" },
			wantSplatSegments: null,
		},
		{
			name: "nested no match",
			patterns: ["/users/:id", "/users/:id/profile"],
			path: "users/123/settings",
			wantPattern: NOT_FOUND,
			wantParams: null,
			wantSplatSegments: null,
		},
	];
}

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

// Helper function to modify patterns based on options
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

describe("FindBestMatch", () => {
	for (const opts of differentOptsToTest) {
		for (const tt of getTestCases()) {
			describe(`with options ${JSON.stringify(opts)}`, () => {
				it(tt.name, () => {
					const registry = createPatternRegistry(opts);
					for (const pattern of modifyPatternsToOpts(
						tt.patterns,
						"",
						opts,
					)) {
						registerPattern(registry, pattern);
					}

					const match = findBestMatch(registry, tt.path);
					const wantMatch = tt.wantPattern !== NOT_FOUND;

					if (wantMatch && match === null) {
						throw new Error(
							`FindBestMatch() match for ${tt.path} = null -- want ${tt.wantPattern}`,
						);
					}

					if (!wantMatch) {
						if (match !== null) {
							throw new Error(
								`FindBestMatch() match for ${tt.path} = ${match.registeredPattern?.normalizedPattern} -- want null`,
							);
						}
						return;
					}

					expect(match!.registeredPattern!.normalizedPattern).toBe(
						tt.wantPattern,
					);

					// Compare params, allowing null == empty map
					if (
						tt.wantParams === null &&
						Object.keys(match!.params).length > 0
					) {
						throw new Error(
							`FindBestMatch() params = ${JSON.stringify(match!.params)}, want null`,
						);
					} else if (tt.wantParams !== null) {
						expect(match!.params).toEqual(tt.wantParams);
					}

					// Compare splat segments
					if (tt.wantSplatSegments === null) {
						expect(match!.splatValues).toEqual([]);
					} else {
						expect(match!.splatValues).toEqual(
							tt.wantSplatSegments,
						);
					}
				});
			});
		}
	}
});

describe("FindBestMatchAdditionalScenarios", () => {
	it("should not match /settings/account with registered patterns", () => {
		const registry = createPatternRegistry({
			explicitIndexSegment: "_index",
		});

		registerPattern(registry, "/");
		registerPattern(registry, "/:slug");
		registerPattern(registry, "/_index");
		registerPattern(registry, "/app");

		const path = "/settings/account";
		const match = findBestMatch(registry, path);

		expect(match).toBe(null);
	});
});
