package ui_test

import (
	"fmt"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestTerminalPane(t *testing.T) {
	t.Run("opens in place and restores on close", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		focus := e.Tree().Focus()
		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)

		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: ']'})
		m = m2.(ui.Model)
		_ = m.View()

		v, ok := e.FocusedView()
		assert.True(t, ok)
		assert.Equal(t, focus, v.ID())
	})

	t.Run("mouse click focuses it", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		_, ok = e.VSplit(doc.ID())
		assert.True(t, ok)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)

		termID := e.Tree().Focus()
		tp, ok := e.Tree().Get(termID).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		area := tp.Area()

		e.FocusNextView()
		assert.NotEqual(t, termID, e.Tree().Focus())

		m2, _ := m.Update(tea.MouseClickMsg{
			X: area.X, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		_ = m.View()

		assert.Equal(t, termID, e.Tree().Focus())
	})

	t.Run("session restore reopens the shell", func(t *testing.T) {
		dir := t.TempDir()
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		sessionPath := filepath.Join(dir, "session.toml")
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(80, 24)
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		nextModel := ui.New(next, command.NewKeymaps())
		nextModel.RestoreTerminalPanes(next)

		focus := next.Tree().Focus()
		reopened, ok := next.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = reopened.Close() })
		assert.Empty(t, next.TakePendingTerminals())
	})

	t.Run("OSC title updates the status label", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		assert.Equal(t, "", tp.Title())

		_, err := tp.Emulator().Write([]byte("\x1b]0;MYTITLE\x07"))
		assert.NoError(t, err)
		assert.Equal(t, "MYTITLE", tp.Title())

		assert.Contains(t, m.View().Content, "MYTITLE")
	})

	t.Run("mouse wheel scrolls into scrollback", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		writeScrollbackLines(t, tp, 50)
		assert.Positive(t, tp.Emulator().ScrollbackLen())
		assert.Equal(t, 0, tp.ScrollOffset())

		area := tp.Area()
		m2, _ := m.Update(tea.MouseWheelMsg{
			X: area.X, Y: area.Y, Button: tea.MouseWheelUp,
		})
		m = m2.(ui.Model)
		_ = m.View()
		assert.Positive(t, tp.ScrollOffset())

		m3, _ := m.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
		_ = m3.(ui.Model)
		assert.Equal(t, 0, tp.ScrollOffset())
	})

	t.Run("mouse mode tracks DEC private mode 1000", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		assert.False(t, tp.MouseEnabled())

		_, err := tp.Emulator().Write([]byte("\x1b[?1000h"))
		assert.NoError(t, err)
		assert.True(t, tp.MouseEnabled())

		_, err = tp.Emulator().Write([]byte("\x1b[?1000l"))
		assert.NoError(t, err)
		assert.False(t, tp.MouseEnabled())
	})

	t.Run("search jumps to a scrollback match", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		writeScrollbackLines(t, tp, 50)

		assert.True(t, tp.SearchScrollback("line 3"))
		assert.Positive(t, tp.ScrollOffset())
		assert.False(t, tp.SearchScrollback("does-not-exist"))
	})
}

func writeScrollbackLines(t *testing.T, tp *ui.TerminalPane, n int) {
	t.Helper()
	for i := range n {
		_, err := tp.Emulator().Write(fmt.Appendf(nil, "line %d\r\n", i))
		assert.NoError(t, err)
	}
}
