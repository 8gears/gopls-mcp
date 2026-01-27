package main
type Stringer interface { String() string }
type Person struct { Name string }
func (p Person) String() string { return "Person: " + p.Name }
func main() { var _ Stringer = Person{Name: "Alice"} }