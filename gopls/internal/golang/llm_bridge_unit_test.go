// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"go/ast"
	"go/token"
	"testing"

	"golang.org/x/tools/gopls/mcpbridge/api"
)

// TestUpdateScopeStack tests the scope stack update logic
func TestUpdateScopeStack(t *testing.T) {
	tests := []struct {
		name           string
		node           ast.Node
		initialStack   []scopeFrame
		wantEnclosing  string
	}{
		{
			name: "function declaration without receiver",
			node: &ast.FuncDecl{
				Name: &ast.Ident{Name: "myFunction"},
			},
			initialStack:  []scopeFrame{{enclosingFunc: ""}},
			wantEnclosing: "myFunction",
		},
		{
			name: "function declaration with pointer receiver",
			node: &ast.FuncDecl{
				Name: &ast.Ident{Name: "Method"},
				Recv: &ast.FieldList{
					List: []*ast.Field{
						{
							Type: &ast.StarExpr{
								X: &ast.Ident{Name: "Server"},
							},
						},
					},
				},
			},
			initialStack:  []scopeFrame{{enclosingFunc: ""}},
			wantEnclosing: "(*Server).Method",
		},
		{
			name: "type declaration",
			node: &ast.TypeSpec{
				Name: &ast.Ident{Name: "MyStruct"},
			},
			initialStack:  []scopeFrame{{enclosingFunc: ""}},
			wantEnclosing: "MyStruct",
		},
		{
			name: "nested function preserves parent",
			node: &ast.FuncDecl{
				Name: &ast.Ident{Name: "inner"},
			},
			initialStack:  []scopeFrame{
				{enclosingFunc: "outer"},
			},
			wantEnclosing: "inner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updateScopeStack(tt.node, tt.initialStack)
			if result.enclosingFunc != tt.wantEnclosing {
				t.Errorf("updateScopeStack() enclosingFunc = %q, want %q",
					result.enclosingFunc, tt.wantEnclosing)
			}
		})
	}
}

// TestMatchesLocatorFilters tests the filter matching logic
func TestMatchesLocatorFilters(t *testing.T) {
	tests := []struct {
		name        string
		locator     api.SymbolLocator
		nodeParent  string
		nodeKind    string
		wantPassed  bool
		wantReason  string // if !wantPassed, the reason should match this prefix
	}{
		{
			name:       "no filters - always passes",
			locator:    api.SymbolLocator{SymbolName: "Func"},
			nodeParent: "Server",
			nodeKind:   "function",
			wantPassed: true,
		},
		{
			name:        "parent scope exact match",
			locator:     api.SymbolLocator{SymbolName: "Func", ParentScope: "Server"},
			nodeParent:  "Server",
			nodeKind:    "function",
			wantPassed:  true,
		},
		{
			name:        "parent scope with pointer normalization",
			locator:     api.SymbolLocator{SymbolName: "Func", ParentScope: "*Server"},
			nodeParent:  "Server",
			nodeKind:    "function",
			wantPassed:  true,
		},
		{
			name:        "parent scope mismatch",
			locator:     api.SymbolLocator{SymbolName: "Func", ParentScope: "Client"},
			nodeParent:  "Server",
			nodeKind:    "function",
			wantPassed:  false,
			wantReason:  "parent scope mismatch",
		},
		{
			name:        "parent scope substring false positive",
			locator:     api.SymbolLocator{SymbolName: "Func", ParentScope: "Server"},
			nodeParent:  "ServerType",
			nodeKind:    "function",
			wantPassed:  false,
			wantReason:  "parent scope mismatch",
		},
		{
			name:        "kind filter passes",
			locator:     api.SymbolLocator{SymbolName: "Func", Kind: "function"},
			nodeParent:  "Server",
			nodeKind:    "function",
			wantPassed:  true,
		},
		{
			name:        "kind filter mismatch (struct vs function)",
			locator:     api.SymbolLocator{SymbolName: "Func", Kind: "struct"},
			nodeParent:  "Server",
			nodeKind:    "function",
			wantPassed:  false,
			wantReason:  "kind mismatch",
		},
		{
			name:        "both filters pass",
			locator:     api.SymbolLocator{SymbolName: "Func", ParentScope: "*Server", Kind: "method"},
			nodeParent:  "Server",
			nodeKind:    "method",
			wantPassed:  true,
		},
		{
			name:        "parent filter fails, kind passes",
			locator:     api.SymbolLocator{SymbolName: "Func", ParentScope: "Client", Kind: "method"},
			nodeParent:  "Server",
			nodeKind:    "method",
			wantPassed:  false,
			wantReason:  "parent scope mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesLocatorFilters(tt.locator, tt.nodeParent, tt.nodeKind)
			if result.passed != tt.wantPassed {
				t.Errorf("matchesLocatorFilters() passed = %v, want %v",
					result.passed, tt.wantPassed)
			}
			if !tt.wantPassed && result.reason == "" {
				t.Errorf("matchesLocatorFilters() returned no reason for failure, want reason to contain %q", tt.wantReason)
			}
			if !tt.wantPassed && tt.wantReason != "" {
				// Check if reason contains expected text
				if len(result.reason) < len(tt.wantReason) ||
					result.reason[:len(tt.wantReason)] != tt.wantReason {
					t.Errorf("matchesLocatorFilters() reason = %q, want to contain %q",
						result.reason, tt.wantReason)
				}
			}
		})
	}
}

