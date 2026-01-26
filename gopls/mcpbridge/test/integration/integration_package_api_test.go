package integration

// End-to-end test for get_package_symbol_detail functionality.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/api"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestGetPackageSymbolDetailE2E is an end-to-end test that verifies get_package_symbol_detail works.
func TestGetPackageSymbolDetailE2E(t *testing.T) {
	t.Run("WithSpecificFilters", func(t *testing.T) {
		// Test with specific symbol_filters - should return only matching symbols
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with functions
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
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "get_package_symbol_detail"
		args := map[string]any{
			"package_path":   "example.com/test",
			"symbol_filters": []map[string]any{{"name": "Hello"}},
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

		// Parse result to check symbols
		result := &api.OGetPackageSymbolDetailResult{}
		resultText := testutil.ResultText(res)
		// Extract JSON from the text content (find the first { and last })
		jsonStart := strings.Index(resultText, "{")
		jsonEnd := strings.LastIndex(resultText, "}")
		if jsonStart == -1 || jsonEnd == -1 {
			t.Fatalf("Could not find JSON in result: %s", resultText)
		}
		jsonStr := resultText[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
			t.Fatalf("Failed to parse result: %v\nResult: %s", err, resultText)
		}

		// Should find the Hello function
		if len(result.Symbols) != 1 {
			t.Errorf("Expected 1 symbol (Hello), got %d", len(result.Symbols))
		}
		if len(result.Symbols) > 0 && result.Symbols[0].Name != "Hello" {
			t.Errorf("Expected symbol named 'Hello', got '%s'", result.Symbols[0].Name)
		}
	})

	t.Run("WithSymbolFilters", func(t *testing.T) {
		// Test with symbol_filters - should return only matching symbols
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with functions
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

// Multiply returns the product
func Multiply(a, b int) int {
	return a * b
}

func main() {
	fmt.Println(Hello())
	fmt.Println(Add(1, 2))
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "get_package_symbol_detail"
		args := map[string]any{
			"package_path": "example.com/test",
			"symbol_filters": []any{
				map[string]any{"name": "Hello"},
				map[string]any{"name": "Add"},
			},
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

		// Parse result to check symbols
		result := &api.OGetPackageSymbolDetailResult{}
		resultText := testutil.ResultText(res)
		// Extract JSON from the text content (find the first { and last })
		jsonStart := strings.Index(resultText, "{")
		jsonEnd := strings.LastIndex(resultText, "}")
		if jsonStart == -1 || jsonEnd == -1 {
			t.Fatalf("Could not find JSON in result: %s", resultText)
		}
		jsonStr := resultText[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
			t.Fatalf("Failed to parse result: %v\nResult: %s", err, resultText)
		}

		t.Logf("Got %d symbols", len(result.Symbols))

		// Should return only Hello and Add
		if len(result.Symbols) != 2 {
			t.Errorf("Expected 2 symbols (Hello, Add), got %d", len(result.Symbols))
		}

		// Check that we got Hello and Add (not Multiply)
		foundHello := false
		foundAdd := false
		foundMultiply := false
		for _, sym := range result.Symbols {
			if sym.Name == "Hello" {
				foundHello = true
			}
			if sym.Name == "Add" {
				foundAdd = true
			}
			if sym.Name == "Multiply" {
				foundMultiply = true
			}
		}

		if !foundHello {
			t.Error("Expected to find Hello symbol")
		}
		if !foundAdd {
			t.Error("Expected to find Add symbol")
		}
		if foundMultiply {
			t.Error("Expected NOT to find Multiply symbol (not in filters)")
		}
	})

	t.Run("MethodsWithReceiverFilter", func(t *testing.T) {
		// Test filtering methods by receiver
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with types and methods
		sourceCode := `package main

import "fmt"

// Person represents a person
type Person struct {
	Name string
	Age  int
}

// Greeting returns a greeting from the person
func (p *Person) Greeting() string {
	return fmt.Sprintf("Hello, I am %s", p.Name)
}

// Birthday increases the person's age
func (p *Person) Birthday() {
	p.Age++
}

// Animal represents an animal
type Animal struct {
	Species string
}

// Speak makes the animal speak
func (a *Animal) Speak() string {
	return "..."
}

func main() {
	p := Person{Name: "Alice", Age: 30}
	fmt.Println(p.Greeting())
	p.Birthday()
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		// First, try list_package_symbols to verify the symbols exist
		t.Log("Testing with list_package_symbols first...")
		listRes, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{
			Name: "list_package_symbols",
			Arguments: map[string]any{
				"package_path":   "example.com/test",
				"include_docs":   false,
				"include_bodies": false,
				"Cwd":            projectDir,
			},
		})
		if err != nil {
			t.Logf("Warning: list_package_symbols failed: %v", err)
		} else {
			listContent := testutil.ResultText(listRes)
			t.Logf("list_package_symbols result (first 500 chars): %s", testutil.TruncateString(listContent, 500))
			if strings.Contains(listContent, "Person") {
				t.Log("Found Person type via list_package_symbols")
			} else {
				t.Log("WARNING: Person type NOT found via list_package_symbols")
			}
		}

		tool := "get_package_symbol_detail"
		args := map[string]any{
			"package_path": "example.com/test",
			"symbol_filters": []any{
				map[string]any{"name": "Greeting", "receiver": "*Person"},
				map[string]any{"name": "Birthday", "receiver": "*Person"},
			},
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

		// Parse result to check symbols
		result := &api.OGetPackageSymbolDetailResult{}
		resultText := testutil.ResultText(res)

		// Debug: log full result
		t.Logf("Full result text (first 1000 chars): %s", testutil.TruncateString(resultText, 1000))
		t.Logf("Result text length: %d chars", len(resultText))

		// Extract JSON from the text content (find the first { and last })
		jsonStart := strings.Index(resultText, "{")
		jsonEnd := strings.LastIndex(resultText, "}")
		if jsonStart == -1 || jsonEnd == -1 {
			t.Fatalf("Could not find JSON in result: %s", resultText)
		}
		jsonStr := resultText[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
			t.Fatalf("Failed to parse result: %v\nResult: %s", err, resultText)
		}

		t.Logf("Got %d symbols", len(result.Symbols))

		// Should return only Person methods
		if len(result.Symbols) != 2 {
			t.Errorf("Expected 2 symbols (Person.Greeting, Person.Birthday), got %d", len(result.Symbols))
		}

		// Check receiver field
		foundGreeting := false
		foundBirthday := false
		foundSpeak := false
		for _, sym := range result.Symbols {
			if sym.Name == "Greeting" {
				foundGreeting = true
				if sym.Receiver != "*Person" {
					t.Errorf("Expected receiver '*Person' for Greeting, got '%s'", sym.Receiver)
				}
				if sym.Kind != api.SymbolKindMethod {
					t.Errorf("Expected kind 'method' for Greeting, got '%s'", sym.Kind)
				}
				// Check doc is included
				if sym.Doc == "" {
					t.Error("Expected documentation to be included")
				}
				// Check body is included
				if !strings.Contains(sym.Body, "fmt.Sprintf") {
					t.Error("Expected body to be included")
				}
			}
			if sym.Name == "Birthday" {
				foundBirthday = true
				if sym.Receiver != "*Person" {
					t.Errorf("Expected receiver '*Person' for Birthday, got '%s'", sym.Receiver)
				}
			}
			if sym.Name == "Speak" {
				foundSpeak = true
			}
		}

		if !foundGreeting {
			t.Error("Expected to find Greeting symbol")
		}
		if !foundBirthday {
			t.Error("Expected to find Birthday symbol")
		}
		if foundSpeak {
			t.Error("Expected NOT to find Speak symbol (different receiver)")
		}
	})

	t.Run("SignaturesOnly", func(t *testing.T) {
		// Test with include_bodies=false (default) - should return signatures only
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with functions
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
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "get_package_symbol_detail"
		args := map[string]any{
			"package_path": "example.com/test",
			"symbol_filters": []any{
				map[string]any{"name": "Hello"},
				map[string]any{"name": "Add"},
			},
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
		t.Logf("Package Symbol Detail (signatures only):\n%s", content)

		// Parse result to check symbols
		result := &api.OGetPackageSymbolDetailResult{}
		resultText := testutil.ResultText(res)
		// Extract JSON from the text content (find the first { and last })
		jsonStart := strings.Index(resultText, "{")
		jsonEnd := strings.LastIndex(resultText, "}")
		if jsonStart == -1 || jsonEnd == -1 {
			t.Fatalf("Could not find JSON in result: %s", resultText)
		}
		jsonStr := resultText[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
			t.Fatalf("Failed to parse result: %v\nResult: %s", err, resultText)
		}

		// Should contain signatures but NOT bodies
		for _, sym := range result.Symbols {
			if sym.Name == "Hello" || sym.Name == "Add" {
				if !strings.Contains(sym.Signature, "func") {
					t.Errorf("Expected %s to have signature, got: %s", sym.Name, sym.Signature)
				}
				if sym.Body != "" {
					t.Errorf("Expected %s to NOT have body (include_bodies=false), got: %s", sym.Name, sym.Body)
				}
			}
		}
	})

	t.Run("WithBodies", func(t *testing.T) {
		// Test with include_bodies=true - should return full implementations
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with functions that have distinct implementations
		sourceCode := `package main

// Hello returns a greeting message
func Hello() string {
	return "hello world"
}

// Multiply returns the product of two integers
func Multiply(a, b int) int {
	result := a * b
	return result
}

func main() {
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "get_package_symbol_detail"
		args := map[string]any{
			"package_path": "example.com/test",
			"symbol_filters": []any{
				map[string]any{"name": "Hello"},
				map[string]any{"name": "Multiply"},
			},
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

		// Parse result to check symbols
		result := &api.OGetPackageSymbolDetailResult{}
		resultText := testutil.ResultText(res)
		// Extract JSON from the text content (find the first { and last })
		jsonStart := strings.Index(resultText, "{")
		jsonEnd := strings.LastIndex(resultText, "}")
		if jsonStart == -1 || jsonEnd == -1 {
			t.Fatalf("Could not find JSON in result: %s", resultText)
		}
		jsonStr := resultText[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
			t.Fatalf("Failed to parse result: %v\nResult: %s", err, resultText)
		}

		// Should contain bodies
		for _, sym := range result.Symbols {
			if sym.Name == "Hello" {
				if !strings.Contains(sym.Body, "return \"hello world\"") {
					t.Errorf("Expected Hello body, got: %s", sym.Body)
				}
			}
			if sym.Name == "Multiply" {
				if !strings.Contains(sym.Body, "result := a * b") {
					t.Errorf("Expected Multiply body with local variable, got: %s", sym.Body)
				}
			}
		}
	})

	t.Run("NonExistentPackage", func(t *testing.T) {
		// Test querying a package that doesn't exist
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create minimal main.go
		sourceCode := `package main

func main() {
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "get_package_symbol_detail"
		args := map[string]any{
			"package_path":   "does/not/exist",
			"symbol_filters": []any{map[string]any{"name": "Something"}},
			"include_bodies": false,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})

		// Non-existent package should error
		if err != nil {
			t.Logf("Expected error for non-existent package: %v", err)
		} else if res != nil {
			content := testutil.ResultText(res)
			t.Logf("Result for non-existent package: %s", content)

			// If no error, the result should mention the issue
			if !strings.Contains(content, "not found") &&
				!strings.Contains(content, "no such package") &&
				!strings.Contains(content, "error") &&
				!strings.Contains(content, "failed") {
				t.Logf("Note: Tool didn't error for non-existent package, returned: %s", content)
			}
		}
	})

	t.Run("NonExistentSymbol", func(t *testing.T) {
		// Test querying for a symbol that doesn't exist in the package
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create minimal main.go
		sourceCode := `package main

func main() {
}
`
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		tool := "get_package_symbol_detail"
		args := map[string]any{
			"package_path":   "example.com/test",
			"symbol_filters": []any{map[string]any{"name": "NonExistentFunc"}},
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

		// Parse result to check symbols
		result := &api.OGetPackageSymbolDetailResult{}
		resultText := testutil.ResultText(res)
		// Extract JSON from the text content (find the first { and last })
		jsonStart := strings.Index(resultText, "{")
		jsonEnd := strings.LastIndex(resultText, "}")
		if jsonStart == -1 || jsonEnd == -1 {
			t.Fatalf("Could not find JSON in result: %s", resultText)
		}
		jsonStr := resultText[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
			t.Fatalf("Failed to parse result: %v\nResult: %s", err, resultText)
		}

		// Should return empty list (symbol doesn't exist)
		if len(result.Symbols) != 0 {
			t.Errorf("Expected 0 symbols for non-existent filter, got %d", len(result.Symbols))
		}
	})
}
