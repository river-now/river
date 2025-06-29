package matcher

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

var NestedPatterns = []string{
	"/_index",                                         // Index
	"/articles/_index",                                // Index
	"/articles/test/articles/_index",                  // Index
	"/bear/_index",                                    // Index
	"/dashboard/_index",                               // Index
	"/dashboard/customers/_index",                     // Index
	"/dashboard/customers/:customer_id/_index",        // Index
	"/dashboard/customers/:customer_id/orders/_index", // Index
	"/dynamic-index/:pagename/_index",                 // Index
	"/lion/_index",                                    // Index
	"/tiger/_index",                                   // Index
	"/tiger/:tiger_id/_index",                         // Index

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
}

type TestNestedScenario struct {
	Path            string
	ExpectedMatches []string
	SplatValues     []string
	Params          Params
}

var NestedScenarios = []TestNestedScenario{
	{
		Path:        "/does-not-exist",
		SplatValues: []string{"does-not-exist"},
		ExpectedMatches: []string{
			"",
			"/*",
		},
	},
	{
		Path:        "/this-should-be-ignored",
		SplatValues: []string{"this-should-be-ignored"},
		ExpectedMatches: []string{
			"",
			"/*",
		},
	},
	{
		Path: "/",
		ExpectedMatches: []string{
			"",
			"/",
		},
	},
	{
		Path: "/lion",
		ExpectedMatches: []string{
			"",
			"/lion",
			"/lion/",
		},
	},
	{
		Path:        "/lion/123",
		SplatValues: []string{"123"},
		ExpectedMatches: []string{
			"",
			"/lion",
			"/lion/*",
		},
	},
	{
		Path:        "/lion/123/456",
		SplatValues: []string{"123", "456"},
		ExpectedMatches: []string{
			"",
			"/lion",
			"/lion/*",
		},
	},
	{
		Path:        "/lion/123/456/789",
		SplatValues: []string{"123", "456", "789"},
		ExpectedMatches: []string{
			"",
			"/lion",
			"/lion/*",
		},
	},
	{
		Path: "/tiger",
		ExpectedMatches: []string{
			"",
			"/tiger",
			"/tiger/",
		},
	},
	{
		Path:   "/tiger/123",
		Params: Params{"tiger_id": "123"},
		ExpectedMatches: []string{
			"",
			"/tiger",
			"/tiger/:tiger_id",
			"/tiger/:tiger_id/",
		},
	},
	{
		Path:   "/tiger/123/456",
		Params: Params{"tiger_id": "123", "tiger_cub_id": "456"},
		ExpectedMatches: []string{
			"",
			"/tiger",
			"/tiger/:tiger_id",
			"/tiger/:tiger_id/:tiger_cub_id",
		},
	},
	{
		Path:        "/tiger/123/456/789",
		Params:      Params{"tiger_id": "123"},
		SplatValues: []string{"456", "789"},
		ExpectedMatches: []string{
			"",
			"/tiger",
			"/tiger/:tiger_id",
			"/tiger/:tiger_id/*",
		},
	},
	{
		Path: "/bear",
		ExpectedMatches: []string{
			"",
			"/bear",
			"/bear/",
		},
	},
	{
		Path:   "/bear/123",
		Params: Params{"bear_id": "123"},
		ExpectedMatches: []string{
			"",
			"/bear",
			"/bear/:bear_id",
		},
	},
	{
		Path:        "/bear/123/456",
		Params:      Params{"bear_id": "123"},
		SplatValues: []string{"456"},
		ExpectedMatches: []string{
			"",
			"/bear",
			"/bear/:bear_id",
			"/bear/:bear_id/*",
		},
	},
	{
		Path:        "/bear/123/456/789",
		Params:      Params{"bear_id": "123"},
		SplatValues: []string{"456", "789"},
		ExpectedMatches: []string{
			"",
			"/bear",
			"/bear/:bear_id",
			"/bear/:bear_id/*",
		},
	},
	{
		Path: "/dashboard",
		ExpectedMatches: []string{
			"",
			"/dashboard",
			"/dashboard/",
		},
	},
	{
		Path:        "/dashboard/asdf",
		SplatValues: []string{"asdf"},
		ExpectedMatches: []string{
			"",
			"/dashboard",
			"/dashboard/*",
		},
	},
	{
		Path: "/dashboard/customers",
		ExpectedMatches: []string{
			"",
			"/dashboard",
			"/dashboard/customers",
			"/dashboard/customers/",
		},
	},
	{
		Path:   "/dashboard/customers/123",
		Params: Params{"customer_id": "123"},
		ExpectedMatches: []string{
			"",
			"/dashboard",
			"/dashboard/customers",
			"/dashboard/customers/:customer_id",
			"/dashboard/customers/:customer_id/",
		},
	},
	{
		Path:   "/dashboard/customers/123/orders",
		Params: Params{"customer_id": "123"},
		ExpectedMatches: []string{
			"",
			"/dashboard",
			"/dashboard/customers",
			"/dashboard/customers/:customer_id",
			"/dashboard/customers/:customer_id/orders",
			"/dashboard/customers/:customer_id/orders/",
		},
	},
	{
		Path:   "/dashboard/customers/123/orders/456",
		Params: Params{"customer_id": "123", "order_id": "456"},
		ExpectedMatches: []string{
			"",
			"/dashboard",
			"/dashboard/customers",
			"/dashboard/customers/:customer_id",
			"/dashboard/customers/:customer_id/orders",
			"/dashboard/customers/:customer_id/orders/:order_id",
		},
	},
	{
		Path: "/articles",
		ExpectedMatches: []string{
			"",
			"/articles/",
		},
	},
	{
		Path:        "/articles/bob",
		SplatValues: []string{"articles", "bob"},
		ExpectedMatches: []string{
			"",
			"/*",
		},
	},
	{
		Path:        "/articles/test",
		SplatValues: []string{"articles", "test"},
		ExpectedMatches: []string{
			"",
			"/*",
		},
	},
	{
		Path: "/articles/test/articles",
		ExpectedMatches: []string{
			"",
			"/articles/test/articles/",
		},
	},
	{
		Path: "/dynamic-index/index",
		ExpectedMatches: []string{
			"",
			// no underscore prefix, so not really an index!
			"/dynamic-index/index",
		},
	},

	/*
		"/a/b/:",
		"/c/d/e/:_",
		"/f/g/h/i/:/:",
		"/j/k/l/m/n/:_/:_",
	*/
	{
		Path: "/a/b/hi",
		ExpectedMatches: []string{
			"",
			"/a/b/:",
		},
		Params: Params{"": "hi"},
	},
	{
		Path: "/c/d/e/hi",
		ExpectedMatches: []string{
			"",
			"/c/d/e/:_",
		},
		Params: Params{"_": "hi"},
	},
	{
		Path: "/f/g/h/i/hi/hi2",
		ExpectedMatches: []string{
			"",
			"/f/g/h/i/:/:",
		},
		Params: Params{"": "hi2"},
	},
	{
		Path: "/j/k/l/m/n/hi/hi2",
		ExpectedMatches: []string{
			"",
			"/j/k/l/m/n/:_/:_",
		},
		Params: Params{"_": "hi2"},
	},
}

