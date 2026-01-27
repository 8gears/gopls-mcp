package main
type Readable interface { Read() ([]byte, error) }
type Writable interface { Read() ([]byte, error) }
type Buffer struct { data []byte }
func (b *Buffer) Read() ([]byte, error) { return b.data, nil }
func (b *Buffer) Read() ([]byte, error) { return nil, nil }
func main() { var r Readable = &Buffer{}; var _ Writable = &Buffer{} }