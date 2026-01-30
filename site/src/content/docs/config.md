---
title: Configure gopls-mcp
description: Customize behavior and response limits.
---

## Quick Setup

gopls-mcp uses a JSON configuration file for customization. Pass it via the `-config` flag:

```bash
gopls-mcp -config /path/to/config.json
```

## Configuration Structure

```json
{
  "workdir": "/path/to/project",
  "max_response_bytes": 64000,
  "gopls": {
    // Native gopls settings (passed directly to gopls)
  }
}
```

## Key Settings

### max_response_bytes

**Type**: `integer` | **Default**: `32000` (32KB)

Global response size limit in bytes. All tools respect this limit automatically. When exceeded, responses are truncated with metadata.

```json
{
  "max_response_bytes": 64000  // 64KB limit
}
```

Increase this if you frequently encounter truncated results from large files or complex queries.

### workdir

**Type**: `string` | **Default**: Current directory

The Go project directory to analyze. Overrides the `-workdir` flag.

```json
{
  "workdir": "/path/to/my/project"
}
```

### gopls

**Type**: `object` | **Default**: `{}`

Native gopls configuration options. These are passed directly to gopls's internal settings system.

> **For complete gopls settings documentation, see [go.dev/gopls/settings](https://go.dev/gopls/settings)**

Common options:

```json
{
  "gopls": {
    "staticcheck": true,
    "analyses": {
      "unusedparams": true
    },
    "buildFlags": ["-tags=integration"],
    "env": {
      "GOFLAGS": "-mod=mod"
    }
  }
}
```

## Default Configuration

If no config file is provided:

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
| `-config` | Path to configuration file (JSON) |
| `-workdir` | Path to Go project directory |
| `-logfile` | Path to log file for debugging |
| `-addr` | HTTP server address (e.g., `localhost:8080`) |
| `-verbose` | Enable verbose logging (HTTP mode only) |

## Learn More

- **[gopls settings reference](https://go.dev/gopls/settings)** - All native gopls options
- **[gopls user guide](https://go.dev/gopls)** - Comprehensive documentation
