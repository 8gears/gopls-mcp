
package inference

func Map[T, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

func UseInference() {
	// Type inference should work here
	numbers := []int{1, 2, 3}
	strings := Map(numbers, func(n int) string {
		return "number"
	})

	// Explicit instantiation
	explicit := Map[int, string](numbers, func(n int) string {
		return "explicit"
	})

	_ = strings
	_ = explicit
}
