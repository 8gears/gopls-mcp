package main
type Comparable[T comparable] interface { Compare(other T) int }
type Number struct { val int }
func (n Number) Compare(other Number) int { if n.val < other.val { return -1 }; if n.val > other.val { return 1 }; return 0 }
func main() { var _ Comparable[int] = Number{val: 5} }