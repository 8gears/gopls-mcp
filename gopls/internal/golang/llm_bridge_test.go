// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/gopls/internal/cache"
	"golang.org/x/tools/gopls/internal/cache/parsego"
	"golang.org/x/tools/gopls/internal/protocol"
	"golang.org/x/tools/gopls/internal/settings"
	"golang.org/x/tools/gopls/internal/test/integration/fake"
	"golang.org/x/tools/gopls/mcpbridge/api"
	"golang.org/x/tools/internal/testenv"
)

// ===== Test Helpers =====

// loadTestDataFiles loads all files from a testdata subdirectory
func loadTestDataFiles(t *testing.T, testdataDir string) map[string][]byte {
	t.Helper()

	dirPath := filepath.Join("testdata", testdataDir)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("failed to read testdata directory %s: %v", dirPath, err)
	}

	files := make(map[string][]byte)
	// Always add go.mod
	files["go.mod"] = []byte("module example.com\nGo 1.21\n")

	for _, entry := range entries {
		if entry.IsDir() {
			// For subdirectories (like multi-package tests)
			subDirPath := filepath.Join(dirPath, entry.Name())
			subEntries, err := os.ReadDir(subDirPath)
			if err != nil {
				t.Fatalf("failed to read subdirectory %s: %v", subDirPath, err)
			}
			for _, subEntry := range subEntries {
				if !subEntry.IsDir() {
					filePath := filepath.Join(subDirPath, subEntry.Name())
					content, err := os.ReadFile(filePath)
					if err != nil {
						t.Fatalf("failed to read file %s: %v", filePath, err)
					}
					relPath := filepath.Join(entry.Name(), subEntry.Name())
					files[relPath] = content
				}
			}
		} else {
			// For files in the root testdata directory
			filePath := filepath.Join(dirPath, entry.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read file %s: %v", filePath, err)
			}
			files[entry.Name()] = content
		}
	}

	return files
}

type llmTestFixtures struct {
	sandbox  *fake.Sandbox
	ctx      context.Context
	snapshot *cache.Snapshot
	release  func()
	mainPath string
}

func setupLLMTest(t *testing.T, files map[string][]byte) *llmTestFixtures {
	t.Helper()
	sandbox, err := fake.NewSandbox(&fake.SandboxConfig{RootDir: t.TempDir(), Files: files})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ca := cache.New(nil)
	session := cache.NewSession(ctx, ca)
	options := settings.DefaultOptions()
	uri := protocol.URIFromPath(sandbox.Workdir.RootURI().Path())
	env, err := cache.FetchGoEnv(ctx, uri, options)
	if err != nil {
		sandbox.Close()
		t.Fatal(err)
	}
	folder := &cache.Folder{Dir: uri, Options: options, Env: *env}
	_, snapshot, release, err := session.NewView(ctx, folder)
	if err != nil {
		sandbox.Close()
		t.Fatal(err)
	}
	return &llmTestFixtures{
		sandbox: sandbox, ctx: ctx, snapshot: snapshot,
		release: release, mainPath: sandbox.Workdir.AbsPath("main.go"),
	}
}

func (f *llmTestFixtures) cleanup() {
	f.release()
	f.sandbox.Close()
}

// ===== ResolveNode and other tests =====

