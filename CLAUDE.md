# CLAUDE.md - AI Agent Guidelines

## 🧠 Core Philosophy
**You are an expert Go developer utilizing the `gopls-mcp` server.**
Your primary goal is to use semantic analysis tools instead of crude text search.
This ensures accuracy, saves tokens, and mimics how human experts use IDEs.

## 🚫 Restrictions (IMPORTANT - Read This First!)

### NEVER Use These Commands:
| ❌ Forbidden Command | ✅ Use This Instead | Why |
|---------------------|---------------------|-----|
| `grep -rn "func MyFunc" .` | `go_search("MyFunc")` | Grep uses text matching; go_search uses semantic understanding with fuzzy matching |
| `grep -rn "MyType" .` | `go_search("MyType")` | Finds the symbol definition with file:line, signature, and docs |
| `grep -rn "myFunction" .` | `go_symbol_references(...)` | Finds all usages, not just text matches (avoids false positives) |
| `find . -name "*.go"` | `list_module_packages()` | Semantic package discovery |
| `go build` | `go_build_check()` | ~500x faster via incremental type checking |

### Critical Rules:
1. **DO NOT use grep/ripgrep/find for ANY symbol-related searches** - This includes finding functions, types, methods, constants, variables, or their usages
2. **DO NOT use go build to check compilation** - Use `go_build_check` instead
3. **DO NOT read full files just to find a function signature** - Use `go_definition` or `go_search`
4. **DO NOT assume grep is faster** - After cache warmup, semantic tools are just as fast and more accurate

## 🛠 Preferred Tool Workflows

### 1. Discovery & Navigation
| Goal | Tool | Why |
|---|---|---|
| **Start Here** | `get_started` | Overview of the project, entry points, and next steps. |
| **Map Project** | `analyze_workspace` | High-level map of packages and dependencies. |
| **Find Packages** | `list_module_packages` | Discover all packages in a module (useful when exploring new codebases). |
| **Find Symbol** | `go_search` | Fuzzy find functions/types by name (semantic). |
| **Explore Pkg** | `list_package_symbols` | List all symbols in a package (replaces `ls` + read). |

### 2. Reading & Understanding
| Goal | Tool | Why |
|---|---|---|
| **Go to Def** | `go_definition` | Jump directly to code definition (replaces `grep`). |
| **Find Usage** | `go_symbol_references` | Find callers/usages (semantic, not text matching). |
| **Implementations**| `go_implementation`| Find interface implementations (Crucial for Go). |
| **Check API** | `go_package_api` | See public interface of a package or dependency. |
| **Quick Docs** | `go_hover` | Read doc comments and signatures instantly. |
| **Hierarchy** | `get_call_hierarchy` | Trace incoming/outgoing calls to understand flow. |
| **Dependencies**| `get_dependency_graph`| Visualize package imports and dependents. |

### 3. Verification (CRITICAL)
| Goal | Tool | Why |
|---|---|---|
| **Check Errors/Build**| `go_build_check` | **REPLACES `go build`.** Run after edits to verify compilation. ~500x faster than `go build` via incremental type checking. Returns detailed errors with file/line/column. |
| **Dry Run Refactor** | `go_rename_symbol` | Preview refactors safely before applying changes. |

## 🏗️ Repository Context: gopls-mcp

This repository contains the `gopls-mcp` server source code, which wraps the official `gopls`.

*   **Project Root:** `gopls/mcpbridge` (Focus your work here)
*   **Build Command:** `go build .`
*   **Test Command:** `go test ./gopls/mcpbridge/...`

### 🧑‍💻 Contributing / Extending gopls-mcp
If you are adding a new tool to the server:
1.  **Implement Handler:** Add the logic in `gopls/mcpbridge/core/handlers.go` (or a new file in `core/`).
2.  **Register Tool:** Add the `GenericTool` definition to `tools` slice in `gopls/mcpbridge/core/server.go`.
3.  **Update Docs:** Run `go generate ./gopls/mcpbridge/core` to update the README automatically.

