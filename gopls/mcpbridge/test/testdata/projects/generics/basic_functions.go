
package generics

// Generic function with type parameter
func First[T any](slice []T) T {
	if len(slice) == 0 {
		var zero T
		return zero
	}
	return slice[0]
}

// Generic function with constraint
func Max[T comparable](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Generic function with multiple type parameters
func Pair[T, U any](t T, u U) (T, U) {
	return t, u
}
