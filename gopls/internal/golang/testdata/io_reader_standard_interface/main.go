package main
type Reader interface { Read(p []byte) (int, error) }
type MyReader struct { data []byte }
func (m *MyReader) Read(p []byte) (int, error) { if len(m.data) == 0 { return 0, nil }; copy(p, m.data); m.data = nil; return len(p), nil }
func main() { var _ Reader = &MyReader{} }