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
		tool := "analyze_workspace"
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

		content := testutil.ResultText(res)
		t.Logf("Analyze workspace output:\n%s", content)

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

	t.Run("MaxResponseSize_Unlimited", func(t *testing.T) {
		tool := "analyze_workspace"
		args := map[string]any{
			"Cwd":               projectDir,
			"max_response_size": -1, // No limit
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)
		t.Logf("Unlimited response size:\n%s", testutil.TruncateString(content, 500))

		// With no limit, should get full response (same as normal for small project)
		if !strings.Contains(content, "Workspace Analysis") {
			t.Errorf("Expected 'Workspace Analysis' section in unlimited output")
		}
	})

	t.Run("MaxResponseSize_SpecificLimit", func(t *testing.T) {
		tool := "analyze_workspace"
		limit := 1000
		args := map[string]any{
			"Cwd":               projectDir,
			"max_response_size": limit, // Very small limit
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)
		t.Logf("Response with %d char limit:\n%s", limit, content)

		// HARD ASSERTION: With small limit, response should be truncated
		// Allow 20% overhead for metadata
		maxAllowed := limit + (limit / 5)
		if len(content) > maxAllowed {
			t.Errorf("Response %d chars exceeds limit %d by too much (max allowed: %d)",
				len(content), limit, maxAllowed)
		}

		// HARD ASSERTION: Response should be shorter than unlimited (or equal if project is small)
		unlimitedRes, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: tool,
			Arguments: map[string]any{
				"Cwd":               projectDir,
				"max_response_size": -1,
			},
		})
		if err == nil {
			unlimitedContent := testutil.ResultText(unlimitedRes)
			// If unlimited response fits within the limit, sizes will be equal (expected)
			// Only fail if limited is strictly larger than unlimited (bug)
			if len(content) > len(unlimitedContent) {
				t.Errorf("Limited response (%d chars) should not be larger than unlimited (%d chars)",
					len(content), len(unlimitedContent))
			} else if len(content) < len(unlimitedContent) {
				t.Logf("✓ Response truncated: %d chars vs %d chars unlimited (%.1fx reduction)",
					len(content), len(unlimitedContent),
					float64(len(unlimitedContent))/float64(len(content)))
			} else {
				t.Logf("✓ Response fits within limit: %d chars (same as unlimited - project is small enough)",
					len(content))
			}
		}
	})

	t.Run("MaxResponseSize_UseGlobalDefault", func(t *testing.T) {
		tool := "analyze_workspace"
		args := map[string]any{
			"Cwd":               projectDir,
			"max_response_size": 0, // Use global default
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)

		// Should use global config and get normal response
		if !strings.Contains(content, "Workspace Analysis") {
			t.Errorf("Expected 'Workspace Analysis' with global default limit")
		}
	})

	t.Run("MaxResponseSize_NotSet", func(t *testing.T) {
		tool := "analyze_workspace"
		args := map[string]any{
			"Cwd": projectDir, // max_response_size not set
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)

		// Should use global config by default
		if !strings.Contains(content, "Workspace Analysis") {
			t.Errorf("Expected 'Workspace Analysis' with global default limit when max_response_size not set")
		}
	})
}
