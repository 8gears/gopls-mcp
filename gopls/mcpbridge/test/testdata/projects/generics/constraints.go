
package constraints

// Custom constraint using interface
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~string
}

func MaxOrdered[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Constraint with methods
type Stringer interface {
	String() string
}

func PrintAll[T Stringer](items []T) {
	for _, item := range items {
		println(item.String())
	}
}
