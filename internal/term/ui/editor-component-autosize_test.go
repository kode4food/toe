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
	assert.Greater(t, e.FocusedView().Area().Width, before)
	assert.Less(t, e.FocusedView().Area().Width, 88)
	feedCmds(m, next)

	assert.Equal(t, 88, e.FocusedView().Area().Width)
}

func TestAutoSizeSimultaneous(t *testing.T) {
	e := view.NewEditor(t.TempDir())
	m := resize(ui.New(e, command.NewKeymaps()), 120, 40)
	e.VSplitNew()
	e.HSplitNew()
	e.Options().SetRulers([]int{80})
	m.SetAutoSize(true)
	prevWidth := e.FocusedView().Area().Width
	prevHeight := e.FocusedView().Area().Height

	m2, cmd := m.Update(tea.FocusMsg{})
	m = m2.(ui.Model)
	var widthArrival, heightArrival, tick int
	for cmd != nil {
		m2, cmd = m.Update(cmd())
		m = m2.(ui.Model)
		a := e.FocusedView().Area()
		if a.Width > prevWidth {
			widthArrival = tick
			prevWidth = a.Width
		}
		if a.Height > prevHeight {
			heightArrival = tick
			prevHeight = a.Height
		}
		tick++
	}

	assert.Positive(t, widthArrival)
	assert.Positive(t, heightArrival)
	assert.Equal(t, widthArrival, heightArrival)
}

func TestAutoSizeNoAnimation(t *testing.T) {
	e := view.NewEditor(t.TempDir())
	m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
	assert.True(t, m.Animation())
	e.HSplitNew()
	m.SetAutoSize(true)
	m.SetAnimation(false)
	parentH, ok := e.Tree().FocusedParentHeight()
	assert.True(t, ok)

	m.Update(tea.FocusMsg{})

	// snaps to target within the single update, no tick loop
	want := parentH * ui.DefaultAutoSizeVerticalPct / 100
	assert.Equal(t, want, e.FocusedView().Area().Height)
}

func TestAutoSizeVertical(t *testing.T) {
	e := view.NewEditor(t.TempDir())
	m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
	e.HSplitNew()
	m.SetAutoSize(true)
	parentH, ok := e.Tree().FocusedParentHeight()
	assert.True(t, ok)
	want := parentH * ui.DefaultAutoSizeVerticalPct / 100
	before := e.FocusedView().Area().Height
	assert.Less(t, before, want)

	m2, cmd := m.Update(tea.FocusMsg{})
	m = m2.(ui.Model)
	feedCmds(m, cmd)

	assert.Equal(t, want, e.FocusedView().Area().Height)
}

func TestAutoSizeVerticalPercent(t *testing.T) {
	t.Run("zero disables growth", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
		e.HSplitNew()
		m.SetAutoSize(true)
		m.SetAutoSizeVerticalPercent(0)
		before := e.FocusedView().Area().Height

		m2, cmd := m.Update(tea.FocusMsg{})
		m = m2.(ui.Model)
		feedCmds(m, cmd)

		assert.Equal(t, before, e.FocusedView().Area().Height)
	})

	t.Run("custom percent", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
		e.HSplitNew()
		m.SetAutoSize(true)
		m.SetAutoSizeVerticalPercent(50)
		parentH, ok := e.Tree().FocusedParentHeight()
		assert.True(t, ok)

		m2, cmd := m.Update(tea.FocusMsg{})
		m = m2.(ui.Model)
		feedCmds(m, cmd)

		assert.Equal(t, parentH*50/100, e.FocusedView().Area().Height)
	})

	t.Run("default percent", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
		assert.Equal(t, ui.DefaultAutoSizeVerticalPct,
			m.AutoSizeVerticalPercent())
	})

	t.Run("hundred keeps sibling alive", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 120, 40)
		e.HSplitNew()
		m.SetAutoSize(true)
		m.SetAutoSizeVerticalPercent(100)
		parentH, ok := e.Tree().FocusedParentHeight()
		assert.True(t, ok)

		m2, cmd := m.Update(tea.FocusMsg{})
		m = m2.(ui.Model)
		feedCmds(m, cmd)

		assert.Less(t, e.FocusedView().Area().Height, parentH)
	})

	t.Run("clamps out of range", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
		m.SetAutoSizeVerticalPercent(500)
		assert.Equal(t, 100, m.AutoSizeVerticalPercent())
		m.SetAutoSizeVerticalPercent(-10)
		assert.Equal(t, 0, m.AutoSizeVerticalPercent())
	})
}
