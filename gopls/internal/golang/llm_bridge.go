// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/gopls/internal/cache"
	"golang.org/x/tools/gopls/internal/cache/metadata"
	"golang.org/x/tools/gopls/internal/cache/methodsets"
	"golang.org/x/tools/gopls/internal/cache/parsego"
	"golang.org/x/tools/gopls/internal/file"
	"golang.org/x/tools/gopls/internal/protocol"
	"golang.org/x/tools/gopls/mcpbridge/api"
)

// This file provides a "Semantic Bridge" for LLMs to query gopls internal APIs
// without going through the LSP protocol layer (no protocol.Position needed).
//
// It directly resolves semantic intent (SymbolLocator) to gopls internal types
// (ast.Node, types.Object) and calls internal logic directly.

// ResolveNodeResult is the output of ResolveNode, containing rich information
// about the resolved symbol.
type ResolveNodeResult struct {
	// Node is the AST node representing the symbol (e.g., *ast.Ident, *ast.TypeSpec, *ast.FuncDecl)
	Node ast.Node
	// Pos is the token position of the symbol
	Pos token.Pos
	// Object is the types.Object representing the symbol (if type-checking succeeded)
	Object types.Object
	// EnclosingFunc is the name of the function containing this symbol
	EnclosingFunc string
	// IsDefinition indicates whether this node is a definition (not just a reference)
	IsDefinition bool
}

