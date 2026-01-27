package main
type Logger interface { Log(msg string, args ...interface{}) }
type ConsoleLogger struct { prefix string }
func (c *ConsoleLogger) Log(msg string, args ...interface{}) {}
func main() { var _ Logger = &ConsoleLogger{} }