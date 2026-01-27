package main
type Container[T any] interface { Put(value T) }
type Box[T any] struct { value T }
func (b *Box[T]) Put(value T) { b.value = value }
type Slice[T any] struct { items []T }
func (s *Slice[T]) Put(value T) { s.items = append(s.items, value) }
func main() { var c Container[int] = &Box[int]{}; c.Put(42) }