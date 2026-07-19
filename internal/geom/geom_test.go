package geom_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
)

func TestPoint(t *testing.T) {
	p := geom.Point{X: 3, Y: 5}
	d := geom.Point{X: 2, Y: 4}
	assert.Equal(t, geom.Point{X: 5, Y: 9}, p.Add(d))
	assert.Equal(t, geom.Point{X: 1, Y: 1}, p.Sub(d))
}

func TestSize(t *testing.T) {
	s := geom.Size{Width: 4, Height: 3}
	assert.False(t, s.Empty())
	assert.True(t, geom.Size{Width: 0, Height: 3}.Empty())
	assert.True(t, s.Contains(geom.Point{X: 3, Y: 2}))
	assert.False(t, s.Contains(geom.Point{X: 4, Y: 2}))
	assert.Equal(t,
		geom.Point{X: 3, Y: 0},
		s.Clamp(geom.Point{X: 8, Y: -1}),
	)
}

func TestArea(t *testing.T) {
	a := geom.Area{
		Point: geom.Point{X: 2, Y: 3},
		Size:  geom.Size{Width: 6, Height: 4},
	}
	assert.True(t, a.Contains(geom.Point{X: 7, Y: 6}))
	assert.False(t, a.Contains(geom.Point{X: 8, Y: 6}))
	assert.Equal(t, 7, a.Right())
	assert.Equal(t, 6, a.Bottom())
	assert.Equal(t,
		geom.Area{
			Point: geom.Point{X: 3, Y: 5},
			Size:  a.Size,
		},
		a.Translate(geom.Point{X: 1, Y: 2}),
	)
	assert.True(t, a.Intersects(geom.Area{
		Point: geom.Point{X: 7, Y: 6},
		Size:  geom.Size{Width: 2, Height: 2},
	}))
	assert.False(t, a.Intersects(geom.Area{
		Point: geom.Point{X: 3, Y: 4},
	}))
	assert.Equal(t,
		geom.Point{X: 4, Y: 4},
		a.Center(geom.Size{Width: 2, Height: 2}),
	)
	inset := a.Inset(geom.Size{Width: 4, Height: 3})
	assert.True(t, inset.Empty())
}
