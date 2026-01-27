package main
type Handler interface { Method() string }
type Base struct { Name string }
func (b Base) Method() string { return b.Name }
type Wrapper struct { Base }
func main() { w := Wrapper{Base{Name: "test"}}; var _ Handler = w }