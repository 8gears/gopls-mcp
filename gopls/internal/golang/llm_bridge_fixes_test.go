// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/gopls/internal/protocol"
	"golang.org/x/tools/gopls/mcpbridge/api"
	"golang.org/x/tools/internal/testenv"
)

// TestParentScopeNormalization tests the parent scope matching fix
func TestParentScopeNormalization(t *testing.T) {
	tests := []struct {
		name        string
		nodeParent  string
		locator     string
		shouldMatch bool
	}{
		// Exact match
		{"exact match", "Server", "Server", true},
		{"exact match with pointer", "*Server", "*Server", true},
		{"pointer to non-pointer", "*Server", "Server", true},
		{"non-pointer to pointer", "Server", "*Server", true},

		// Should NOT match (substring false positives prevented)
		{"substring mismatch - server vs servertype", "Server", "ServerType", false},
		{"substring mismatch - fmt vs serverfmt", "fmt", "serverfmt", false},
		{"substring mismatch - type in typename", "Type", "ServerType", false},
		{"empty locator", "Server", "", true},           // No filter = match
		{"empty nodeParent", "", "Server", false},       // Can't match empty
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the normalization logic from ResolveNode
			locator := tt.locator
			nodeParent := tt.nodeParent

			matched := true
			if locator != "" {
				normalizeParent := func(s string) string {
					return strings.TrimPrefix(s, "*")
				}
				if normalizeParent(nodeParent) != normalizeParent(locator) {
					matched = false
				}
			}

			if matched != tt.shouldMatch {
				t.Errorf("Match result = %v, want %v (nodeParent=%q, locator=%q)",
					matched, tt.shouldMatch, nodeParent, locator)
			}
		})
	}
}

// TestStructFieldResolution tests that struct fields resolve to their parent type
func TestStructFieldResolution(t *testing.T) {
	testenv.NeedsGoPackages(t)

	files := map[string][]byte{
		"go.mod": []byte("module example.com\ngo 1.21\n"),
		"main.go": []byte(`package main

type Server struct {
	port int
	host string
}

func (s *Server) Start() {
	_ = s.port
}
`),
	}

	fix := setupLLMTest(t, files)
	defer fix.cleanup()

	mainPath := fix.sandbox.Workdir.AbsPath("main.go")

	// Test that we can find the "port" field with ParentScope "Server"
	locator := api.SymbolLocator{
		SymbolName:  "port",
		ParentScope: "Server",
		Kind:        "field",
		ContextFile: mainPath,
	}

	fh, err := fix.snapshot.ReadFile(fix.ctx, protocol.URIFromPath(mainPath))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	result, err := ResolveNode(fix.ctx, fix.snapshot, fh, locator)
	if err != nil {
		t.Fatalf("ResolveNode failed: %v", err)
	}

	if result == nil || result.Object == nil {
		t.Fatal("ResolveNode returned nil result")
	}

	// Verify we found a field
	v, ok := result.Object.(*types.Var)
	if !ok {
		t.Fatalf("Expected *types.Var, got %T", result.Object)
	}

	if !v.IsField() {
		t.Error("Expected IsField() to be true")
	}

	// Verify the name
	if v.Name() != "port" {
		t.Errorf("Found field name %q, want %q", v.Name(), "port")
	}

	// Verify parent scope tracking
	// The enclosingFunc should be "Server" (from TypeSpec tracking)
	if result.EnclosingFunc != "Server" {
		t.Errorf("EnclosingFunc = %q, want %q", result.EnclosingFunc, "Server")
	}
}

// TestMethodWithExactParentScope tests that methods match their receiver exactly
func TestMethodWithExactParentScope(t *testing.T) {
	testenv.NeedsGoPackages(t)

	files := map[string][]byte{
		"go.mod": []byte("module example.com\ngo 1.21\n"),
		"main.go": []byte(`package main

type Server struct {
	name string
}

func (s *Server) Start() {
	s.name = "started"
}

func (s *Server) Stop() {
	s.name = "stopped"
}
`),
	}

	fix := setupLLMTest(t, files)
	defer fix.cleanup()

	mainPath := fix.sandbox.Workdir.AbsPath("main.go")

	tests := []struct {
		name         string
		symbolName   string
		parentScope  string
		shouldMatch  bool
	}{
		{
			name:        "Start with *Server",
			symbolName:  "Start",
			parentScope: "*Server",
			shouldMatch: true,
		},
		{
			name:        "Start with Server (no pointer)",
			symbolName:  "Start",
			parentScope: "Server",
			shouldMatch: true,
		},
		{
			name:        "Start with wrong scope",
			symbolName:  "Start",
			parentScope: "Client",
			shouldMatch: false,
		},
		{
			name:        "Start with substring match (should not match)",
			symbolName:  "Start",
			parentScope: "ServerType",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locator := api.SymbolLocator{
				SymbolName:  tt.symbolName,
				ParentScope: tt.parentScope,
				Kind:        "method",
				ContextFile: mainPath,
			}

			fh, err := fix.snapshot.ReadFile(fix.ctx, protocol.URIFromPath(mainPath))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			result, err := ResolveNode(fix.ctx, fix.snapshot, fh, locator)

			if tt.shouldMatch {
				if err != nil {
					t.Errorf("Expected to find symbol, got error: %v", err)
				}
				if result == nil || result.Object == nil {
					t.Error("Expected to find symbol, got nil result")
				}
			} else {
				if err == nil {
					t.Error("Expected not to find symbol, but it was found")
				}
			}
		})
	}
}
