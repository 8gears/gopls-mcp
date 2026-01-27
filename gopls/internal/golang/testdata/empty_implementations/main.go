package main
type UnusedInterface interface { DoSomething() }
func main() { var _ UnusedInterface }