package main
type Processor interface { Process() string }
type ValueReceiver struct { name string }
func (v ValueReceiver) Process() string { return "value: " + v.name }
type PointerReceiver struct { name string }
func (p *PointerReceiver) Process() string { return "pointer: " + p.name }
func main() { var p1 Processor = ValueReceiver{name: "a"}; var p2 Processor = &PointerReceiver{name: "b"}; _ = p1.Process(); _ = p2.Process() }