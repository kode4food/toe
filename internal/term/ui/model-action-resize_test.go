package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestResizeViewAction(t *testing.T) {
	e := view.NewEditor(t.TempDir())
	e.ResizeTree(geom.Size{Width: 80, Height: 24})
	e.VSplitNew()
	e.TogglePaneMaximized()

	m := ui.New(e, command.NewKeymaps())
	m.ResizeViewAction(e)

	assert.False(t, e.Tree().Maximized())
}
