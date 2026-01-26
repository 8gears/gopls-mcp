package core

import (
	"bytes"
	"os"
	"testing"
)

// TestGenerate ensures that the generated reference.md is up to date.
// It generates the complete file content to a buffer and compares it with the actual file.
func TestGenerate(t *testing.T) {
	// Read the actual reference.md file
	referencePath := "reference.md"
	actualContent, err := os.ReadFile(referencePath)
	if err != nil {
		t.Fatalf("Failed to read reference.md: %v", err)
	}

	// Generate the complete file content to a buffer
	var buf bytes.Buffer
	if err := GenerateReference(&buf); err != nil {
		t.Fatalf("GenerateReference failed: %v", err)
	}

	generatedContent := buf.Bytes()

	// Compare generated content with actual file content
	if string(generatedContent) != string(actualContent) {
		t.Errorf("reference.md is out of sync with generated content.\n"+
			"Run 'go generate ./gopls/mcpbridge/core' to update.\n\n"+
			"Generated length: %d, Actual length: %d", len(generatedContent), len(actualContent))
	}

	t.Logf("reference.md is up to date with %d tools", len(tools))
}
