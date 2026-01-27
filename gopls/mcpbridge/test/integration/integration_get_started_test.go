package integration

// End-to-end tests for get_started functionality.

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestGetStartedE2E is an end-to-end test that verifies get_started works.
func TestGetStartedE2E(t *testing.T) {
	// Use the simple test project
	projectDir := testutil.CopyProjectTo(t, "simple")

	// Start gopls-mcp

	t.Run("GetStartedBasic", func(t *testing.T) {
		tool := "go_get_started"
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

		content := testutil.ResultText(t, res, testutil.GoldenGetStarted)
		t.Logf("Get Started output:\n%s", content)

		// Compare against golden file (documentation + regression check)

		// Should have project identity section
		if !strings.Contains(content, "## Project") {
			t.Errorf("Expected '## Project' section in output, got: %s", content)
		}

		// Should have quick stats section
		if !strings.Contains(content, "## Quick Stats") {
			t.Errorf("Expected '## Quick Stats' section in output, got: %s", content)
		}

		// Should mention the module path
		if !strings.Contains(content, "example.com/simple") {
			t.Errorf("Expected to find 'example.com/simple' module, got: %s", content)
		}

		// Should have total packages count
		if !strings.Contains(content, "Total Packages:") {
			t.Errorf("Expected 'Total Packages:' in output, got: %s", content)
		}

		// Should have next steps section
		if !strings.Contains(content, "## Next Steps") {
			t.Errorf("Expected '## Next Steps' section in output, got: %s", content)
		}

		// Should suggest using go_search
		if !strings.Contains(content, "go_search") {
			t.Errorf("Expected to suggest 'go_search', got: %s", content)
		}
	})

	t.Run("GetStartedHasCategories", func(t *testing.T) {
		tool := "go_get_started"
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

		content := testutil.ResultText(t, res, testutil.GoldenGetStarted)
		t.Logf("Get Started categories output:\n%s", content)

		// Compare against golden file (documentation + regression check)

		// Should have package categories section
		if !strings.Contains(content, "## Package Categories") {
			t.Errorf("Expected '## Package Categories' section in output, got: %s", content)
		}
	})

	t.Run("GetStartedHasEntryPoints", func(t *testing.T) {
		tool := "go_get_started"
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

		content := testutil.ResultText(t, res, testutil.GoldenGetStarted)
		t.Logf("Get Started entry points output:\n%s", content)

		// Should have entry points section
		if !strings.Contains(content, "## Suggested Entry Points") {
			t.Errorf("Expected '## Suggested Entry Points' section in output, got: %s", content)
		}
	})
}
