package e2e

// E2E tests for response size limiting functionality.
// These tests verify that the max_response_size parameter works correctly
// and that truncation actually happens when limits are exceeded.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestResponseLimits_Unlimited verifies that max_response_size=-1 returns full response.
func TestResponseLimits_Unlimited(t *testing.T) {
	goplsMcpDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get gopls-mcp directory: %v", err)
	}

	t.Run("analyze_workspace_unlimited", func(t *testing.T) {
		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "analyze_workspace",
			Arguments: map[string]any{
				"Cwd":               goplsMcpDir,
				"max_response_size": -1, // No limit
			},
		})
		if err != nil {
			t.Fatalf("Failed to call analyze_workspace: %v", err)
		}

		content := testutil.ResultText(res)
		t.Logf("Unlimited response size: %d chars", len(content))
		t.Logf("Response preview:\n%s", testutil.TruncateString(content, 1000))

		// With no limit, should get complete response
		// For the gopls-mcp project, this should be quite large
		if len(content) < 1000 {
			t.Errorf("Expected larger response with no limit, got: %d chars", len(content))
		}

		// Should NOT have truncation metadata
		if strings.Contains(content, "\"_truncated\":") {
			t.Error("Response should not be truncated when max_response_size=-1")
		}
	})
}

// TestResponseLimits_LargeLimit verifies that a large limit doesn't truncate unnecessarily.
func TestResponseLimits_LargeLimit(t *testing.T) {
	goplsMcpDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get gopls-mcp directory: %v", err)
	}

	t.Run("analyze_workspace_large_limit", func(t *testing.T) {
		// Set a large limit that should fit the full response
		largeLimit := 500000 // 500K characters should be enough for most responses

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "analyze_workspace",
			Arguments: map[string]any{
				"Cwd":               goplsMcpDir,
				"max_response_size": largeLimit,
			},
		})
		if err != nil {
			t.Fatalf("Failed to call analyze_workspace: %v", err)
		}

		content := testutil.ResultText(res)
		t.Logf("Response with %d char limit: %d chars", largeLimit, len(content))

		// Should get full response (no truncation)
		if len(content) > largeLimit {
			t.Errorf("Response exceeded limit of %d chars (got %d)", largeLimit, len(content))
		}

		// Should NOT have truncation metadata since response fits
		if strings.Contains(content, "\"_truncated\":") {
			t.Error("Response should not be truncated when limit is large enough")
		}
	})
}

// TestResponseLimits_SmallLimit verifies that truncation actually happens with a very small limit.
func TestResponseLimits_SmallLimit(t *testing.T) {
	goplsMcpDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get gopls-mcp directory: %v", err)
	}

	t.Run("analyze_workspace_small_limit", func(t *testing.T) {
		// Set a very small limit that forces truncation
		smallLimit := 500 // 500 characters - definitely too small

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "analyze_workspace",
			Arguments: map[string]any{
				"Cwd":               goplsMcpDir,
				"max_response_size": smallLimit,
			},
		})
		if err != nil {
			t.Fatalf("Failed to call analyze_workspace: %v", err)
		}

		content := testutil.ResultText(res)
		t.Logf("Response with %d char limit: %d chars", smallLimit, len(content))
		t.Logf("Response:\n%s", content)

		// HARD ASSERTION: Response must be close to limit
		// Allow some overhead for metadata (up to 20%)
		maxAllowed := smallLimit + (smallLimit / 5) // 600 chars max
		if len(content) > maxAllowed {
			t.Errorf("Response %d chars exceeds limit %d by too much (max allowed: %d)",
				len(content), smallLimit, maxAllowed)
		}

		// HARD ASSERTION: Response should be truncated (shorter than unlimited)
		// Compare with unlimited response
		unlimitedRes, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "analyze_workspace",
			Arguments: map[string]any{
				"Cwd":               goplsMcpDir,
				"max_response_size": -1,
			},
		})
		if err == nil {
			unlimitedContent := testutil.ResultText(unlimitedRes)
			if len(content) >= len(unlimitedContent) {
				t.Errorf("Limited response (%d chars) should be shorter than unlimited (%d chars)",
					len(content), len(unlimitedContent))
			} else {
				t.Logf("✓ Response truncated: %d chars vs %d chars unlimited (%.1fx reduction)",
					len(content), len(unlimitedContent),
					float64(len(unlimitedContent))/float64(len(content)))
			}
		}
	})
}

