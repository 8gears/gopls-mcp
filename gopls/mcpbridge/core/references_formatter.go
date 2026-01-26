package core

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/internal/cache"
	"golang.org/x/tools/gopls/internal/protocol"
)

// formatReferences formats symbol references into a human-readable output.
// Adapted from gopls/internal/mcp/references.go
func formatReferences(ctx context.Context, snapshot *cache.Snapshot, refs []protocol.Location) (*mcp.CallToolResult, error) {
	if len(refs) == 0 {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "No references found."}}}, nil
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "The object has %v reference(s). Their locations are listed below:\n", len(refs))
	for i, r := range refs {
		fmt.Fprintf(&builder, "\nReference %d\n", i+1)
		fmt.Fprintf(&builder, "Located in the file: %s\n", filepath.ToSlash(r.URI.Path()))
		refFh, err := snapshot.ReadFile(ctx, r.URI)
		// If for some reason there is an error reading the file content, we should still
		// return the references URIs.
		if err != nil {
			continue
		}
		content, err := refFh.Content()
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		var lineContent string
		if int(r.Range.Start.Line) < len(lines) {
			lineContent = strings.TrimLeftFunc(lines[r.Range.Start.Line], unicode.IsSpace)
		} else {
			continue
		}
		fmt.Fprintf(&builder, "Line %d: %s\n", r.Range.Start.Line+1, lineContent)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: builder.String()}}}, nil
}

// formatReferencesWithCount formats symbol references into a human-readable output
// with truncation metadata.
func formatReferencesWithCount(ctx context.Context, snapshot *cache.Snapshot, refs []protocol.Location, totalCount int, truncated bool, hint string) (string, error) {
	if len(refs) == 0 {
		return "No references found.", nil
	}

	var builder strings.Builder

	// Build header with truncation info
	if truncated {
		fmt.Fprintf(&builder, "The object has %v reference(s) (showing first %d):\n", totalCount, len(refs))
	} else {
		fmt.Fprintf(&builder, "The object has %v reference(s):\n", totalCount)
	}

	// Format each reference
	for i, r := range refs {
		fmt.Fprintf(&builder, "\nReference %d\n", i+1)
		fmt.Fprintf(&builder, "Located in the file: %s\n", filepath.ToSlash(r.URI.Path()))
		refFh, err := snapshot.ReadFile(ctx, r.URI)
		// If for some reason there is an error reading the file content, we should still
		// return the references URIs.
		if err != nil {
			continue
		}
		content, err := refFh.Content()
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		var lineContent string
		if int(r.Range.Start.Line) < len(lines) {
			lineContent = strings.TrimLeftFunc(lines[r.Range.Start.Line], unicode.IsSpace)
		} else {
			continue
		}
		fmt.Fprintf(&builder, "Line %d: %s\n", r.Range.Start.Line+1, lineContent)
	}

	// Add truncation hint if applicable
	if truncated && hint != "" {
		builder.WriteString("\n")
		builder.WriteString(hint)
	}

	return builder.String(), nil
}
