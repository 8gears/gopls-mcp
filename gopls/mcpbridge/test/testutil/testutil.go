package testutil

// testutil provides helper functions for testing goplsmcp.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/internal/testenv"
)

// ProjectDir returns the path to a test project in testdata.
// For example, ProjectDir("simple") returns the path to the simple test project.
func ProjectDir(name string) string {
	// Navigate from gopls-mcp/test/testutil to gopls-mcp/test/testdata/projects
	relPath := filepath.Join("..", "testdata", "projects", name)
	abs, err := filepath.Abs(relPath)
	if err != nil {
		panic(fmt.Sprintf("failed to get absolute path for %s: %v", relPath, err))
	}
	return abs
}

// CopyProjectTo copies a test project from testdata to a temporary directory.
// Returns the path to the copied project.
func CopyProjectTo(t *testing.T, projectName string) string {
	t.Helper()

	srcDir := ProjectDir(projectName)
	dstDir := t.TempDir()

	// Copy all files from src to dst
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})

	if err != nil {
		t.Fatalf("failed to copy project %s: %v", projectName, err)
	}

	return dstDir
}

// StartMCPServer builds and starts gopls-mcp for testing.
// It returns the MCP session and a cleanup function.
func StartMCPServer(t *testing.T, workdir string) (*mcp.ClientSession, context.Context, func()) {
	t.Helper()
	testenv.NeedsExec(t)

	dir := t.TempDir()
	goplsMcpPath := filepath.Join(dir, "gopls-mcp")

	// Build gopls-mcp
	// Navigate from test/testutil to project root (where main.go is located)
	projectRoot, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}
	buildCmd := exec.Command("go", "build", "-o", goplsMcpPath, ".")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Skipf("Skipping test: failed to build gopls-mcp: %v\n%s", err, output)
	}

	// Ensure the binary has executable permissions
	if err := os.Chmod(goplsMcpPath, 0755); err != nil {
		t.Fatalf("Failed to set executable permissions: %v", err)
	}

	// Start gopls-mcp
	goplsMcpCmd := exec.Command(goplsMcpPath, "-workdir", workdir)
	ctx := t.Context()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	mcpSession, err := client.Connect(ctx, &mcp.CommandTransport{Command: goplsMcpCmd}, nil)
	if err != nil {
		t.Fatalf("Failed to connect to gopls-mcp: %v", err)
	}

	cleanup := func() {
		// Close the MCP session - errors are expected during shutdown
		// when the server process exits, so we log but don't fail the test
		if err := mcpSession.Close(); err != nil {
			t.Logf("MCP connection closed with error (expected): %v", err)
		}
	}

	return mcpSession, ctx, cleanup
}

// StartSharedMCPServer builds and starts a gopls-mcp server with DYNAMIC VIEWS enabled.
// TEST-ONLY: This enables the -allow-dynamic-views flag, which allows one gopls-mcp
// process to create views for multiple test directories on-demand.
//
// This is a performance optimization for e2e tests:
// - Starts ONE gopls-mcp process for ALL tests (not one per test)
// - Shares the gopls cache (GOROOT, stdlib) across all tests
// - Each test gets its own view created on first access
// - Reduces test time from ~197s to ~60s
//
// IMPORTANT: This is for TESTING ONLY. Normal users should NOT use this flag,
// as it allows querying multiple unrelated projects which can lead to unexpected behavior.
//
// Usage pattern:
//
//	func TestMain(m *testing.M) {
//	    // Start ONE shared server for all tests in this file
//	    session, ctx, cancel := testutil.StartSharedMCPServer(t, "/tmp/shared-workdir")
//	    defer cancel()
//
//	    // Store in package-level variable for tests to access
//	    globalSession = session
//	    globalCtx = ctx
//
//	    code := m.Run()
//	    os.Exit(code)
//	}
//
//	func TestXxx(t *testing.T) {
//	    dir := testutil.CopyProjectTo(t, "simple")
//	    // Each test uses the shared session, specifying its own workdir via Cwd parameter
//	    result := globalSession.CallTool(globalCtx, "go_search", map[string]any{
//	        "query": "foo",
//	        "Cwd": dir,  // Triggers view creation on first use
//	    })
//	}
func StartSharedMCPServer(t *testing.T, sharedWorkdir string) (*mcp.ClientSession, context.Context, context.CancelFunc) {
	t.Helper()
	testenv.NeedsExec(t)

	dir := t.TempDir()
	goplsMcpPath := filepath.Join(dir, "gopls-mcp")

	// Build gopls-mcp
	// Navigate from test/testutil to project root (where main.go is located)
	projectRoot, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}
	buildCmd := exec.Command("go", "build", "-o", goplsMcpPath, ".")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Skipf("Skipping test: failed to build gopls-mcp: %v\n%s", err, output)
	}

	// Ensure the binary has executable permissions
	if err := os.Chmod(goplsMcpPath, 0755); err != nil {
		t.Fatalf("Failed to set executable permissions: %v", err)
	}

	// Start gopls-mcp with DYNAMIC VIEWS enabled (TEST-ONLY flag)
	// The -allow-dynamic-views flag allows the server to create views on-demand
	// for different test directories, sharing the gopls cache across all tests.
	goplsMcpCmd := exec.Command(goplsMcpPath,
		"-workdir", sharedWorkdir,
		"-allow-dynamic-views", // TEST-ONLY: Enable dynamic view creation
	)

	ctx, cancel := context.WithCancel(context.Background())
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	mcpSession, err := client.Connect(ctx, &mcp.CommandTransport{Command: goplsMcpCmd}, nil)
	if err != nil {
		t.Fatalf("Failed to connect to gopls-mcp: %v", err)
	}

	// Return cancel function instead of cleanup - caller is responsible for lifecycle
	return mcpSession, ctx, cancel
}