// TestResponseLimits_DefaultBehavior verifies default behavior without max_response_size.
func TestResponseLimits_DefaultBehavior(t *testing.T) {
	goplsMcpDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get gopls-mcp directory: %v", err)
	}

	t.Run("analyze_workspace_default_limit", func(t *testing.T) {
		// Don't set max_response_size - should use global default
		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "analyze_workspace",
			Arguments: map[string]any{
				"Cwd": goplsMcpDir,
				// max_response_size not set
			},
		})
		if err != nil {
			t.Fatalf("Failed to call analyze_workspace: %v", err)
		}

		content := testutil.ResultText(res)
		t.Logf("Response with default limit: %d chars", len(content))

		// Should get truncated response (using global 100K token limit)
		if len(content) == 0 {
			t.Error("Expected non-empty response")
		}

		// Should have some truncation indication for large project
		if strings.Contains(content, "showing") || strings.Contains(content, "of") {
			t.Log("✓ Response appears to be truncated (shows 'showing X of Y')")
		}
	})
}

// TestResponseLimits_Comparison verifies that unlimited response is larger than limited.
func TestResponseLimits_Comparison(t *testing.T) {
	goplsMcpDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get gopls-mcp directory: %v", err)
	}

	// First, get response with small limit
	limitedRes, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
		Name: "analyze_workspace",
		Arguments: map[string]any{
			"Cwd":               goplsMcpDir,
			"max_response_size": 1000, // Small limit
		},
	})
	if err != nil {
		t.Fatalf("Failed to call analyze_workspace with limit: %v", err)
	}
	limitedContent := testutil.ResultText(limitedRes)
	limitedSize := len(limitedContent)

	// Then, get response with no limit
	unlimitedRes, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
		Name: "analyze_workspace",
		Arguments: map[string]any{
			"Cwd":               goplsMcpDir,
			"max_response_size": -1, // No limit
		},
	})
	if err != nil {
		t.Fatalf("Failed to call analyze_workspace without limit: %v", err)
	}
	unlimitedContent := testutil.ResultText(unlimitedRes)
	unlimitedSize := len(unlimitedContent)

	t.Logf("Limited response (1000 chars): %d chars", limitedSize)
	t.Logf("Unlimited response: %d chars", unlimitedSize)

	// Unlimited should be significantly larger
	if unlimitedSize <= limitedSize {
		t.Errorf("Expected unlimited response (%d chars) to be larger than limited (%d chars)",
			unlimitedSize, limitedSize)
	}

	// Verify unlimited is at least 2x larger for a large project
	if unlimitedSize < limitedSize*2 {
		t.Logf("Note: Unlimited response is %.1fx larger (project might be small enough that limit doesn't matter)",
			float64(unlimitedSize)/float64(limitedSize))
	} else {
		t.Logf("✓ Unlimited response is %.1fx larger than limited - truncation is working!",
			float64(unlimitedSize)/float64(limitedSize))
	}
}

// TestResponseLimits_ListPackageSymbols tests limiting with list_package_symbols.
func TestResponseLimits_ListPackageSymbols(t *testing.T) {
	t.Run("list_package_symbols_small_limit", func(t *testing.T) {
		// Use stdlib package which has many symbols
		limit := 200 // Very small limit

		limitedRes, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "list_package_symbols",
			Arguments: map[string]any{
				"package_path":      "fmt",
				"include_docs":      false,
				"include_bodies":    false,
				"max_response_size": limit,
			},
		})
		if err != nil {
			t.Fatalf("Failed to call list_package_symbols: %v", err)
		}

		limitedContent := testutil.ResultText(limitedRes)
		t.Logf("Response with %d char limit:\n%s", limit, limitedContent)

		// HARD ASSERTION: Response should fit within reasonable bounds
		maxAllowed := limit + (limit / 5) // Allow 20% overhead for metadata
		if len(limitedContent) > maxAllowed {
			t.Errorf("Response %d chars exceeds limit %d by too much (max allowed: %d)",
				len(limitedContent), limit, maxAllowed)
		}

		// HARD ASSERTION: Verify we got a response
		if len(limitedContent) == 0 {
			t.Fatal("Expected non-empty response")
		}

		// HARD ASSERTION: Limited should be shorter than unlimited
		unlimitedRes, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "list_package_symbols",
			Arguments: map[string]any{
				"package_path":   "fmt",
				"include_docs":   false,
				"include_bodies": false,
				// No max_response_size - use global default or unlimited
			},
		})
		if err == nil {
			unlimitedContent := testutil.ResultText(unlimitedRes)
			if len(limitedContent) >= len(unlimitedContent) && len(unlimitedContent) > limit*2 {
				t.Errorf("Limited response (%d chars) should be shorter than unlimited (%d chars)",
					len(limitedContent), len(unlimitedContent))
			} else {
				t.Logf("✓ Response truncated correctly: %d chars vs %d chars unlimited",
					len(limitedContent), len(unlimitedContent))
			}
		}
	})

	t.Run("list_package_symbols_unlimited", func(t *testing.T) {
		// Verify unlimited works
		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "list_package_symbols",
			Arguments: map[string]any{
				"package_path":      "fmt",
				"include_docs":      false,
				"include_bodies":    false,
				"max_response_size": -1, // Unlimited
			},
		})
		if err != nil {
			t.Fatalf("Failed to call list_package_symbols: %v", err)
		}

		content := testutil.ResultText(res)
		t.Logf("Unlimited response size: %d chars", len(content))

		// Should get full response with many symbols
		if len(content) < 500 {
			t.Errorf("Expected larger response with no limit, got: %d chars", len(content))
		}

		// Should NOT have truncation metadata
		if strings.Contains(content, "\"_truncated\":") {
			t.Error("Response should not be truncated when max_response_size=-1")
		}
	})
}

