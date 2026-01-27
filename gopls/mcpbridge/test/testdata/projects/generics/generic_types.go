
package generictypes

// Generic struct
type Container[T any] struct {
	Value T
}

// Generic struct with multiple type parameters
type Pair[T, U any] struct {
	First  T
	Second U
}

// Generic interface
type Wrapper[T any] interface {
	Wrap(T) T
	Unwrap() T
}

// Method on generic type
func (c Container[T]) Get() T {
	return c.Value
}

func (c Container[T]) Set(v T) {
	c.Value = v
}
