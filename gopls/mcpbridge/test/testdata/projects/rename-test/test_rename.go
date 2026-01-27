
package core

// TestRenameFunction is a test function for rename operations
func TestRenameFunction(x int) int {
	return x * 2
}

// useTestRenameFunction uses the test function
func useTestRenameFunction() {
	// First usage
	result := TestRenameFunction(10)

	// Second usage in a different context
	another := TestRenameFunction(20)

	// Third usage
	third := TestRenameFunction(result)

	_ = result
	_ = another
	_ = third
}
