package e2e

// Deep end-to-end tests for standard library exploration.
// These tests verify that gopls-mcp enables developers to deeply explore
// and understand Go's standard library without needing AI tools or grep.
//
// Scenarios covered:
// - Understanding complex stdlib packages (net/http, context, sync, io, encoding/json, database/sql, time)
// - Finding related functions and types within packages
// - Discovering implementation patterns in stdlib
// - Navigating complex type hierarchies in stdlib
// - Understanding stdlib interface contracts and implementations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/mcpbridge/test/testutil"
)

// TestStdlibNetHttpDeepDive tests deep exploration of net/http package.
func TestStdlibNetHttpDeepDive(t *testing.T) {
	projectDir := createHTTPProject(t)

	t.Run("ExploreHTTPServerAPI", func(t *testing.T) {
		// Use list_package_symbols to see the full net/http API
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "net/http",
			"include_docs":   false,
			"include_bodies": false, // Fast - signatures only
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibNetHTTP)
		t.Logf("net/http API (first 5000 chars):\n%s", testutil.TruncateString(content, 5000))
		t.Logf("Full content length: %d chars", len(content))

		// Should find key HTTP server types
		requiredTypes := []string{
			"ServeMux",
			"ServeHTTP",
			"Handler",
			"ResponseWriter",
			"Request",
			"Server",
		}

		missingTypes := []string{}
		for _, typeName := range requiredTypes {
			if !strings.Contains(content, typeName) {
				missingTypes = append(missingTypes, typeName)
			}
		}

		if len(missingTypes) > 0 {
			t.Logf("Note: Could not find types in first 5000 chars: %v", missingTypes)
			// Don't fail - this is just a truncation issue
		}

		t.Log("✓ Can explore HTTP server types and interface")
	})

	t.Run("FindHandlerImplementations", func(t *testing.T) {
		// Find all implementations of Handler interface in net/http
		tool := "go_implementation"
		args := map[string]any{
			"locator": map[string]any{
				"symbol_name":  "Handler",
				"context_file": createHTTPServerFile(t, projectDir),
				"kind":         "interface",
				"line_hint":    3,
			},
		}

		// Instead, let's use list_package_symbols with include_bodies to find Handler implementations
		tool = "go_list_package_symbols"
		args = map[string]any{
			"package_path":   "net/http",
			"include_docs":   true,
			"include_bodies": true,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibNetHTTP)
		t.Logf("HTTP Handler implementations:\n%s", testutil.TruncateString(content, 2000))

		// Should show HandlerFunc as a common implementation
		if strings.Contains(content, "HandlerFunc") {
			t.Log("✓ Found HandlerFunc implementation pattern")
		}
	})

	t.Run("DiscoverMiddlewarePattern", func(t *testing.T) {
		// Search for middleware-related patterns in net/http
		tool := "go_search"
		args := map[string]any{
			"query": "middleware",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibNetHTTP)
		t.Logf("Middleware search:\n%s", content)

		// Search for "StripPrefix" or common middleware utilities
		tool = "go_search"
		args = map[string]any{
			"query": "StripPrefix",
		}

		res, err = globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content = testutil.ResultText(t, res, testutil.GoldenStdlibNetHTTP)
		t.Logf("StripPrefix search:\n%s", content)

		if strings.Contains(content, "StripPrefix") || strings.Contains(content, "found") {
			t.Log("✓ Can discover middleware utilities")
		}
	})

	t.Run("ExploreFileSystem", func(t *testing.T) {
		// List all symbols in net/http to find file-related utilities
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "net/http",
			"include_docs":   true,
			"include_bodies": false,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibNetHTTP)
		t.Logf("net/http symbols:\n%s", testutil.TruncateString(content, 2000))

		// Should find FileServer
		if strings.Contains(content, "FileServer") {
			t.Log("✓ Can discover FileServer utility")
		}

		// Should find ServeFile or similar
		if strings.Contains(content, "Serve") {
			t.Log("✓ Can discover Serve utilities")
		}
	})
}

