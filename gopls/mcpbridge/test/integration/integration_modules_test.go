package integration

// End-to-end tests for module and package discovery functionality.

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestListModulesE2E is an end-to-end test that verifies list_modules works.
func TestListModulesE2E(t *testing.T) {
	// Use the simple test project
	projectDir := testutil.CopyProjectTo(t, "simple")

	// Start gopls-mcp

	t.Run("ListModules", func(t *testing.T) {
		tool := "go_list_modules"
		args := map[string]any{
			"direct_only": true,
			"Cwd":         projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenListModules)
		t.Logf("Modules:\n%s", content)

		// Should find the main module
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find 'example.com/simple' module, got: %s", content)
		}

		// Should mention total modules
		if !strings.Contains(content, "Modules (") {
			t.Errorf("Expected 'Modules (' in output, got: %s", content)
		}

		// Should have [MAIN] marker for main module
		if !strings.Contains(content, "[MAIN]") {
			t.Errorf("Expected '[MAIN]' marker for main module, got: %s", content)
		}
	})
}

// TestListModulePackagesE2E is an end-to-end test that verifies list_module_packages works.
func TestListModulePackagesE2E(t *testing.T) {
	// Use the simple test project
	projectDir := testutil.CopyProjectTo(t, "simple")

	// Start gopls-mcp

	t.Run("ListPackagesWithoutDocs", func(t *testing.T) {
		tool := "go_list_module_packages"
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

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenListModulePackages)
		t.Logf("Module packages (no docs):\n%s", content)

		// Should find the main package
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find 'example.com/simple' module, got: %s", content)
		}

		// Should mention total packages
		if !strings.Contains(content, "packages)") {
			t.Errorf("Expected 'packages)' in output, got: %s", content)
		}
	})

	t.Run("ListPackagesWithDocs", func(t *testing.T) {
		tool := "go_list_module_packages"
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

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenListModulePackages)
		t.Logf("Module packages (with docs):\n%s", testutil.TruncateString(content, 500))

		// Should find the main package
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find 'example.com/simple' module, got: %s", content)
		}
	})

	t.Run("ListPackagesForSpecificModule", func(t *testing.T) {
		tool := "go_list_module_packages"
		args := map[string]any{
			"Cwd":              projectDir,
			"module_path":      "example.com/simple",
			"include_docs":     false,
			"exclude_tests":    false,
			"exclude_internal": false,
			"top_level_only":   false,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenListModulePackages)
		t.Logf("Module packages (specific module):\n%s", content)

		// Should find the requested module
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find 'example.com/simple' module, got: %s", content)
		}
	})
}

// TestListPackageSymbolsE2E is an end-to-end test that verifies list_package_symbols works.
func TestListPackageSymbolsE2E(t *testing.T) {
	// Use the simple test project
	projectDir := testutil.CopyProjectTo(t, "simple")

	// Start gopls-mcp

	t.Run("ListSymbolsWithoutDocs", func(t *testing.T) {
		tool := "go_list_package_symbols"
		args := map[string]any{
			"Cwd":            projectDir,
			"package_path":   "example.com/simple",
			"include_docs":   false,
			"include_bodies": false,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenListPackageSymbols)
		t.Logf("Package symbols (no docs):\n%s", content)

		// Should mention the package
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find 'example.com/simple' package, got: %s", content)
		}

		// Should mention total symbols
		if !strings.Contains(content, "symbols)") {
			t.Errorf("Expected 'symbols)' in output, got: %s", content)
		}

		// Should find exported symbols (Hello, Add, or Person)
		if !strings.Contains(content, "Hello") && !strings.Contains(content, "Add") && !strings.Contains(content, "Person") {
			t.Errorf("Expected to find exported symbols, got: %s", content)
		}
	})

	t.Run("ListSymbolsWithDocs", func(t *testing.T) {
		tool := "go_list_package_symbols"
		args := map[string]any{
			"Cwd":            projectDir,
			"package_path":   "example.com/simple",
			"include_docs":   true,
			"include_bodies": false,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenListPackageSymbols)
		t.Logf("Package symbols (with docs):\n%s", testutil.TruncateString(content, 500))

		// Should mention the package
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find 'example.com/simple' package, got: %s", content)
		}

		// Should mention "with docs"
		if !strings.Contains(content, "with docs") {
			t.Errorf("Expected output to mention 'with docs', got: %s", content)
		}
	})

	t.Run("ListSymbolsWithBodies", func(t *testing.T) {
		tool := "go_list_package_symbols"
		args := map[string]any{
			"Cwd":            projectDir,
			"package_path":   "example.com/simple",
			"include_docs":   false,
			"include_bodies": true,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenListPackageSymbols)
		t.Logf("Package symbols (with bodies):\n%s", testutil.TruncateString(content, 500))

		// Should mention the package
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find 'example.com/simple' package, got: %s", content)
		}

		// Should mention "with bodies"
		if !strings.Contains(content, "with bodies") {
			t.Errorf("Expected output to mention 'with bodies', got: %s", content)
		}
	})
}
