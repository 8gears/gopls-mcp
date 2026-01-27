---
title: Configuration
---

gopls-mcp supports configuration through a JSON file, allowing you to customize both gopls-mcp specific behavior and native gopls settings.

## Configuration Methods

### 1. Command-Line Flag

```bash
gopls-mcp -config /path/to/config.json
```

### 2. MCP Initialization

When starting the MCP server via your MCP client (e.g., Claude Desktop), you can pass configuration in the initialization parameters:

```json
{
  "gopls-mcp": {
    "gopls": {
      "staticcheck": true
    },
    "max_response_bytes": 64000
  }
}
```

## Configuration Structure

```json
{
  "gopls": {
    // Native gopls settings (passed directly to gopls)
    // See https://go.dev/gopls/settings for all available options
  },
  "workdir": "/path/to/go/project",
  "max_response_bytes": 32000  // Global response size limit in bytes
}
```

## Configuration Options

### gopls

**Type**: `map[string]any`
**Default**: `{}`

Native gopls configuration options. These are passed directly to gopls's internal settings system. All standard gopls options are supported.

> **For a complete list of gopls settings, see the [official gopls documentation](https://go.dev/gopls/settings).**

Common options include:

- `analyses` - Enable/disable analyses (e.g., `{"unusedparams": true}`)
- `staticcheck` - Enable staticcheck (default: `false`)
- `buildFlags` - Build tags and flags (e.g., `["-tags=integration"]`)
- `env` - Environment variables for builds
- `directoryFilters` - Exclude directories from indexing
- `experimentalPostfixCompletions` - Enable postfix completions
- `noIncrementalSync` - Disable incremental sync
- `completeUnimported` - Enable completion for unimported packages
- `matcher` - Type matching algorithm (`caseInsensitive`, `caseSensitive`, `fuzzy`)

### workdir

**Type**: `string`
**Default**: Current directory

The working directory for the Go project. This can also be set via the `-workdir` command-line flag.

### max_response_bytes

**Type**: `integer`
**Default**: `32000` (32KB)

Global maximum response size in bytes. All tools respect this limit automatically. When a response exceeds this limit, it is truncated and includes metadata indicating the truncation.

## Examples

### Basic Configuration

```json
{
  "workdir": "/path/to/my/project"
}
```

### Enable Static Analysis

```json
{
  "gopls": {
    "staticcheck": true,
    "analyses": {
      "unusedparams": true,
      "unusedwrite": true
    }
  }
}
```

### Build Tags and Response Limits

```json
{
  "gopls": {
    "buildFlags": ["-tags=integration", "-tags=dev"]
  },
  "max_response_bytes": 64000
}
```

### Project-Specific Settings

```json
{
  "workdir": "/path/to/project",
  "gopls": {
    "env": {
      "GOFLAGS": "-mod=mod"
    },
    "directoryFilters": [
      "-/vendor",
      "-/node_modules",
      "-/cache"
    ]
  }
}
```

### Completion and Formatting

```json
{
  "gopls": {
    "experimentalPostfixCompletions": true,
    "completeUnimported": true,
    "matcher": "fuzzy"
  }
}
```

## Gopls Settings Reference

For comprehensive documentation on all available gopls settings, see:

- **Official gopls settings**: [https://go.dev/gopls/settings](https://go.dev/gopls/settings)
- **User guide**: [https://go.dev/gopls](https://go.dev/gopls)

## Default Configuration

If no configuration is provided, gopls-mcp uses these defaults:

```json
{
  "gopls": {},
  "max_response_bytes": 32000,
  "workdir": "<current directory>"
}
```

## Command-Line Flags

| Flag | Description |
|------|-------------|
| `-config` | Path to configuration file (JSON format) |
| `-workdir` | Path to Go project directory |
| `-logfile` | Path to log file for debugging (writes logs even in stdio mode) |
| `-addr` | HTTP server address (e.g., `localhost:8080`) |
| `-allow-dynamic-views` | **TEST-ONLY**: Allow dynamic view creation for e2e testing |

## Notes

- Configuration options specified via the config file or MCP initialization take precedence over defaults
- The `gopls` section is passed directly to gopls using `opts.Set()`, so all standard gopls options are supported without any hardcoding
- Use the `-logfile` flag to write logs to a file for debugging (especially useful in stdio mode where logs are otherwise discarded)
- For production use, it's recommended to run one gopls-mcp instance per project directory