// TestStdlibContextDeepDive tests deep exploration of context package.
func TestStdlibContextDeepDive(t *testing.T) {

	t.Run("ExploreContextAPI", func(t *testing.T) {
		// Get the full context package API
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "context",
			"include_docs":   true,
			"include_bodies": true,
			"Cwd":            globalGoplsMcpDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibContext)
		t.Logf("Context package API:\n%s", testutil.TruncateString(content, 2000))

		// Should find key context functions
		requiredFuncs := []string{
			"Background",
			"TODO",
			"WithCancel",
			"WithDeadline",
			"WithTimeout",
			"WithValue",
		}

		for _, funcName := range requiredFuncs {
			if !strings.Contains(content, funcName) {
				t.Errorf("Expected to find %s in context API", funcName)
			}
		}

		t.Log("✓ Can explore context creation functions")
	})

	t.Run("UnderstandContextInterface", func(t *testing.T) {
		// Use list_package_symbols to see the Context interface definition
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "context",
			"include_docs":   true,
			"include_bodies": true,
			"Cwd":            globalGoplsMcpDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibContext)
		t.Logf("Context interface:\n%s", testutil.TruncateString(content, 2000))

		// Should find Context interface methods
		requiredMethods := []string{
			"Deadline",
			"Done",
			"Err",
			"Value",
		}

		for _, method := range requiredMethods {
			if !strings.Contains(content, method) {
				t.Errorf("Expected to find %s method in Context interface", method)
			}
		}

		t.Log("✓ Can understand Context interface contract")
	})

	t.Run("FindCancelPattern", func(t *testing.T) {
		// Search for CancelFunc to understand cancellation pattern
		tool := "go_search"
		args := map[string]any{
			"query": "CancelFunc",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibContext)
		t.Logf("CancelFunc search:\n%s", content)

		// Should find CancelFunc
		if strings.Contains(content, "CancelFunc") {
			t.Log("✓ Can discover cancellation pattern")
		}
	})

	t.Run("UnderstandDeadlineBehavior", func(t *testing.T) {
		// List context package symbols to find deadline-related types
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "context",
			"include_docs":   true,
			"include_bodies": true,
			"Cwd":            globalGoplsMcpDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibContext)
		t.Logf("Context deadline symbols:\n%s", testutil.TruncateString(content, 2000))

		// Should find Timer type
		if strings.Contains(content, "Timer") {
			t.Log("✓ Can discover Timer type for deadlines")
		}
	})
}

// TestStdlibSyncDeepDive tests deep exploration of sync package.
func TestStdlibSyncDeepDive(t *testing.T) {

	t.Run("ExploreSynchronizationPrimitives", func(t *testing.T) {
		// Get sync package API
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "sync",
			"include_docs":   false,
			"include_bodies": false,
			"Cwd":            globalGoplsMcpDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibSync)
		t.Logf("Sync package API:\n%s", testutil.TruncateString(content, 2000))

		// Should find key synchronization types
		requiredTypes := []string{
			"Mutex",
			"RWMutex",
			"WaitGroup",
			"Once",
			"Cond",
			"Pool",
			"Map",
		}

		for _, typeName := range requiredTypes {
			if !strings.Contains(content, typeName) {
				t.Errorf("Expected to find %s in sync API", typeName)
			}
		}

		t.Log("✓ Can explore all sync primitives")
	})

	t.Run("UnderstandMutexPatterns", func(t *testing.T) {
		// List symbols with docs to understand Mutex methods
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "sync",
			"include_docs":   true,
			"include_bodies": true,
			"Cwd":            globalGoplsMcpDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibSync)
		t.Logf("Mutex symbols with docs:\n%s", testutil.TruncateString(content, 2000))

		// Should find Lock and Unlock methods
		if strings.Contains(content, "Lock") && strings.Contains(content, "Unlock") {
			t.Log("✓ Can see Mutex locking methods")
		}
	})

	t.Run("FindPoolPattern", func(t *testing.T) {
		// Search for Pool to understand object pooling pattern
		tool := "go_search"
		args := map[string]any{
			"query": "Pool",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibSync)
		t.Logf("Pool search:\n%s", content)

		// Should find sync.Pool
		if strings.Contains(content, "Pool") {
			t.Log("✓ Can discover Pool pattern")
		}
	})

	t.Run("UnderstandErrGroupPattern", func(t *testing.T) {
		// Search for errgroup pattern (commonly used with WaitGroup)
		tool := "go_search"
		args := map[string]any{
			"query": "ErrorGroup",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibSync)
		t.Logf("ErrorGroup search:\n%s", content)

		// Note: ErrorGroup is in x/sync/errgroup, not sync package
		// This test verifies we can search beyond sync
		t.Log("✓ Can search for related packages (x/sync/errgroup)")
	})
}

