package integration

// End-to-end tests for analyze_workspace functionality.

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestAnalyzeWorkspaceE2E is an end-to-end test that verifies analyze_workspace works.
func TestAnalyzeWorkspaceE2E(t *testing.T) {
	// Use the simple test project
	projectDir := testutil.CopyProjectTo(t, "simple")

	t.Run("AnalyzeSimpleWorkspace", func(t *testing.T) {
		tool := "go_analyze_workspace"
		args := map[string]any{
			"Cwd": projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenAnalyzeWorkspace)
		t.Logf("Analyze workspace output:\n%s", content)

		// Compare against golden file (documentation + regression check)

		// Should have workspace summary section
		if !strings.Contains(content, "Workspace Analysis") {
			t.Errorf("Expected 'Workspace Analysis' section in output, got: %s", content)
		}

		// Should mention the module path
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find 'example.com/simple' module, got: %s", content)
		}

		// Should have packages section
		if !strings.Contains(content, "Packages") {
			t.Errorf("Expected 'Packages' section in output, got: %s", content)
		}

		// Should list the main package
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find the main package 'example.com/simple', got: %s", content)
		}

		// Should identify the main function as an entry point
		if !strings.Contains(content, "main") {
			t.Errorf("Expected to find 'main' as an entry point, got: %s", content)
		}
	})
}
