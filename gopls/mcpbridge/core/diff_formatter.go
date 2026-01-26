package core

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/gopls/internal/cache"
	"golang.org/x/tools/gopls/internal/protocol"
	"golang.org/x/tools/internal/diff"
)

// formatRenameChanges converts the list of DocumentChange to a unified diff and writes them to the specified buffer.
// Adapted from gopls/internal/mcp/rename_symbol.go and file_diagnostics.go
//
// DRY RUN: This only formats proposed changes - no files are modified.
func formatRenameChanges(ctx context.Context, snapshot *cache.Snapshot, changes []protocol.DocumentChange) (string, error) {
	diffText, err := toUnifiedDiff(ctx, snapshot, changes)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "=== DRY RUN: Symbol Rename Preview ===\n")
	fmt.Fprintf(&builder, "The following changes would be made to rename the symbol:\n")
	fmt.Fprintf(&builder, "NO FILES HAVE BEEN MODIFIED - this is a preview only.\n\n")
	fmt.Fprintf(&builder, "%s\n", diffText)
	return builder.String(), nil
}

// toUnifiedDiff converts a list of DocumentChange operations into a unified diff format.
// Adapted from gopls/internal/mcp/file_diagnostics.go
func toUnifiedDiff(ctx context.Context, snapshot *cache.Snapshot, changes []protocol.DocumentChange) (string, error) {
	var res strings.Builder
	for _, change := range changes {
		switch {
		case change.CreateFile != nil:
			res.WriteString(diff.Unified("/dev/null", change.CreateFile.URI.Path(), "", ""))
		case change.DeleteFile != nil:
			fh, err := snapshot.ReadFile(ctx, change.DeleteFile.URI)
			if err != nil {
				return "", err
			}
			content, err := fh.Content()
			if err != nil {
				return "", err
			}
			res.WriteString(diff.Unified(change.DeleteFile.URI.Path(), "/dev/null", string(content), ""))
		case change.RenameFile != nil:
			fh, err := snapshot.ReadFile(ctx, change.RenameFile.OldURI)
			if err != nil {
				return "", err
			}
			content, err := fh.Content()
			if err != nil {
				return "", err
			}
			res.WriteString(diff.Unified(filepath.ToSlash(change.RenameFile.OldURI.Path()), filepath.ToSlash(change.RenameFile.NewURI.Path()), string(content), string(content)))
		case change.TextDocumentEdit != nil:
			// Assumes gopls never return AnnotatedTextEdit.
			sorted := protocol.AsTextEdits(change.TextDocumentEdit.Edits)

			// As stated by the LSP, text edits ranges must never overlap.
			// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textEditArray
			slices.SortFunc(sorted, func(a, b protocol.TextEdit) int {
				if a.Range.Start.Line != b.Range.Start.Line {
					return int(a.Range.Start.Line) - int(b.Range.Start.Line)
				}
				return int(a.Range.Start.Character) - int(b.Range.Start.Character)
			})

			fh, err := snapshot.ReadFile(ctx, change.TextDocumentEdit.TextDocument.URI)
			if err != nil {
				return "", err
			}
			content, err := fh.Content()
			if err != nil {
				return "", err
			}

			var newSrc bytes.Buffer
			{
				mapper := protocol.NewMapper(fh.URI(), content)

				start := 0
				for _, edit := range sorted {
					l, r, err := mapper.RangeOffsets(edit.Range)
					if err != nil {
						return "", err
					}

					newSrc.Write(content[start:l])
					newSrc.WriteString(edit.NewText)

					start = r
				}
				newSrc.Write(content[start:])
			}

			res.WriteString(diff.Unified(filepath.ToSlash(fh.URI().Path()), filepath.ToSlash(fh.URI().Path()), string(content), newSrc.String()))
		default:
			continue // this shouldn't happen
		}
		res.WriteString("\n")
	}
	return res.String(), nil
}
