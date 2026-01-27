package main
type FileWriter2 struct { path string }
func (f *FileWriter2) Write(p []byte) (int, error) { return len(p), nil }