func TestFindAllMatches(t *testing.T) {
	for _, opts := range differentOptsToTest {
		m := New(opts)

		for _, p := range modifyPatternsToOpts(NestedPatterns, "_index", opts) {
			m.RegisterPattern(p)
		}

		for _, tc := range NestedScenarios {
			t.Run(tc.Path, func(t *testing.T) {
				results, ok := m.FindNestedMatches(tc.Path)

				if !equalParams(tc.Params, results.Params) {
					t.Errorf("Expected params %v, got %v", tc.Params, results.Params)
				}
				if !equalSplat(tc.SplatValues, results.SplatValues) {
					t.Errorf("Expected splat values %v, got %v", tc.SplatValues, results.SplatValues)
				}

				actualMatches := results.Matches

				var errors []string

				// Check if there's a failure
				expectedCount := len(tc.ExpectedMatches)
				actualCount := len(actualMatches)

				fail := (!ok && expectedCount > 0) || (expectedCount != actualCount)

				// Compare each matched pattern
				for i := range max(expectedCount, actualCount) {
					if i < expectedCount && i < actualCount {
						expected := tc.ExpectedMatches[i]
						actual := actualMatches[i]

						// ---- Use helper functions to compare maps/slices ----
						if expected != actual.normalizedPattern {
							fail = true
							break
						}
					} else {
						fail = true
						break
					}
				}

				// Only output errors if a failure occurred
				if fail {
					errors = append(errors, fmt.Sprintf("\n===== Path: %q =====", tc.Path))

					// Expected matches exist but got none
					if !ok && expectedCount > 0 {
						errors = append(errors, "Expected matches but got none.")
					}

					// Length mismatch
					if expectedCount != actualCount {
						errors = append(errors, fmt.Sprintf("Expected %d matches, got %d", expectedCount, actualCount))
					}

					// Always output all expected and actual matches for debugging
					errors = append(errors, "Expected Matches:")
					for i, expected := range tc.ExpectedMatches {
						errors = append(errors, fmt.Sprintf(
							"  [%d] {Pattern: %q}",
							i, expected,
						))
					}

					errors = append(errors, "Actual Matches:")
					for i, actual := range actualMatches {
						errors = append(errors, fmt.Sprintf(
							"  [%d] {Pattern: %q}",
							i, actual.normalizedPattern,
						))
					}

					// Print only if something went wrong
					t.Error(strings.Join(errors, "\n"))
				}
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Helper functions to treat nil maps/slices as empty, avoiding false mismatches
// ---------------------------------------------------------------------------

func equalParams(a, b Params) bool {
	// Consider nil and empty as the same
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func equalSplat(a, b []string) bool {
	// Consider nil and empty slice as the same
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func modifyPatternsToOpts(incomingPatterns []string, incomingIndexSegment string, opts_ *Options) []string {
	opts := mungeOptsToDefaults(opts_)

	m := New(&Options{ExplicitIndexSegment: incomingIndexSegment, Quiet: true})

	rps := make([]*RegisteredPattern, len(incomingPatterns))
	for i, p := range incomingPatterns {
		rps[i] = m.NormalizePattern(p)
	}

	newPatterns := make([]string, 0, len(rps))

	for _, rp := range rps {
		var sb strings.Builder

		for _, seg := range rp.normalizedSegments {
			sb.WriteString("/")
			switch seg.segType {
			case segTypes.static:
				sb.WriteString(seg.normalizedVal)
			case segTypes.dynamic:
				sb.WriteString(string(opts.DynamicParamPrefixRune))
				sb.WriteString(seg.normalizedVal[1:])
			case segTypes.splat:
				sb.WriteString(string(opts.SplatSegmentRune))
			case segTypes.index:
				sb.WriteString(string(opts.ExplicitIndexSegment))
			}
		}

		newPatterns = append(newPatterns, sb.String())
	}

	return newPatterns
}

/////////////////////////////////////////////////////////////////////
/////// BENCHMARKS
/////////////////////////////////////////////////////////////////////

func setupNestedMatcherForBenchmark() *Matcher {
	m := New(&Options{Quiet: true})

	for _, pattern := range NestedPatterns {
		m.RegisterPattern(pattern)
	}
	return m
}

func generateNestedPathsForBenchmark() []string {
	return []string{
		"/",                                   // Root index
		"/dashboard",                          // Static path with index
		"/dashboard/customers",                // Nested static path
		"/dashboard/customers/123",            // Path with params
		"/dashboard/customers/123/orders",     // Deep nested path
		"/dashboard/customers/123/orders/456", // Deep nested path with multiple params
		"/tiger",                              // Another static path
		"/tiger/123",                          // Dynamic path
		"/tiger/123/456",                      // Dynamic path with multiple params
		"/tiger/123/456/789",                  // Path with splat
		"/bear/123/456/789",                   // Different path with splat
		"/articles/test/articles",             // Deeply nested static path
		"/does-not-exist",                     // Non-existent path (tests splat handling)
		"/dashboard/unknown/path",             // Tests dashboard splat path
	}
}

func BenchmarkFindNestedMatches(b *testing.B) {
	cases := []struct {
		name     string
		pathType string
		paths    []string
	}{
		{
			name:     "StaticPatterns",
			pathType: "static",
			paths:    []string{"/", "/dashboard", "/dashboard/customers", "/tiger", "/lion"},
		},
		{
			name:     "DynamicPatterns",
			pathType: "dynamic",
			paths: []string{
				"/dashboard/customers/123",
				"/dashboard/customers/456/orders",
				"/tiger/123",
				"/bear/123",
			},
		},
		{
			name:     "DeepNestedPatterns",
			pathType: "deep",
			paths: []string{
				"/dashboard/customers/123/orders/456",
				"/tiger/123/456/789",
				"/bear/123/456/789",
				"/articles/test/articles",
			},
		},
		{
			name:     "SplatPatterns",
			pathType: "splat",
			paths: []string{
				"/does-not-exist",
				"/dashboard/unknown/path",
				"/tiger/123/456/789/extra",
				"/bear/123/456/789/extra",
			},
		},
		{
			name:     "MixedPatterns",
			pathType: "mixed",
			paths:    generateNestedPathsForBenchmark(),
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			m := setupNestedMatcherForBenchmark()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				path := tc.paths[i%len(tc.paths)]
				matches, _ := m.FindNestedMatches(path)
				runtime.KeepAlive(matches)
			}
		})
	}
}
