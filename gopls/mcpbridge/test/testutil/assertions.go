package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ParseJSONResult parses the MCP tool result and returns the structured data.
// It extracts the JSON from the first TextContent block.
func ParseJSONResult(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()

	if result == nil {
		t.Fatal("result is nil")
	}

	if len(result.Content) == 0 {
		t.Fatal("result.Content is empty")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	text := textContent.Text
	if text == "" {
		t.Fatal("text content is empty")
	}

	// Try to parse as JSON (wrapped in { } if needed)
	var data map[string]any
	var err error

	// Try direct JSON parse first
	err = json.Unmarshal([]byte(text), &data)
	if err != nil {
		// If that fails, the result might be just text with no JSON
		// Return empty map rather than failing
		return map[string]any{}
	}

	return data
}

// AssertStringContains asserts that the content contains the expected substring.
func AssertStringContains(t *testing.T, content, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Errorf("Expected content to contain:\n  %q\n\nGot:\n  %s", expected, content)
	}
}

// AssertStringNotContains asserts that the content does NOT contain the substring.
func AssertStringNotContains(t *testing.T, content, unexpected string) {
	t.Helper()
	if strings.Contains(content, unexpected) {
		t.Errorf("Expected content NOT to contain:\n  %q\n\nGot:\n  %s", unexpected, content)
	}
}

// AssertIntEqual asserts that two integers are equal.
func AssertIntEqual(t *testing.T, got, expected int, msg string) {
	t.Helper()
	if got != expected {
		t.Errorf("%s: expected %d, got %d", msg, expected, got)
	}
}

// AssertContainsAny asserts that content contains at least one of the expected strings.
func AssertContainsAny(t *testing.T, content string, expected []string, msg string) {
	t.Helper()
	for _, e := range expected {
		if strings.Contains(content, e) {
			return // Found at least one
		}
	}
	t.Errorf("%s: expected content to contain one of %v, got:\n%s", msg, expected, content)
}

// AssertHasPrefix asserts that content starts with the expected prefix.
func AssertHasPrefix(t *testing.T, content, prefix string) {
	t.Helper()
	if !strings.HasPrefix(content, prefix) {
		t.Errorf("Expected content to start with:\n  %q\n\nGot:\n  %s", prefix, content)
	}
}

// ExtractCount extracts a count from content like "Found 3 reference(s)" -> 3
func ExtractCount(t *testing.T, content, prefix string) int {
	t.Helper()
	if !strings.Contains(content, prefix) {
		return -1
	}

	// Find the number after the prefix
	idx := strings.Index(content, prefix)
	if idx == -1 {
		return -1
	}

	remaining := content[idx+len(prefix):]
	var count int
	_, err := fmt.Sscanf(remaining, "%d", &count)
	if err != nil {
		t.Errorf("Failed to extract count after %q: %v", prefix, err)
		return -1
	}
	return count
}

// Position represents a file position for testing
type Position struct {
	Line   int
	Column int
}

// FindPositionInContent finds the line/column of a substring in test content.
// Returns -1 for both if not found.
func FindPositionInContent(t *testing.T, content, search string) Position {
	t.Helper()
	lines := strings.Split(content, "\n")
	for lineNum, line := range lines {
		colIdx := strings.Index(line, search)
		if colIdx != -1 {
			// Line is 1-indexed, Column is 1-indexed (UTF-16)
			return Position{
				Line:   lineNum + 1,
				Column: colIdx + 1,
			}
		}
	}
	return Position{Line: -1, Column: -1}
}

// WriteFile is a convenience wrapper around os.WriteFile for tests.
func WriteFile(dir, name, content string) error {
	return os.WriteFile(dir+"/"+name, []byte(content), 0644)
}

// TruncateString truncates a string to a maximum length for logging.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
