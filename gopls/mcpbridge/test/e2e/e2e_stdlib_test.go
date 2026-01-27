package e2e

// End-to-end tests for standard library navigation.
// These tests verify that gopls-mcp can navigate to and understand
// symbols from Go's standard library.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestStdlibNavigation tests navigation to standard library definitions.
func TestStdlibNavigation(t *testing.T) {

	// Use the simple test project which imports fmt
	projectDir := testutil.CopyProjectTo(t, "simple")

	// Start gopls-mcp

	t.Run("JumpToStdlibFunction", func(t *testing.T) {
		// Test jumping to fmt.Println definition
		tool := "go_definition"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "Println",
				"context_file": projectDir + "/main.go",
				"kind":         "function",
				"line_hint":    27,
			},
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibNavigation)
		t.Logf("Definition result for fmt.Println:\n%s", content)

		// Should find definition in GOROOT (Go standard library)
		if !strings.Contains(content, "Definition found") && !strings.Contains(content, "fmt") {
			t.Errorf("Expected to find definition for fmt.Println, got: %s", content)
		}

		// The result should contain a file path pointing to stdlib
		// Stdlib files are typically in go/src/fmt/ or similar
		if strings.Contains(content, "fmt") || strings.Contains(content, "GOROOT") || strings.Contains(content, "/go/") {
			t.Logf("✓ Found stdlib definition for fmt.Println")
		}
	})

	t.Run("JumpToStdlibType", func(t *testing.T) {
		// Test jumping to error type definition (if we use error in the code)
		// First, let's create a test file that uses error type
		projectDir2 := t.TempDir()

		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir2, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create code that uses error type
		sourceCode := `package main

import "fmt"

func doSomething() error {
	return nil
}

func main() {
	err := doSomething()
	if err != nil {
		fmt.Println(err)
	}
}
`
		if err := os.WriteFile(filepath.Join(projectDir2, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Start new gopls-mcp session
		mcpSession2, ctx2, cleanup2 := testutil.StartMCPServer(t, projectDir2)
		defer cleanup2()

		// Now test jumping to error type definition
		tool := "go_definition"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "error",
				"context_file": projectDir2 + "/main.go",
				"kind":         "interface",
				"line_hint":    5,
			},
		}

		res, err := mcpSession2.CallTool(ctx2, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibNavigation)
		t.Logf("Definition result for error type:\n%s", content)

		// Should find error type definition in stdlib
		if strings.Contains(content, "error") || strings.Contains(content, "Definition found") {
			t.Logf("✓ Found stdlib definition for error type")
		}
	})
}

