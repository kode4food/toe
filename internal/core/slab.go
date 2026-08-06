package core

// Slab batches values into contiguous backing arrays, handing out
// individually stable pointers to each one
type Slab[T any] struct {
	items []T
}

const (
	// SlabInitSize is the capacity of a Slab's first backing array
	SlabInitSize = 256

	// SlabMaxSize is the largest Slab will allocate for any later backing array
	SlabMaxSize = 4096
)

// Add copies v into the slab and returns a pointer to its slot
func (s *Slab[T]) Add(v T) *T {
	if len(s.items) == cap(s.items) {
		next := min(max(cap(s.items)*2, SlabInitSize), SlabMaxSize)
		s.items = make([]T, 0, next)
	}
	s.items = append(s.items, v)
	return &s.items[len(s.items)-1]
}