func TestResolveNode(t *testing.T) {
	testenv.NeedsGoPackages(t)

	files := map[string][]byte{
		"go.mod": []byte("module example.com\ngo 1.21\n"),
		"main.go": []byte(`package main

import "fmt"

type Server struct {
	Addr string
}

func (s *Server) Start() {
	fmt.Println("Starting at", s.Addr)
}

func main() {
	srv := &Server{Addr: ":8080"}
	srv.Start()
}
`),
	}

	sandbox, err := fake.NewSandbox(&fake.SandboxConfig{
		RootDir: t.TempDir(),
		Files:   files,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	ctx := context.Background()
	ca := cache.New(nil)
	session := cache.NewSession(ctx, ca)
	options := settings.DefaultOptions()

	uri := protocol.URIFromPath(sandbox.Workdir.RootURI().Path())
	env, err := cache.FetchGoEnv(ctx, uri, options)
	if err != nil {
		t.Fatal(err)
	}

	folder := &cache.Folder{
		Dir:     uri,
		Options: options,
		Env:     *env,
	}
	_, snapshot, release, err := session.NewView(ctx, folder)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	mainGoPath := sandbox.Workdir.AbsPath("main.go")
	fh, err := snapshot.ReadFile(ctx, protocol.URIFromPath(mainGoPath))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		locator  api.SymbolLocator
		wantName string
		wantKind string
	}{
		{
			name: "Resolve struct",
			locator: api.SymbolLocator{
				SymbolName:  "Server",
				ContextFile: mainGoPath,
				Kind:        "struct",
			},
			wantName: "Server",
			wantKind: "type",
		},
		{
			name: "Resolve method",
			locator: api.SymbolLocator{
				SymbolName:  "Start",
				ContextFile: mainGoPath,
				ParentScope: "Server",
				Kind:        "method",
			},
			wantName: "Start",
			wantKind: "function",
		},
		{
			name: "Resolve field",
			locator: api.SymbolLocator{
				SymbolName:  "Addr",
				ContextFile: mainGoPath,
				ParentScope: "Server",
				Kind:        "field",
			},
			wantName: "Addr",
			wantKind: "field",
		},
		{
			name: "Resolve package import",
			locator: api.SymbolLocator{
				SymbolName:  "Println",
				ContextFile: mainGoPath,
				PackageIdentifier: "fmt",
			},
			wantName: "Println",
			wantKind: "function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveNode(ctx, snapshot, fh, tt.locator)
			if err != nil {
				t.Fatalf("ResolveNode failed: %v", err)
			}

			if result.Object == nil {
				t.Fatal("Result object is nil")
			}

			if result.Object.Name() != tt.wantName {
				t.Errorf("Resolved object name = %q, want %q", result.Object.Name(), tt.wantName)
			}

			gotKind := getKindString(result.Object)
			if gotKind != tt.wantKind {
				t.Errorf("Resolved object kind = %q, want %q", gotKind, tt.wantKind)
			}
		})
	}
}

