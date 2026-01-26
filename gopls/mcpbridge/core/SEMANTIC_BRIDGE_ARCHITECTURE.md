# Semantic Bridge Architecture

## Overview

The **Semantic Bridge** (`internal/golang/llm_bridge.go`) provides a direct path from LLM-friendly semantic queries to gopls internal APIs, **bypassing the LSP protocol layer entirely**.

## Architecture

```
┌─────────────────┐
│  LLM Query       │
│  SymbolLocator   │  (semantic: name, scope, kind)
└────────┬────────┘
         │
         ▼
┌────────────────────────────────────────┐
│  ResolveNode()                         │
│  → Stateful AST walk with parent      │
│  → Uses gopls types.Info              │
│  → Returns ast.Node + types.Object     │
│  → No protocol.Position needed!        │
└────────┬───────────────────────────────┘
         │
         ├──────────────────────┬──────────────────────┐
         ▼                      ▼                      ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  Implementation  │   │  References      │   │  Definition      │
│  (LLMImpl)       │   │  (future)        │   │  (future)        │
└────────┬────────┘   └─────────────────┘   └─────────────────┘
         │
         ▼
┌────────────────────────────────────────┐
│  SourceContext[]                        │
│  - File/Line/Col                         │
│  - Symbol + Signature                    │
│  - Code snippet                         │
│  - Documentation                        │
└────────────────────────────────────────┘
```

## Key Components

### 1. SymbolLocator (Input)
```go
type SymbolLocator struct {
    SymbolName   string // e.g., "HandleRequest"
    ContextFile  string // Absolute file path
    PackageName  string // Import alias: "json"
    ParentScope  string // Receiver: "*Server" or function: "Main"
    Kind         string // "function", "method", "struct"
    LineHint     int    // Fuzzy hint for disambiguation
}
```

### 2. ResolveNode() (Core Resolver)
**Purpose**: Map semantic intent → gopls internal types

**Key Features**:
- **Parent Scope Tracking**: Uses stack-based AST traversal to track enclosing functions
- **Type-Based Resolution**: Leverages `types.Info.Uses` and `types.Info.Defs`
- **Multi-Level Filtering**: Name → Package → Scope → Kind → LineHint
- **Confidence Scoring**: Picks the best match when multiple candidates exist

**Returns**:
```go
type ResolveNodeResult struct {
    Node         ast.Node      // AST node
    Pos          token.Pos     // Position
    Object       types.Object // Type info
    EnclosingFunc string        // Function containing this symbol
    IsDefinition bool          // Is this a definition?
}
```

### 3. LLMImplementation() (High-Level API)
**Purpose**: Find implementations without LSP

**Algorithm**:
1. Call `ResolveNode()` to get `types.Object`
2. Extract cursor and type information
3. Call internal `implementationsMsets()` directly
4. Extract source context for each implementation

**Returns**:
```go
type SourceContext struct {
    // File allows the LLM to know which file to modify later
    File        string

    // Line numbers only (columns removed for cleaner output)
    StartLine   int
    EndLine     int

    // Symbol name (e.g., "FileWriter")
    Symbol      string

    // Kind helps distinguish "struct", "interface", "method", "func"
    Kind        string

    // Full signature (e.g., "func (f *FileWriter) Write(p []byte) error")
    Signature   string

    // Documentation comment (if available)
    DocComment  string

    // Snippet is the HERO field - full code body
    Snippet     string
}
```

**Limitations**:
- Only finds implementations of interfaces defined in the codebase
- Does NOT work with standard library interfaces (io.Reader, error, fmt.Stringer)
- May not detect method promotion from embedded structs
- Requires the interface to be defined in the workspace

## Usage Examples

### Example 1: Find Function Implementation
```go
locator := api.SymbolLocator{
    SymbolName:  "ServeHTTP",
    ContextFile: "/path/to/server.go",
    ParentScope: "*Server",  // Method receiver
    Kind:        "method",
    LineHint:    42,
}

impls, err := golang.LLMImplementation(ctx, snapshot, locator)
for _, impl := range impls {
    fmt.Printf("Implementation: %s at %s:%d\n",
        impl.Symbol, impl.File, impl.StartLine)
}
```

### Example 2: Find Type Definition
```go
locator := api.SymbolLocator{
    SymbolName:  "Person",
    ContextFile: "/path/to/main.go",
    Kind:        "struct",
}

fh, _ := snapshot.ReadFile(ctx, protocol.URIFromPath(locator.ContextFile))
result, err := golang.ResolveNode(ctx, snapshot, fh, locator)

if result.IsDefinition {
    fmt.Printf("Found definition at %v\n", result.Pos)
}
```

## Comparison: Old vs New Approach

| Aspect | Old (LSP-based) | New (Semantic Bridge) |
|--------|----------------|-------------------|
| **Input** | `file:line:col` (error-prone) | `SymbolLocator` (semantic) |
| **Resolution** | Manual line/col parsing | Type-based (accurate) |
| **API Layer** | LSP protocol → gopls | Direct gopls internals |
| **Context Loss** | Position conversion errors | Full type information |
| **Error Rate** | High (race conditions) | Low (type-checked) |
| **Performance** | Double conversion | Single pass |

## Future Extensions

The semantic bridge pattern can be extended to other tools:

1. **References**: `LLMReferences(locator) []SourceContext`
2. **Callers/Callees**: `LLMCallHierarchy(locator) []SourceContext`
3. **Rename**: `LLMRename(locator, newName) []SourceContext`

All using the same `ResolveNode()` core resolver!

## Testing

```bash
# Test the semantic bridge
go test ./gopls/internal/golang -run TestLLMBridge

# Integration test with real codebase
cd gopls/mcpbridge/test/e2e
go test -run TestLLM*
```

## Benefits

1. **LLM-Friendly**: Uses semantic descriptions, not error-prone coordinates
2. **Accurate**: Type-based resolution (no off-by-one errors)
3. **Fast**: Single-pass resolution, no protocol conversions
4. **Unified**: Single `ResolveNode()` for all tools
5. **Robust**: Handles fuzzy inputs with confidence scoring
