package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestRopePosition(t *testing.T) {
	rope := core.NewRope("l0\nabcdef\n")

	t.Run("start is 1:1", func(t *testing.T) {
		at, err := rope.Position(0)
		assert.NoError(t, err)
		assert.Equal(t, core.Position{Line: 1, Col: 1}, at)
	})

	t.Run("reports 1-based line and column", func(t *testing.T) {
		// char 5 is 'c' on the second line
		at, err := rope.Position(5)
		assert.NoError(t, err)
		assert.Equal(t, core.Position{Line: 2, Col: 3}, at)
	})
}
