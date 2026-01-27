package main
type Writer interface { Write([]byte) (int, error) }
type W1 struct{}; func (w *W1) Write(p []byte) (int, error) { return len(p), nil }
type W2 struct{}; func (w *W2) Write(p []byte) (int, error) { return len(p), nil }
type W3 struct{}; func (w *W3) Write(p []byte) (int, error) { return len(p), nil }
type W4 struct{}; func (w *W4) Write(p []byte) (int, error) { return len(p), nil }
type W5 struct{}; func (w *W5) Write(p []byte) (int, error) { return len(p), nil }
func main() { var _ Writer = &W1{} }