package core

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/tools/gopls/internal/cache"
	"golang.org/x/tools/gopls/internal/cache/metadata"
	"golang.org/x/tools/gopls/internal/cache/parsego"
	"golang.org/x/tools/gopls/internal/golang"
	"golang.org/x/tools/gopls/internal/protocol"
	"golang.org/x/tools/internal/astutil"
)

// This file provides utility functions for truncating output to prevent
// excessive token usage. The truncation happens in the wrapper layer
// AFTER gopls has fetched the full results.

const (
	// TruncationIndicator is appended to truncated output.
	TruncationIndicator = "\n\n[TRUNCATED]"
)

// summarizePackageWithBodyLimit generates a summary of a Go package with optional
// function body truncation based on max_body_tokens.
// Copied from gopls/internal/mcp/context.go and enhanced with body token limiting.
func summarizePackageWithBodyLimit(ctx context.Context, snapshot *cache.Snapshot, mp *metadata.Package, includeBodies bool, maxBodyTokens int) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%q (package %s)\n", mp.PkgPath, mp.Name)
	for _, f := range mp.CompiledGoFiles {
		fmt.Fprintf(&buf, "%s:\n", f.Base())
		buf.WriteString("```go\n")
		if err := writeFileSummaryWithBodyLimit(ctx, snapshot, f, &buf, true, nil, includeBodies, maxBodyTokens); err != nil {
			return "" // ignore error
		}
		buf.WriteString("```\n\n")
	}
	return buf.String()
}

// writeFileSummaryWithBodyLimit writes the file summary to the string builder based on
// the input file URI, with optional function body truncation.
// Copied from gopls/internal/mcp/context.go and enhanced with body token limiting.
func writeFileSummaryWithBodyLimit(ctx context.Context, snapshot *cache.Snapshot, f protocol.DocumentURI, out *strings.Builder, onlyExported bool, declsToSummarize map[string]bool, includeBodies bool, maxBodyTokens int) error {
	fh, err := snapshot.ReadFile(ctx, f)
	if err != nil {
		return err
	}
	pgf, err := snapshot.ParseGo(ctx, fh, parsego.Full)
	if err != nil {
		return err
	}

	// If we're summarizing specific declarations, we don't need to copy the header.
	if declsToSummarize == nil {
		// Copy everything before the first non-import declaration:
		// package decl, imports decl(s), and all comments (excluding copyright).
		{
			endPos := pgf.File.FileEnd

		outerloop:
			for _, decl := range pgf.File.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
					if decl.Doc != nil {
						endPos = decl.Doc.Pos()
					} else {
						endPos = decl.Pos()
					}
					break outerloop
				case *ast.GenDecl:
					if decl.Tok == token.IMPORT {
						continue
					}
					if decl.Doc != nil {
						endPos = decl.Doc.Pos()
					} else {
						endPos = decl.Pos()
					}
					break outerloop
				}
			}

			startPos := pgf.File.FileStart
			if copyright := golang.CopyrightComment(pgf.File); copyright != nil {
				startPos = copyright.End()
			}

			text, err := pgf.PosText(startPos, endPos)
			if err != nil {
				return err
			}

			out.WriteString(strings.TrimSpace(string(text)))
			out.WriteString("\n\n")
		}
	}

	// Track total body tokens for truncation
	totalBodyTokens := 0

	// Write func decl and gen decl.
	for _, decl := range pgf.File.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if declsToSummarize != nil {
				if _, ok := declsToSummarize[decl.Name.Name]; !ok {
					continue
				}
			}
			if onlyExported {
				if !decl.Name.IsExported() {
					continue
				}

				if decl.Recv != nil && len(decl.Recv.List) > 0 {
					_, rname, _ := astutil.UnpackRecv(decl.Recv.List[0].Type)
					if !rname.IsExported() {
						continue
					}
				}
			}

			// Write doc comment and func signature/body.
			startPos := decl.Pos()
			if decl.Doc != nil {
				startPos = decl.Doc.Pos()
			}

			var endPos token.Pos
			if includeBodies {
				// Check if we should include this body or truncate it
				bodyText, err := pgf.PosText(decl.Type.Pos(), decl.End())
				if err == nil {
					bodyTokenEstimate := len(bodyText) / 4 // ~4 chars per token
					if maxBodyTokens > 0 && (totalBodyTokens+bodyTokenEstimate) > maxBodyTokens {
						// Skip this body due to token limit
						endPos = decl.Type.End()
						out.WriteString("[Function body omitted due to max_body_tokens limit]\n")
					} else {
						// Include the full body
						endPos = decl.End()
						totalBodyTokens += bodyTokenEstimate
					}
				} else {
					endPos = decl.Type.End()
				}
			} else {
				// Include only the function signature (up to Type.End())
				endPos = decl.Type.End()
			}

			text, err := pgf.PosText(startPos, endPos)
			if err != nil {
				return err
			}

			out.Write(text)
			out.WriteString("\n\n")

		case *ast.GenDecl:
			if decl.Tok == token.IMPORT {
				continue
			}

			// If we are summarizing specific decls, check if any of them are in this GenDecl.
			if declsToSummarize != nil {
				found := false
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						if _, ok := declsToSummarize[spec.Name.Name]; ok {
							found = true
						}
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							if _, ok := declsToSummarize[name.Name]; ok {
								found = true
							}
						}
					}
				}
				if !found {
					continue
				}
			}

			// Dump the entire GenDecl (exported or unexported)
			// including doc comment without any filtering to the output.
			if !onlyExported {
				startPos := decl.Pos()
				if decl.Doc != nil {
					startPos = decl.Doc.Pos()
				}
				text, err := pgf.PosText(startPos, decl.End())
				if err != nil {
					return err
				}

				out.Write(text)
				out.WriteString("\n")
				continue
			}

			// Write only the GenDecl with exported identifier to the output.
			var buf strings.Builder
			if decl.Doc != nil {
				text, err := pgf.NodeText(decl.Doc)
				if err != nil {
					return err
				}
				buf.Write(text)
				buf.WriteString("\n")
			}

			buf.WriteString(decl.Tok.String() + " ")
			if decl.Lparen.IsValid() {
				buf.WriteString("(\n")
			}

			var anyExported bool
			for _, spec := range decl.Specs {
				// Captures the full byte range of the spec, including
				// its associated doc comments and line comments.
				// This range also covers any floating comments as these
				// can be valuable for context. Like
				// ```
				// type foo struct { // floating comment.
				// 		// floating comment.
				// 	x int
				// }
				// ```
				var startPos, endPos token.Pos

				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if declsToSummarize != nil {
						if _, ok := declsToSummarize[spec.Name.Name]; !ok {
							continue
						}
					}
					if !spec.Name.IsExported() {
						continue
					}
					anyExported = true

					// Include preceding doc comment, if any.
					if spec.Doc == nil {
						startPos = spec.Pos()
					} else {
						startPos = spec.Doc.Pos()
					}

					// Include trailing line comment, if any.
					if spec.Comment == nil {
						endPos = spec.End()
					} else {
						endPos = spec.Comment.End()
					}

				case *ast.ValueSpec:
					if declsToSummarize != nil {
						found := false
						for _, name := range spec.Names {
							if _, ok := declsToSummarize[name.Name]; ok {
								found = true
							}
						}
						if !found {
							continue
						}
					}
					if !slices.ContainsFunc(spec.Names, (*ast.Ident).IsExported) {
						continue
					}
					anyExported = true

					if spec.Doc == nil {
						startPos = spec.Pos()
					} else {
						startPos = spec.Doc.Pos()
					}

					if spec.Comment == nil {
						endPos = spec.End()
					} else {
						endPos = spec.Comment.End()
					}
				}

				indent, err := pgf.Indentation(startPos)
				if err != nil {
					return err
				}

				buf.WriteString(indent)

				text, err := pgf.PosText(startPos, endPos)
				if err != nil {
					return err
				}

				buf.Write(text)
				buf.WriteString("\n")
			}

			if decl.Lparen.IsValid() {
				buf.WriteString(")\n")
			}

			// Only write the summary of the genDecl if there is
			// any exported spec.
			if anyExported {
				out.WriteString(buf.String())
				out.WriteString("\n")
			}
		}
	}
	return nil
}

