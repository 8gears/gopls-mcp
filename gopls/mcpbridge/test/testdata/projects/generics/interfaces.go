
package genericiface

// Generic interface
type Processor[T any] interface {
	Process(T) T
}

// Multiple implementations
type StringProcessor struct{}

func (s StringProcessor) Process(str string) string {
	return "processed: " + str
}

type IntProcessor struct{}

func (i IntProcessor) Process(n int) int {
	return n * 2
}

// Generic function using interface
func RunProcessor[T any](p Processor[T], input T) T {
	return p.Process(input)
}
