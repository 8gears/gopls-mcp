/*
Package gopls-mcp provides MCP (Model Context Protocol) server functionality for gopls.

This package enables Large Language Models (LLMs) to better understand Go codebases through
static analysis. It exposes gopls's existing analysis capabilities as MCP tools that can be
used to query project structure, symbols, dependencies, and documentation.

# Architecture

The gopls-mcp package is designed to integrate with gopls's existing architecture:

  - Session/View: Uses gopls's session and view concepts to manage workspace state
  - Snapshot: Leverages gopls's snapshot mechanism for consistent package state
  - Metadata: Reuses gopls's metadata package for package/module information
  - Symbol Extraction: Integrates with gopls's symbol extraction from internal/golang

# MCP Tools

The following MCP tools are provided:

  - get_go_env: Returns Go environment (version, GOROOT, GOBIN)
  - list_package_details: Returns detailed information about a specific package
  - list_stdlib_packages_symbols: Returns all stdlib packages with full symbol details
  - list_stdlib_packages: Returns all stdlib packages (docs only, no symbols)
  - fetch_project_build_required_modules: Returns all modules in go.mod
  - list_project_defined_packages: Returns all packages defined by the project
  - check_package_exists: Validates package existence in the project
  - check_package_symbol_exists: Validates symbol existence in a package

# Implementation Status

This package is currently a stub with TODOs indicating where gopls's existing
functionality should be integrated. The design goal is to reuse existing gopls
features rather than reimplementing them.

Key integration points:

  - internal/cache: Session, Snapshot, Metadata for package loading
  - internal/golang: DocumentSymbols, PackageSymbols for symbol extraction
  - internal/protocol: DocumentSymbol, Range, Position for LSP types
  - golang.org/x/tools/go/packages: Package loading (already used by gopls)
  - golang.org/x/tools/internal/gocommand: Go command execution

# Design Considerations

 1. Reuse over reinvention: This package should delegate to existing gopls
    functionality wherever possible rather than duplicating code.

 2. Session lifecycle: MCP server should integrate with gopls's session lifecycle
    rather than running as a separate process.

 3. Performance: Leverage gopls's caching and incremental analysis to avoid
    redundant work.

 4. Consistency: Use the same package loading and analysis as gopls to ensure
    consistent results with LSP features.

# Future Work

  - [ ] Implement handlers using gopls's internal/cache
  - [ ] Implement symbol extraction using internal/golang
  - [ ] Integrate with gopls's session/view lifecycle
  - [ ] Add tests using gopls's testing framework
  - [ ] Add documentation for each tool handler
  - [ ] Consider streaming responses for large packages
*/
package core