## 🎯 Common Patterns & Examples

### Pattern 1: Finding Functions by Name (FUZZY MATCHING!)
**Scenario**: You need to find functions related to "formatting symbols"

❌ **WRONG** (text search):
```bash
grep -rn "format.*Symbol" mcpbridge/
```

✅ **CORRECT** (semantic fuzzy search):
```
go_search(query="formatSymbol")
# Returns: formatPackageSymbols, formatPackageSymbolDetail, FormatSymbolSummary
# Each with: file path, line number, signature, documentation
```

**Key Insight**: `go_search` supports fuzzy matching! You don't need exact names.

---

### Pattern 2: Finding All Usages of a Symbol
**Scenario**: You need to find everywhere `Handler` type is used

❌ **WRONG** (text search - gives false positives):
```bash
grep -rn "Handler" .
# Matches: "Handler", "handler", "myHandler", comment text, etc.
```

✅ **CORRECT** (semantic search - exact matches only):
```
go_symbol_references(
    symbol_name="Handler",
    context_file="mcpbridge/core/handlers.go"
)
# Returns: Only actual references to the Handler type
# With: file:line:column, code context, full symbol info
```

---

### Pattern 3: Understanding a Function's Implementation
**Scenario**: You need to understand what `handleGoDefinition` does

❌ **WRONG** (grep + read file):
```bash
grep -rn "func handleGoDefinition" .
# Then read the entire file
```

✅ **CORRECT** (jump to definition):
```
go_definition(
    symbol_name="handleGoDefinition",
    context_file="mcpbridge/core/gopls_wrappers.go",
    line_hint=571
)
# Returns: Full function signature, documentation, implementation body
```

---

### Pattern 4: Exploring Package Contents
**Scenario**: You want to understand what's in the `core` package

❌ **WRONG** (list + read files):
```bash
ls mcpbridge/core/*.go
# Then read each file
```

✅ **CORRECT** (semantic package overview):
```
go_list_package_symbols(
    package_path="golang.org/x/tools/gopls/mcpbridge/core",
    include_docs=true
)
# Returns: All exported symbols with signatures and documentation
```

---

### Pattern 5: Tracing Function Call Chains
**Scenario**: You need to understand the call flow

❌ **WRONG** (grep for function calls):
```bash
grep -rn "snapshot()" .
# Misses indirect calls, interface calls, stdlib calls
```

✅ **CORRECT** (semantic call graph):
```
go_get_call_hierarchy(
    locator={
        symbol_name: "handleGoDefinition",
        context_file: "mcpbridge/core/gopls_wrappers.go",
        line_hint: 571
    },
    direction: "outgoing"
)
# Returns: All functions this calls, with signatures and documentation
```

---

## 🔍 Mental Model Shift

### Old Mental Model (Text-Based):
```
I want to find → grep
I want to read → cat/read
I want to understand → analyze
```

### New Mental Model (Semantic):
```
I want to find symbols → go_search (fuzzy matching!)
I want to see usage → go_symbol_references
I want to understand → go_definition + go_hover
I want to explore → go_list_package_symbols
I want to trace → go_get_call_hierarchy
```

**Remember**: `go_search` REPLACES grep for ANY symbol-related search, including pattern matching!

---

## 💡 Tips & Edge Cases

### File Reading (Use Sparingly)
Most of the time, you don't need to read full files. Use semantic tools instead:
- Use `go_read_file` only when you need to see actual implementation details
- Prefer `go_definition` or `go_list_package_symbols` for understanding code structure
- Reading files should be the LAST resort, not the first step

### When to Use Grep (Rare Cases ONLY)
The ONLY acceptable use cases for grep/ripgrep:
- Searching non-Go files (config files, markdown, logs)
- Searching for specific text strings (not symbols)
- Searching build output or error messages

**If you're searching Go code for symbols → USE SEMANTIC TOOLS**

### Module Debugging
- Use `list_modules` to debug module dependency problems

---

## 🚀 Quick Decision Guide

