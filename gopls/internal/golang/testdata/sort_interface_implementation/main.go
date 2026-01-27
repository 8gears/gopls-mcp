package main
type Sortable interface { Len() int; Less(i, j int) bool; Swap(i, j int) }
type Person []string
func (p Person) Len() int { return len(p) }
func (p Person) Less(i, j int) bool { return p[i] < p[j] }
func (p Person) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func main() { sort.Sort(Person{}) }