package main
type Shape interface { Area() float64 }
type Circle struct { Radius float64 }
func (c Circle) Area() float64 { return 3.14 * c.Radius * c.Radius }
func main() { var s Shape = Circle{Radius: 10}; _ = s.Area() }