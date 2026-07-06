package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestVersionControl(t *testing.T) {
	h := view.DiffHunk{BaseFrom: 1, BaseTo: 1, From: 1, To: 2}
	assert.True(t, h.PureInsertion())
	assert.False(t, h.PureRemoval())

	h = view.DiffHunk{BaseFrom: 1, BaseTo: 2, From: 1, To: 1}
	assert.False(t, h.PureInsertion())
	assert.True(t, h.PureRemoval())

	e := view.NewEditor(t.TempDir())
	e.SetVersionControl(nil)
	assert.Nil(t, e.VersionControl())
}
