package main
type Writer interface { Write([]byte) (int, error) }
type File struct { path string }
func (f *File) Write(p []byte) (int, error) { return len(p), nil }
type Buffer struct { data []byte }
func (b *Buffer) Write(p []byte) (int, error) { b.data = append(b.data, p...); return len(p), nil }
type Network struct { addr string }
func (n *Network) Write(p []byte) (int, error) { return len(p), nil }
func main() { var w Writer = &File{}; w.Write(nil) }