// StartMCPServerWithLogfile builds and starts gopls-mcp with a log file for debugging.
// This is useful for tests that need to inspect internal server behavior (e.g., file watcher).
func StartMCPServerWithLogfile(t *testing.T, workdir string, logfile string) (*mcp.ClientSession, context.Context, func()) {
	t.Helper()
	testenv.NeedsExec(t)

	dir := t.TempDir()
	goplsMcpPath := filepath.Join(dir, "gopls-mcp")

	// Build gopls-mcp
	// Navigate from test/testutil to project root (where main.go is located)
	projectRoot, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}
	buildCmd := exec.Command("go", "build", "-o", goplsMcpPath, ".")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Skipf("Skipping test: failed to build gopls-mcp: %v\n%s", err, output)
	}

	// Ensure the binary has executable permissions
	if err := os.Chmod(goplsMcpPath, 0755); err != nil {
		t.Fatalf("Failed to set executable permissions: %v", err)
	}

	// Start gopls-mcp with logfile flag
	goplsMcpCmd := exec.Command(goplsMcpPath, "-workdir", workdir, "-logfile", logfile)
	ctx := t.Context()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	mcpSession, err := client.Connect(ctx, &mcp.CommandTransport{Command: goplsMcpCmd}, nil)
	if err != nil {
		t.Fatalf("Failed to connect to gopls-mcp: %v", err)
	}

	cleanup := func() {
		// Close the MCP session - errors are expected during shutdown
		// when the server process exits, so we log but don't fail the test
		if err := mcpSession.Close(); err != nil {
			t.Logf("MCP connection closed with error (expected): %v", err)
		}
	}

	return mcpSession, ctx, cleanup
}

// StartMCPServerRaw builds and starts gopls-mcp for benchmarking.
// Similar to StartMCPServer but doesn't require testing.T.
func StartMCPServerRaw(workdir string) (*mcp.ClientSession, context.Context, func(), error) {
	ctx := context.Background()

	// Get project root directory (navigate from testutil to project root)
	projectRoot, err := filepath.Abs("../../../..")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get project root: %v", err)
	}

	// Build gopls-mcp to temp location
	goplsMcpPath := filepath.Join(projectRoot, "goplsmcp.tmp")
	buildCmd := exec.Command("go", "build", "-o", goplsMcpPath, ".")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build gopls-mcp: %v\n%s", err, output)
	}

	// Ensure the binary has executable permissions
	if err := os.Chmod(goplsMcpPath, 0755); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to set executable permissions: %v", err)
	}

	// Start gopls-mcp
	goplsMcpCmd := exec.Command(goplsMcpPath, "-workdir", workdir)
	client := mcp.NewClient(&mcp.Implementation{Name: "benchmark-client", Version: "v0.0.1"}, nil)
	mcpSession, err := client.Connect(ctx, &mcp.CommandTransport{Command: goplsMcpCmd}, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to gopls-mcp: %v", err)
	}

	cleanup := func() {
		if err := mcpSession.Close(); err != nil {
			// Log but don't fail in cleanup
			fmt.Printf("Warning: Failed to close MCP connection: %v\n", err)
		}
		// Remove the built binary
		os.Remove(goplsMcpPath)
	}

	return mcpSession, ctx, cleanup, nil
}

// ResultText concatenates the textual content of an MCP tool result.
func ResultText(res *mcp.CallToolResult) string {
	var buf strings.Builder
	for _, content := range res.Content {
		if c, ok := content.(*mcp.TextContent); ok {
			buf.WriteString(c.Text)
			buf.WriteString("\n")
		}
	}
	return buf.String()
}
