package main
type Writer interface { Write([]byte) (int, error) }