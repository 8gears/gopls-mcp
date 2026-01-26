package core

//go:generate go run generate.go

import (
	"fmt"
	"io"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/api"
)

// tools is the registry of all MCP tools provided by goplsmcp.
// This includes both the original gopls-mcp tools and the integrated gopls MCP tools.
// Note: list_tools is handled separately to avoid initialization cycles.
var tools = []Tool{
	// ===== Original gopls-mcp Tools =====
	// Module and package discovery tools
	GenericTool[api.IListModules, *api.OListModules]{
		Name:        "list_modules",
		Description: "List current module and direct dependencies only. Returns module paths only (no packages). By default, transitive dependencies are excluded. Set direct_only=false to show all dependencies including transitive ones. Use this to understand the module structure before exploring packages.",
		Handler:     handleListModules, // implemented in handlers.go
		Doc: `list_modules allows the LLM to get all module dependencies of the current project.

This MCP tool offers a comprehensive overview of existing go.mod usage,
it covers the direct and indirect dependencies and a semantic understanding of go.mod directives.
The tool offers a roadmap of current project by separating scopes to internal submodules and third-party modules.

LLM should use this MCP tool to retrieve a well-structured roadmap for dependencies instead of reading go.mod by itself.
LLM should be careful that the third-party modules might be huge depending on current modules.
`,
	},
	GenericTool[api.IListModulePackages, *api.OListModulePackages]{
		Name:        "list_module_packages",
		Description: "List all packages in a given module. Returns package names and optionally documentation. Use this to discover packages within a module before exploring symbols.",
		Handler:     handleListModulePackages, // implemented in handlers.go
		Doc: `list_module_packages allows the LLM to get all packages of a given module.

This MCP tool offers a semantically accurate, fast, and token-saving way to discover packages.

Standard LLM workflows require multiple turns of 'ls' and file readings to finally understand the package structure.
The pain points of this procedure are:

(1) Attention Dilution: The output of raw file readings dilutes the LLM's attention—and attention is a scarce resource.
(2) Token Cost: It forces the model to consume tokens proportional to file length, just to find a package name.
(3) Low Precision: The understanding remains at a textual level, not a semantic-level accurate overview.

This tool offers a comprehensive and fast overview of module packages.
It effectively prevents the attention dilution and heavy token cost caused by reading files.
It performs best when you want to understand the module structure without looking into code details.

LLM should use this tool to get package information instead of reading project files manually.
It preserves the LLM's attention context, eliminates textual analysis hallucinations, and significantly saves tokens.
`},
	GenericTool[api.IListPackageSymbols, *api.OListPackageSymbols]{
		Name:        "list_package_symbols",
		Description: "List all defined symbols in a package. Returns symbols with optional documentation and function bodies. Use this to understand a package's API, find functions, or learn implementation patterns.",
		Handler:     handleListPackageSymbols, // implemented in handlers.go
		Doc: `list_package_symbols allows the LLM to get all exported symbols of a given package.

This MCP tool offers a semantically accurate, fast, and token-saving way to discover package APIs.

Standard LLM workflows require multiple turns of file readings and textual search to understand the symbols defined in a package.
The pain points of this procedure are:

(1) Attention Dilution: The output of full file content dilutes the LLM's attention with implementation details.
(2) Token Cost: It forces the model to consume tokens for function bodies and comments, just to find function signatures.
(3) Low Precision: The understanding relies on textual matching, which often misses cross-file definitions or struct methods.

This tool offers a comprehensive and structural overview of package symbols.
It effectively prevents the attention dilution and heavy token cost caused by reading entire files.
It performs best when you want to understand the package interface or locate specific symbols without reading the implementation.

LLM should use this tool to get symbol information instead of reading package files manually.
It preserves the LLM's attention context, eliminates textual analysis hallucinations, and significantly saves tokens.
`,
	},

	// ===== Integrated gopls MCP Tools =====
	// These wrap the existing implementations from gopls/internal/mcp
	// Origin: gopls/internal/mcp/*.go

	GenericTool[api.IGetPackageSymbolDetailParams, *api.OGetPackageSymbolDetailResult]{
		Name:        "get_package_symbol_detail",
		Description: "Get detailed symbol information from a package. Returns Symbol objects with name, kind, signature, receiver (for methods), parent (for fields), documentation, and optional bodies. Symbol filters are REQUIRED - provide symbol_filters to retrieve specific symbols by name and receiver (e.g., [{name: \"Start\", receiver: \"*Server\"}]). For methods, receiver matching uses exact string match (e.g., \"*Server\" != \"Server\"). Use include_docs=true for documentation and include_bodies=true for function implementations. Use list_package_symbols to get all symbols in a package.",
		Handler:     handleGetPackageSymbolDetail, // wrapper for outlineHandler()
		Doc: `get_package_symbol_detail is a precision tool to get specific symbol details only.

This MCP tool offers lightweight, fast and fine-grained details for interested symbols.

Standard LLM workflows go through every file and do textual search to get selected symbol details,
the pain points are:

(1) Attention Dilution: it shares the same problem of list_package_symbols,
and after context grows, the LLM loses the attention of previous full list_package_symbols responses,
and requires another work for interested symbols, and this further dilutes the attention in a long thread.
The frequent symbol finding populates the context.
(2) Token Cost: It forces the model to consume tokens for function bodies and comments just to find several symbols only.
(3) Low Precision: The understanding relies on textual matching, which often misses cross-file definitions or struct methods.

LLM should use this tool to get symbol details of package instead of reading package files manually.
`,
	},
	GenericTool[api.IWorkspaceParams, *api.OWorkspaceResult]{
		Name:        "go_workspace",
		Description: "Get workspace information (module path, type). Use this for initial project overview.",
		Handler:     handleGoWorkspace, // wrapper for workspaceHandler()
		Doc: `go_workspace allows the LLM to understand the current workspace configuration.

This MCP tool offers a quick overview of the module path and build settings.

Standard LLM workflows might try to read go.mod or go.work files to understand the workspace.
The pain points of this procedure are:

(1) Attention Dilution: Reading configuration files might distract from the main code.
(2) Token Cost: Configuration files can be verbose.

This tool offers a concise summary of the workspace.

LLM should use this tool to get an initial understanding of the project structure.
`,
	},

	GenericTool[api.IDiagnosticsParams, *api.ODiagnosticsResult]{
		Name:        "go_diagnostics",
		Description: "Check for compilation and type errors. FAST: uses incremental type checking (faster than 'go build'). Use this to verify code correctness and populate the workspace cache for other tools. Returns detailed error information with file/line/column.",
		Handler:     handleGoDiagnostics, // wrapper for workspaceDiagnosticsHandler()
		Doc: `go_diagnostics allows the LLM to check syntax and type errors in a lightweight and fast way.

Standard LLM workflows verify code change works by triggering 'go build' by shell command
to ensure syntax and type usages are correct.
However, 'go build' is too heavy in this case because it generates the real executable code we don't need.
This slows down the feedback to the LLM and user-facing latency is high.

This MCP tool offers a lightweight tool to satisfy the LLM verification requirement after code change,
by eliminating compiler backend code generation, it offers a faster tool to meet the requirement.

LLM should use this tool to verify code change instead of using 'go build' to make the latency smaller
to gain a fluent user experience.
`,
	},

	GenericTool[api.ISearchParams, *api.OSearchResult]{
		Name:        "go_search",
		Description: "Find symbols (functions, types, constants) by name with fuzzy matching. Use this when user knows part of a symbol name but not the full name or location. Returns rich symbol information (name, kind, file, line) for fast exploration.",
		Handler:     handleGoSearch, // wrapper for searchHandler()
		Doc: `go_search allows the LLM to fuzzy find symbols by name.

This MCP tool offers a flexible way to locate symbols when the exact name is unknown.

Standard LLM workflows might resort to 'grep' or repetitive file reading to find symbols.
The pain points of this procedure are:

(1) Attention Dilution: Grep output can be noisy and lack context.
(2) Token Cost: Reading many files to find a symbol is expensive.
(3) Low Precision: Textual search lacks semantic understanding.

This tool uses gopls's index for fast and relevant results.

OUTPUT:
  Returns rich Symbol information including:
  - name: Symbol name
  - kind: Symbol kind (function, method, struct, interface, etc.)
  - file_path: Source file path
  - line: Line number where symbol is defined
  - signature: NOT included (use go_definition for full details)
  - doc: NOT included (use go_definition for full details)
  - body: NOT included (use go_definition for full details)

NOTE: For full signature and documentation, use go_definition on the specific symbol.
This tool is optimized for fast exploration and discovery.

LLM should use this tool when exploring the codebase or looking for specific functionality.
`,
	},

	GenericTool[api.ISymbolReferencesParams, *api.OSymbolReferencesResult]{
		Name:        "go_symbol_references",
		Description: "Find all usages of a symbol across the codebase using semantic location (symbol name, package, scope). Use this before refactoring to assess impact or to understand how a symbol is used. REPLACES: grep + manual file reading for finding references.",
		Handler:     handleGoSymbolReferences, // uses semantic bridge (resolveSymbolLocatorToPosition)
		Doc: `go_symbol_references allows the LLM to get all usages of a symbol under current project.

Standard LLM workflows cannot handle this as it requires textual matches over almost every file under the current project.
It greatly dilutes LLM attention and costs a lot of tokens, and it's quite slow due to the nature of IO and matching.

This MCP tool offers fast, lightweight and accurate results to report all usages of a given symbol,
and it can fully prevent LLM mixed the similar symbols as the same one due to textual matching nature.

It makes the LLM get rid of tedious and heavy file reading and matching workflow,
and makes the LLM focus on the user assignments instead of getting lost in retrieving information.

LLM should use this tool to retrieve symbol usages to understand how to use a symbol,
evaluate the impacts of a symbol, and so on. LLM should never consider doing this stuff by itself.

INPUT FORMAT:
  Use semantic locator instead of error-prone file/line/column numbers:
  - symbol_name: The symbol name to find references for (e.g., "HandleRequest", "Start")
  - context_file: File where the symbol is used (absolute path)
  - package_name: Package import alias (e.g., "http", "fmt") - optional
  - parent_scope: Receiver type for methods (e.g., "*Server") - optional
  - kind: Symbol kind ("function", "method", "variable", "const") - optional
  - line_hint: Approximate line number for disambiguation - optional

EXAMPLE:
  Find all references to a function:
  {
    "locator": {
      "symbol_name": "HandleRequest",
      "context_file": "/path/to/handler.go",
      "kind": "function"
    }
  }

OUTPUT:
  Returns both:
  - references: List of reference locations (file, line, column)
  - symbols: Rich symbol information (signature, documentation, snippet) for the referenced symbol
		`,
	},

	GenericTool[api.IRenameSymbolParams, *api.ORenameSymbolResult]{
		Name:        "go_dryrun_rename_symbol",
		Description: "Preview a symbol rename operation across all files (DRY RUN - no changes are applied). Use go_symbol_references first to assess impact, then use this to preview the exact changes that would be made. Returns a unified diff showing all proposed modifications.",
		Handler:     handleGoRenameSymbol, // wrapper for renameSymbolHandler()
		Doc: `go_dryrun_rename_symbol allows the LLM to get semantic-level suggestions for rename symbol action.

Standard LLM renaming workflows is an evolving process; it consistently finds symbols by textual matches,
renames them and runs 'go build' to ensure whether it succeeds. If not, it will check the failure and repeat until succeeds.
This causes unnecessary context growth and lacks the ability to give the user a correct and accurate overview of renaming impact.
Besides, it dilutes attention and costs tokens.

This MCP tool offers a fast, dry-run output for the LLM to understand all necessary changes for a renaming action.
It's fast and semantically correct.

LLM should use this tool to evaluate renaming impacts and based on the results to rename,
instead of trying to analyze and change the findings by itself.
`,
	},

	// todo: what does the metadata mean? and the context is quite vague,
	// from user view and llm view, it's too internal.
	// also, the return summary is quite vague as well.
	GenericTool[api.IFileContextParams, *api.OFileContextResult]{
		Name:        "go_file_context",
		Description: "Get file metadata (package name, path). FAST: uses cached metadata with minimal IO. Use this first when you only need package info, then use go_read_file only if you need the actual file content.",
		Handler:     handleGoFileContext, // wrapper for fileContextHandler()
		Doc: `go_file_context allows the LLM to get metadata about a file without reading its full content.

This MCP tool offers a low-cost way to check package names and paths.

Standard LLM workflows often read the full file just to check the package declaration.
The pain points of this procedure are:

(1) Token Cost: Reading the full file is wasteful if only metadata is needed.
(2) Attention Dilution: Full content crowds the context window.

This tool provides just the essential metadata.

LLM should use this tool to verify file identity before reading it fully.
`,
	},

	// todo: let's rethink about the location, can LLM give us a correct location?
	// or we can think about what's better.
	// todo: what happened if the location symbol is not an interface?
	//
	// todo: regarding on the response, is the location enough?
	// location means file + line + column, but it requires further reading.
	// do we want to handle this or leave it to LLM to read the file again?
	// it's very likely for LLM to mis-understand the location and read wrong code.
	// I feel we should return more, location is too vague, let's return snippet.
	GenericTool[api.IImplementationParams, *api.OImplementationResult]{
		Name:        "go_implementation",
		Description: "Find all implementations of an interface or all interfaces implemented by a type using semantic location (symbol name, package, scope). Use this to understand type hierarchies, find all implementations of an interface, or discover design patterns in the codebase. REPLACES: grep + manual file reading for interface implementations.",
		Handler:     handleGoImplementation, // uses semantic bridge (golang.LLMImplementation)
		Doc: `go_implementation allows the LLM to retrieve implementation of an interface.

Standard LLM is blind when trying to find implementations of an interface due to Go interface nature.
It goes through files and textual matches structures. As interface implementation is type sensitive
and the implementation may be put in different package files, LLM needs a lot of attention and tokens
to handle it but output an incomplete or wrong result.

This MCP tool offers a fast, accurate and correct output to find implementations of an interface.
LLM MUST use this MCP tool rather than trying to find implementations by itself.

INPUT FORMAT:
  Use semantic locator instead of error-prone line/column numbers:
  - symbol_name: The interface or type name (e.g., "Handler", "Server")
  - context_file: File where the symbol is used (absolute path)
  - package_name: Package import alias (e.g., "http", "fmt") - optional
  - parent_scope: Receiver type for methods (e.g., "*Server") - optional
  - kind: Symbol kind ("interface", "type") - optional, helps disambiguate
  - line_hint: Approximate line number for disambiguation - optional

OUTPUT FORMAT:
  Returns a list of symbols with rich information:
  - name: The implementing type or method name (e.g., "FileWriter", "Write")
  - signature: Full function signature (e.g., "func (f *FileWriter) Write(data string) error")
  - file_path: Absolute path to the implementation file
  - line: Starting line number
  - kind: Symbol kind ("struct", "interface", "method", etc.)
  - doc: Documentation comment (if available)
  - body: Full code snippet of the implementation (HERO field)

COMMON PITFALLS:
  ✗ Standard library interfaces: io.Reader, error, fmt.Stringer don't work
    → Only interfaces defined in your codebase are supported
  ✗ Wrong context_file: Pointing to usage instead of definition
    → Point context_file to where the interface is defined
  ✗ Missing parent_scope for methods
    → For interface methods, set parent_scope to the interface name
  ✗ Empty result doesn't mean error
    → Check if the interface actually has implementations

EXAMPLES:
  1. Find implementations of an interface:
  {
    "locator": {
      "symbol_name": "Handler",
      "context_file": "/path/to/interfaces.go",
      "kind": "interface"
    }
  }
  → Returns: FileHandler, MemoryHandler, MockHandler, etc.

  2. Find implementations of an interface method:
  {
    "locator": {
      "symbol_name": "ServeHTTP",
      "parent_scope": "Handler",
      "kind": "method",
      "context_file": "/path/to/handler.go"
    }
  }
  → Returns: All ServeHTTP method implementations

  3. Find what interfaces a type implements:
  {
    "locator": {
      "symbol_name": "FileWriter",
      "context_file": "/path/to/file.go",
      "kind": "struct"
    }
  }
  → Returns: Writer, io.Closer, etc.
`,
	},

	// todo: is it necessary to support partially read?
	// e.g., read first N lines, or read from line X to line Y,
	// it will greatly reduce token cost and attention dilution.
	GenericTool[api.IReadFileParams, *api.OReadFileResult]{
		Name:        "go_read_file",
		Description: "Read file content through gopls. SLOWER: reads full file from disk. Use go_file_context first for quick metadata check, then use this only when you need to see actual code or implementation details. Note: unsaved editor changes not included.",
		Handler:     handleGoReadFile, // wrapper for snapshot.ReadFile()
		Doc: `go_read_file offers a lightweight file reading without involving disk IO.

As this MCP server natively loads files into memory and does further analysis, it offers a faster way to read file content.
`,
	},

	// Navigation tools
	GenericTool[api.IDefinitionParams, *api.ODefinitionResult]{
		Name: "go_definition",
		// TODO: description is too long, don't use one line.
		Description: "Jump to the definition of a symbol using semantic location (symbol name, package, scope). REPLACES: grep + manual file reading. Use this when you see a function call or type reference and need to find where it's defined. Faster and more accurate than text search - uses type information from gopls.",
		Handler:     handleGoDefinition, // wrapper for golang.Definition()
		Doc: `go_definition offers LSP feature to return function declaration details.
It's designed for LLM to use during reading code, unlike get_package_symbol_detail which is used for exploring.

Standard LLM workflow to read code requires to keep locating and loading files to understand function calls across a lot of packages.
This requires to frequently locate and load files and parse them via a textual matching way.

This MCP tool offers an approach of retrieving precise definitions when the LLM focuses on code reading in a file.
It doesn't require the LLM to quickly switch via multiple files to understand functions and symbols it needs to proceed.

LLM should use it to get more precise information during reading code, instead of reading files and parsing them by the LLM itself.

The tool uses SymbolLocator for LLM-friendly input:
- symbol_name: The exact name of the symbol (e.g., "HandleRequest", not "fmt.Println")
- context_file: The absolute path of the file where you see the symbol
- package_name: Optional import alias if the symbol is imported (e.g., "fmt", "json")
- parent_scope: Optional enclosing scope (receiver type for methods, function name for local variables)
- kind: Optional symbol kind filter ("function", "method", "struct", "interface", "variable", "const")
- line_hint: Optional approximate line number for disambiguation (treated as fuzzy search hint)

Example: To find definition of "Add" function call in main.go:
{
  "symbol_name": "Add",
  "context_file": "/path/to/main.go"
}
`,
	},

	// ===== Call Hierarchy Tools =====

	GenericTool[api.ICallHierarchyParams, *api.OCallHierarchyResult]{
		Name:        "get_call_hierarchy",
		Description: "Get the call hierarchy for a function using semantic location (symbol name, package, scope). Returns both incoming calls (what functions call this one) and outgoing calls (what functions this one calls). Use this to understand code flow, debug call chains, and trace execution paths through the codebase. REPLACES: grep + manual file reading for call graph analysis.",
		Handler:     handleGoCallHierarchy, // uses semantic bridge (golang.ResolveNode)
		Doc: `get_call_hierarchy allows the LLM to explore the call graph of a function.

This MCP tool offers a powerful way to understand code flow and dependencies.

Standard LLM workflows struggle to trace call chains across multiple files.
The pain points of this procedure are:

(1) Attention Dilution: Manually tracking calls requires keeping many files in context.
(2) Token Cost: Reading all involved files consumes significant tokens.
(3) Complexity: It is hard to maintain a mental model of deep call stacks.

This tool provides a structured view of incoming and outgoing calls.

LLM should use this tool to debug, refactor, or understand complex logic.

INPUT FORMAT:
  Use semantic locator instead of error-prone line/column numbers:
  - symbol_name: The function name (e.g., "HandleRequest", "Main")
  - context_file: File where the function is called or defined (absolute path)
  - package_name: Package import alias (e.g., "http") - optional
  - parent_scope: Receiver type for methods (e.g., "*Server") - optional
  - kind: Symbol kind ("function", "method") - optional, helps disambiguate
  - line_hint: Approximate line number for disambiguation - optional

DIRECTION:
  - "incoming": Show what functions call this function
  - "outgoing": Show what this function calls
  - "both" (default): Show both directions

EXAMPLE:
  Get call hierarchy for a function:
  {
    "locator": {
      "symbol_name": "HandleRequest",
      "context_file": "/path/to/server.go",
      "parent_scope": "*Server",
      "kind": "method"
    },
    "direction": "both"
  }
`,
	},

	// ===== New Discovery Tools =====

	GenericTool[api.IAnalyzeWorkspaceParams, *api.OAnalyzeWorkspaceResult]{
		Name:        "analyze_workspace",
		Description: "Analyze the entire workspace to discover packages, entry points, and dependencies. Use this when exploring a new codebase to understand the project structure, find main packages, API endpoints, and get a comprehensive overview of the codebase.",
		Handler:     handleAnalyzeWorkspace, // new tool for workspace discovery
		Doc: `analyze_workspace allows the LLM to get a comprehensive overview of the project.

This MCP tool offers a high-level map of packages, entry points, and dependencies.

Standard LLM workflows lack a good way to "see" the whole project at once.
The pain points of this procedure are:

(1) Exploration Overhead: It takes many steps to discover the project structure manually.
(2) Missing Context: It is easy to miss important parts of the codebase.

This tool provides a starting point for exploration.

LLM should use this tool when first encountering a new codebase.
`,
	},

	GenericTool[api.IGetStarted, *api.OGetStarted]{
		Name:        "get_started",
		Description: "Get a beginner-friendly guide to start exploring the Go project. Returns project identity, quick stats, entry points, package categories, and recommended next steps. Use this when you're new to a codebase and want to understand where to start.",
		Handler:     handleGetStarted, // new tool for getting started
		Doc: `get_started allows the LLM to get a beginner-friendly guide to the project.

This MCP tool offers curated information to help the LLM (and user) hit the ground running.

Standard LLM workflows require the user to explain the project or the LLM to guess where to start.
The pain points of this procedure are:

(1) Ramp-up Time: It takes time to figure out the "what" and "how" of a project.
(2) Lack of Direction: Without guidance, exploration can be aimless.

This tool provides a structured introduction.

LLM should use this tool to orient itself and the user.
`,
	},

	// ===== Dependency Analysis Tools =====

	GenericTool[api.IDependencyGraphParams, *api.ODependencyGraphResult]{
		Name:        "get_dependency_graph",
		Description: "Get the dependency graph for a package. Returns both dependencies (packages it imports) and dependents (packages that import it). Use this to understand architectural relationships, analyze coupling, and visualize the package's place in the codebase.",
		Handler:     handleGetDependencyGraph, // new tool for dependency graph analysis
		Doc: `get_dependency_graph allows the LLM to visualize package dependencies.

This MCP tool offers a clear view of how packages relate to each other.

Standard LLM workflows can infer dependencies from imports but lack a holistic view.
The pain points of this procedure are:

(1) Limited Scope: It is hard to see the "big picture" of architecture from individual files.
(2) Coupling Analysis: Detecting cycles or tight coupling is difficult manually.

This tool provides a graph structure of dependencies.

LLM should use this tool to understand architecture and impact of changes.
`,
	},
}

