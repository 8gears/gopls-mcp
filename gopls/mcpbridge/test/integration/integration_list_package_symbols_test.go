package integration

// End-to-end tests for list_package_symbols functionality.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestListPackageSymbolsE2E_Comprehensive is a comprehensive end-to-end test for list_package_symbols.
// Note: Basic tests exist in e2e_modules_test.go - this file covers edge cases.
func TestListPackageSymbolsE2E_Comprehensive(t *testing.T) {
	t.Run("BasicSymbols", func(t *testing.T) {
		// Create a test project
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with various symbols
		sourceCode := `package main

import "fmt"

// Counter counts things
type Counter struct {
	count int
}

// Increment increments the counter
func (c *Counter) Increment() {
	c.count++
}

// GetCount returns the current count
func (c *Counter) GetCount() int {
	return c.count
}

// Add adds two numbers
func Add(a, b int) int {
	return a + b
}

const (
	// MaxCount is the maximum count
	MaxCount = 100
)

var (
	// globalCounter is a global counter
	globalCounter int
)

func main() {
	fmt.Println("hello")
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Start gopls-mcp

		tool := "list_package_symbols"
		args := map[string]any{
			"package_path":   "example.com/test",
			"include_docs":   false,
			"include_bodies": false,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)
		t.Logf("Package symbols:\n%s", content)

		// Should find the type
		if !strings.Contains(content, "Counter") {
			t.Errorf("Expected to find Counter type, got: %s", content)
		}

		// Should find functions
		if !strings.Contains(content, "Add") {
			t.Errorf("Expected to find Add function, got: %s", content)
		}

		if !strings.Contains(content, "Increment") {
			t.Errorf("Expected to find Increment method, got: %s", content)
		}

		// Should find constants
		if !strings.Contains(content, "MaxCount") {
			t.Errorf("Expected to find MaxCount constant, got: %s", content)
		}
	})

	t.Run("WithDocumentation", func(t *testing.T) {
		// Test with include_docs=true
		projectDir := t.TempDir()

		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create documented symbols
		sourceCode := `package main

// Greeting returns a friendly greeting
func Greeting(name string) string {
	return "Hello, " + name
}

// Farewell says goodbye
func Farewell(name string) string {
	return "Goodbye, " + name
}

func main() {
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "list_package_symbols"
		args := map[string]any{
			"package_path":   "example.com/test",
			"include_docs":   true,
			"include_bodies": false,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)
		t.Logf("Package symbols with docs:\n%s", content)

		// Should contain documentation
		if !strings.Contains(content, "friendly greeting") {
			t.Errorf("Expected to find 'friendly greeting' in docs, got: %s", content)
		}

		if !strings.Contains(content, "goodbye") {
			t.Errorf("Expected to find 'goodbye' in docs, got: %s", content)
		}
	})

	t.Run("WithBodies", func(t *testing.T) {
		// Test with include_bodies=true
		projectDir := t.TempDir()

		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create functions with distinct implementations
		sourceCode := `package main

// Double returns twice the value
func Double(n int) int {
	return n * 2
}

// Triple returns three times the value
func Triple(n int) int {
	return n * 3
}

func main() {
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "list_package_symbols"
		args := map[string]any{
			"package_path":   "example.com/test",
			"include_docs":   false,
			"include_bodies": true,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)
		t.Logf("Package symbols with bodies:\n%s", content)

		// Should contain function bodies
		if !strings.Contains(content, "return n * 2") {
			t.Errorf("Expected to find Double body, got: %s", content)
		}

		if !strings.Contains(content, "return n * 3") {
			t.Errorf("Expected to find Triple body, got: %s", content)
		}
	})

	t.Run("EmptyPackage", func(t *testing.T) {
		// Test with an empty package
		projectDir := t.TempDir()

		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create empty package
		sourceCode := `package main

func main() {
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "list_package_symbols"
		args := map[string]any{
			"package_path":   "example.com/test",
			"include_docs":   false,
			"include_bodies": false,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)
		t.Logf("Empty package symbols:\n%s", content)

		// Should still return a result, even if empty
		if content == "" {
			t.Error("Expected some output even for empty package")
		}
	})

	t.Run("NonExistentPackage", func(t *testing.T) {
		// Test querying a package that doesn't exist
		projectDir := t.TempDir()

		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		sourceCode := `package main

func main() {
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "list_package_symbols"
		args := map[string]any{
			"package_path":   "does/not/exist",
			"include_docs":   false,
			"include_bodies": false,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})

		// Non-existent package should error
		if err != nil {
			t.Logf("Expected error for non-existent package: %v", err)
		} else if res != nil {
			content := testutil.ResultText(res)
			t.Logf("Result for non-existent package: %s", content)

			// If no error, should mention the issue
			if !strings.Contains(content, "not found") &&
				!strings.Contains(content, "no such package") &&
				!strings.Contains(content, "error") &&
				!strings.Contains(content, "failed") {
				t.Logf("Warning: Tool didn't error for non-existent package")
			}
		}
	})

	t.Run("ComplexTypeHierarchy", func(t *testing.T) {
		// Test with interfaces, structs, and methods
		projectDir := t.TempDir()

		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create complex type hierarchy
		sourceCode := `package main

// Shape is an interface for geometric shapes
type Shape interface {
	Area() float64
	Perimeter() float64
}

// Circle is a circular shape
type Circle struct {
	radius float64
}

// Area returns the circle's area
func (c Circle) Area() float64 {
	return 3.14159 * c.radius * c.radius
}

// Perimeter returns the circle's perimeter
func (c Circle) Perimeter() float64 {
	return 2 * 3.14159 * c.radius
}

// Rectangle is a rectangular shape
type Rectangle struct {
	width, height float64
}

// Area returns the rectangle's area
func (r Rectangle) Area() float64 {
	return r.width * r.height
}

// Perimeter returns the rectangle's perimeter
func (r Rectangle) Perimeter() float64 {
	return 2 * (r.width + r.height)
}

// NewCircle creates a new circle
func NewCircle(radius float64) Circle {
	return Circle{radius: radius}
}

func main() {
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "list_package_symbols"
		args := map[string]any{
			"package_path":   "example.com/test",
			"include_docs":   true,
			"include_bodies": true,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(res)
		t.Logf("Complex type hierarchy:\n%s", content)

		// Should find interface
		if !strings.Contains(content, "Shape") {
			t.Errorf("Expected to find Shape interface, got: %s", content)
		}

		// Should find implementing types
		if !strings.Contains(content, "Circle") {
			t.Errorf("Expected to find Circle struct, got: %s", content)
		}

		if !strings.Contains(content, "Rectangle") {
			t.Errorf("Expected to find Rectangle struct, got: %s", content)
		}

		// Should find methods
		if !strings.Contains(content, "Area") {
			t.Errorf("Expected to find Area methods, got: %s", content)
		}

		if !strings.Contains(content, "Perimeter") {
			t.Errorf("Expected to find Perimeter methods, got: %s", content)
		}
	})
}
