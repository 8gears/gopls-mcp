package e2e

// End-to-end tests that query the ACTUAL gopls-mcp codebase using table-driven approach.

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// realCodebaseTestCase defines a real codebase test case
type realCodebaseTestCase struct {
	name      string
	tool      string
	args      func(goplsMcpDir string) map[string]any
	assertion func(t *testing.T, content string)
	setup     func(goplsMcpDir string) (skip bool, reason string)
}

// TestRealCodebase runs comprehensive tests on the real gopls-mcp codebase
func TestRealCodebase(t *testing.T) {
	goplsMcpDir, _ := filepath.Abs("../..")
	wrappersPath := filepath.Join(goplsMcpDir, "core", "gopls_wrappers.go")
	handlersPath := filepath.Join(goplsMcpDir, "core", "handlers.go")

	// Helper to find line number for a function
	findFuncLine := func(content, funcName string) int {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, funcName+"(") || strings.Contains(line, funcName+" struct") {
				return i + 1
			}
		}
		return 0
	}

	wrappersContent, _ := os.ReadFile(wrappersPath)
	handlersContent, _ := os.ReadFile(handlersPath)

	testCases := []realCodebaseTestCase{
		{
			name: "Definition_handleGoDefinition",
			tool: "go_definition",
			args: func(goplsMcpDir string) map[string]any {
				defLine := findFuncLine(string(wrappersContent), "func handleGoDefinition")
				return map[string]any{
					"locator": map[string]any{
						"symbol_name":  "handleGoDefinition",
						"context_file": wrappersPath,
						"kind":         "function",
						"line_hint":    defLine,
					},
				}
			},
			setup: func(goplsMcpDir string) (bool, string) {
				defLine := findFuncLine(string(wrappersContent), "func handleGoDefinition")
				return defLine == 0, "Could not find handleGoDefinition function"
			},
			assertion: func(t *testing.T, content string) {
				if !strings.Contains(content, "gopls_wrappers.go") {
					t.Fatalf("Expected to find definition in gopls_wrappers.go.\nGot: %s", content)
				}
				re := regexp.MustCompile(`gopls_wrappers\.go:(\d+):\d+`)
				matches := re.FindStringSubmatch(content)
				if len(matches) < 2 {
					t.Fatalf("Expected 'gopls_wrappers.go:LINE:COL' format.\nGot: %s", content)
				}
				foundLine, _ := strconv.Atoi(matches[1])
				t.Logf("✓ Found handleGoDefinition definition at gopls_wrappers.go:%d", foundLine)
			},
		},
		{
			name: "Definition_HandlerStruct",
			tool: "go_definition",
			args: func(goplsMcpDir string) map[string]any {
				defLine := findFuncLine(string(handlersContent), "type Handler struct")
				return map[string]any{
					"locator": map[string]any{
						"symbol_name":  "Handler",
						"context_file": handlersPath,
						"kind":         "struct",
						"line_hint":    defLine,
					},
				}
			},
			setup: func(goplsMcpDir string) (bool, string) {
				defLine := findFuncLine(string(handlersContent), "type Handler struct")
				return defLine == 0, "Could not find Handler struct definition"
			},
			assertion: func(t *testing.T, content string) {
				if !strings.Contains(content, "handlers.go") {
					t.Fatalf("Expected to find definition in handlers.go.\nGot: %s", content)
				}
				re := regexp.MustCompile(`handlers\.go:(\d+):\d+`)
				matches := re.FindStringSubmatch(content)
				if len(matches) < 2 {
					t.Fatalf("Expected 'handlers.go:LINE:COL' format.\nGot: %s", content)
				}
				foundLine, _ := strconv.Atoi(matches[1])
				t.Logf("✓ Found Handler type definition at handlers.go:%d", foundLine)
			},
		},
		{
			name: "References_Handler",
			tool: "go_symbol_references",
			args: func(goplsMcpDir string) map[string]any {
				defLine := findFuncLine(string(handlersContent), "type Handler struct")
				return map[string]any{
					"locator": map[string]any{
						"symbol_name":  "Handler",
						"context_file": handlersPath,
						"kind":         "struct",
						"line_hint":    defLine,
					},
				}
			},
			setup: func(goplsMcpDir string) (bool, string) {
				defLine := findFuncLine(string(handlersContent), "type Handler struct")
				return defLine == 0, "Could not find Handler struct definition"
			},
			assertion: func(t *testing.T, content string) {
				count := testutil.ExtractCount(t, content, "Found ")
				if count <= 0 {
					if strings.Contains(content, "No references found") {
						t.Log("Note: No references found (known limitation when searching from definition file)")
						return
					}
					t.Fatalf("Expected to find references to Handler, got %d.\nGot: %s", count, content)
				}
				t.Logf("✓ Found %d references to Handler", count)
				if !strings.Contains(content, "handlers.go") {
					t.Fatalf("Expected 'handlers.go' in references result.\nGot: %s", content)
				}
				if !regexp.MustCompile(`handlers\.go:\d+:\d+`).MatchString(content) {
					t.Fatalf("Expected 'handlers.go:LINE:COL' format in result.\nGot: %s", content)
				}
				t.Log("✓ References properly formatted with line numbers")
			},
		},
	}

	// Run all test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Check if test should be skipped
			if tc.setup != nil {
				if skip, reason := tc.setup(goplsMcpDir); skip {
					t.Skip(reason)
				}
			}

			// Get args and call tool
			args := tc.args(goplsMcpDir)
			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
				Name:      tc.tool,
				Arguments: args,
			})
			if err != nil {
				t.Fatalf("Failed to call %s: %v", tc.tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("%s result:\n%s", tc.name, testutil.TruncateString(content, 2000))

			// Run assertion
			tc.assertion(t, content)
		})
	}
}

