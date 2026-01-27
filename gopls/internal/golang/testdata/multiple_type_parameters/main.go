package main
type Mapper[K comparable, V any] interface { Get(key K) (V, bool); Set(key K, value V) }
type HashMap[K comparable, V any] struct { data map[K]V }
func (m *HashMap[K, V]) Get(key K) (V, bool) { val, ok := m.data[key]; return val, ok }
func (m *HashMap[K, V]) Set(key K, value V) { if m.data == nil { m.data = make(map[K]V) }; m.data[key] = value }
func main() { var _ Mapper[string, int] = &HashMap[string, int]{} }