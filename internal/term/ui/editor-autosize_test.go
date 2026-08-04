package ui_test

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
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

func TestAutoSizeNoAnimation(t *testing.T) {
	e := view.NewEditor(t.TempDir())
	m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
	assert.True(t, m.Animation())
	e.VSplitNew()
	e.Options().SetRulers([]int{80})
	m.SetAutoSize(true)
	m.SetAnimation(false)
	before := e.FocusedView().Area().Width

	m.Update(tea.FocusMsg{})

	// snaps to target within the single update, no tick loop
	assert.Greater(t, e.FocusedView().Area().Width, before)
	assert.Equal(t, 88, e.FocusedView().Area().Width)
}

func TestAutoSizePane(t *testing.T) {
	t.Run("hex fits 16 bytes per row", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blob.bin")
		assert.NoError(t, os.WriteFile(path, make([]byte, 1024), 0o644))
		e := view.NewEditor(dir)
		m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
		m.SetAutoSize(true)
		m.SetAnimation(false)
		e.VSplitNew()
		pane, err := ui.NewBinaryPane(e, path)
		assert.NoError(t, err)
		e.ReplacePane(e.Tree().Focus(), pane)
		assert.Less(t, pane.Area().Width, 78)

		m.Update(tea.FocusMsg{})

		// 8 offset columns, padding, and two 8-byte groups
		assert.Equal(t, 78, pane.Area().Width)
	})

	t.Run("terminal fits 80 columns", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 120, 40)
		m.SetAutoSize(true)
		m.SetAnimation(false)
		e.VSplitNew()
		e.HSplitNew()
		pane, err := ui.NewTerminalPane(e, "cat", geom.Size{})
		assert.NoError(t, err)
		t.Cleanup(func() { _ = pane.Stop() })
		e.ReplacePane(e.Tree().Focus(), pane)
		before := pane.Area()
		assert.Less(t, before.Width, 80)

		m.Update(tea.FocusMsg{})

		assert.Equal(t, 80, pane.Area().Width)
		assert.Equal(t, before.Height, pane.Area().Height)
	})

	t.Run("image has no width target", func(t *testing.T) {
		dir := t.TempDir()
		path := writeRenderImage(t, dir, 40, 20, color.RGBA{G: 255, A: 255})
		e := view.NewEditor(dir)
		m := resize(ui.New(e, command.NewKeymaps()), 120, 24)
		m.SetAutoSize(true)
		m.SetAnimation(false)
		e.VSplitNew()
		pane, err := ui.NewImagePane(e, path)
		assert.NoError(t, err)
		e.ReplacePane(e.Tree().Focus(), pane)
		before := pane.Area().Width

		m.Update(tea.FocusMsg{})

		assert.Equal(t, before, pane.Area().Width)
	})
}

func TestAutoSizeScrolled(t *testing.T) {
	e := view.NewEditor(t.TempDir())
	m := resize(ui.New(e, command.NewKeymaps()), 300, 24)
	e.VSplitNew()
	e.Options().SetRulers([]int{80})
	m.SetAutoSize(true)
	m.SetAnimation(false)

	vs := e.AllViews()
	sepX := vs[0].Area().X + vs[0].Area().Width
	res, ok := e.Tree().SeparatorAt(geom.Point{X: sepX})
	assert.True(t, ok)
	e.Tree().MoveSeparator(res.ContainerID, res.ChildIdx, res.Layout, 75)

	// the target is where the ruler sits at offset zero, so scrolling right
	// does not shrink it even though the ruler is already painted
	v := e.AllViews()[0]
	off := v.Offset()
	off.HorizontalOffset = 15
	v.SetOffset(off)
	e.FocusView(v.ID())

	m.Update(tea.FocusMsg{})

	assert.Equal(t, 88, v.Area().Width)
}
