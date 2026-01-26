package integration

// Test that the cache is properly warmed on server startup.
// This is a regression test for the bug where warmCache called
// WorkspacePackages() without triggering metadata loading first,
// causing go_search to fail with "cache is empty" errors.

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestCacheIsWarmedOnStartup verifies that the workspace cache is populated
// when the server starts, without requiring any prior tool calls.
//
// This test specifically catches the bug where warmCache() called
// snapshot.WorkspacePackages() directly, which just returns the cached
// field (empty in a fresh snapshot) without triggering metadata loading.
//
// The fix ensures warmCache calls snapshot.LoadMetadataGraph() first,
// which triggers reloadWorkspace() and populates the cache.
func TestCacheIsWarmedOnStartup(t *testing.T) {
	// Use the simple test project

	// IMMEDIATELY call go_search without any warmup operations.
	// This tests that the background warmCache goroutine has completed
	// and populated the cache before we need it.
	t.Run("SearchWorksImmediately", func(t *testing.T) {
		tool := "go_search"
		args := map[string]any{
			"query": "Hello",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)

		// CRITICAL: If warmCache didn't trigger LoadMetadataGraph,
		// this will fail with "cache is empty" error message
		if strings.Contains(content, "cache is empty") {
			t.Errorf("BUG: Cache was not warmed on startup! go_search returned:\n%s", content)
		}

		// Verify we actually found the symbol
		if !strings.Contains(content, "Hello") {
			t.Errorf("Expected to find 'Hello' in search results, got:\n%s", content)
		}

		t.Logf("✓ Cache was properly warmed on startup, found 'Hello' immediately")
	})

	// Test another symbol to ensure cache is fully populated
	t.Run("MultipleSymbolsWork", func(t *testing.T) {
		tool := "go_search"
		args := map[string]any{
			"query": "Person",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)

		// Verify no cache empty error
		if strings.Contains(content, "cache is empty") {
			t.Errorf("BUG: Cache not warmed! got:\n%s", content)
		}

		// Should find the Person type
		if !strings.Contains(content, "Person") {
			t.Errorf("Expected to find 'Person' type, got:\n%s", content)
		}

		t.Logf("✓ Multiple symbols work, cache is fully populated")
	})
}

// TestCacheWarmupRaceCondition tests for race conditions in cache warming.
// It runs multiple searches in rapid succession to ensure the warmCache
// goroutine doesn't cause intermittent failures.
func TestCacheWarmupRaceCondition(t *testing.T) {

	// Run multiple searches immediately without any delays
	queries := []string{"Hello", "Person", "Add"}

	for i, query := range queries {
		t.Run("RapidSearch_"+query, func(t *testing.T) {
			tool := "go_search"
			args := map[string]any{
				"query": query,
			}

			res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("Search %d failed: %v", i, err)
			}

			if res == nil {
				t.Fatal("Expected non-nil result")
			}

			content := testutil.ResultText(res)

			// If we see "cache is empty", the warmCache didn't finish
			// before this tool call - a race condition!
			if strings.Contains(content, "cache is empty") {
				t.Errorf("RACE CONDITION: Search %d hit empty cache!\n%s", i, content)
			}

			t.Logf("✓ Search %d (%s) completed without cache error", i, query)
		})
	}
}
