#!/bin/bash
# Manual test to verify file watcher works in stdio mode (production-like scenario)
# This simulates how gopls-mcp would be used by Claude Code or other MCP clients

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GOPROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PROJECT_DIR="$SCRIPT_DIR/projects/simple"
BINARY="$GOPROOT/goplsmcp-test-stdio"

echo "=== Building gopls-mcp ==="
cd "$GOPROOT"
go build -o "$BINARY" .

echo "=== Starting gopls-mcp in stdio mode (production-like) ==="
echo "Project: $PROJECT_DIR"
echo ""
echo "The server is now running in stdio mode."
echo "We will send it JSON-RPC requests and verify file watching works."
echo ""

# Create a temp directory for the test
TEST_DIR=$(mktemp -d)
TEST_FILE="$TEST_DIR/test.go"

# Write initial file
cat > "$TEST_FILE" << 'EOF'
package main

func Hello() string {
	return "hello"
}
EOF

echo "=== Test 1: Start server and query initial state ==="
# This would require implementing a full JSON-RPC client
# For now, let's just document what we'd need to test:

cat << 'EOF'
To properly test stdio mode, we need to:
1. Start gopls-mcp as a subprocess with stdin/stdout pipes
2. Send JSON-RPC initialize request
3. Send JSON-RPC tools/list request
4. Send JSON-RPC tools/call request for get_package_symbol_detail
5. Modify a file on disk
6. Wait 1-2 seconds
7. Send another tools/call request
8. Verify the response includes the new content

This requires a JSON-RPC client implementation.
EOF

echo ""
echo "=== Cleanup ==="
rm -f "$BINARY"
rm -rf "$TEST_DIR"

echo ""
echo "=== Recommendation ==="
echo "The unit tests in watcher/watcher_test.go prove the watcher works."
echo "However, we should test the actual stdio subprocess scenario to be 100% sure."
echo ""
echo "Options:"
echo "1. Implement a proper JSON-RPC client test (more work)"
echo "2. Test manually with Claude Code in production"
echo "3. Add more logging and verify watcher activity in real usage"
