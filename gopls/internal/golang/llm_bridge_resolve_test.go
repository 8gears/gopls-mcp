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

// TestResolveNodeWithLocator tests the core symbol locator resolution logic
// with various testdata fixtures.
//
// This test verifies that ResolveNode can correctly:
// 1. Parse SymbolLocator inputs (mocked LLM input)
// 2. Resolve to the correct AST node and position
// 3. Return accurate type information
//
// This is a focused unit test for symbol resolution, separate from
// integration tests that test the full pipeline.
func TestResolveNodeWithLocator(t *testing.T) {
	testenv.NeedsGoPackages(t)

	tests := []struct {
		name        string
		testdataDir string
		locator     api.SymbolLocator
		fileHint    string // which file to look in (for multi-file tests)
		wantName    string // expected symbol name
		wantRecv    string // expected receiver name (for methods)
		wantFound   bool   // whether the symbol should be found
	}{
		// Basic interface method
		{
			name:        "basic interface method",
			testdataDir: "basic_interface",
			locator: api.SymbolLocator{
				SymbolName:  "Area",
				ParentScope: "Shape",
				Kind:        "method",
			},
			wantName:  "Area",
			wantFound: true,
		},
		// Interface method with pointer receiver
		{
			name:        "pointer receiver method",
			testdataDir: "pointer_vs_value_receivers",
			locator: api.SymbolLocator{
				SymbolName:  "Process",
				ParentScope: "*Processor",
				Kind:        "method",
			},
			wantName:  "Process",
			wantRecv:  "Processor",
			wantFound: true,
		},
		// Value receiver method
		{
			name:        "value receiver method",
			testdataDir: "pointer_vs_value_receivers",
			locator: api.SymbolLocator{
				SymbolName:  "Process",
				ParentScope: "Processor",
				Kind:        "method",
			},
			wantName:  "Process",
			wantRecv:  "Processor",
			wantFound: true,
		},
		// Generic method
		{
			name:        "generic method",
			testdataDir: "generic_types",
			locator: api.SymbolLocator{
				SymbolName: "Put",
				Kind:       "method",
			},
			wantName:  "Put",
			wantFound: true,
		},
		// Variadic method
		{
			name:        "variadic method",
			testdataDir: "variadic_methods",
			locator: api.SymbolLocator{
				SymbolName:  "Log",
				ParentScope: "Logger",
				Kind:        "method",
			},
			wantName:  "Log",
			wantFound: true,
		},
		// Method with error return
		{
			name:        "error return type",
			testdataDir: "error_return_types",
			locator: api.SymbolLocator{
				SymbolName:  "Get",
				ParentScope: "Repository",
				Kind:        "method",
			},
			wantName:  "Get",
			wantFound: true,
		},
		// Multiple type parameters
		{
			name:        "multiple type parameters",
			testdataDir: "multiple_type_parameters",
			locator: api.SymbolLocator{
				SymbolName: "Get",
				Kind:       "method",
			},
			wantName:  "Get",
			wantFound: true,
		},
		// Nested interface
		{
			name:        "nested interface",
			testdataDir: "nested_interfaces",
			locator: api.SymbolLocator{
				SymbolName:  "Read",
				ParentScope: "Readable",
				Kind:        "method",
			},
			wantName:  "Read",
			wantFound: true,
		},
		// Sort interface implementation
		{
			name:        "sort interface implementation",
			testdataDir: "sort_interface_implementation",
			locator: api.SymbolLocator{
				SymbolName:  "Len",
				ParentScope: "Sortable",
				Kind:        "method",
			},
			wantName:  "Len",
			wantFound: true,
		},
		// Error interface
		{
			name:        "error interface",
			testdataDir: "error_interface",
			locator: api.SymbolLocator{
				SymbolName:  "Error",
				ParentScope: "MyError",
				Kind:        "method",
			},
			wantName:  "Error",
			wantFound: true,
		},
		// Stringer interface
		{
			name:        "stringer interface",
			testdataDir: "fmt_stringer_interface",
			locator: api.SymbolLocator{
				SymbolName:  "String",
				ParentScope: "Stringer",
				Kind:        "method",
			},
			wantName:  "String",
			wantFound: true,
		},
		// Method with no return value
		{
			name:        "method with no return value",
			testdataDir: "method_with_no_return_value",
			locator: api.SymbolLocator{
				SymbolName:  "Init",
				ParentScope: "Initializer",
				Kind:        "method",
			},
			wantName:  "Init",
			wantFound: true,
		},
		// Symbol not found - should error
		{
			name:        "symbol not found",
			testdataDir: "symbol_not_found_should_error",
			locator: api.SymbolLocator{
				SymbolName:  "NonExistentMethod",
				ParentScope: "MyInterface",
				Kind:        "method",
			},
			wantFound: false,
		},
		// Ambiguous method names - same method different interfaces
		{
			name:        "ambiguous method names",
			testdataDir: "ambiguous_method_names",
			locator: api.SymbolLocator{
				SymbolName:  "Read",
				ParentScope: "Readable",
				Kind:        "method",
			},
			wantName:  "Read",
			wantFound: true,
		},
		// Multiple methods - test each
		{
			name:        "multiple methods",
			testdataDir: "multiple_methods_test_each",
			locator: api.SymbolLocator{
				SymbolName:  "Close",
				ParentScope: "ReadWriter",
				Kind:        "method",
			},
			wantName:  "Close",
			wantFound: true,
		},
		// Complex generics with constraints
		{
			name:        "complex generics with constraints",
			testdataDir: "complex_generics_with_constraints",
			locator: api.SymbolLocator{
				SymbolName: "Compare",
				Kind:       "method",
			},
			wantName:  "Compare",
			wantFound: true,
		},
		// Complex signatures
		{
			name:        "complex signatures",
			testdataDir: "complex_signatures",
			locator: api.SymbolLocator{
				SymbolName:  "Process",
				ParentScope: "Processor",
				Kind:        "method",
			},
			wantName:  "Process",
			wantFound: true,
		},
		// io.Reader standard interface
		{
			name:        "io reader standard interface",
			testdataDir: "io_reader_standard_interface",
			locator: api.SymbolLocator{
				SymbolName:  "Read",
				ParentScope: "Reader",
				Kind:        "method",
			},
			wantName:  "Read",
			wantFound: true,
		},
		// Pointer receiver with nil safety
		{
			name:        "pointer receiver with nil safety",
			testdataDir: "pointer_receiver_with_nil_safety",
			locator: api.SymbolLocator{
				SymbolName:  "Close",
				ParentScope: "Closer",
				Kind:        "method",
			},
			wantName:  "Close",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load test files from testdata directory
			files := loadTestDataFiles(t, tt.testdataDir)

			// Setup test environment
			fix := setupLLMTest(t, files)
			defer fix.cleanup()

			// Determine the context file
			contextFile := tt.fileHint
			if contextFile == "" {
				// Find main.go or first non-go.mod file
				for fname := range files {
					if fname != "go.mod" {
						contextFile = fname
						if strings.HasSuffix(fname, "main.go") {
							break
						}
					}
				}
			}

			tt.locator.ContextFile = fix.sandbox.Workdir.AbsPath(contextFile)

			// Call ResolveNode
			fh, err := fix.snapshot.ReadFile(fix.ctx, protocol.URIFromPath(tt.locator.ContextFile))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			result, err := ResolveNode(fix.ctx, fix.snapshot, fh, tt.locator)

			// Check expected result
			if tt.wantFound {
				if err != nil {
					t.Fatalf("ResolveNode failed: %v", err)
				}
				if result == nil {
					t.Fatal("ResolveNode returned nil result")
				}
				if result.Object == nil {
					t.Errorf("ResolveNode found no type object for symbol %s", tt.locator.SymbolName)
					return
				}
				if result.Object.Name() != tt.wantName {
					t.Errorf("ResolveNode returned name %q, want %q", result.Object.Name(), tt.wantName)
				}

				// Check receiver if specified
				if tt.wantRecv != "" {
					if fn, ok := result.Object.(*types.Func); ok {
						if sig, ok := fn.Type().(*types.Signature); ok {
							recv := sig.Recv()
							if recv == nil {
								t.Errorf("Expected method with receiver %s, got no receiver", tt.wantRecv)
							} else {
								named := getNamedType(recv.Type())
								if named == nil {
									t.Errorf("Could not get named type for receiver")
								} else if named.Obj().Name() != tt.wantRecv {
									t.Errorf("Got receiver type %q, want %q", named.Obj().Name(), tt.wantRecv)
								}
							}
						}
					}
				}
			} else {
				if err == nil {
					t.Error("ResolveNode expected error for non-existent symbol, got nil")
				}
			}
		})
	}
}

