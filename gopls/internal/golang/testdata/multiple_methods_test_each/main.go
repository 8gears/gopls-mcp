package main
type ReadWriter interface { Read() ([]byte, error); Write([]byte) (int, error); Close() error }
type Buffer struct { data []byte }
func (b *Buffer) Read() ([]byte, error) { return b.data, nil }
func (b *Buffer) Write(p []byte) (int, error) { b.data = append(b.data, p...); return len(p), nil }
func (b *Buffer) Close() error { b.data = nil; return nil }
func main() { var _ ReadWriter = &Buffer{} }