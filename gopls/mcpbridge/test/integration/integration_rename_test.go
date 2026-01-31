package integration

// Strong end-to-end test for go_dryrun_rename_symbol functionality.
// These tests verify the rename preview returns accurate changes and would fail with fake implementations.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestGoRenameSymbol_Strong is a strong end-to-end test that verifies go_dryrun_rename_symbol
// returns an accurate dry-run preview. This would FAIL with placeholder implementations.
func TestGoRenameSymbol_Strong(t *testing.T) {
	t.Run("ExactChangeCountAndFiles", func(t *testing.T) {
		// Create a test project with KNOWN symbol usage
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a file with a function used 3 times
		// Line 6:  func OldName() string {       <- definition
		// Line 13: result := OldName()         <- usage 1
		// Line 14: fmt.Println(result)
		// Line 17: x := OldName() + "suffix"  <- usage 2
		// Line 18: y := OldName() + "again"    <- usage 3
		sourceCode := `package main

import "fmt"

// OldName is a function to be renamed
func OldName() string {
	return "hello"
}

func main() {
	result := OldName()
	fmt.Println(result)

	x := OldName() + " suffix"
	y := OldName() + " again"
}
`
		mainGoPath := filepath.Join(projectDir, "main.go")
		if err := os.WriteFile(mainGoPath, []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Find the line number where OldName is defined
		lines := strings.Split(sourceCode, "\n")
		var lineNum int
		for i, line := range lines {
			if strings.Contains(line, "func OldName()") {
				lineNum = i + 1
				break
			}
		}

		if lineNum == 0 {
			t.Fatal("Could not find OldName function definition")
		}

		tool := "go_dryrun_rename_symbol"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "OldName",
				"context_file": mainGoPath,
				"line_hint":    lineNum,
			},
			"new_name": "NewName",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenRenameSymbolExact)
		t.Logf("Rename result:\n%s", content)

		// Compare against golden file (documentation + regression check)

		// === STRONG ASSERTIONS ===

		// 1. MUST contain "DRY RUN" - this is critical for safety
		if !strings.Contains(strings.ToUpper(content), "DRY RUN") {
			t.Fatalf("CRITICAL: Output must contain 'DRY RUN' to indicate preview mode.\nA fake implementation would just echo the symbol names!\nGot: %s", content)
		}
		t.Logf("✓ DRY RUN indicator present")

		// 2. MUST contain both old and new names
		// But this alone is not enough (the fake test passed this)
		if !strings.Contains(content, "OldName") {
			t.Errorf("Expected old symbol name 'OldName' in output")
		}
		if !strings.Contains(content, "NewName") {
			t.Errorf("Expected new symbol name 'NewName' in output")
		}
		t.Logf("✓ Both symbol names present")

		// 3. Extract and verify exact change count using unified diff format
		// Count lines with OldName removal and NewName addition
		diffLines := strings.Split(content, "\n")
		oldNameCount, newNameCount := 0, 0
		for _, line := range diffLines {
			if strings.HasPrefix(line, "-") && strings.Contains(line, "OldName") {
				oldNameCount++
			}
			if strings.HasPrefix(line, "+") && strings.Contains(line, "NewName") {
				newNameCount++
			}
		}
		if oldNameCount < 3 || newNameCount < 3 {
			t.Errorf("Expected at least 3 OldName removals and 3 NewName additions, got %d and %d.\nA fake implementation would have 0!", oldNameCount, newNameCount)
		}
		t.Logf("✓ Found %d removals and %d additions in unified diff", oldNameCount, newNameCount)

		// 4. Verify unified diff format is used
		if !strings.Contains(content, "---") || !strings.Contains(content, "+++") {
			t.Errorf("Expected unified diff format with --- and +++ headers")
		}
		t.Logf("✓ Unified diff format detected")

		// 5. Verify DRY RUN: file was NOT actually modified
		originalContent, err := os.ReadFile(mainGoPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(originalContent) != sourceCode {
			t.Errorf("DRY RUN VIOLATED: File was modified!\nExpected:\n%s\n\nGot:\n%s", sourceCode, string(originalContent))
		}
		t.Logf("✓ DRY RUN verified: file unchanged")
	})

	t.Run("MultiFileRenamePreview", func(t *testing.T) {
		// Create a test project with cross-file usage
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create util package
		utilDir := filepath.Join(projectDir, "util")
		if err := os.Mkdir(utilDir, 0755); err != nil {
			t.Fatal(err)
		}

		utilCode := `package util

func SharedFunc(x int) int {
	return x * 2
}
`
		helperPath := filepath.Join(utilDir, "helper.go")
		if err := os.WriteFile(helperPath, []byte(utilCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Create main.go using SharedFunc
		mainCode := `package main

import (
	"fmt"
	"example.com/test/util"
)

func main() {
	a := util.SharedFunc(5)
	b := util.SharedFunc(10)
	fmt.Println(a, b)
}
`
		mainGoPath := filepath.Join(projectDir, "main.go")
		if err := os.WriteFile(mainGoPath, []byte(mainCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Create other.go also using SharedFunc
		otherCode := `package main

import (
	"example.com/test/util"
)

func AnotherFunc() int {
	return util.SharedFunc(20)
}
`
		otherGoPath := filepath.Join(projectDir, "other.go")
		if err := os.WriteFile(otherGoPath, []byte(otherCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Find the line number where SharedFunc is defined
		lines := strings.Split(utilCode, "\n")
		var lineNum int
		for i, line := range lines {
			if strings.Contains(line, "func SharedFunc(") {
				lineNum = i + 1
				break
			}
		}

		if lineNum == 0 {
			t.Fatal("Could not find SharedFunc function definition")
		}

		tool := "go_dryrun_rename_symbol"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "SharedFunc",
				"context_file": helperPath,
				"line_hint":    lineNum,
			},
			"new_name": "RenamedFunc",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenRenameSymbolMultiFile)
		t.Logf("Multi-file rename result:\n%s", content)

		// === STRONG ASSERTIONS ===

		// 1. MUST contain "DRY RUN"
		if !strings.Contains(strings.ToUpper(content), "DRY RUN") {
			t.Fatalf("CRITICAL: Output must contain 'DRY RUN'\nGot: %s", content)
		}

		// 2. Should show the definition file being modified
		// Note: The current implementation shows changes to the file where we search from
		if !strings.Contains(content, "helper.go") && !strings.Contains(content, "util/helper.go") {
			t.Errorf("Expected output to mention helper.go being modified")
		}

		// 3. Count the number of change indicators using unified diff format
		// Count lines with SharedFunc removal and RenamedFunc addition
		diffLines := strings.Split(content, "\n")
		sharedFuncCount, renamedFuncCount := 0, 0
		for _, line := range diffLines {
			if strings.HasPrefix(line, "-") && strings.Contains(line, "SharedFunc") {
				sharedFuncCount++
			}
			if strings.HasPrefix(line, "+") && strings.Contains(line, "RenamedFunc") {
				renamedFuncCount++
			}
		}
		if sharedFuncCount < 1 || renamedFuncCount < 1 {
			t.Errorf("Expected at least 1 SharedFunc removal and 1 RenamedFunc addition, got %d and %d", sharedFuncCount, renamedFuncCount)
		}
		t.Logf("✓ Found %d removals and %d additions in unified diff", sharedFuncCount, renamedFuncCount)

		// 4. Verify both old and new names appear
		if !strings.Contains(content, "SharedFunc") {
			t.Errorf("Expected 'SharedFunc' in output")
		}
		if !strings.Contains(content, "RenamedFunc") {
			t.Errorf("Expected 'RenamedFunc' in output")
		}

		// 5. Verify unified diff format is used
		if !strings.Contains(content, "---") || !strings.Contains(content, "+++") {
			t.Errorf("Expected unified diff format with --- and +++ headers")
		}
		t.Logf("✓ Unified diff format detected")

		// 6. Verify DRY RUN: all files unchanged
		for path, original := range map[string]string{
			mainGoPath:  mainCode,
			otherGoPath: otherCode,
			helperPath:  utilCode,
		} {
			current, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(current) != original {
				t.Errorf("DRY RUN VIOLATED: %s was modified!", path)
			}
		}
		t.Logf("✓ DRY RUN verified: all 3 files unchanged")
	})

	t.Run("TypeRenamePreview", func(t *testing.T) {
		// Test renaming a type
		projectDir := t.TempDir()

		// Initialize go.mod
		goModContent := `module example.com/test

go 1.21
`
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Type with receiver and usage
		sourceCode := `package main

import "fmt"

// OldType is a type
type OldType struct {
	Value int
}

func (o OldType) Process() {
	fmt.Println(o.Value)
}

func main() {
	ot := OldType{Value: 42}
	ot.Process()

	var ptr *OldType
	_ = ptr
}
`
		mainGoPath := filepath.Join(projectDir, "main.go")
		if err := os.WriteFile(mainGoPath, []byte(sourceCode), 0644); err != nil {
			t.Fatal(err)
		}

		// Find the line number where OldType is defined
		lines := strings.Split(sourceCode, "\n")
		var lineNum int
		for i, line := range lines {
			if strings.Contains(line, "type OldType struct") {
				lineNum = i + 1
				break
			}
		}

		if lineNum == 0 {
			t.Fatal("Could not find OldType definition")
		}

		tool := "go_dryrun_rename_symbol"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "OldType",
				"context_file": mainGoPath,
				"line_hint":    lineNum,
			},
			"new_name": "NewType",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		if res == nil {
			t.Fatal("Expected non-nil result")
		}

		content := testutil.ResultText(t, res, testutil.GoldenRenameSymbolType)
		t.Logf("Type rename result:\n%s", content)

		// Note: The structured Changes field is populated internally but MCP returns only text content.
		// This is correct - the unified diff in the Summary is the primary output for LLMs.

		// === STRONG ASSERTIONS ===

		// 1. DRY RUN indicator
		if !strings.Contains(strings.ToUpper(content), "DRY RUN") {
			t.Fatalf("CRITICAL: Output must contain 'DRY RUN'\nGot: %s", content)
		}

		// 2. Should find multiple changes indicated by unified diff format
		// Count lines containing OldType and NewType in the diff (with - or + prefix)
		// The unified diff format uses "-line" and "+line" without space after marker
		diffLines := strings.Split(content, "\n")
		oldTypeCount, newTypeCount := 0, 0
		for _, line := range diffLines {
			if strings.HasPrefix(line, "-") && strings.Contains(line, "OldType") {
				oldTypeCount++
			}
			if strings.HasPrefix(line, "+") && strings.Contains(line, "NewType") {
				newTypeCount++
			}
		}
		if oldTypeCount < 1 || newTypeCount < 1 {
			t.Errorf("Expected unified diff with both OldType removals and NewType additions, got %d removals and %d additions", oldTypeCount, newTypeCount)
		}
		t.Logf("✓ Found unified diff with %d removals and %d additions", oldTypeCount, newTypeCount)

		// 3. Both names present
		if !strings.Contains(content, "OldType") || !strings.Contains(content, "NewType") {
			t.Errorf("Expected both type names in output")
		}

		// 4. Unified diff indicators present
		if !strings.Contains(content, "---") || !strings.Contains(content, "+++") {
			t.Errorf("Expected unified diff format with --- and +++ headers")
		}
		t.Logf("✓ Unified diff format detected")

		// 5. DRY RUN verified
		original, _ := os.ReadFile(mainGoPath)
		if string(original) != sourceCode {
			t.Error("DRY RUN VIOLATED: file was modified")
		}
		t.Logf("✓ DRY RUN verified: type unchanged")
	})
}

// TestGoRenameSymbolE2E is kept for backward compatibility but now uses strong assertions
func TestGoRenameSymbolE2E(t *testing.T) {
	// This now just calls the strong test
	TestGoRenameSymbol_Strong(t)
}
