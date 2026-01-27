package main
type Readable interface { Read() ([]byte, error) }
type Writable interface { Write([]byte) (int, error) }
type ReadWritable interface { Readable; Writable }
type Buffer struct { data []byte }
func (b *Buffer) Read() ([]byte, error) { return b.data, nil }
func (b *Buffer) Write(p []byte) (int, error) { b.data = append(b.data, p...); return len(p), nil }
func main() { var _ ReadWritable = &Buffer{} }