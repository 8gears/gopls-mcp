//go:build ignore

package main

import (
	"fmt"
	"os"

	"golang.org/x/tools/gopls/mcpbridge/core"
)

func main() {
	referencePath := "reference.md"

	// Create file for writing
	f, err := os.Create(referencePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating reference.md: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Generate and write the complete reference (all logic in server.go)
	if err := core.GenerateReference(f); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating reference: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("reference.md updated successfully")
}
