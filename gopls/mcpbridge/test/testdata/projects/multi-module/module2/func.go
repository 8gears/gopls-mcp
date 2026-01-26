package module2

// FuncInModule2 is a function defined in module2.
// This is used to test multi-module symbol search.
// Searching for this symbol tests that we search ALL views, not just views[0].
func FuncInModule2() string {
	return "function from module2"
}
