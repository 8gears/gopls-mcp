
package nested

import "container/list"

// Nested generic types
type Matrix[T any] [][]T

type TripleContainer[T, U, V any] struct {
	First  Container[T]
	Second Container[U]
	Third  Container[V]
}

type Container[T any] struct {
	Value T
}

// Generic function returning nested generic type
func NestedSlice[T any](n int) [][]T {
	return make([][]T, n)
}

func UseNested() {
	// Nested usage
	var m Matrix[int]
	m = append(m, []int{1, 2, 3})

	// Triple nested
	_ = TripleContainer[int, string, bool]{}
}
