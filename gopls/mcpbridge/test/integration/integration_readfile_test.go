package integration

// End-to-end test for go_read_file functionality.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestGoReadFileE2E is an end-to-end test that verifies go_read_file works.
func TestGoReadFileE2E(t *testing.T) {
	t.Run("ReadExistingFile", func(t *testing.T) {
		// Create a test project
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with known content
		sourceCode := `package main

import "fmt"

// Hello returns a greeting message
func Hello() string {
	return "hello world"
}

// Add returns the sum of two integers
func Add(a, b int) int {
	return a + b
}

func main() {
	fmt.Println(Hello())
	fmt.Println(Add(1, 2))
}
`
		mainGoPath := filepath.Join(projectDir, "main.go")
		if err := os.WriteFile(mainGoPath, []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Start gopls-mcp

		tool := "go_read_file"
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

		content := testutil.ResultText(t, res, testutil.GoldenReadFileExisting)
		t.Logf("File content:\n%s", content)

		// Verify the content matches what we wrote
		if !strings.Contains(content, "package main") {
			t.Errorf("Expected content to contain 'package main', got: %s", content)
		}

		if !strings.Contains(content, "func Hello() string") {
			t.Errorf("Expected content to contain Hello function, got: %s", content)
		}

		if !strings.Contains(content, "func Add(a, b int) int") {
			t.Errorf("Expected content to contain Add function, got: %s", content)
		}

		// Verify the file path is mentioned in summary
		if !strings.Contains(content, mainGoPath) && !strings.Contains(content, "main.go") {
			t.Errorf("Expected content to mention file path, got: %s", content)
		}
	})

	t.Run("ReadFileWithSpecialCharacters", func(t *testing.T) {
		// Create a test project with special characters in comments
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with special characters and UTF-8 content
		sourceCode := `package main

import "fmt"

// Special characters: ©, ®, ™, €, £, ¥
// Unicode: 你好世界, 🚀, 🎯
func main() {
	// Test various string literals
	s1 := "Hello, World!"
	s2 := "Привет, мир!"
	s3 := "こんにちは世界"

	fmt.Println(s1)
	fmt.Println(s2)
	fmt.Println(s3)
}
`
		mainGoPath := filepath.Join(projectDir, "main.go")
		if err := os.WriteFile(mainGoPath, []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Start gopls-mcp

		tool := "go_read_file"
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

		content := testutil.ResultText(t, res, testutil.GoldenReadFileSpecialCharacters)
		t.Logf("File content with special characters:\n%s", content)

		// Verify special characters are preserved
		if !strings.Contains(content, "你好世界") {
			t.Errorf("Expected content to contain Chinese characters, got: %s", content)
		}

		if !strings.Contains(content, "🚀") {
			t.Errorf("Expected content to contain rocket emoji, got: %s", content)
		}
	})

	t.Run("ReadNonExistentFile", func(t *testing.T) {
		// This test verifies that go_read_file properly handles non-existent files.
		// Expected behavior: The tool should either return an error OR return a result
		// containing an error message about the file not being found.
		//
		// This is important for graceful error handling - we don't want the tool to
		// crash or return an empty/empty result when a file doesn't exist.

		// Create a test project
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a minimal main.go (we need at least one valid Go file for the project)
		sourceCode := `package main

func main() {
}
`
		mainGoPath := filepath.Join(projectDir, "main.go")
		if err := os.WriteFile(mainGoPath, []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Start gopls-mcp

		// Attempt to read a file that doesn't exist
		tool := "go_read_file"
		nonExistentPath := filepath.Join(projectDir, "does_not_exist.go")

		args := map[string]any{
			"file": nonExistentPath,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})

		// Verify we get proper error handling
		// Case 1: Tool call itself returns an error (acceptable)
		if err != nil {
			return
		}

		// Case 2: Tool returns a result - it should contain an error message
		if res != nil {
			content := testutil.ResultText(t, res, testutil.GoldenReadFileNonExistent)

			if !strings.Contains(content, "failed to get file content") {
				// This is unexpected - the tool should have indicated an error somehow
				t.Errorf("Tool should return an error or error message for non-existent file, but got: %s", content)
			}
		} else {
			t.Errorf("Tool returned nil result and nil error for non-existent file (should indicate error)")
		}
	})

	t.Run("ReadLargeFile", func(t *testing.T) {
		// Create a test project with a larger file
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with many functions
		var builder strings.Builder
		builder.WriteString("package main\n\n")
		builder.WriteString("import \"fmt\"\n\n")

		// Generate 20 functions
		for i := 1; i <= 20; i++ {
			builder.WriteString(fmt.Sprintf("// Function%d does some work\n", i))
			builder.WriteString(fmt.Sprintf("func Function%d() int {\n", i))
			builder.WriteString(fmt.Sprintf("\treturn %d\n", i))
			builder.WriteString("}\n\n")
		}

		builder.WriteString("func main() {\n")
		for i := 1; i <= 20; i++ {
			builder.WriteString(fmt.Sprintf("\tfmt.Println(Function%d())\n", i))
		}
		builder.WriteString("}\n")

		largeFilePath := filepath.Join(projectDir, "large.go")
		if err := os.WriteFile(largeFilePath, []byte(builder.String()), 0644); err != nil {
			t.Fatal(err)
		}

		// Start gopls-mcp

		tool := "go_read_file"
		args := map[string]any{
			"file": largeFilePath,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenReadFileLarge)
		t.Logf("Large file read (length: %d chars)", len(content))

		// Verify we got the complete file
		if !strings.Contains(content, "Function1()") {
			preview := content
			if len(preview) > 100 {
				preview = content[:100] + "..."
			}
			t.Errorf("Expected content to contain Function1, got: %s", preview)
		}

		if !strings.Contains(content, "Function20()") {
			t.Errorf("Expected content to contain Function20, file may be truncated")
		}

		// Count the number of functions to ensure complete content
		functionCount := strings.Count(content, "func Function")
		if functionCount != 20 {
			t.Errorf("Expected 20 functions, got %d", functionCount)
		}
	})
}