// SourceContext provides rich context about a symbol, suitable for LLM consumption.
type SourceContext struct {
	// File allows the LLM to know which file to modify later.
	File string `json:"file"`

	// Line numbers are sufficient for human reference or coarse-grained location.
	// Columns are removed because they are noisy and prone to tokenizer errors.
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`

	// Symbol name (e.g., "Start")
	Symbol string `json:"symbol"`

	// Kind helps the LLM distinguish between "struct", "interface", "method", "func".
	// This is cheaply available from types.Object.
	Kind string `json:"kind"`

	// Signature helps the LLM index results quickly (e.g., "func (s *Server) Start(ctx context.Context) error")
	Signature string `json:"signature"`

	// DocComment conveys intent (e.g., "Start initializes the server...")
	DocComment string `json:"doc_comment,omitempty"`

	// Snippet is the HERO field. Zero-RTT access to the code.
	Snippet string `json:"snippet"`
}

// ResolveNode resolves a SymbolLocator to the matching AST node and types.Object.
//
// It performs a stateful AST walk with proper parent scope tracking to match
// the semantic criteria in the SymbolLocator:
//   - Strict symbol name matching
//   - Optional package name filtering (for imports)
//   - Optional parent scope filtering (for receivers or local variables)
//   - Optional kind filtering (function, method, struct, etc.)
//   - Fuzzy line hint matching (for disambiguation)
//
// This function bypasses the LSP protocol layer entirely and works directly
// with gopls internal types.
func ResolveNode(ctx context.Context, snapshot *cache.Snapshot, fh file.Handle, locator api.SymbolLocator) (*ResolveNodeResult, error) {
	// Get the package and parse tree for the context file
	pkg, pgf, err := NarrowestPackageForFile(ctx, snapshot, fh.URI())
	if err != nil {
		return nil, fmt.Errorf("failed to get package for %s: %w", locator.ContextFile, err)
	}

	// Build a cursor for efficient AST traversal
	info := pkg.TypesInfo()

	// Find all matching symbols with scope tracking
	type scopeFrame struct {
		node          ast.Node
		enclosingFunc string
	}
	scopeStack := []scopeFrame{{node: pgf.File, enclosingFunc: ""}}

	var candidates []ResolveNodeResult
	var bestCandidate *ResolveNodeResult

	// Walk the AST to find matching symbols
	ast.Inspect(pgf.File, func(n ast.Node) bool {
		if n == nil {
			// Pop from scope stack when returning
			if len(scopeStack) > 1 {
				scopeStack = scopeStack[:len(scopeStack)-1]
			}
			return false
		}

		// Push current node onto scope stack
		currentFrame := scopeFrame{node: n, enclosingFunc: scopeStack[len(scopeStack)-1].enclosingFunc}

		// Update enclosing function if this is a function declaration
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Recv == nil {
				currentFrame.enclosingFunc = fn.Name.Name
			} else {
				recvTypeName := getReceiverTypeName(fn.Recv)
				currentFrame.enclosingFunc = fmt.Sprintf("(%s).%s", recvTypeName, fn.Name.Name)
			}
		}
		scopeStack = append(scopeStack, currentFrame)

		// Only process identifiers that match the symbol name
		ident, ok := n.(*ast.Ident)
		if !ok || ident.Name != locator.SymbolName {
			return true
		}

		// Get the type information for this identifier
		obj := info.Uses[ident]
		isDef := false
		if obj == nil {
			obj = info.Defs[ident]
			isDef = (obj != nil)
		}

		// Extract parent scope information
		var nodeParent string
		var nodePackage string
		var nodeKind string

		if obj != nil {
			// Get parent from object (for methods, fields, etc.)
			nodeParent = getParentScope(obj, scopeStack[len(scopeStack)-1].enclosingFunc)
			nodeKind = getKindString(obj)
		} else {
			// No type info available - use scope stack
			nodeParent = scopeStack[len(scopeStack)-1].enclosingFunc
		}

		// Check if this is a selector expression (e.g., fmt.Println)
		// We need to check the parent to see if this is part of a SelectorExpr
		if parent := scopeStack[len(scopeStack)-1].node; parent != nil {
			if sel, ok := parent.(*ast.SelectorExpr); ok && sel.Sel == ident {
				// This ident is the selector part of a SelectorExpr
				if xIdent, ok := sel.X.(*ast.Ident); ok {
					// Check if X is a package import
					if pkgObj := info.Defs[xIdent]; pkgObj != nil {
						if _, ok := pkgObj.(*types.PkgName); ok {
							nodePackage = xIdent.Name
						}
					} else {
						// X might be a receiver/variable
						nodeParent = xIdent.Name
					}
				}
			}
		}

		// Apply filters
		// 1. Package name filter
		if locator.PackageName != "" && nodePackage != "" && nodePackage != locator.PackageName {
			return true
		}

		// 2. Parent scope filter
		if locator.ParentScope != "" {
			// Support both "FuncName" and "(Receiver).FuncName" formats
			if !strings.Contains(nodeParent, locator.ParentScope) &&
				!strings.Contains(locator.ParentScope, nodeParent) {
				return true
			}
		}

		// 3. Kind filter
		if locator.Kind != "" && nodeKind != "" {
			if !normalizeKindMatches(nodeKind, locator.Kind) {
				return true
			}
		}

		// Found a match!
		candidate := ResolveNodeResult{
			Node:          n,
			Pos:           ident.Pos(),
			Object:        obj,
			EnclosingFunc: scopeStack[len(scopeStack)-1].enclosingFunc,
			IsDefinition:  isDef,
		}

		candidates = append(candidates, candidate)

		// Use line hint to pick the best match
		if locator.LineHint > 0 {
			pos := pgf.Tok.Position(ident.Pos())
			line := pos.Line

			// Calculate confidence score based on line proximity
			distance := abs(line - int(locator.LineHint))
			confidence := 1.0 / float64(distance+1)

			if bestCandidate == nil || confidence > calculateCandidateConfidence(bestCandidate, pgf, locator) {
				bestCandidate = &candidate
			}
		} else {
			// No line hint, prefer definitions over references
			if bestCandidate == nil || (isDef && !bestCandidate.IsDefinition) {
				bestCandidate = &candidate
			}
		}

		return true
	})

	if bestCandidate == nil {
		if len(candidates) == 0 {
			return nil, fmt.Errorf("symbol '%s' not found in file", locator.SymbolName)
		}
		bestCandidate = &candidates[0]
	}

	return bestCandidate, nil
}

// LLMImplementation finds all implementations of an interface method or type.
//
// It takes a SymbolLocator (semantic identifier) and returns SourceContext for each
// implementation, including the symbol's location, signature, documentation, and code snippet.
//
// Use cases:
//   - Find all concrete types implementing an interface
//   - Find all implementations of an interface method
//   - Discover type hierarchies in the codebase
//
// Parameters:
//
//	ctx - Context for cancellation
//	snapshot - gopls snapshot for type checking
//	locator - SymbolLocator with symbol_name, context_file, and optional filters
//	          * symbol_name: The method or type name to find implementations for
//	          * context_file: Absolute path to file where symbol is defined/used
//	          * parent_scope: For methods, the interface name (e.g., "Writer")
//	          * kind: Optional filter ("interface", "method", "struct")
//
// Returns:
//
//	[]SourceContext - Rich information about each implementation:
//	  * File: Absolute path to implementation file
//	  * StartLine/EndLine: Line range (columns removed for cleaner output)
//	  * Symbol: Implementing type or method name
//	  * Kind: Symbol kind ("struct", "interface", "method", etc.)
//	  * Signature: Full signature (e.g., "func (f *File) Write(p []byte) error")
//	  * DocComment: Documentation if available
//	  * Snippet: Full code body (HERO field - zero-RTT access)
//	error - Error if:
//	  * Symbol not found in context file
//	  * File cannot be read
//	  * No type information available
//	  * Context file is not part of the workspace
//
// Limitations:
//   - Only finds implementations of interfaces defined in the codebase
//   - Does NOT work with standard library interfaces (io.Reader, error, fmt.Stringer, etc.)
//   - May not detect method promotion from embedded structs
//   - Requires the interface to be defined in the workspace (not vendored/external)
//
// Example - Find implementations of Writer interface:
//
//	locator := api.SymbolLocator{
//	    SymbolName:  "Write",
//	    ParentScope: "Writer",  // interface name
//	    Kind:        "method",
//	    ContextFile: "/path/to/interfaces.go",
//	}
//	impls, err := golang.LLMImplementation(ctx, snapshot, locator)
//	// Returns: File, FileWriter, ConsoleWriter, etc.
//
// Example - Find what interfaces a type implements:
//
//	locator := api.SymbolLocator{
//	    SymbolName:  "FileWriter",
//	    Kind:        "struct",
//	    ContextFile: "/path/to/file.go",
//	}
//	impls, err := golang.LLMImplementation(ctx, snapshot, locator)
//	// Returns: Writer, io.Closer, etc.
//
// This bypasses the LSP protocol layer and works directly with gopls internals,
// making it faster and more accurate than text-based search.
func LLMImplementation(ctx context.Context, snapshot *cache.Snapshot, locator api.SymbolLocator) ([]SourceContext, error) {
	// First, resolve the node to get the types.Object
	fh, err := snapshot.ReadFile(ctx, protocol.URIFromPath(locator.ContextFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result, err := ResolveNode(ctx, snapshot, fh, locator)
	if err != nil {
		return nil, err
	}

	if result.Object == nil {
		return nil, fmt.Errorf("symbol '%s' has no type information", locator.SymbolName)
	}

	// Get the package for this file
	pkg, pgf, err := NarrowestPackageForFile(ctx, snapshot, fh.URI())
	if err != nil {
		return nil, fmt.Errorf("failed to get package: %w", err)
	}

	// Build a cursor from the resolved position
	cur, ok := pgf.Cursor().FindByPos(result.Pos, result.Pos)
	if !ok {
		return nil, fmt.Errorf("failed to create cursor for position")
	}

	// Use the internal implementations logic
	var implementations []SourceContext

	// relation=0 means infer direction (Supertypes/Subtypes) from concreteness
	const relation = methodsets.TypeRelation(0)

	err = implementationsMsets(ctx, snapshot, pkg, cur, relation, func(_ metadata.PackagePath, _ string, _ bool, loc protocol.Location) {
		// Get symbol information from the package containing the implementation
		implPkg, implPgf, err := NarrowestPackageForFile(ctx, snapshot, loc.URI)
		if err != nil {
			// Fallback to minimal source context
			implementations = append(implementations, sourceContextFromLocation(snapshot, loc))
			return
		}

		// Find the identifier at the implementation location
		implIdent := findIdentifierAtPos(implPgf, loc.Range.Start.Line, loc.Range.Start.Character)
		if implIdent == nil {
			implementations = append(implementations, sourceContextFromLocation(snapshot, loc))
			return
		}

		// Get the types.Object for this identifier
		implObj := implPkg.TypesInfo().Defs[implIdent]
		if implObj == nil {
			implObj = implPkg.TypesInfo().Uses[implIdent]
		}

		// Find the AST node for this implementation (contains doc comment)
		implNode := findNodeAtPos(implPgf, loc.Range.Start.Line, loc.Range.Start.Character)

		// Build rich source context if we have both object and node
		if implObj != nil && implNode != nil {
			srcCtx := buildSourceContext(implPkg.FileSet(), implObj, implNode)
			implementations = append(implementations, srcCtx)
		} else {
			// Fallback to minimal source context
			srcCtx := sourceContextFromLocation(snapshot, loc)
			srcCtx.Symbol = implIdent.Name
			if implObj != nil {
				srcCtx.Signature = formatObjectString(implObj)
			}
			implementations = append(implementations, srcCtx)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find implementations: %w", err)
	}

	return implementations, nil
}

// Helper functions

// getReceiverTypeName extracts the receiver type name from a receiver field list
func getReceiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	expr := recv.List[0].Type
	switch t := expr.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr, *ast.IndexListExpr:
		// Generic types - skip for now
		return ""
	}
	return ""
}

// getParentScope extracts the parent scope name from a types.Object
func getParentScope(obj types.Object, enclosingFunc string) string {
	if obj == nil {
		return enclosingFunc
	}

	// Check if it's a method
	if fn, ok := obj.(*types.Func); ok {
		if sig, ok := fn.Type().(*types.Signature); ok {
			recv := sig.Recv()
			if recv != nil {
				// Method with receiver
				named := getNamedType(recv.Type())
				if named != nil {
					return "*" + named.Obj().Name()
				}
			}
		}
	}

	// Check if it's a field
	if v, ok := obj.(*types.Var); ok && v.IsField() {
		named := getNamedType(v.Type())
		if named != nil {
			return named.Obj().Name()
		}
	}

	return enclosingFunc
}

// getNamedType extracts the *types.Named from a type
func getNamedType(t types.Type) *types.Named {
	switch t := t.(type) {
	case *types.Named:
		return t
	case *types.Pointer:
		return getNamedType(t.Elem())
	default:
		return nil
	}
}

// getKindString returns the kind string for a types.Object
func getKindString(obj types.Object) string {
	switch obj.(type) {
	case *types.Func:
		return "function"
	case *types.Var:
		if obj.(*types.Var).IsField() {
			return "field"
		}
		return "variable"
	case *types.Const:
		return "const"
	case *types.TypeName:
		return "type"
	default:
		return ""
	}
}

// normalizeKindMatches checks if two kind strings match after normalization
func normalizeKindMatches(a, b string) bool {
	return normalizeKind(a) == normalizeKind(b)
}

func normalizeKind(kind string) string {
	kind = strings.ToLower(kind)
	switch kind {
	case "function", "method":
		return "function"
	case "struct", "interface", "type":
		return "type"
	case "variable", "field":
		return kind
	case "const":
		return kind
	default:
		return kind
	}
}

// calculateCandidateConfidence calculates a confidence score for a candidate
func calculateCandidateConfidence(candidate *ResolveNodeResult, pgf *parsego.File, locator api.SymbolLocator) float64 {
	pos := pgf.Tok.Position(candidate.Pos)
	line := pos.Line

	if locator.LineHint > 0 {
		distance := abs(line - int(locator.LineHint))
		return 1.0 / float64(distance+1)
	}
	return 0.5
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// sourceContextFromLocation converts a protocol.Location to a minimal SourceContext.
// This is used as a fallback when we can't get full symbol information.
func sourceContextFromLocation(snapshot *cache.Snapshot, loc protocol.Location) SourceContext {
	return SourceContext{
		File:      loc.URI.Path(),
		StartLine: int(loc.Range.Start.Line + 1),
		EndLine:   int(loc.Range.End.Line + 1),
	}
}

// findNodeAtPos finds the declaration node at a given position in a file.
// It searches upward from the identifier to find the enclosing declaration
// (e.g., *ast.FuncDecl, *ast.TypeSpec, *ast.ValueSpec) that contains the doc comment.
func findNodeAtPos(pgf *parsego.File, line, col uint32) ast.Node {
	// Convert LSP position (0-indexed) to offset
	offset, err := pgf.Mapper.PositionOffset(protocol.Position{Line: line, Character: col})
	if err != nil {
		return nil
	}

	// Convert offset to token.Pos
	pos := pgf.Tok.Pos(offset)
	if !pos.IsValid() {
		return nil
	}

	// Use cursor to find the node at this position
	cur, ok := pgf.Cursor().FindByPos(pos, pos)
	if !ok {
		return nil
	}

	// Walk up the tree to find a declaration node with documentation
	for {
		node := cur.Node()
		if node == nil {
			break
		}

		// Check if this is a declaration node type
		switch node.(type) {
		case *ast.FuncDecl, *ast.GenDecl, *ast.TypeSpec, *ast.ValueSpec:
			return node
		}

		cur = cur.Parent()
	}

	return nil
}

// findIdentifierAtPos finds the identifier at a given position in a file
func findIdentifierAtPos(pgf *parsego.File, line, col uint32) *ast.Ident {
	// Convert LSP position (0-indexed) to offset
	offset, err := pgf.Mapper.PositionOffset(protocol.Position{Line: line, Character: col})
	if err != nil {
		return nil
	}

	// Convert offset to token.Pos
	pos := pgf.Tok.Pos(offset)
	if !pos.IsValid() {
		return nil
	}

	// Use cursor to find the node at this position
	// FindByPos requires both start and end positions
	cur, ok := pgf.Cursor().FindByPos(pos, pos)
	if !ok {
		return nil
	}

	// Check if the current node is an identifier
	if ident, ok := cur.Node().(*ast.Ident); ok {
		return ident
	}

	// Walk up to find the enclosing identifier
	for {
		node := cur.Node()
		if node == nil {
			break
		}

		if ident, ok := node.(*ast.Ident); ok {
			return ident
		}

		cur = cur.Parent()
	}
	return nil
}

// formatObjectString returns a string representation of a types.Object
func formatObjectString(obj types.Object) string {
	if obj == nil {
		return ""
	}

	switch obj := obj.(type) {
	case *types.Func:
		return obj.String()
	case *types.Var:
		return obj.String()
	case *types.Const:
		return obj.String()
	case *types.TypeName:
		return obj.String()
	default:
		return fmt.Sprintf("%v", obj)
	}
}

// extractDocComment extracts the documentation comment for an object from its AST node.
func extractDocComment(node ast.Node) string {
	if node == nil {
		return ""
	}

	// Check different node types for documentation
	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Doc != nil {
			return n.Doc.Text()
		}
	case *ast.GenDecl:
		if n.Doc != nil {
			return n.Doc.Text()
		}
	case *ast.TypeSpec:
		if n.Doc != nil {
			return n.Doc.Text()
		}
	}

	return ""
}

// buildSourceContext creates a rich SourceContext from types.Object and ast.Node.
// This is the core function that populates all fields in SourceContext.
//
// Parameters:
//   - fset: FileSet for position information
//   - obj: types.Object containing type information
//   - node: ast.Node containing syntax tree and documentation
func buildSourceContext(fset *token.FileSet, obj types.Object, node ast.Node) SourceContext {
	// 1. Position information
	pos := node.Pos()
	end := node.End()
	position := fset.Position(pos)
	endPos := fset.Position(end)

	// 2. Extract Kind
	kind := getKindFromObject(obj)

	// 3. Extract Snippet (HERO field)
	var snippet string
	var snippetBuf bytes.Buffer
	if err := printer.Fprint(&snippetBuf, fset, node); err != nil {
		// Fallback: try to format the node
		snippet = ""
	} else {
		// Try to format the snippet for better readability
		formatted, err := format.Source(snippetBuf.Bytes())
		if err != nil {
			// If formatting fails, use the raw output
			snippet = snippetBuf.String()
		} else {
			snippet = string(formatted)
		}
	}

	// 4. Extract DocComment from AST node
	docComment := extractDocComment(node)

	// 5. Build signature using gopls's type formatter
	signature := types.ObjectString(obj, nil) // nil = no qualifier needed

	return SourceContext{
		File:       position.Filename,
		StartLine:  position.Line,
		EndLine:    endPos.Line,
		Symbol:     obj.Name(),
		Kind:       kind,
		Signature:  signature,
		Snippet:    snippet,
		DocComment: docComment,
	}
}

// getKindFromObject determines the kind of a types.Object.
func getKindFromObject(obj types.Object) string {
	if obj == nil {
		return "unknown"
	}

	switch obj.(type) {
	case *types.Func:
		// Distinguish between method and function
		if sig, ok := obj.Type().(*types.Signature); ok && sig.Recv() != nil {
			return "method"
		}
		return "function"
	case *types.TypeName:
		// Try to determine if it's a struct or interface from the AST
		// This is a best-effort approximation
		return "type"
	case *types.Var:
		if v, ok := obj.(*types.Var); ok && v.IsField() {
			return "field"
		}
		return "variable"
	case *types.Const:
		return "const"
	default:
		return "unknown"
	}
}

// ===== AST Utility Functions =====
// These functions provide AST-level operations for symbol extraction and analysis.

// FindSymbolPosition searches for a symbol by name in a parsed Go file.
// It returns the token.Pos of the symbol declaration.
//
// This is useful for locating symbols without full type information,
// or for quick lookups when you only need the position.
func FindSymbolPosition(pgf *parsego.File, symbolName string) (token.Pos, bool) {
	// Search for function declarations
	for _, decl := range pgf.File.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Name != nil && decl.Name.Name == symbolName {
				return decl.Name.Pos(), true
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if spec.Name != nil && spec.Name.Name == symbolName {
						return spec.Name.Pos(), true
					}
				case *ast.ValueSpec:
					for _, n := range spec.Names {
						if n.Name == symbolName {
							return n.Pos(), true
						}
					}
				}
			}
		}
	}
	return token.NoPos, false
}

// ExtractBodyText extracts the source text of an AST block statement.
// It cleans up the text by removing extra whitespace and newlines.
//
// This is useful for extracting function bodies for display or analysis.
func ExtractBodyText(pgf *parsego.File, body *ast.BlockStmt) string {
	if body == nil {
		return ""
	}

	// Get positions as token.Pos
	startPos := body.Pos()
	endPos := body.End()

	// Convert to byte offsets, adjusting for file start
	fileStart := pgf.File.FileStart
	start := startPos - fileStart
	end := endPos - fileStart

	// Get the source content
	content := pgf.Src

	// Extract the body text
	if start >= 0 && int32(end) <= int32(len(content)) && start < end {
		bodyText := string(content[start:end])

		// Clean up: remove extra whitespace and newlines
		bodyText = strings.TrimSpace(bodyText)
		bodyText = strings.ReplaceAll(bodyText, "\n", " ")
		bodyText = strings.ReplaceAll(bodyText, "\t", " ")
		// Collapse multiple spaces
		for strings.Contains(bodyText, "  ") {
			bodyText = strings.ReplaceAll(bodyText, "  ", " ")
		}

		return bodyText
	}

	return ""
}

// ExtractBodyForSymbol extracts the function body text for a symbol by name.
// It searches for the function declaration in the given file and returns
// the cleaned body text.
//
// This is a convenience function that combines FindSymbolPosition and ExtractBodyText.
func ExtractBodyForSymbol(ctx context.Context, snapshot *cache.Snapshot, symbolName string, defFile string) string {
	// Read the file at the definition location
	uri := protocol.URIFromPath(defFile)
	defFh, err := snapshot.ReadFile(ctx, uri)
	if err != nil {
		return ""
	}

	pgf, err := snapshot.ParseGo(ctx, defFh, parsego.Full)
	if err != nil {
		return ""
	}

	// Find the function declaration by name
	for _, decl := range pgf.File.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Body != nil {
			if fd.Name != nil && fd.Name.Name == symbolName {
				return ExtractBodyText(pgf, fd.Body)
			}
		}
	}

	return ""
}

// ConvertLSPSymbolKind converts LSP SymbolKind to API SymbolKind.
// This provides a mapping between the LSP protocol's symbol kind enumeration
// and our API's SymbolKind type.
func ConvertLSPSymbolKind(kind protocol.SymbolKind) api.SymbolKind {
	switch kind {
	case protocol.Function:
		return api.SymbolKindFunction
	case protocol.Method:
		return api.SymbolKindMethod
	case protocol.Struct:
		return api.SymbolKindStruct
	case protocol.Interface:
		return api.SymbolKindInterface
	case protocol.Variable:
		return api.SymbolKindVariable
	case protocol.Constant:
		return api.SymbolKindConstant
	case protocol.Field:
		return api.SymbolKindField
	case protocol.Property:
		return api.SymbolKindField // Treat properties as fields
	case protocol.Class:
		return api.SymbolKindStruct // Treat classes as structs
	case protocol.Module:
		return api.SymbolKindType // Treat modules as types
	case protocol.Package:
		return api.SymbolKindType // Treat packages as types
	default:
		return api.SymbolKindType
	}
}

// ===== LLMRename - Semantic Bridge for Rename Operations =====

// LLMRename performs a dry-run rename operation and returns both unified diff and line changes.
//
// This is a high-level function that:
// 1. Uses ResolveNode to find the symbol position
// 2. Calls the internal Rename logic to compute changes
// 3. Returns both a unified diff format and LLM-friendly line changes
//
// This bypasses the LSP protocol layer and works directly with gopls internals.
func LLMRename(ctx context.Context, snapshot *cache.Snapshot, locator api.SymbolLocator, newName string) (string, []api.RenameChange, error) {
	// First, resolve the node to get the position
	fh, err := snapshot.ReadFile(ctx, protocol.URIFromPath(locator.ContextFile))
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	result, err := ResolveNode(ctx, snapshot, fh, locator)
	if err != nil {
		return "", nil, err
	}

	// Convert token.Pos to protocol.Position
	pkg, _, err := NarrowestPackageForFile(ctx, snapshot, fh.URI())
	if err != nil {
		return "", nil, fmt.Errorf("failed to get package: %w", err)
	}

	posn := pkg.FileSet().Position(result.Pos)
	if !posn.IsValid() {
		return "", nil, fmt.Errorf("invalid position for symbol '%s'", locator.SymbolName)
	}

	position := protocol.Position{
		Line:      uint32(posn.Line - 1),
		Character: uint32(posn.Column - 1),
	}

	// Call the internal Rename function
	changes, err := Rename(ctx, snapshot, fh, position, newName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to compute rename: %w", err)
	}

	// Convert changes to unified diff format
	unifiedDiff, err := generateUnifiedDiff(ctx, snapshot, changes)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate unified diff: %w", err)
	}

	// Convert changes to LLM-friendly line-based format
	lineChanges, err := generateLineChanges(ctx, snapshot, changes)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate line changes: %w", err)
	}

	return unifiedDiff, lineChanges, nil
}

// generateUnifiedDiff converts DocumentChange results to a unified diff format.
func generateUnifiedDiff(ctx context.Context, snapshot *cache.Snapshot, changes []protocol.DocumentChange) (string, error) {
	var diff strings.Builder

	// Group changes by file
	type fileChange struct {
		uri      protocol.DocumentURI
		edits    []protocol.TextEdit
		content  string
		original string
	}

	fileChanges := make(map[protocol.DocumentURI]*fileChange)

	for _, docChange := range changes {
		if docChange.TextDocumentEdit == nil {
			continue
		}

		edit := docChange.TextDocumentEdit
		uri := edit.TextDocument.URI

		// Read the original file content
		fh, err := snapshot.ReadFile(ctx, uri)
		if err != nil {
			continue // Skip files we can't read
		}

		contentBytes, err := fh.Content()
		if err != nil {
			continue
		}

		originalContent := string(contentBytes)

		// Get text edits
		textEdits := protocol.AsTextEdits(edit.Edits)

		if _, exists := fileChanges[uri]; !exists {
			fileChanges[uri] = &fileChange{
				uri:      uri,
				edits:    textEdits,
				original: originalContent,
			}
		} else {
			// Append edits if file already exists
			fileChanges[uri].edits = append(fileChanges[uri].edits, textEdits...)
		}
	}

	// Generate diff for each file
	for uri, fc := range fileChanges {
		// Sort edits by position (descending order to apply from end to start)
		sortEdits(fc.edits)

		// Apply edits to get modified content
		modified := applyEdits(fc.original, fc.edits)

		// Generate unified diff for this file
		fileDiff := unifiedDiffForFile(uri.Path(), fc.original, modified)
		diff.WriteString(fileDiff)
		diff.WriteString("\n")
	}

	return diff.String(), nil
}

// generateLineChanges converts DocumentChange results to LLM-friendly line-based format.
// This format provides complete line content for easy verification and rewriting by LLMs.
func generateLineChanges(ctx context.Context, snapshot *cache.Snapshot, changes []protocol.DocumentChange) ([]api.RenameChange, error) {
	var lineChanges []api.RenameChange

	for _, docChange := range changes {
		if docChange.TextDocumentEdit == nil {
			continue
		}

		edit := docChange.TextDocumentEdit
		uri := edit.TextDocument.URI
		filePath := uri.Path()

		// Read the original file content
		fh, err := snapshot.ReadFile(ctx, uri)
		if err != nil {
			continue // Skip files we can't read
		}

		contentBytes, err := fh.Content()
		if err != nil {
			continue
		}

		originalContent := string(contentBytes)
		lines := strings.Split(originalContent, "\n")

		// Get text edits
		textEdits := protocol.AsTextEdits(edit.Edits)

		// Convert each text edit to a line-based change
		for _, textEdit := range textEdits {
			startLine := int(textEdit.Range.Start.Line)
			endLine := int(textEdit.Range.End.Line)

			// Get the original line(s) affected
			if startLine >= len(lines) {
				continue
			}

			oldLine := ""
			if endLine < len(lines) {
				// Multi-line edit: collect all affected lines
				var oldLines []string
				for i := startLine; i <= endLine; i++ {
					oldLines = append(oldLines, lines[i])
				}
				oldLine = strings.Join(oldLines, "\n")
			} else if startLine < len(lines) {
				oldLine = lines[startLine]
			}

			// Apply the edit to get the new line content
			newLine := applyEditToLine(oldLine, textEdit)

			lineChanges = append(lineChanges, api.RenameChange{
				File:    filePath,
				Line:    startLine + 1, // 1-indexed for display
				OldLine: oldLine,
				NewLine: newLine,
			})
		}
	}

	return lineChanges, nil
}

// applyEditToLine applies a single text edit to a line (or multi-line) content.
func applyEditToLine(content string, edit protocol.TextEdit) string {
	startCol := int(edit.Range.Start.Character)
	endCol := int(edit.Range.End.Character)

	// Handle single-line edit
	if edit.Range.Start.Line == edit.Range.End.Line {
		if startCol <= len(content) && endCol <= len(content) {
			return content[:startCol] + edit.NewText + content[endCol:]
		}
	}

	// For multi-line edits or edge cases, just return the new text
	return edit.NewText
}

// sortEdits sorts text edits in descending order by position.
// This ensures that applying edits from end to start doesn't invalidate positions.
func sortEdits(edits []protocol.TextEdit) {
	// Sort by start position (descending), then by end position (descending)
	for i := 0; i < len(edits)-1; i++ {
		for j := i + 1; j < len(edits); j++ {
			iStart := edits[i].Range.Start
			jStart := edits[j].Range.Start

			// Compare positions
			if jStart.Line > iStart.Line ||
				(jStart.Line == iStart.Line && jStart.Character > iStart.Character) {
				edits[i], edits[j] = edits[j], edits[i]
			}
		}
	}
}

// applyEdits applies text edits to the original content.
func applyEdits(original string, edits []protocol.TextEdit) string {
	// Convert original content to lines for easier editing
	lines := strings.Split(original, "\n")

	// Apply edits from end to start (already sorted)
	for _, edit := range edits {
		startLine := int(edit.Range.Start.Line)
		startCol := int(edit.Range.Start.Character)
		endLine := int(edit.Range.End.Line)
		endCol := int(edit.Range.End.Character)

		if startLine >= len(lines) {
			continue
		}

		// Get the prefix and suffix for the edit range
		prefix := ""
		suffix := ""

		if startLine == endLine {
			// Single-line edit
			line := lines[startLine]
			if startCol <= len(line) && endCol <= len(line) {
				prefix = line[:startCol]
				suffix = line[endCol:]
				lines[startLine] = prefix + edit.NewText + suffix
			}
		} else {
			// Multi-line edit
			if startLine < len(lines) {
				prefix = lines[startLine][:min(startCol, len(lines[startLine]))]
			}
			if endLine < len(lines) {
				suffix = lines[endLine][min(endCol, len(lines[endLine])):]
			}

			// Replace the range with new text
			newLines := strings.Split(edit.NewText, "\n")
			result := make([]string, 0, len(lines)-(endLine-startLine)+len(newLines))
			result = append(result, lines[:startLine]...)
			result = append(result, newLines...)
			result = append(result, lines[endLine+1:]...)

			// Join first/last line with prefix/suffix
			if len(result) > startLine {
				result[startLine] = prefix + result[startLine]
			}
			lastIdx := startLine + len(newLines) - 1
			if lastIdx < len(result) && lastIdx >= 0 {
				result[lastIdx] = result[lastIdx] + suffix
			}

			lines = result
		}
	}

	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// unifiedDiffForFile generates a unified diff for a single file.
func unifiedDiffForFile(filePath string, original, modified string) string {
	var diff strings.Builder

	// Split into lines
	origLines := strings.Split(original, "\n")
	modLines := strings.Split(modified, "\n")

	// Generate diff header
	diff.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
	diff.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

	// Generate hunks using simple line-by-line comparison
	hunks := generateHunks(origLines, modLines)

	for _, hunk := range hunks {
		diff.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			hunk.origStart, hunk.origCount,
			hunk.modStart, hunk.modCount))

		for _, line := range hunk.lines {
			diff.WriteString(line)
			diff.WriteString("\n")
		}
	}

	return diff.String()
}

// hunk represents a single hunk in a unified diff.
type hunk struct {
	origStart int // 1-indexed
	origCount int
	modStart  int // 1-indexed
	modCount  int
	lines     []string
}

// generateHunks generates hunks for the unified diff.
// This is a simplified implementation that finds contiguous changes.
func generateHunks(origLines, modLines []string) []hunk {
	var hunks []hunk

	i, j := 0, 0
	for i < len(origLines) || j < len(modLines) {
		// Find the next difference
		if i < len(origLines) && j < len(modLines) && origLines[i] == modLines[j] {
			i++
			j++
			continue
		}

		// Found a difference, collect the hunk
		// Context size (number of unchanged lines before/after)
		const contextSize = 3

		// Add context lines before the change
		contextStart := max(0, i-contextSize)
		if contextStart > 0 {
			// We have previous context, this is not the first hunk
		}

		var lines []string

		// Add context before
		for k := contextStart; k < i; k++ {
			lines = append(lines, " "+origLines[k])
		}

		// Collect changed lines
		origCount := i - contextStart
		modCount := j - contextStart

		// Find the end of the change
		origEnd := i
		modEnd := j

		// Count deletions
		for origEnd < len(origLines) && (modEnd >= len(modLines) || origLines[origEnd] != modLines[modEnd]) {
			lines = append(lines, "-"+origLines[origEnd])
			origEnd++
			origCount++
		}

		// Count additions
		for modEnd < len(modLines) && (origEnd >= len(origLines) || origLines[origEnd] != modLines[modEnd]) {
			lines = append(lines, "+"+modLines[modEnd])
			modEnd++
			modCount++
		}

		// Add context after
		contextEnd := min(len(origLines), origEnd+contextSize)
		for k := origEnd; k < contextEnd; k++ {
			lines = append(lines, " "+origLines[k])
			origCount++
			modCount++
		}

		hunks = append(hunks, hunk{
			origStart: contextStart + 1,
			origCount: origCount,
			modStart:  contextStart + 1,
			modCount:  modCount,
			lines:     lines,
		})

		i = origEnd
		j = modEnd
	}

	return hunks
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
