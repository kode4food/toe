package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestPosition(t *testing.T) {
	t.Run("adds positions", func(t *testing.T) {
		p := core.Position{Row: 2, Col: 3}
		q := core.Position{Row: 4, Col: 5}

		assert.Equal(t, core.Position{Row: 6, Col: 8}, p.Add(q))
	})

	t.Run("subtracts positions", func(t *testing.T) {
		p := core.Position{Row: 8, Col: 13}
		q := core.Position{Row: 3, Col: 5}

		assert.Equal(t, core.Position{Row: 5, Col: 8}, p.Sub(q))
	})

	t.Run("reports zero position", func(t *testing.T) {
		assert.True(t, core.Position{}.IsZero())
		assert.False(t, core.Position{Col: 1}.IsZero())
	})

	t.Run("traverses line feeds", func(t *testing.T) {
		p := core.Position{Row: 3, Col: 4}

		assert.Equal(t, core.Position{Row: 5, Col: 3},
			p.Traverse("ab\ncd\nefg"),
		)
	})

	t.Run("reference carriage return handling", func(t *testing.T) {
		p := core.Position{}

		assert.Equal(t, core.Position{Row: 1, Col: 1},
			p.Traverse("a\r\nb"),
		)
	})
}