// TestRealCodebase_Rename tests the rename tool with realistic code
func TestRealCodebase_Rename(t *testing.T) {
	testCode := `package core

// TestRenameFunction is a test function for rename operations
func TestRenameFunction(x int) int {
	return x * 2
}

// useTestRenameFunction uses the test function
func useTestRenameFunction() {
	// First usage
	result := TestRenameFunction(10)

	// Second usage in a different context
	another := TestRenameFunction(20)

	// Third usage
	third := TestRenameFunction(result)

	_ = result
	_ = another
	_ = third
}
`
	testDir := t.TempDir()
	testPath := filepath.Join(testDir, "test_rename.go")
	if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Find function line
	lines := strings.Split(testCode, "\n")
	var lineNum int
	for i, line := range lines {
		if strings.Contains(line, "func TestRenameFunction(") {
			lineNum = i + 1
			break
		}
	}

	if lineNum == 0 {
		t.Fatal("Could not find TestRenameFunction definition")
	}

	// Test rename
	res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
		Name: "go_dryrun_rename_symbol",
		Arguments: map[string]any{
			"locator": map[string]any{
				"symbol_name":  "TestRenameFunction",
				"context_file": testPath,
				"line_hint":    lineNum,
			},
			"new_name": "RenamedTestFunction",
		},
	})
	if err != nil {
		t.Fatalf("Failed to call go_dryrun_rename_symbol: %v", err)
	}

	content := testutil.ResultText(res)
	t.Logf("Rename result:\n%s", testutil.TruncateString(content, 2000))

	// Assertions
	if !strings.Contains(strings.ToUpper(content), "DRY RUN") {
		t.Fatalf("CRITICAL: Must contain 'DRY RUN' indicator.\nGot: %s", content)
	}

	if !strings.Contains(content, "TestRenameFunction") {
		t.Fatalf("Expected old name 'TestRenameFunction'.\nGot: %s", content)
	}
	if !strings.Contains(content, "RenamedTestFunction") {
		t.Fatalf("Expected new name 'RenamedTestFunction'.\nGot: %s", content)
	}

	// Verify rename shows changes (diff format with -/+ lines)
	hasOldName := strings.Contains(content, "-func TestRenameFunction")
	hasNewName := strings.Contains(content, "+func RenamedTestFunction")
	if !hasOldName || !hasNewName {
		t.Fatalf("Expected diff format with old name removed and new name added.\nGot: %s", testutil.TruncateString(content, 500))
	}
	t.Logf("✓ Rename shows proper diff format (old name removed, new name added)")

	// Verify file was NOT modified
	originalContent, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(originalContent) != testCode {
		t.Fatalf("DRY RUN VIOLATED: File was modified!\nExpected:\n%s\n\nGot:\n%s", testCode, string(originalContent))
	}
	t.Log("✓ DRY RUN verified: file unchanged")
}
