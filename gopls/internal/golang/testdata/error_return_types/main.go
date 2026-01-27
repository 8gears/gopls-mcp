package main
type Repository interface { Get(id string) (interface{}, error) }
type MockRepository struct {}
func (m *MockRepository) Get(id string) (interface{}, error) { return nil, nil }
func main() { var _ Repository = &MockRepository{} }