// RegisterTools registers all tools with the MCP server.
// The handler provides access to gopls's session and snapshot for all tool implementations.
// Integration point: called from gopls/internal/cmd/mcp.go or gopls/internal/mcp/mcp.go
func RegisterTools(server *mcp.Server, handler *Handler) {
	// Register the list_tools meta-tool first (special case to avoid init cycle)
	GenericTool[api.IListToolsParams, *api.OListToolsResult]{
		Name:        "list_tools",
		Description: "List all available gopls-mcp tools with documentation and parameter schemas. These semantic analysis tools are more accurate (type-aware vs text matching), faster after warm-up (17x-587x per operation, breaks even after 3 queries), and save tokens compared to text-based search by leveraging gopls's cached type information.",
		Handler:     handleListTools,
	}.Register(server, handler)

	// Register all other tools
	for _, tool := range getTools() {
		tool.Register(server, handler)
	}
}

// GenerateReference writes the complete tool reference documentation to the provided writer.
// This is called by generate.go via go:generate and by tests for validation.
// It always writes the full documentation from scratch.
func GenerateReference(w io.Writer) error {
	// Write header and section

	fmt.Fprintf(w, `---
title: Reference
sidebar:
  order: 1
---
`)

	// Write each tool as a ### title with description
	for _, tool := range tools {
		name, des := tool.Details()
		fmt.Fprintf(w, "### `%s`\n\n", name)
		fmt.Fprintf(w, "> %s\n\n", des)
		fmt.Fprintf(w, "%s\n\n", tool.Docs())
	}

	return nil
}

// UpdateReference updates the reference.md with the current tool list.
// This is called by generate.go via go:generate.
// Deprecated: Use GenerateReference with io.Writer directly.
func UpdateReference(content string) string {
	var builder strings.Builder
	GenerateReference(&builder)
	return builder.String()
}

// Note: Tool handler implementations are in handlers.go
// to keep this file focused on registration and metadata.
