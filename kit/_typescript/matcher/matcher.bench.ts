import { bench, describe } from "vitest";
import { NestedPatterns } from "./find_nested_matches.test";
import { Matcher, ParseSegments } from "./matcher";

// Setup functions
function setupNonNestedMatcherForBenchmark(scale: string): Matcher {
	const m = new Matcher({ Quiet: true });

	switch (scale) {
		case "small":
			// Basic patterns for simple tests
			m.RegisterPattern("/");
			m.RegisterPattern("/users");
			m.RegisterPattern("/users/:id");
			m.RegisterPattern("/users/:id/profile");
			m.RegisterPattern("/api/v1/users");
			m.RegisterPattern("/api/:version/users");
			m.RegisterPattern("/api/v1/users/:id");
			m.RegisterPattern("/files/*");
			break;

		case "medium":
			// RESTful API-style patterns
			for (let i = 0; i < 1_000; i++) {
				m.RegisterPattern(`/api/v${i % 5}/users`);
				m.RegisterPattern(`/api/v${i % 5}/users/:id`);
				m.RegisterPattern(`/api/v${i % 5}/users/:id/posts/:post_id`);
				m.RegisterPattern(`/files/bucket${i % 10}/*`);
			}
			break;

		case "large":
			// Complex application patterns
			for (let i = 0; i < 10_000; i++) {
				// Static patterns
				m.RegisterPattern(`/api/v${i % 10}/users`);
				m.RegisterPattern(`/api/v${i % 10}/products`);
				m.RegisterPattern(`/docs/section${i % 100}`);

				// Dynamic patterns
				m.RegisterPattern(`/api/v${i % 10}/users/:id/posts/:post_id`);
				m.RegisterPattern(`/api/v${i % 10}/products/:category/:id`);

				// Splat patterns
				m.RegisterPattern(`/files/bucket${i % 20}/*`);
			}
			break;
	}

	return m;
}

function generateNonNestedPathsForBenchmark(scale: string): string[] {
	switch (scale) {
		case "small":
			return [
				"/",
				"/users",
				"/users/123",
				"/users/123/profile",
				"/api/v1/users",
				"/api/v2/users",
				"/files/document.pdf",
			];
		case "medium":
		case "large":
			const paths: string[] = [];

			// Static paths (40%)
			for (let i = 0; i < 400; i++) {
				paths.push(`/api/v${i % 5}/users`);
			}

			// Dynamic paths (40%)
			for (let i = 0; i < 400; i++) {
				paths.push(`/api/v${i % 5}/users/${i}/posts/${i % 100}`);
			}

			// Splat paths (20%)
			for (let i = 0; i < 200; i++) {
				paths.push(`/files/bucket${i % 10}/path/to/file${i}.txt`);
			}

			return paths;
	}
	return [];
}

function setupNestedMatcherForBenchmark(): Matcher {
	const m = new Matcher({ Quiet: true });

	for (const pattern of NestedPatterns) {
		m.RegisterPattern(pattern);
	}
	return m;
}

function generateNestedPathsForBenchmark(): string[] {
	return [
		"/", // Root index
		"/dashboard", // Static path with index
		"/dashboard/customers", // Nested static path
		"/dashboard/customers/123", // Path with params
		"/dashboard/customers/123/orders", // Deep nested path
		"/dashboard/customers/123/orders/456", // Deep nested path with multiple params
		"/tiger", // Another static path
		"/tiger/123", // Dynamic path
		"/tiger/123/456", // Dynamic path with multiple params
		"/tiger/123/456/789", // Path with splat
		"/bear/123/456/789", // Different path with splat
		"/articles/test/articles", // Deeply nested static path
		"/does-not-exist", // Non-existent path (tests splat handling)
		"/dashboard/unknown/path", // Tests dashboard splat path
	];
}

describe("FindBestMatch Benchmarks", () => {
	// Pre-setup matchers outside of benchmark loops
	const mediumMatcher = setupNonNestedMatcherForBenchmark("medium");
	const smallMatcher = setupNonNestedMatcherForBenchmark("small");
	const largeMatcher = setupNonNestedMatcherForBenchmark("large");

	const scenarios = [
		{ name: "StaticPattern", path: "/api/v1/users" },
		{ name: "DynamicPattern", path: "/api/v1/users/123/posts/456" },
		{ name: "SplatPattern", path: "/files/bucket1/deep/path/file.txt" },
	];

	for (const s of scenarios) {
		bench(`FindBestMatchSimple - ${s.name}`, () => {
			// Single operation per benchmark iteration
			const [match, ok] = mediumMatcher.FindBestMatch(s.path);
			if (!ok) throw new Error("Expected match");
		});
	}

	bench("FindBestMatchAtScale - Scale_small", () => {
		const paths = generateNonNestedPathsForBenchmark("small");
		// Use a stable path index to avoid modulo operation in benchmark
		const path = paths[0];
		const [match, _] = smallMatcher.FindBestMatch(path || "");
	});

	bench("FindBestMatchAtScale - Scale_medium", () => {
		const paths = generateNonNestedPathsForBenchmark("medium");
		const path = paths[0];
		const [match, _] = mediumMatcher.FindBestMatch(path || "");
	});

	bench("FindBestMatchAtScale - Scale_large", () => {
		const paths = generateNonNestedPathsForBenchmark("large");
		const path = paths[0];
		const [match, _] = largeMatcher.FindBestMatch(path || "");
	});

	bench("WorstCase_DeepNested", () => {
		const path = "/api/v9/users/999/posts/999";
		const [match, _] = largeMatcher.FindBestMatch(path);
	});
});

describe("FindNestedMatches Benchmarks", () => {
	// Pre-setup matcher
	const nestedMatcher = setupNestedMatcherForBenchmark();

	const cases = [
		{
			name: "StaticPatterns",
			paths: [
				"/",
				"/dashboard",
				"/dashboard/customers",
				"/tiger",
				"/lion",
			],
		},
		{
			name: "DynamicPatterns",
			paths: [
				"/dashboard/customers/123",
				"/dashboard/customers/456/orders",
				"/tiger/123",
				"/bear/123",
			],
		},
		{
			name: "DeepNestedPatterns",
			paths: [
				"/dashboard/customers/123/orders/456",
				"/tiger/123/456/789",
				"/bear/123/456/789",
				"/articles/test/articles",
			],
		},
		{
			name: "SplatPatterns",
			paths: [
				"/does-not-exist",
				"/dashboard/unknown/path",
				"/tiger/123/456/789/extra",
				"/bear/123/456/789/extra",
			],
		},
		{
			name: "MixedPatterns",
			paths: generateNestedPathsForBenchmark(),
		},
	];

	for (const tc of cases) {
		bench(`FindNestedMatches - ${tc.name}`, () => {
			// Single operation per benchmark iteration
			const path = tc.paths[0];
			const [matches, _] = nestedMatcher.FindNestedMatches(path || "");
		});
	}
});

describe("ParseSegments Benchmarks", () => {
	const paths = [
		"/",
		"/api/v1/users",
		"/api/v1/users/123/posts/456/comments",
		"/files/documents/reports/quarterly/q3-2023.pdf",
	];

	bench("ParseSegments", () => {
		// Single operation per benchmark iteration
		const path = paths[0];
		const segments = ParseSegments(path || "");
		// Keep reference to prevent optimization
		if (segments.length < 0) throw new Error();
	});
});