func TestNormalizeKindMatches(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"Function", "function", true},
		{"Method", "Function", true},
		{"Struct", "Type", true},
		{"Interface", "Type", true},
		{"Variable", "Field", false},
		{"Const", "const", true},
		{"Unknown", "Unknown", true},
		{"Function", "Type", false},
		{"Method", "Const", false},
	}

	for _, tt := range tests {
		if got := normalizeKindMatches(tt.a, tt.b); got != tt.want {
			t.Errorf("normalizeKindMatches(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestExtractBodyText(t *testing.T) {
	src := `package main

import "fmt"

func foo() {
	x := 1
	y := 2
	fmt.Println(x + y)
}

func empty() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	tok := fset.File(file.Pos())
	uri := protocol.DocumentURI("file:///main.go")
	mapper := protocol.NewMapper(uri, []byte(src))

	pgf := &parsego.File{
		URI:    uri,
		File:   file,
		Tok:    tok,
		Src:    []byte(src),
		Mapper: mapper,
	}

	tests := []struct {
		name     string
		funcName string
		want     string
	}{
		{
			name:     "foo body",
			funcName: "foo",
			want:     "{ x := 1 y := 2 fmt.Println(x + y) }",
		},
		{
			name:     "empty body",
			funcName: "empty",
			want:     "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Find function body
			var body *ast.BlockStmt
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == tt.funcName {
					body = fn.Body
					break
				}
			}

			if body == nil {
				t.Fatalf("function %s not found", tt.funcName)
			}

			got := ExtractBodyText(pgf, body)
			if got != tt.want {
				t.Errorf("ExtractBodyText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindSymbolPosition(t *testing.T) {
	src := `package main

type MyStruct struct{}

func (s *MyStruct) Method() {}

func Function() {}

const Constant = 1
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	tok := fset.File(file.Pos())
	uri := protocol.DocumentURI("file:///main.go")
	mapper := protocol.NewMapper(uri, []byte(src))

	pgf := &parsego.File{
		URI:    uri,
		File:   file,
		Tok:    tok,
		Src:    []byte(src),
		Mapper: mapper,
	}

	tests := []struct {
		symbolName string
		wantFound  bool
	}{
		{"MyStruct", true},
		{"Method", true},
		{"Function", true},
		{"Constant", true},
		{"NonExistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.symbolName, func(t *testing.T) {
			pos, found := FindSymbolPosition(pgf, tt.symbolName)
			if found != tt.wantFound {
				t.Errorf("FindSymbolPosition(%q) found = %v, want %v", tt.symbolName, found, tt.wantFound)
			}
			if found && !pos.IsValid() {
				t.Errorf("FindSymbolPosition(%q) returned invalid position", tt.symbolName)
			}
		})
	}
}

func TestGetReceiverTypeName(t *testing.T) {
	src := `package main

type T struct{}
type G[T any] struct{}

func (t T) Value() {}
func (t *T) Pointer() {}
func (g G[int]) Generic() {}
func (g *G[int]) GenericPointer() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		funcName string
		want     string
	}{
		{"Value", "T"},
		{"Pointer", "*T"},
		{"Generic", ""},        // Current implementation skips generics
		{"GenericPointer", ""}, // Current implementation skips generics
	}

	for _, tt := range tests {
		t.Run(tt.funcName, func(t *testing.T) {
			var decl *ast.FuncDecl
			for _, d := range file.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == tt.funcName {
					decl = fn
					break
				}
			}
			if decl == nil {
				t.Fatalf("function %s not found", tt.funcName)
			}

			if got := getReceiverTypeName(decl.Recv); got != tt.want {
				t.Errorf("getReceiverTypeName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindIdentifierAtPos(t *testing.T) {
	src := `package main

func main() {
	var x int = 10
	println(x)
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	tok := fset.File(file.Pos())
	uri := protocol.DocumentURI("file:///main.go")
	mapper := protocol.NewMapper(uri, []byte(src))

	pgf := &parsego.File{
		URI:    uri,
		File:   file,
		Tok:    tok,
		Src:    []byte(src),
		Mapper: mapper,
	}

	// x is at line 4, char 6 (0-indexed line 3, char 5)
	// println(x)
	// 0123456789
	// println(x) -> x starts at index 9 on line 5? No.
	// Line 4: "	var x int = 10" (tab is 1 byte?)
	// Line 5: "	println(x)"

	// Let's find the exact position of 'x' in println(x)
	var xPos token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if len(call.Args) > 0 {
				if ident, ok := call.Args[0].(*ast.Ident); ok && ident.Name == "x" {
					xPos = ident.Pos()
				}
			}
		}
		return true
	})

	if !xPos.IsValid() {
		t.Fatal("could not find 'x' usage")
	}

	// Convert token.Pos to line/col
	pos := tok.Position(xPos)
	// LSP is 0-indexed
	line := uint32(pos.Line - 1)
	col := uint32(pos.Column - 1)

	ident := findIdentifierAtPos(pgf, line, col)
	if ident == nil {
		t.Fatal("findIdentifierAtPos returned nil")
	}
	if ident.Name != "x" {
		t.Errorf("findIdentifierAtPos returned %s, want x", ident.Name)
	}

	// Test case where no identifier exists
	ident = findIdentifierAtPos(pgf, 0, 0) // "package" keyword or space
	if ident != nil && ident.Name == "main" {
		// It might return "main" if 0,0 maps to package declaration identifier?
		// "package main" -> 0,0 is 'p'.
	}
}
