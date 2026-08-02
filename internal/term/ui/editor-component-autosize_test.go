package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestAutoSize(t *testing.T) {
	e := view.NewEditor(t.TempDir())
	m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
	e.VSplitNew()
	e.Options().SetRulers([]int{120, 80, 120})
	m.SetAutoSize(true)
	before := e.FocusedView().Area().Width

	m2, cmd := m.Update(tea.FocusMsg{})
	m = m2.(ui.Model)
	assert.Equal(t, before, e.FocusedView().Area().Width)
	m2, next := m.Update(cmd())
	m = m2.(ui.Model)
	assert.Equal(t, before+3, e.FocusedView().Area().Width)
	feedCmds(m, next)

	assert.Equal(t, 88, e.FocusedView().Area().Width)
}