// TestSelectBestCandidate tests the candidate selection logic
func TestSelectBestCandidate(t *testing.T) {
	// Create mock candidates
	candidate1 := &ResolveNodeResult{
		Pos:          token.Pos(100),
		IsDefinition: true,
	}
	candidate2 := &ResolveNodeResult{
		Pos:          token.Pos(200),
		IsDefinition: false,
	}

	tests := []struct {
		name          string
		current       *ResolveNodeResult
		new           *ResolveNodeResult
		lineHint      int
		newIsDef      bool
		wantSelectNew bool
	}{
		{
			name:          "nil current selects new",
			current:       nil,
			new:           candidate1,
			lineHint:      0,
			newIsDef:      true,
			wantSelectNew: true,
		},
		{
			name:          "definition preferred over reference (no line hint)",
			current:       candidate2, // reference
			new:           candidate1, // definition
			lineHint:      0,
			newIsDef:      true,
			wantSelectNew: true,
		},
		{
			name:          "reference not preferred over definition (no line hint)",
			current:       candidate1, // definition
			new:           candidate2, // reference
			lineHint:      0,
			newIsDef:      false,
			wantSelectNew: false,
		},
		{
			name:          "keep first when neither is definition",
			current:       candidate2, // reference
			new:           &ResolveNodeResult{Pos: token.Pos(300), IsDefinition: false},
			lineHint:      0,
			newIsDef:      false,
			wantSelectNew: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal parsego.File mock (in real tests you'd use a proper file)
			// For unit testing, we can't easily create a full parsego.File,
			// so we only test cases that don't require it

			if tt.lineHint > 0 {
				t.Skip("line hint tests require full parsego.File setup")
				return
			}

			locator := api.SymbolLocator{LineHint: tt.lineHint}

			// Note: selectBestCandidate needs a parsego.File for line hint calculation
			// For now, we'll pass nil and test only the non-line-hint cases
			result := selectBestCandidate(tt.current, tt.new, nil, locator, tt.newIsDef)

			selectedNew := (result == tt.new)
			if selectedNew != tt.wantSelectNew {
				t.Errorf("selectBestCandidate() selected new = %v, want %v",
					selectedNew, tt.wantSelectNew)
			}
		})
	}
}

// TestNodeParentInfo tests the nodeParentInfo struct
func TestNodeParentInfo(t *testing.T) {
	info := nodeParentInfo{
		parent: "Server",
		kind:   "struct",
	}

	if info.parent != "Server" {
		t.Errorf("parent = %q, want %q", info.parent, "Server")
	}
	if info.kind != "struct" {
		t.Errorf("kind = %q, want %q", info.kind, "struct")
	}
}

// TestScopeFrame tests the scopeFrame struct
func TestScopeFrame(t *testing.T) {
	frame := scopeFrame{
		node:          &ast.Ident{Name: "test"},
		enclosingFunc: "myFunction",
	}

	if frame.enclosingFunc != "myFunction" {
		t.Errorf("enclosingFunc = %q, want %q", frame.enclosingFunc, "myFunction")
	}

	ident, ok := frame.node.(*ast.Ident)
	if !ok {
		t.Fatal("node is not *ast.Ident")
	}
	if ident.Name != "test" {
		t.Errorf("node name = %q, want %q", ident.Name, "test")
	}
}
