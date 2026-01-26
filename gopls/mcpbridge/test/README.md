# gopls-mcp Tests

This directory contains integration tests, end-to-end (E2E) scenarios, and performance benchmarks for the gopls-mcp server.
For detailed documentation, please visit [https://gopls-mcp](https://gopls-mcp).

## Running Tests

```bash
# Run integration tests (tool-level API verification)
go test -v ./integration/...

# Run end-to-end tests (real user scenarios)
go test -v ./e2e/...

# Run performance benchmarks
cd benchmark && go run benchmark_main.go -compare
```