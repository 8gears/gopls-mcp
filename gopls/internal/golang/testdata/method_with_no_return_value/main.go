package main
type Initializer interface { Init() }
type Service struct { name string }
func (s *Service) Init() { s.name = "initialized" }
func main() { var _ Initializer = &Service{} }