// TruncateFileContent truncates file content based on byte and line limits.
// Supports starting from a specific line (1-indexed).
// Returns the truncated content, lines read, and truncation info.
func TruncateFileContent(content string, maxBytes, maxLines, startLine int) (string, int, string) {
	lines := strings.Split(content, "\n")

	// Handle start line (1-indexed)
	if startLine > 1 && startLine <= len(lines) {
		lines = lines[startLine-1:]
	} else if startLine > len(lines) {
		return "", 0, fmt.Sprintf("[START_LINE %d exceeds file length %d]", startLine, len(lines))
	}

	actualContent := strings.Join(lines, "\n")
	linesRead := len(lines)
	var truncationInfo []string

	// Apply line limit
	if maxLines > 0 && len(lines) > maxLines {
		actualContent = strings.Join(lines[:maxLines], "\n")
		truncationInfo = append(truncationInfo, fmt.Sprintf("showing lines %d-%d of %d", startLine, startLine+maxLines-1, len(lines)+startLine-1))
		linesRead = maxLines
	}

	// Apply byte limit
	if maxBytes > 0 {
		truncated, wasTruncated := truncateByBytes(actualContent, maxBytes)
		if wasTruncated {
			actualContent = truncated
			truncationInfo = append(truncationInfo, fmt.Sprintf("limited to %d bytes", maxBytes))
		}
	}

	// Add truncation indicator
	if len(truncationInfo) > 0 {
		actualContent = strings.TrimSuffix(actualContent, TruncationIndicator)
		actualContent += "\n\n[TRUNCATED: " + strings.Join(truncationInfo, ", ") + "]"
	}

	return actualContent, linesRead, ""
}

// truncateByBytes truncates content to a maximum number of bytes.
// The truncation happens at a UTF-8 boundary to avoid invalid sequences.
// Returns the truncated content and whether truncation occurred.
func truncateByBytes(content string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(content) <= maxBytes {
		return content, false
	}

	// Find a safe UTF-8 boundary to truncate at
	// Try to truncate at maxBytes minus room for the indicator
	safeMax := maxBytes - len(TruncationIndicator)
	if safeMax <= 0 {
		safeMax = 0
	}

	// Truncate at UTF-8 boundary
	truncated := content
	for i := safeMax; i < len(content); i++ {
		if utf8.RuneStart(content[i]) {
			truncated = content[:i]
			break
		}
	}

	// Try to truncate at a newline for better readability
	lastNewline := strings.LastIndex(truncated, "\n")
	if lastNewline > safeMax/2 { // Only use newline if it's not too far back
		truncated = truncated[:lastNewline]
	}

	return truncated + TruncationIndicator, true
}
