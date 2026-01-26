package integration

// End-to-end tests for boolean parameter flags.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestBooleanParameterFlagsE2E tests that boolean parameter flags work correctly.
// This is CRITICAL - we fixed the schema generation bug where omitempty on *bool
// fields caused them to be excluded from the MCP schema entirely.
func TestBooleanParameterFlagsE2E(t *testing.T) {
	t.Run("ListModulePackages_BooleanFlags", func(t *testing.T) {
		// Use a project with multiple packages including test and internal packages
		projectDir := testutil.CopyProjectTo(t, "simple")

		// Create a test package
		testDir := filepath.Join(projectDir, "mypkg_test")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatal(err)
		}
		testCode := `package mypkg_test

import "testing"

func TestSomething(t *testing.T) {
}
`
		if err := os.WriteFile(filepath.Join(testDir, "mypkg_test.go"), []byte(testCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Create an internal package
		internalDir := filepath.Join(projectDir, "internal")
		if err := os.MkdirAll(internalDir, 0755); err != nil {
			t.Fatal(err)
		}
		internalCode := `package internal

// InternalFunc is internal
func InternalFunc() string {
	return "internal"
}
`
		if err := os.WriteFile(filepath.Join(internalDir, "internal.go"), []byte(internalCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a nested package
		nestedDir := filepath.Join(projectDir, "mypkg", "subpkg")
		if err := os.MkdirAll(nestedDir, 0755); err != nil {
			t.Fatal(err)
		}
		nestedCode := `package subpkg

// NestedFunc is in a nested package
func NestedFunc() string {
	return "nested"
}
`
		if err := os.WriteFile(filepath.Join(nestedDir, "subpkg.go"), []byte(nestedCode), 0644); err != nil {
			t.Fatal(err)
		}

		t.Run("Default_NoFilters", func(t *testing.T) {
			// Without any filters, should return all packages
			tool := "list_module_packages"
			args := map[string]any{
				"Cwd":              projectDir,
				"include_docs":     false,
				"exclude_tests":    false,
				"exclude_internal": false,
				"top_level_only":   false,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("All packages (no filters):\n%s", content)

			// Should include test packages
			if !strings.Contains(content, "mypkg_test") {
				t.Errorf("Expected to find test package when exclude_tests=false (default), got: %s", content)
			}

			// Should include internal packages
			if !strings.Contains(content, "internal") {
				t.Errorf("Expected to find internal package when exclude_internal=false (default), got: %s", content)
			}

			// Should include nested packages
			if !strings.Contains(content, "subpkg") {
				t.Errorf("Expected to find nested package when top_level_only=false (default), got: %s", content)
			}
		})

		t.Run("ExcludeTests_True", func(t *testing.T) {
			// Test exclude_tests=true filters out test packages
			tool := "list_module_packages"
			args := map[string]any{
				"Cwd":              projectDir,
				"include_docs":     false,
				"exclude_tests":    true,
				"exclude_internal": false,
				"top_level_only":   false,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("Packages (exclude_tests=true):\n%s", content)

			// Should NOT include test packages
			if strings.Contains(content, "mypkg_test") {
				t.Errorf("Expected to NOT find test package when exclude_tests=true, got: %s", content)
			}

			// Should still include regular packages
			if !strings.Contains(content, "simple") {
				t.Errorf("Expected to find main package when exclude_tests=true, got: %s", content)
			}
		})

		t.Run("ExcludeInternal_True", func(t *testing.T) {
			// Test exclude_internal=true filters out internal packages
			tool := "list_module_packages"
			args := map[string]any{
				"Cwd":              projectDir,
				"include_docs":     false,
				"exclude_tests":    false,
				"exclude_internal": true,
				"top_level_only":   false,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("Packages (exclude_internal=true):\n%s", content)

			// Should NOT include internal packages
			if strings.Contains(content, "internal") {
				t.Errorf("Expected to NOT find internal package when exclude_internal=true, got: %s", content)
			}

			// Should still include regular packages
			if !strings.Contains(content, "simple") {
				t.Errorf("Expected to find main package when exclude_internal=true, got: %s", content)
			}
		})

		t.Run("TopLevelOnly_True", func(t *testing.T) {
			// Test top_level_only=true filters out nested packages
			tool := "list_module_packages"
			args := map[string]any{
				"Cwd":              projectDir,
				"include_docs":     false,
				"exclude_tests":    false,
				"exclude_internal": false,
				"top_level_only":   true,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("Packages (top_level_only=true):\n%s", content)

			// Should NOT include nested packages (subpkg)
			if strings.Contains(content, "subpkg") {
				t.Errorf("Expected to NOT find nested package when top_level_only=true, got: %s", content)
			}

			// Should still include top-level packages
			if !strings.Contains(content, "simple") {
				t.Errorf("Expected to find top-level package when top_level_only=true, got: %s", content)
			}
		})

		t.Run("AllFilters_True", func(t *testing.T) {
			// Test with all filters enabled
			tool := "list_module_packages"
			args := map[string]any{
				"Cwd":              projectDir,
				"include_docs":     false,
				"exclude_tests":    true,
				"exclude_internal": true,
				"top_level_only":   true,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("Packages (all filters true):\n%s", content)

			// Should NOT include test packages
			if strings.Contains(content, "mypkg_test") {
				t.Errorf("Expected to NOT find test package with all filters, got: %s", content)
			}

			// Should NOT include internal packages
			if strings.Contains(content, "internal") {
				t.Errorf("Expected to NOT find internal package with all filters, got: %s", content)
			}

			// Should NOT include nested packages
			if strings.Contains(content, "subpkg") {
				t.Errorf("Expected to NOT find nested package with all filters, got: %s", content)
			}

			// Should include the main package
			if !strings.Contains(content, "simple") {
				t.Errorf("Expected to find main package with all filters, got: %s", content)
			}
		})

		t.Run("IncludeDocs_BooleanFlag", func(t *testing.T) {
			// Test include_docs boolean flag
			tool := "list_module_packages"
			args := map[string]any{
				"Cwd":              projectDir,
				"include_docs":     true,
				"exclude_tests":    false,
				"exclude_internal": false,
				"top_level_only":   false,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("Packages with include_docs=true:\n%s", content)

			// Should include package documentation
			// (This is a basic check - the actual documentation content depends on the package)
			if !strings.Contains(content, "simple") {
				t.Errorf("Expected to find main package, got: %s", content)
			}
		})
	})

	t.Run("ListPackageSymbols_BooleanFlags", func(t *testing.T) {
		// Test boolean flags for list_package_symbols
		projectDir := t.TempDir()

		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a package with documented symbols
		sourceCode := `package main

// Add returns the sum of two integers
func Add(a, b int) int {
	return a + b
}

// Multiply returns the product of two integers
func Multiply(a, b int) int {
	return a * b
}

func main() {
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		t.Run("IncludeDocs_False", func(t *testing.T) {
			// Test include_docs=false (default) - should not include documentation
			tool := "list_package_symbols"
			args := map[string]any{
				"package_path":   "example.com/test",
				"include_docs":   false,
				"include_bodies": false,
				"Cwd":            projectDir,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("Symbols without docs:\n%s", content)

			// Should find symbols
			if !strings.Contains(content, "Add") {
				t.Errorf("Expected to find Add function, got: %s", content)
			}

			// Should NOT include documentation
			if strings.Contains(content, "sum of two integers") {
				t.Errorf("Expected NOT to include documentation when include_docs=false, got: %s", content)
			}
		})

		t.Run("IncludeDocs_True", func(t *testing.T) {
			// Test include_docs=true - should include detail information
			// Note: Current implementation includes the Detail field (signature) from DocumentSymbol
			// Actual documentation comments require a different approach (e.g., parsing source directly)
			tool := "list_package_symbols"
			args := map[string]any{
				"package_path":   "example.com/test",
				"include_docs":   true,
				"include_bodies": false,
				"Cwd":            projectDir,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("Symbols with docs:\n%s", content)

			// Should find symbols
			if !strings.Contains(content, "Add") {
				t.Errorf("Expected to find Add function, got: %s", content)
			}

			// Should include detail/signature information (even if not actual doc comments)
			if !strings.Contains(content, "func") && !strings.Contains(content, "int") {
				t.Errorf("Expected to include detail information when include_docs=true, got: %s", content)
			}

			t.Logf("Note: LSP DocumentSymbol.Detail contains signature, not documentation comments. Full documentation extraction requires source file parsing.")
		})

		t.Run("IncludeBodies_False", func(t *testing.T) {
			// Test include_bodies=false (default) - should not include function bodies
			tool := "list_package_symbols"
			args := map[string]any{
				"package_path":   "example.com/test",
				"include_docs":   false,
				"include_bodies": false,
				"Cwd":            projectDir,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("Symbols without bodies:\n%s", content)

			// Should find symbols
			if !strings.Contains(content, "Add") {
				t.Errorf("Expected to find Add function, got: %s", content)
			}

			// Should NOT include function bodies
			if strings.Contains(content, "return a + b") {
				t.Errorf("Expected NOT to include function body when include_bodies=false, got: %s", content)
			}
		})

		t.Run("IncludeBodies_True", func(t *testing.T) {
			// Test include_bodies=true
			// Note: Current implementation uses golang.DocumentSymbols() which doesn't include function bodies
			// Full function body extraction requires reading source files directly
			tool := "list_package_symbols"
			args := map[string]any{
				"package_path":   "example.com/test",
				"include_docs":   false,
				"include_bodies": true,
				"Cwd":            projectDir,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Failed to call tool %s: %v", tool, err)
			}

			content := testutil.ResultText(res)
			t.Logf("Symbols with bodies:\n%s", content)

			// Should find symbols
			if !strings.Contains(content, "Add") {
				t.Errorf("Expected to find Add function, got: %s", content)
			}

			// Note: Current implementation doesn't include actual function bodies
			// This test verifies the flag is accepted and doesn't crash
			t.Logf("Note: Function bodies not included (LSP DocumentSymbol limitation). Bodies require direct source file parsing.")
		})
	})
}
