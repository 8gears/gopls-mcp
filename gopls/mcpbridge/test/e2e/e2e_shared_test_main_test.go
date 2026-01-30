package e2e

// Shared test setup for all e2e tests.
// This file provides a single TestMain and shared MCP session for the entire e2e package.
// OPTIMIZATION: Uses ONE gopls-mcp process for all e2e tests (instead of starting a new one per test).
// This reduces test time from ~210s to ~60-70s by sharing GOROOT and stdlib cache across all tests.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Global shared MCP session for all tests in the e2e package.
// All test functions across all files should use globalSession and globalCtx.
//
// TEST-ONLY: The GOPMCS_ALLOW_DYNAMIC_VIEWS environment variable is enabled for performance optimization.
// This allows one gopls-mcp process to create views for multiple test directories on-demand.
var globalSession *mcp.ClientSession
var globalCtx context.Context

// Global shared path to the gopls-mcp directory.
// This is the parent directory of the e2e test directory.
var globalGoplsMcpDir string

// TestMain sets up the shared MCP server before running any tests in the e2e package.
// This function runs ONCE for the entire e2e test package, not per test file.
//
// The GOPMCS_ALLOW_DYNAMIC_VIEWS environment variable enables TEST-ONLY dynamic view creation:
// - Starts ONE gopls-mcp process for ALL tests (not one per test)
// - Shares the gopls cache (GOROOT, stdlib) across all tests
// - Each test gets its own view created on first access via Cwd parameter
// - Reduces test time from ~210s to ~60-70s
//
// IMPORTANT: This is for TESTING ONLY. Normal users should NOT use this.
func TestMain(m *testing.M) {
	// Build gopls-mcp first (outside of test context)
	// NOTE: The main package is at the project root (4 levels up), not in gopls/
	projectRoot, _ := filepath.Abs("../../../..")
	goplsMcpPath := filepath.Join(projectRoot, "gopls", "mcpbridge", "test", "e2e", ".tmp", "gopls-mcp-test")
	buildCmd := exec.Command("go", "build", "-o", goplsMcpPath, ".")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to build gopls-mcp: %v\n%s", err, output)
		os.Exit(1)
	}

	// Initialize the shared gopls-mcp directory path (parent of e2e test directory)
	globalGoplsMcpDir, _ = filepath.Abs("../..")

	// Ensure the binary has executable permissions
	if err := os.Chmod(goplsMcpPath, 0755); err != nil {
		fmt.Printf("Failed to set executable permissions: %v\n", err)
		os.Exit(1)
	}

	// Create a minimal shared workdir (must be a valid Go project for gopls)
	sharedWorkdir := filepath.Join(projectRoot, "gopls", "mcpbridge", "test", "testdata", "projects", "simple")

	// Start ONE shared gopls-mcp server for ALL tests in the e2e package
	var err error
	globalSession, globalCtx, err = startSharedServer(goplsMcpPath, sharedWorkdir)
	if err != nil {
		fmt.Printf("Failed to start gopls-mcp server: %v\n", err)
		os.Exit(1)
	}

	// Run all tests
	code := m.Run()

	// Cleanup
	if globalSession != nil {
		globalSession.Close()
	}

	// Clean up the binary
	os.Remove(goplsMcpPath)

	os.Exit(code)
}

// startSharedServer starts a gopls-mcp server with dynamic views enabled.
// Helper function to avoid using testing.T() in TestMain context.
func startSharedServer(goplsMcpPath, sharedWorkdir string) (*mcp.ClientSession, context.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	goplsMcpCmd := exec.Command(goplsMcpPath, "-workdir", sharedWorkdir)
	goplsMcpCmd.Env = append(os.Environ(), "GOPMCS_ALLOW_DYNAMIC_VIEWS=true") // TEST-ONLY: Enable dynamic view creation

	mcpSession, err := client.Connect(ctx, &mcp.CommandTransport{Command: goplsMcpCmd}, nil)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to connect to gopls-mcp: %w", err)
	}

	// Store cancel function for cleanup
	// We'll use a closure to capture both the session and the cancel function
	_ = cancel // Will be called when session is closed

	return mcpSession, ctx, nil
}
