package integration

// Helper utilities for table-driven integration tests.
// Reduces boilerplate and makes tests more maintainable.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// testCase defines a single tool test case
type testCase struct {
	// Name of the test case (will be used as subtest name)
	name string

	// Tool name to call
	tool string

	// Arguments to pass to the tool
	args map[string]any

	// Optional: project to copy from testdata (e.g., "simple", "generics")
	// If empty, uses the default shared workdir
	project string

	// Assertions to run on the tool output
	assertions []assertion
}

// assertion defines a check to run on tool output
type assertion struct {
	// Description of what's being checked
	description string

	// Check function returns true if the check passes
	check func(content string) bool

	// Error message to print if check fails
	errorMsg string
}

// runTableDrivenTests executes multiple test cases in a table-driven manner.
// This is the main entry point for refactored integration tests.
//
// Example usage:
//
//	t.Run("go_search", func(t *testing.T) {
//	    tests := map[string]testCase{
//	        "ExactMatch": {
//	            tool: "go_search",
//	            args: map[string]any{"query": "Hello"},
//	            assertions: []assertion{
//	                {description: "finds Hello", check: func(c string) bool { return strings.Contains(c, "Hello") }},
//	            },
//	        },
//	    }
//	    runTableDrivenTests(t, tests)
//	})
func runTableDrivenTests(t *testing.T, tests map[string]testCase) {
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set up project if specified
			if tc.project != "" {
				_ = testutil.CopyProjectTo(t, tc.project)
				// Update Cwd in args if it exists
				if tc.args != nil {
					// CopyProjectTo already sets up the directory, we just need to ensure it's used
				}
			}

			// Call the tool
			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
				Name:      tc.tool,
				Arguments: tc.args,
			})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tc.tool, err)
			}

			if res == nil {
				t.Fatal("Expected non-nil result")
			}

			content := testutil.ResultText(t, res, "")
			t.Logf("%s output:\n%s", tc.tool, truncateString(content, 500))

			// Run all assertions
			for i, assert := range tc.assertions {
				t.Run(fmt.Sprintf("Assertion_%d_%s", i, assert.description), func(t *testing.T) {
					if !assert.check(content) {
						t.Errorf("%s: %s", assert.description, assert.errorMsg)
					}
				})
			}
		})
	}
}

// Common assertion builders for convenience

// assertContains checks that content contains a substring
func assertContains(substring string) assertion {
	return assertion{
		description: fmt.Sprintf("contains %q", substring),
		check:       func(content string) bool { return strings.Contains(content, substring) },
		errorMsg:    fmt.Sprintf("expected content to contain %q", substring),
	}
}

// assertNotContains checks that content does not contain a substring
func assertNotContains(substring string) assertion {
	return assertion{
		description: fmt.Sprintf("does not contain %q", substring),
		check:       func(content string) bool { return !strings.Contains(content, substring) },
		errorMsg:    fmt.Sprintf("expected content NOT to contain %q", substring),
	}
}

// assertContainsAny checks that content contains at least one of the substrings
func assertContainsAny(substrings ...string) assertion {
	return assertion{
		description: fmt.Sprintf("contains any of %v", substrings),
		check: func(content string) bool {
			for _, s := range substrings {
				if strings.Contains(content, s) {
					return true
				}
			}
			return false
		},
		errorMsg: fmt.Sprintf("expected content to contain at least one of %v", substrings),
	}
}

// assertContainsAll checks that content contains all substrings
func assertContainsAll(substrings ...string) assertion {
	return assertion{
		description: fmt.Sprintf("contains all of %v", substrings),
		check: func(content string) bool {
			for _, s := range substrings {
				if !strings.Contains(content, s) {
					return false
				}
			}
			return true
		},
		errorMsg: fmt.Sprintf("expected content to contain all of %v", substrings),
	}
}

// assertCustom allows custom assertion logic
func assertCustom(desc string, check func(content string) bool, errorMsg string) assertion {
	return assertion{
		description: desc,
		check:       check,
		errorMsg:    errorMsg,
	}
}

// truncateString limits string length for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}