// TestResolveNode_LineHint tests that line hints work correctly for disambiguation
func TestResolveNode_LineHint(t *testing.T) {
	testenv.NeedsGoPackages(t)

	files := map[string][]byte{
		"go.mod": []byte("module example.com\ngo 1.21\n"),
		"main.go": []byte(`package main

type Shape interface {
	Area() float64
}

type Circle struct {
	radius float64
}

func (c Circle) Area() float64 {
	return 3.14 * c.radius * c.radius
}

type Rectangle struct {
	width, height float64
}

func (r Rectangle) Area() float64 {
	return r.width * r.height
}

func main() {
	c := Circle{radius: 5}
	r := Rectangle{width: 10, height: 20}
	_ = c.Area()
	_ = r.Area()
}
`),
	}

	fix := setupLLMTest(t, files)
	defer fix.cleanup()

	mainPath := fix.sandbox.Workdir.AbsPath("main.go")

	tests := []struct {
		name     string
		locator  api.SymbolLocator
		wantRecv string // expected receiver type
	}{
		{
			name: "Circle Area",
			locator: api.SymbolLocator{
				SymbolName:  "Area",
				Kind:        "method",
				ContextFile: mainPath,
				LineHint:    14, // Circle.Area is around line 14
			},
			wantRecv: "Circle",
		},
		{
			name: "Rectangle Area",
			locator: api.SymbolLocator{
				SymbolName:  "Area",
				Kind:        "method",
				ContextFile: mainPath,
				LineHint:    20, // Rectangle.Area is around line 20
			},
			wantRecv: "Rectangle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh, err := fix.snapshot.ReadFile(fix.ctx, protocol.URIFromPath(tt.locator.ContextFile))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			result, err := ResolveNode(fix.ctx, fix.snapshot, fh, tt.locator)
			if err != nil {
				t.Fatalf("ResolveNode failed: %v", err)
			}

			if result == nil || result.Object == nil {
				t.Fatal("ResolveNode returned nil result")
			}

			// Check if the receiver type matches
			if fn, ok := result.Object.(*types.Func); ok {
				if sig, ok := fn.Type().(*types.Signature); ok {
					recv := sig.Recv()
					if recv == nil {
						t.Errorf("Expected method with receiver, got none")
					} else {
						named := getNamedType(recv.Type())
						if named == nil {
							t.Errorf("Could not get named type for receiver")
						} else if named.Obj().Name() != tt.wantRecv {
							t.Errorf("Got receiver type %q, want %q", named.Obj().Name(), tt.wantRecv)
						}
					}
				}
			} else {
				t.Errorf("Expected Func type, got %T", result.Object)
			}
		})
	}
}

