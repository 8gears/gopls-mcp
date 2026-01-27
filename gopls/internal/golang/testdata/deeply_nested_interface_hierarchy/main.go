package main
type Base interface { BaseMethod() string }
type Middle1 interface { Base; M1() }
type Middle2 interface { Middle1; M2() }
type Final interface { Middle2; M3() }
type Impl struct{}
func (i Impl) BaseMethod() string { return "base" }
func (i Impl) M1() {}
func (i Impl) M2() {}
func (i Impl) M3() {}
func main() { var _ Final = Impl{} }