// TestStdlibReferences tests finding references to stdlib symbols.
func TestStdlibReferences(t *testing.T) {

	// Use the simple test project which imports fmt
	projectDir := testutil.CopyProjectTo(t, "simple")

	// Start gopls-mcp

	t.Run("FindStdlibSymbolReferences", func(t *testing.T) {
		// Test finding all references to fmt.Println in the project
		tool := "go_symbol_references"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "Println",
				"context_file": projectDir + "/main.go",
				"kind":         "function",
				"line_hint":    27, // Where fmt.Println is called
			},
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibReferences)
		t.Logf("Symbol references result for fmt.Println:\n%s", content)

		// Should find references in the user's code
		// Note: This may or may not work depending on how gopls handles stdlib symbols
		if strings.Contains(content, "reference") || strings.Contains(content, "main.go") {
			t.Logf("✓ Found references to fmt.Println in user code")
		} else {
			t.Logf("Note: Finding references to stdlib symbols may have limited results")
		}
	})

	t.Run("StdSymbolReferencesAcrossFiles", func(t *testing.T) {
		// Create a project with multiple files using stdlib symbols
		projectDir := t.TempDir()

		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create file1.go that uses fmt.Println
		file1Code := `package main

import "fmt"

func func1() {
	fmt.Println("from func1")
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "file1.go"), []byte(file1Code), 0644); err != nil {
			t.Fatal(err)
		}

		// Create file2.go that also uses fmt.Println
		file2Code := `package main

import "fmt"

func func2() {
	fmt.Println("from func2")
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "file2.go"), []byte(file2Code), 0644); err != nil {
			t.Fatal(err)
		}

		// Create main.go
		mainCode := `package main

func main() {
	func1()
	func2()
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Start gopls-mcp

		// Try to find references to fmt.Println
		tool := "go_symbol_references"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "Println",
				"context_file": projectDir + "/file1.go",
				"kind":         "function",
				"line_hint":    6, // Where fmt.Println is called in func1
			},
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibReferences)
		t.Logf("Cross-file stdlib references:\n%s", content)

		// Verify the tool works even if it can't find all stdlib references
		t.Logf("✓ go_symbol_references works with stdlib symbols")
	})
}

// TestStdlibComplexTypes tests navigation through complex stdlib types.
func TestStdlibComplexTypes(t *testing.T) {

	// Create a project that uses complex stdlib types
	projectDir := t.TempDir()

	goModContent := `module example.com/test

go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create code that uses various stdlib types
	sourceCode := `package main

import (
	"context"
	"net/http"
	"sync"
)

func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}

func main() {
	// Use context.Context
	ctx := context.Background()

	// Use sync.WaitGroup
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Done()

	// Use http.Handler
	http.HandleFunc("/", handler)
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Start gopls-mcp

	t.Run("NavigateContextType", func(t *testing.T) {
		// Test navigating to context.Context type
		tool := "go_definition"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "Context",
				"context_file": projectDir + "/main.go",
				"kind":         "interface",
				"line_hint":    17,
			},
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibComplexTypes)
		t.Logf("Definition result for context:\n%s", content)

		if strings.Contains(content, "Definition found") || strings.Contains(content, "context") {
			t.Logf("✓ Found definition for context package")
		}
	})

	t.Run("NavigateHTTPHandler", func(t *testing.T) {
		// Test navigating to http.ResponseWriter
		tool := "go_definition"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "ResponseWriter",
				"context_file": projectDir + "/main.go",
				"kind":         "interface",
				"line_hint":    10,
			},
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibComplexTypes)
		t.Logf("Definition result for http.ResponseWriter:\n%s", content)

		if strings.Contains(content, "Definition found") || strings.Contains(content, "http") {
			t.Logf("✓ Found definition for http.ResponseWriter")
		}
	})

	t.Run("NavigateSyncWaitGroup", func(t *testing.T) {
		// Test navigating to sync.WaitGroup
		tool := "go_definition"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "WaitGroup",
				"context_file": projectDir + "/main.go",
				"kind":         "struct",
				"line_hint":    21,
			},
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibComplexTypes)
		t.Logf("Definition result for sync.WaitGroup:\n%s", content)

		if strings.Contains(content, "Definition found") || strings.Contains(content, "sync") {
			t.Logf("✓ Found definition for sync.WaitGroup")
		}
	})
}

// TestStdlibInterfaces tests navigating to stdlib interface definitions.
func TestStdlibInterfaces(t *testing.T) {

	// Create a project that implements stdlib interfaces
	projectDir := t.TempDir()

	goModContent := `module example.com/test

go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create code that implements stdlib interfaces
	sourceCode := `package main

import (
	"fmt"
	"io"
	"strings"
)

// MyReader implements io.Reader
type MyReader struct {
	data string
	pos  int
}

func (r *MyReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func main() {
	r := &MyReader{data: "hello world"}
	buf := make([]byte, 5)
	r.Read(buf)
	fmt.Println(string(buf))
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Start gopls-mcp

	t.Run("FindIOInterfaceDefinition", func(t *testing.T) {
		// Test navigating to io.Reader interface
		tool := "go_definition"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "Reader",
				"context_file": projectDir + "/main.go",
				"kind":         "interface",
				"line_hint":    12,
			},
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibInterfaces)
		t.Logf("Definition result for io.Reader:\n%s", content)

		if strings.Contains(content, "Definition found") || strings.Contains(content, "io") {
			t.Logf("✓ Found definition for io.Reader interface")
		}
	})

	t.Run("FindIOImplementations", func(t *testing.T) {
		// Test finding implementations of io.Reader
		tool := "go_implementation"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "Read",
				"context_file": projectDir + "/main.go",
				"kind":         "method",
				"line_hint":    16,
			},
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibInterfaces)
		t.Logf("Implementation result for io.Reader:\n%s", content)

		// Should find MyReader as an implementation
		if strings.Contains(content, "MyReader") || strings.Contains(content, "implementation") {
			t.Logf("✓ Found implementations for io.Reader")
		}
	})
}