// TestResolveNode_ParentScope tests parent scope filtering
func TestResolveNode_ParentScope(t *testing.T) {
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

type Client struct{}

func (c *Client) Start() {
	// Client start logic
}
`),
	}

	fix := setupLLMTest(t, files)
	defer fix.cleanup()

	mainPath := fix.sandbox.Workdir.AbsPath("main.go")

	tests := []struct {
		name         string
		locator      api.SymbolLocator
		wantRecvName string // expected receiver name
		wantFound    bool
	}{
		{
			name: "Server.Start with exact parent scope",
			locator: api.SymbolLocator{
				SymbolName:  "Start",
				ParentScope: "*Server",
				Kind:        "method",
				ContextFile: mainPath,
			},
			wantRecvName: "Server",
			wantFound:    true,
		},
		{
			name: "Client.Start with exact parent scope",
			locator: api.SymbolLocator{
				SymbolName:  "Start",
				ParentScope: "Client",
				Kind:        "method",
				ContextFile: mainPath,
			},
			wantRecvName: "Client",
			wantFound:    true,
		},
		{
			name: "Server.Stop",
			locator: api.SymbolLocator{
				SymbolName:  "Stop",
				ParentScope: "*Server",
				Kind:        "method",
				ContextFile: mainPath,
			},
			wantRecvName: "Server",
			wantFound:    true,
		},
		{
			name: "Start without parent scope (ambiguous)",
			locator: api.SymbolLocator{
				SymbolName:  "Start",
				Kind:        "method",
				ContextFile: mainPath,
			},
			wantFound: true, // Should find one of them (first match)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh, err := fix.snapshot.ReadFile(fix.ctx, protocol.URIFromPath(tt.locator.ContextFile))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			result, err := ResolveNode(fix.ctx, fix.snapshot, fh, tt.locator)
			if !tt.wantFound {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveNode failed: %v", err)
			}

			if result == nil || result.Object == nil {
				t.Fatal("ResolveNode returned nil result")
			}

			if tt.wantRecvName != "" {
				if fn, ok := result.Object.(*types.Func); ok {
					if sig, ok := fn.Type().(*types.Signature); ok {
						recv := sig.Recv()
						if recv != nil {
							named := getNamedType(recv.Type())
							if named != nil {
								gotName := named.Obj().Name()
								if gotName != tt.wantRecvName {
									t.Errorf("Got receiver %q, want %q", gotName, tt.wantRecvName)
								}
							}
						}
					}
				}
			}
		})
	}
}
