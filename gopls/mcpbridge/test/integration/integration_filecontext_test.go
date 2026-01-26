package integration

// End-to-end test for go_file_context functionality.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestGoFileContextE2E is an end-to-end test that verifies go_file_context works.
func TestGoFileContextE2E(t *testing.T) {
	// Use the simple test project
	projectDir := testutil.CopyProjectTo(t, "simple")

	t.Run("GetFileContext", func(t *testing.T) {
		tool := "go_file_context"
		mainGoPath := filepath.Join(projectDir, "main.go")
		args := map[string]any{
			"file": mainGoPath,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)
		t.Logf("File context:\n%s", content)

		// Should mention the file and package
		if !strings.Contains(content, "main.go") {
			t.Errorf("Expected file context to mention 'main.go', got: %s", content)
		}

		if !strings.Contains(content, "package") {
			t.Errorf("Expected file context to mention package, got: %s", content)
		}
	})
}
