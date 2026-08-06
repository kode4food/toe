package core

// SlabSize is the chunk size Slab grows by
const SlabSize = 256

// Slab batches values into contiguous backing arrays, handing out
// individually stable pointers to each one
type Slab[T any] struct {
	items []T
}

// Add copies v into the slab and returns a pointer to its slot
func (s *Slab[T]) Add(v T) *T {
	if len(s.items) == cap(s.items) {
		s.items = make([]T, 0, SlabSize)
	}
	s.items = append(s.items, v)
	return &s.items[len(s.items)-1]
}
