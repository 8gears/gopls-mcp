package main
type MyError interface { Error() string }
type CustomError struct { msg string }
func (e *CustomError) Error() string { return e.msg }
func main() { var _ MyError = &CustomError{msg: "failed"} }