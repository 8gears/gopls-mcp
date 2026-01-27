package integration

// End-to-end test for file watching functionality.
// This test verifies that when files are modified, gopls-mcp's
// file watcher detects the changes and invalidates the gopls cache,
// so subsequent tool calls see the updated content.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestFileWatchingE2E simulates the real-world Claude Code workflow:
// 1. User has a Go project with gopls-mcp running
// 2. User asks "what functions are in this package?" → gets initial list
// 3. Claude suggests adding a new function → user accepts
// 4. Claude writes the file to disk
// 5. User asks again "what functions are in this package?" → should see new function
//
// This test verifies that gopls-mcp detects file changes made by external editors
// (like Claude Code) and updates its internal state accordingly.
func TestFileWatchingE2E(t *testing.T) {

	// Use the simple test project
	projectDir := testutil.CopyProjectTo(t, "simple")

	// Create a log file to capture watcher activity
	logFile := filepath.Join(projectDir, "gopls-mcp.log")

	// Start gopls-mcp with logging enabled (stdio mode, like Claude Code uses)
	mcpSession, ctx, _ := testutil.StartMCPServerWithLogfile(t, projectDir, logFile)

	// Give the file watcher time to initialize
	time.Sleep(500 * time.Millisecond)

	// ========================================
	// SCENARIO 1: User asks "what functions exist?"
	// ========================================
	t.Log("=== SCENARIO 1: User asks 'what functions are in this package?' ===")
	t.Log("Calling get_package_symbol_detail to get initial state...")

	tool := "go_get_package_symbol_detail"
	args := map[string]any{
		"package_path": "example.com/simple",
		"symbol_filters": []any{
			map[string]any{"name": "Hello"},
		},
		"include_docs":   false,
		"include_bodies": false,
	}

	res, err := mcpSession.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("Failed to call tool %s: %v", tool, err)
	}

	baselineContent := testutil.ResultText(t, res, testutil.GoldenFileWatching)
	t.Logf("Initial package state:\n%s", baselineContent)

	// Verify we can see the original functions
	if !strings.Contains(baselineContent, "Hello") {
		t.Errorf("Expected to see 'Hello' function, got:\n%s", baselineContent)
	}
	if strings.Contains(baselineContent, "Goodbye") {
		t.Errorf("Should NOT see 'Goodbye' function yet, got:\n%s", baselineContent)
	}
	t.Log("✓ Initial state: Hello function exists, Goodbye does not")

	// ========================================
	// SCENARIO 2: Claude Code suggests adding a function
	// ========================================
	t.Log("=== SCENARIO 2: Claude suggests adding Goodbye() function ===")
	t.Log("Simulating Claude Code writing the suggestion to disk...")

	mainGoPath := filepath.Join(projectDir, "main.go")
	originalContent, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatal(err)
	}

	// Claude Code writes the new function to disk
	// Append the new function after the main function
	newContent := string(originalContent)
	// Find the last } which should be the end of main()
	lastBraceIndex := strings.LastIndex(newContent, "}")
	if lastBraceIndex == -1 {
		t.Fatal("Could not find closing brace in main.go")
	}
	// Insert Goodbye function AFTER the last }
	newContent = newContent[:lastBraceIndex+1] + "\n\n// Goodbye returns a farewell message\nfunc Goodbye() string {\n\treturn \"goodbye world\"\n}"

	if err := os.WriteFile(mainGoPath, []byte(newContent), 0644); err != nil {
		t.Fatal(err)
	}
	t.Log("✓ File written to disk (simulating Claude Code editor)")

	// ========================================
	// SCENARIO 3: Wait for gopls-mcp to detect the change
	// ========================================
	t.Log("=== SCENARIO 3: Waiting for gopls-mcp to detect file change ===")
	t.Log("(In real usage, this happens in background while user is reading/deciding)")

	// The watcher batches events with 500ms delay, plus metadata reload takes time
	// We wait a bit longer to ensure everything is processed
	time.Sleep(2 * time.Second)

	// Check logs to verify watcher detected the change
	logContent, _ := os.ReadFile(logFile)
	logText := string(logContent)

	if strings.Contains(logText, "[gopls-mcp/watcher] Detected") {
		t.Log("✓ Watcher detected file change")
	} else {
		t.Error("✗ Watcher did NOT detect file change")
	}

	if strings.Contains(logText, "[gopls-mcp/watcher] Cache invalidated") {
		t.Log("✓ Cache was invalidated")
	} else {
		t.Error("✗ Cache was NOT invalidated")
	}

	// ========================================
	// SCENARIO 4: User asks again "what functions exist?"
	// ========================================
	t.Log("=== SCENARIO 4: User asks again 'what functions are in this package?' ===")
	t.Log("Calling get_package_symbol_detail to get updated state...")

	// Update the filter to include both Hello and Goodbye
	updatedArgs := map[string]any{
		"package_path": "example.com/simple",
		"symbol_filters": []any{
			map[string]any{"name": "Hello"},
			map[string]any{"name": "Goodbye"},
		},
		"include_docs":   false,
		"include_bodies": false,
	}

	res2, err := mcpSession.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: updatedArgs})
	if err != nil {
		t.Fatalf("Failed to call tool %s after modification: %v", tool, err)
	}

	updatedContent := testutil.ResultText(t, res2, testutil.GoldenFileWatching)
	t.Logf("Updated package state:\n%s", updatedContent)

	// ========================================
	// VERIFICATION: Did gopls-mcp notice the change?
	// ========================================
	t.Log("=== VERIFICATION: Did gopls-mcp update its internal state? ===")

	// Check 1: New function should be visible
	if !strings.Contains(updatedContent, "Goodbye") {
		t.Errorf("FAIL: gopls-mcp did NOT detect the file change!")
		t.Errorf("Expected 'Goodbye' function in output, but got:\n%s", updatedContent)

		// Show diagnostic info
		t.Logf("\n=== Full log file contents for debugging ===")
		t.Logf("%s", logText)
		t.Logf("=== End of logs ===\n")
	} else {
		t.Log("✓ SUCCESS: gopls-mcp detected the file change and updated its state!")
		t.Log("✓ The new 'Goodbye' function is now visible in the API")
	}

	// Check 2: Old function should still be there
	if !strings.Contains(updatedContent, "Hello") {
		t.Errorf("Old 'Hello' function disappeared, got:\n%s", updatedContent)
	} else {
		t.Log("✓ Original 'Hello' function is still present")
	}
}