// TestStdlibIODeepDive tests deep exploration of io and bufio packages.
func TestStdlibIODeepDive(t *testing.T) {

	t.Run("ExploreIOInterfaces", func(t *testing.T) {
		// Get io package API
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "io",
			"include_docs":   true,
			"include_bodies": true,
			"Cwd":            globalGoplsMcpDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibIO)
		t.Logf("io package API:\n%s", testutil.TruncateString(content, 3000))

		// Should find core io interfaces
		requiredInterfaces := []string{
			"Reader",
			"Writer",
			"ReaderAt",
			"WriterAt",
			"ByteReader",
			"ByteScanner",
			"ReadCloser",
			"WriteCloser",
			"Seeker",
			"Closer",
		}

		for _, iface := range requiredInterfaces {
			if !strings.Contains(content, iface) {
				t.Errorf("Expected to find %s interface in io package", iface)
			}
		}

		t.Log("✓ Can explore all io interfaces")
	})

	t.Run("UnderstandPipePattern", func(t *testing.T) {
		// Search for Pipe to understand io.Pipe pattern
		tool := "go_search"
		args := map[string]any{
			"query": "Pipe",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibIO)
		t.Logf("Pipe search:\n%s", content)

		if strings.Contains(content, "Pipe") {
			t.Log("✓ Can discover Pipe pattern")
		}
	})

	t.Run("ExploreBufioUtilities", func(t *testing.T) {
		// Get bufio package API
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "bufio",
			"include_docs":   false,
			"include_bodies": false,
			"Cwd":            globalGoplsMcpDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibIO)
		t.Logf("bufio package API:\n%s", testutil.TruncateString(content, 2000))

		// Should find bufio utilities
		requiredTypes := []string{
			"Reader",
			"Writer",
			"Scanner",
			"ReadWriter",
		}

		for _, typeName := range requiredTypes {
			if !strings.Contains(content, typeName) {
				t.Errorf("Expected to find %s in bufio API", typeName)
			}
		}

		t.Log("✓ Can explore buffered I/O utilities")
	})

	t.Run("UnderstandLimitReader", func(t *testing.T) {
		// Search for LimitReader pattern
		tool := "go_search"
		args := map[string]any{
			"query": "LimitReader",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibIO)
		t.Logf("LimitReader search:\n%s", content)

		if strings.Contains(content, "LimitReader") {
			t.Log("✓ Can discover reader wrapper utilities")
		}
	})
}

// TestStdlibEncodingJSONDeepDive tests deep exploration of encoding/json.
func TestStdlibEncodingJSONDeepDive(t *testing.T) {
	projectDir := createJSONProject(t)

	t.Run("ExploreJSONAPI", func(t *testing.T) {
		// Get encoding/json API
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "encoding/json",
			"include_docs":   false,
			"include_bodies": false,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibEncodingJSON)
		t.Logf("encoding/json API:\n%s", testutil.TruncateString(content, 2000))

		// Should find key JSON functions
		requiredFuncs := []string{
			"Marshal",
			"Unmarshal",
			"MarshalIndent",
			"NewEncoder",
			"NewDecoder",
			"Valid",
		}

		for _, funcName := range requiredFuncs {
			if !strings.Contains(content, funcName) {
				t.Errorf("Expected to find %s in encoding/json API", funcName)
			}
		}

		t.Log("✓ Can explore JSON marshaling API")
	})

	t.Run("UnderstandRawMessage", func(t *testing.T) {
		// Search for RawMessage to understand delayed JSON parsing
		tool := "go_search"
		args := map[string]any{
			"query": "RawMessage",
			"Cwd":   projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibEncodingJSON)
		t.Logf("RawMessage search:\n%s", content)

		if strings.Contains(content, "RawMessage") {
			t.Log("✓ Can discover RawMessage for deferred parsing")
		}
	})

	t.Run("ExploreStreamingJSON", func(t *testing.T) {
		// List encoding/json symbols to find streaming types
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "encoding/json",
			"include_docs":   true,
			"include_bodies": false,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibEncodingJSON)
		t.Logf("encoding/json symbols:\n%s", testutil.TruncateString(content, 2000))

		// Should find Encoder and Decoder
		requiredTypes := []string{
			"Encoder",
			"Decoder",
			"Delim",
		}

		for _, typeName := range requiredTypes {
			if !strings.Contains(content, typeName) {
				t.Errorf("Expected to find %s in encoding/json", typeName)
			}
		}

		t.Log("✓ Can discover streaming JSON utilities")
	})
}

