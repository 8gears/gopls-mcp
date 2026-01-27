package main
type Handler interface { Handle() }
func main() { h := struct{}{}; h.Handle = func() {}; var _ Handler = h }