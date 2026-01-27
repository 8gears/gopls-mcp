package main
type FileWriter1 struct { path string }
func (f *FileWriter1) Write(p []byte) (int, error) { return len(p), nil }