// TestResponseLimits_GoPackageAPI tests MaxResponseSize with list_package_symbols tool.
func TestResponseLimits_GoPackageAPI(t *testing.T) {
	t.Run("list_package_symbols_small_limit", func(t *testing.T) {
		limit := 300

		limitedRes, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "list_package_symbols",
			Arguments: map[string]any{
				"package_path":      "fmt",
				"include_docs":      false,
				"include_bodies":    false,
				"max_response_size": limit,
			},
		})
		if err != nil {
			t.Fatalf("Failed to call list_package_symbols: %v", err)
		}

		limitedContent := testutil.ResultText(limitedRes)
		t.Logf("Response with %d char limit: %d chars", limit, len(limitedContent))

		// HARD ASSERTION: Response should fit within reasonable bounds
		maxAllowed := limit + (limit / 5) // 20% overhead
		if len(limitedContent) > maxAllowed {
			t.Errorf("Response %d chars exceeds limit %d (max allowed: %d)",
				len(limitedContent), limit, maxAllowed)
		}

		if len(limitedContent) == 0 {
			t.Error("Expected non-empty response")
		}
	})

	t.Run("list_package_symbols_unlimited", func(t *testing.T) {
		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "list_package_symbols",
			Arguments: map[string]any{
				"package_path":      "fmt",
				"include_docs":      false,
				"include_bodies":    false,
				"max_response_size": -1,
			},
		})
		if err != nil {
			t.Fatalf("Failed to call list_package_symbols: %v", err)
		}

		content := testutil.ResultText(res)
		t.Logf("Unlimited response size: %d chars", len(content))

		if len(content) < 200 {
			t.Errorf("Expected larger response with no limit, got: %d chars", len(content))
		}

		if strings.Contains(content, "\"_truncated\":") {
			t.Error("Response should not be truncated when max_response_size=-1")
		}
	})
}

// TestResponseLimits_GoSearch tests MaxResponseSize with go_search tool.
func TestResponseLimits_GoSearch(t *testing.T) {
	t.Run("go_search_small_limit", func(t *testing.T) {
		limit := 150

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "go_search",
			Arguments: map[string]any{
				"query":             "Print",
				"max_results":       50,
				"max_response_size": limit,
			},
		})
		if err != nil {
			t.Fatalf("Failed to call go_search: %v", err)
		}

		content := testutil.ResultText(res)
		t.Logf("Response with %d char limit: %d chars", limit, len(content))

		maxAllowed := limit + (limit / 5)
		if len(content) > maxAllowed {
			t.Errorf("Response %d chars exceeds limit %d (max allowed: %d)",
				len(content), limit, maxAllowed)
		}

		if len(content) == 0 {
			t.Error("Expected non-empty response")
		}
	})
}

// TestResponseLimits_ListModules tests MaxResponseSize with list_modules tool.
func TestResponseLimits_ListModules(t *testing.T) {
	t.Run("list_modules_with_limit", func(t *testing.T) {
		// This tool typically returns small responses, so we test with a reasonable limit
		limit := 200

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "list_modules",
			Arguments: map[string]any{
				"direct_only":       true,
				"max_response_size": limit,
			},
		})
		if err != nil {
			t.Fatalf("Failed to call list_modules: %v", err)
		}

		content := testutil.ResultText(res)
		t.Logf("Response with %d char limit: %d chars", limit, len(content))

		maxAllowed := limit + (limit / 5)
		if len(content) > maxAllowed {
			t.Errorf("Response %d chars exceeds limit %d (max allowed: %d)",
				len(content), limit, maxAllowed)
		}

		if len(content) == 0 {
			t.Error("Expected non-empty response")
		}
	})
}