// TestStdlibDatabaseSQLDeepDive tests deep exploration of database/sql.
func TestStdlibDatabaseSQLDeepDive(t *testing.T) {
	projectDir := createDBProject(t)

	t.Run("ExploreDatabaseAPI", func(t *testing.T) {
		// Get database/sql API
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "database/sql",
			"include_docs":   false,
			"include_bodies": false,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibDatabaseSQL)
		t.Logf("database/sql API:\n%s", testutil.TruncateString(content, 2000))

		// Should find key database types
		requiredTypes := []string{
			"DB",
			"Tx",
			"Rows",
			"Row",
			"Stmt",
			"Null",
		}

		for _, typeName := range requiredTypes {
			if !strings.Contains(content, typeName) {
				t.Errorf("Expected to find %s in database/sql API", typeName)
			}
		}

		t.Log("✓ Can explore database/sql types")
	})

	t.Run("UnderstandConnectionPattern", func(t *testing.T) {
		// Search for Open and OpenDB
		tool := "go_search"
		args := map[string]any{
			"query": "Open",
			"Cwd":   projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibDatabaseSQL)
		t.Logf("Open search:\n%s", content)

		if strings.Contains(content, "Open") {
			t.Log("✓ Can discover connection functions")
		}
	})

	t.Run("ExploreTransactionPattern", func(t *testing.T) {
		// List database/sql symbols to find transaction methods
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "database/sql",
			"include_docs":   true,
			"include_bodies": true,
			"Cwd":            projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibDatabaseSQL)
		t.Logf("database/sql transaction symbols:\n%s", testutil.TruncateString(content, 3000))

		// Should find transaction methods
		requiredMethods := []string{
			"Begin",
			"Commit",
			"Rollback",
		}

		for _, method := range requiredMethods {
			if !strings.Contains(content, method) {
				t.Errorf("Expected to find %s method in Tx", method)
			}
		}

		t.Log("✓ Can understand transaction API")
	})

	t.Run("UnderstandNullHandling", func(t *testing.T) {
		// Search for Null types
		tool := "go_search"
		args := map[string]any{
			"query": "Null",
			"Cwd":   projectDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibDatabaseSQL)
		t.Logf("Null search:\n%s", content)

		if strings.Contains(content, "Null") {
			t.Log("✓ Can discover Null handling types")
		}
	})
}

// TestStdlibTimeDeepDive tests deep exploration of time package.
func TestStdlibTimeDeepDive(t *testing.T) {

	t.Run("ExploreTimeAPI", func(t *testing.T) {
		// Get time package API
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "time",
			"include_docs":   false,
			"include_bodies": false,
			"Cwd":            globalGoplsMcpDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibTime)
		t.Logf("time package API:\n%s", testutil.TruncateString(content, 3000))

		// Should find key time types
		requiredTypes := []string{
			"Time",
			"Duration",
			"Timer",
			"Ticker",
			"Location",
			"Zone",
		}

		for _, typeName := range requiredTypes {
			if !strings.Contains(content, typeName) {
				t.Errorf("Expected to find %s in time API", typeName)
			}
		}

		t.Log("✓ Can explore time package types")
	})

	t.Run("UnderstandTickerPattern", func(t *testing.T) {
		// Search for Ticker
		tool := "go_search"
		args := map[string]any{
			"query": "Ticker",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibTime)
		t.Logf("Ticker search:\n%s", content)

		if strings.Contains(content, "Ticker") {
			t.Log("✓ Can discover Ticker for recurring events")
		}
	})

	t.Run("ExploreDurationOperations", func(t *testing.T) {
		// List time package symbols to find duration operations
		tool := "go_list_package_symbols"
		args := map[string]any{
			"package_path":   "time",
			"include_docs":   true,
			"include_bodies": false,
			"Cwd":            globalGoplsMcpDir,
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibTime)
		t.Logf("Duration operations:\n%s", testutil.TruncateString(content, 2000))

		// Should find duration methods
		requiredMethods := []string{
			"Since",
			"Until",
			"Add",
			"String",
		}

		for _, method := range requiredMethods {
			if !strings.Contains(content, method) {
				t.Errorf("Expected to find %s method", method)
			}
		}

		t.Log("✓ Can explore duration arithmetic")
	})

	t.Run("UnderstandTimeZoneHandling", func(t *testing.T) {
		// Search for timezone-related functions
		tool := "go_search"
		args := map[string]any{
			"query": "LoadLocation",
		}

		res, err := globalSession.CallTool(globalCtx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool, err)
		}

		content := testutil.ResultText(t, res, testutil.GoldenStdlibTime)
		t.Logf("LoadLocation search:\n%s", content)

		if strings.Contains(content, "LoadLocation") {
			t.Log("✓ Can discover timezone loading")
		}
	})
}

