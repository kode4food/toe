package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestSlab(t *testing.T) {
	t.Run("returns the stored value", func(t *testing.T) {
		var slab core.Slab[int]
		p := slab.Add(7)
		assert.Equal(t, 7, *p)
	})

	t.Run("pointers survive later segments", func(t *testing.T) {
		var slab core.Slab[int]
		n := core.SlabInitSize * 3
		ptrs := make([]*int, 0, n)
		for i := range n {
			ptrs = append(ptrs, slab.Add(i))
		}
		for i, p := range ptrs {
			assert.Equal(t, i, *p)
		}
	})

	t.Run("later segments grow", func(t *testing.T) {
		n := core.SlabInitSize * 20
		fixedSegments := n / core.SlabInitSize

		segments := testing.AllocsPerRun(1, func() {
			var slab core.Slab[int]
			for range n {
				slab.Add(0)
			}
		})

		// fixed SlabInitSize chunking would need one array per SlabInitSize
		// items; doubling growth covers the same set in far fewer arrays
		assert.Less(t, int(segments), fixedSegments)
	})
}
