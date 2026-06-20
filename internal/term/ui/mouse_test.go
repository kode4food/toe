package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
)

func TestMouseMiddlePaste(t *testing.T) {
	t.Run("pastes at clicked position", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("XY"), 0o644))
		e := editorWithText(t, "abcd")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 6, Y: 0, Button: tea.MouseMiddle,
		})
		_ = m2

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abXYcd", doc.Text().String())
		assert.Equal(t, 2, cursorPos(t, e))
	})

	t.Run("disabled leaves document unchanged", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("XY"), 0o644))
		e := editorWithText(t, "abcd")
		e.Options().MiddleClickPaste = false
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 6, Y: 0, Button: tea.MouseMiddle,
		})
		_ = m2

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abcd", doc.Text().String())
	})

	t.Run("alt replaces selection", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("XY"), 0o644))
		e := editorWithText(t, "abcd")
		setSelection(t, e, []core.Range{core.NewRange(1, 3)}, 0)
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 0, Y: 0, Button: tea.MouseMiddle, Mod: tea.ModAlt,
		})
		_ = m2

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "aXYd", doc.Text().String())
	})
}

func TestMouseWheelScroll(t *testing.T) {
	// renderedModel gives a 40×8 window; row 7 is the status/command line,
	// which is outside all editor panes.

	t.Run("wheel over pane scrolls that pane", func(t *testing.T) {
		e := editorWithText(t, "a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
		e.SetViewHeight(6)
		m := renderedModel(e)

		v, ok := e.FocusedView()
		assert.True(t, ok)
		before := v.Offset().Anchor

		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 5, Y: 0, Button: tea.MouseWheelDown,
		})
		_ = m2

		assert.Greater(t, v.Offset().Anchor, before)
	})

	t.Run("status bar wheel ignored", func(t *testing.T) {
		e := editorWithText(t, "a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
		e.SetViewHeight(6)
		m := renderedModel(e)

		v, ok := e.FocusedView()
		assert.True(t, ok)
		before := v.Offset().Anchor

		// 40×8 window, no bufferline: ResizeTree gives the pane height=7,
		// so Y=6 is the pane's own status bar row (not content)
		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 5, Y: 6, Button: tea.MouseWheelDown,
		})
		_ = m2

		assert.Equal(t, before, v.Offset().Anchor)
	})

	t.Run("wheel outside all panes does not scroll", func(t *testing.T) {
		e := editorWithText(t, "a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
		e.SetViewHeight(6)
		m := renderedModel(e)

		v, ok := e.FocusedView()
		assert.True(t, ok)
		before := v.Offset().Anchor

		// Y=7 is the command line — outside any editor pane in this 40×8 window
		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 5, Y: 7, Button: tea.MouseWheelDown,
		})
		_ = m2

		assert.Equal(t, before, v.Offset().Anchor)
	})
}

func TestMouseClickPositioning(t *testing.T) {
	t.Run("positions cursor on content click", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 7, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2

		// X 7 lands past the line-number gutter at content column 3
		assert.Equal(t, 3, cursorPos(t, e))
	})

	t.Run("ignores click on status or command line", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)

		// place the cursor in the content area first
		m2, _ := m.Update(tea.MouseClickMsg{
			X: 7, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.Equal(t, 3, cursorPos(t, e))

		// the bottom row of the 8-high window is the status/command line, never
		// editor content, so a click there must leave the cursor put
		m3, _ := m.Update(tea.MouseClickMsg{
			X: 1, Y: 7, Button: tea.MouseLeft,
		})
		_ = m3

		assert.Equal(t, 3, cursorPos(t, e))
	})
}

func renderedModel(e *view.Editor) ui.Model {
	m := ui.New(e, command.NewKeymaps())
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m = m2.(ui.Model)
	_ = m.View()
	return m
}

func editorWithText(t *testing.T, text string) *view.Editor {
	t.Helper()
	e := view.NewEditor("/tmp")
	e.SetConfig(config.DefaultConfig())
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	rope := doc.Text()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(0, 0, text),
	})
	assert.NoError(t, err)
	tx := core.NewTransaction(rope).
		WithChanges(cs).
		WithSelection(core.PointSelection(0))
	assert.NoError(t, e.Apply(tx))
	return e
}

func setSelection(
	t *testing.T, e *view.Editor, ranges []core.Range, primary int,
) {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	sel, err := core.NewSelection(ranges, primary)
	assert.NoError(t, err)
	doc.SetSelectionFor(v.ID(), sel)
}

func cursorPos(t *testing.T, e *view.Editor) int {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	return doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
}
