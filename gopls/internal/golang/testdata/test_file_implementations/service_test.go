package main
import "testing"
type MockService struct { started bool }
func (m *MockService) Start() error { m.started = true; return nil }
func (m *MockService) Stop() error { m.started = false; return nil }
func TestService(t *testing.T) { var _ Service = &MockService{} }