package e2e

import "testing"

// Shared test types for table-driven E2E tests.

// testCase defines a single tool test case with basic fields.
type testCase struct {
	name       string
	tool       string
	args       map[string]any
	assertion  func(t *testing.T, content string)
	skip       bool
	skipReason string
}

// setupTestCase defines a test case with a setup function for creating files and returning args.
// Used in error scenarios that need to create temporary files.
type setupTestCase struct {
	name       string
	tool       string
	args       map[string]any
	assertion  func(t *testing.T, content string)
	skip       bool
	skipReason string
	setup      func(t *testing.T) map[string]any
}

// fileTestCase defines a test case with fields for test file creation.
// Used in generics tests that need to create test files with specific source code.
type fileTestCase struct {
	name       string
	tool       string
	args       map[string]any
	assertion  func(t *testing.T, content string)
	skip       bool
	skipReason string
	tmpDir     string
	moduleName string
	sourceCode string
}