### Tool Categories

| 🔍 FIND / DISCOVER | 📊 ANALYZE / UNDERSTAND | ⚡ CHECK / VERIFY | 🔨 REFACTOR / CHANGE |
|-------------------|----------------------|------------------|-------------------|
| `go_search` | `go_definition` | `go_build_check` | `go_dryrun_rename_symbol` |
| `go_symbol_references` | `go_get_call_hierarchy` | | |
| `go_implementation` | `go_get_dependency_graph` | | |
| `go_list_package_symbols` | `go_get_package_symbol_detail` | | |
| `go_list_module_packages` | `go_get_started` | | |
| `go_list_modules` | | | |

### "I want to..." → **Use this tool**

| I want to... | Tool | Don't Use |
|-------------|------|-----------|
| Find a function/type by name | `go_search()` | grep |
| Find where a symbol is used | `go_symbol_references()` | grep |
| Jump to a symbol's definition | `go_definition()` | grep + read |
| Find interface implementations | `go_implementation()` | grep |
| See what a function calls | `go_get_call_hierarchy(direction="outgoing")` | grep |
| See what calls a function | `go_get_call_hierarchy(direction="incoming")` | grep |
| List all symbols in a package | `go_list_package_symbols()` | ls + read |
| Check for compilation errors | `go_build_check()` | go build |
| Preview a rename operation | `go_dryrun_rename_symbol()` | Manual grep |
| Understand package dependencies | `go_get_dependency_graph()` | grep "import" |
| Get started with a new codebase | `go_get_started()` | Read README |
| See all packages in a module | `go_list_module_packages()` | find |

---

## 📚 Example Conversations (Few-Shot Learning)

### Example 1: Finding Where a Function is Defined
```
❌ BAD INTERACTION:
User: "Where is ResolveNode defined?"
Assistant: [Uses Grep] → grep -rn "func ResolveNode" .

✅ GOOD INTERACTION:
User: "Where is ResolveNode defined?"
Assistant: [Uses go_definition]
go_definition(
    symbol_name="ResolveNode",
    context_file="internal/golang/llm_bridge.go"
)
→ Returns: Definition at llm_bridge.go:412 with signature and docs
```

### Example 2: Finding All Usages
```
❌ BAD INTERACTION:
User: "Show me all places that call Start()"
Assistant: [Uses Grep] → grep -rn "Start(" .

✅ GOOD INTERACTION:
User: "Show me all places that call Start()"
Assistant: [Uses go_symbol_references]
go_symbol_references(
    symbol_name="Start",
    parent_scope="Server",
    kind="method"
)
→ Returns: 3 usages with file:line:column and code context
```

### Example 3: Understanding a Package
```
❌ BAD INTERACTION:
User: "What's in the core package?"
Assistant: [Reads all files] → Read handlers.go, Read server.go, Read ...

✅ GOOD INTERACTION:
User: "What's in the core package?"
Assistant: [Uses go_list_package_symbols]
go_list_package_symbols(
    package_path="golang.org/x/tools/gopls/mcpbridge/core",
    include_docs=true
)
→ Returns: 47 exported symbols with signatures and docs
```

### Example 4: Finding Related Functions
```
❌ BAD INTERACTION:
User: "Find all functions related to formatting symbols"
Assistant: [Uses Grep] → grep -rn "format.*Symbol"

✅ GOOD INTERACTION:
User: "Find all functions related to formatting symbols"
Assistant: [Uses go_search with fuzzy matching]
go_search(query="formatSymbol")
→ Returns: formatPackageSymbols, formatPackageSymbolDetail, FormatSymbolSummary
```

---

## ⚡ Performance Notes

- **First query**: ~200-500ms (cache warmup)
- **Subsequent queries**: ~50-100ms (cache is warm)
- **go_build_check**: ~500x faster than `go build` due to incremental type checking
- **Semantic accuracy**: 100% (no false positives from text matching)

**The cache warmup cost is paid once. All subsequent queries are fast.**