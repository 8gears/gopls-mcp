package main
type Service interface { Start() error; Stop() error }
type RealService struct { name string }
func (s *RealService) Start() error { return nil }
func (s *RealService) Stop() error { return nil }