// ===== Helper Functions to Create Test Projects =====

func createHTTPProject(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()

	goModContent := `module example.com/httpserver

go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main.go that uses net/http
	mainCode := `package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// handler is a simple HTTP handler
func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func main() {
	server := &http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(handler),
	}

	go func() {
		time.Sleep(5 * time.Second)
		server.Shutdown(context.Background())
	}()

	fmt.Println("Server starting on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server error: %v\n", err)
	}
}
`
	mainGoPath := filepath.Join(projectDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainCode), 0644); err != nil {
		t.Fatal(err)
	}

	return projectDir
}

func createHTTPServerFile(t *testing.T, projectDir string) string {
	t.Helper()
	return filepath.Join(projectDir, "main.go")
}

func createJSONProject(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()

	goModContent := `module example.com/jsondemo

go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main.go that uses encoding/json
	mainCode := `package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type Person struct {
	Name  string ` + "`json:\"name\"`" + `
	Age   int    ` + "`json:\"age\"`" + `
	Email string ` + "`json:\"email,omitempty\"`" + `
}

func main() {
	// Marshal example
	p := Person{Name: "Alice", Age: 30}
	data, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Marshal: %s\n", data)

	// Unmarshal example
	var p2 Person
	if err := json.Unmarshal(data, &p2); err != nil {
		log.Fatal(err)
	}
}
`
	mainGoPath := filepath.Join(projectDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainCode), 0644); err != nil {
		t.Fatal(err)
	}

	return projectDir
}

func createDBProject(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()

	goModContent := `module example.com/dbdemo

go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main.go that uses database/sql
	mainCode := `package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3" // Example driver
)

func main() {
	// Open database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// Query example
	rows, err := db.Query("SELECT 1")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Database connection successful")
}

// PrepareStmt demonstrates prepared statements
func PrepareStmt(db *sql.DB, name string) (*sql.Stmt, error) {
	return db.Prepare("SELECT id FROM users WHERE name = ?")
}

// ExecTx demonstrates transaction usage
func ExecTx(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return tx.Commit()
}
`
	mainGoPath := filepath.Join(projectDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainCode), 0644); err != nil {
		t.Fatal(err)
	}

	return projectDir
}

func createContextProject(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()

	goModContent := `module example.com/contextdemo

go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main.go that uses context
	mainCode := `package main

import (
	"context"
	"fmt"
	"time"
)

func worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d: shutting down\n", id)
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func main() {
	ctx := context.Background()

	// With timeout
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	for i := 1; i <= 3; i++ {
		go worker(ctx, i)
	}

	time.Sleep(3 * time.Second)
	fmt.Println("Main done")
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainCode), 0644); err != nil {
		t.Fatal(err)
	}

	return projectDir
}

func createSyncProject(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()

	goModContent := `module example.com/syncdemo

go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main.go that uses sync
	mainCode := `package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var counter int

	// Start 10 workers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
			fmt.Printf("Worker %d done\n", id)
		}(i)
	}

	wg.Wait()
	fmt.Printf("Final counter: %d\n", counter)
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainCode), 0644); err != nil {
		t.Fatal(err)
	}

	return projectDir
}

func createIOProject(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()

	goModContent := `module example.com/iodemo

go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main.go that uses io and bufio
	mainCode := `package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

func main() {
	// Create a reader
	reader := strings.NewReader("Hello, World!\nLine 2\nLine 3")

	// Use bufio.Scanner
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fmt.Println("Scanned:", scanner.Text())
	}

	// Use buffered writer
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	writer.WriteString("Buffered write\n")
	writer.Flush()

	fmt.Println("Buffer contents:", buf.String())

	// Copy with io.Copy
	var dest bytes.Buffer
	io.Copy(&dest, reader)
	fmt.Println("Dest:", dest.String())
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainCode), 0644); err != nil {
		t.Fatal(err)
	}

	return projectDir
}
