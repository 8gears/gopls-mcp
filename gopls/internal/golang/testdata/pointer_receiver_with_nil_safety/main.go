package main
type Closer interface { Close() error }
type File struct { path string }
func (f *File) Close() error { return nil }
func main() { var _ Closer = &File{} }