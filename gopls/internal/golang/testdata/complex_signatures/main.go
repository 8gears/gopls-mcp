package main
import ("context"; "io")
type Processor interface { Process(ctx context.Context, r io.Reader, opts map[string]interface{}) (<-chan []byte, error) }
type AsyncProcessor struct { bufSize int }
func (a *AsyncProcessor) Process(ctx context.Context, r io.Reader, opts map[string]interface{}) (<-chan []byte, error) { return nil, nil }
func main() { var _ Processor = &AsyncProcessor{} }