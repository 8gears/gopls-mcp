package api

// SymbolLocator defines a semantic location for a code symbol.
//
// Unlike LSP "Location" (which relies on volatile File/Line/Column coordinates),
// SymbolLocator relies on stable semantic anchors (Names, Scopes, Kinds).
//
// DESIGN PHILOSOPHY:
// 1. "Identity over Coordinates": Finding "Function X in Struct Y" is safer than "Line 10".
// 2. "Hints over Constraints": Line numbers are treated as fuzzy search hints, not hard addresses.
// 3. "Visual over Logical": Inputs should be what the LLM *sees* in the text buffer.
type SymbolLocator struct {
	// SymbolName is the exact identifier name of the symbol to find.
	// This is the primary search key.
	//
	// Usage:
	// - For functions: "HandleRequest"
	// - For methods: "Start" (not "Server.Start")
	// - For variables: "retryCount"
	//
	// Example: "ServeHTTP"
	SymbolName string `json:"symbol_name" jsonschema:"The exact name of the function, struct, method, or variable to find. Do not include package prefixes (e.g., use 'Println', not 'fmt.Println')."`

	// ContextFile is the absolute path of the file where the LLM is currently reading
	// or where the reference to the symbol occurs.
	//
	// This is the "Anchor File". The MCP server uses this to:
	// 1. Resolve import aliases (e.g., knows that 'json' maps to 'encoding/json').
	// 2. Resolve local scopes (if the symbol is a local variable).
	//
	// Example: "/Users/dev/project/server/http.go"
	ContextFile string `json:"context_file" jsonschema:"The absolute path of the file you are currently reading or analyzing. This serves as the starting point for resolution."`

	// PackageIdentifier is the package name or alias *as seen in the ContextFile*.
	//
	// Usage:
	// - If the symbol is imported (e.g., "fmt.Println"), use "fmt".
	// - If the symbol is defined in the current package, leave empty.
	//
	// The MCP server will resolve this string to the full import path (e.g., "fmt" -> "fmt", "u" -> "github.com/google/uuid").
	//
	// Example: "json" (for encoding/json), "myapi" (for project/api/v2)
	PackageIdentifier string `json:"package_identifier,omitempty" jsonschema:"The package name or import alias if the symbol is imported (e.g., 'fmt', 'json'). Leave empty if the symbol is defined in the current package."`

	// ParentScope is the name of the enclosing structure used for disambiguation.
	//
	// Usage:
	// - For Methods: The Receiver Type name (e.g., "Server", "*User").
	// - For Local Variables: The enclosing Function name (e.g., "ProcessData").
	// - For Top-Level Functions/Structs: Leave empty.
	//
	// This is the "Fence". It tells the resolver to only look inside this scope.
	//
	// Example: "Server" (if looking for func (s *Server) Start), "main" (if looking for var cfg inside main)
	ParentScope string `json:"parent_scope,omitempty" jsonschema:"The name of the enclosing scope: the receiver name for methods (e.g., 'Server') or the function name for local variables. Leave empty for top-level symbols."`

	// Kind specifies the type of symbol to narrow down the search.
	//
	// Usage:
	// - Helps distinguish between a struct named "Config" and a function named "Config".
	// - Allowed values: "function", "method", "struct", "interface", "variable", "const".
	//
	// Example: "method"
	Kind string `json:"kind,omitempty" jsonschema:"The type of the symbol to filter results. Enum: function, method, struct, interface, variable, const."`

	// LineHint is an APPROXIMATE line number where the symbol might be located.
	//
	// Usage:
	// - Treat this as a "Search Optimization Hint", NOT a hard constraint.
	// - Useful when multiple identical symbols exist (e.g., closures, table-driven tests).
	// - The resolver will look around this line (e.g., +/- 50 lines).
	//
	// Example: 42
	LineHint int `json:"line_hint,omitempty" jsonschema:"The approximate line number where you see the symbol. Used only as a fuzzy search hint for disambiguation."`

	// SignatureSnippet is a short code snippet of the symbol's definition or usage.
	//
	// Usage:
	// - Used for "Hallucination Verification".
	// - If the resolver finds a symbol, it compares it against this snippet.
	// - If they don't look similar, the tool may return a warning or try finding a better match.
	//
	// Example: "func (s *Server) Start(ctx context.Context)"
	SignatureSnippet string `json:"signature_snippet,omitempty" jsonschema:"A distinct snippet of code (like the function signature) used to verify the match."`
}
