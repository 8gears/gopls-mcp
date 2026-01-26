package integration

// End-to-end test for go_workspace functionality.

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestGoWorkspaceE2E is an end-to-end test that verifies go_workspace works.
func TestGoWorkspaceE2E(t *testing.T) {
	// Use the simple test project

	// Start gopls-mcp

	t.Run("BasicWorkspaceSummary", func(t *testing.T) {
		tool := "go_workspace"
		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: map[string]any{}})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)
		t.Logf("Workspace summary:\n%s", content)

		// Verify the summary mentions the directory
		if !strings.Contains(content, "directory uses Go modules") {
			t.Errorf("Expected workspace summary to mention 'directory uses Go modules', got: %s", content)
		}
	})
}
