package ui_test

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
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

		select {
		case <-tp.Updates():
		case <-time.After(2 * time.Second):
			t.Fatal("expected an update signal for the shell's startup output")
		}

		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: ']'})
		m = m2.(ui.Model)
		_ = m.View()

		v, ok := e.FocusedView()
		assert.True(t, ok)
		assert.Equal(t, focus, v.ID())
	})

	t.Run("second invocation is a no-op", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		focus := e.Tree().Focus()
		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)

		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		select {
		case <-tp.Updates():
		case <-time.After(2 * time.Second):
			t.Fatal("expected an update signal for the shell's startup output")
		}

		cont = m.TerminalAction()(e)
		assert.Nil(t, cont)

		tp2, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		assert.Same(t, tp, tp2)
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

	t.Run("falls back when $SHELL is unset", func(t *testing.T) {
		t.Setenv("SHELL", "")
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)
		focus := e.Tree().Focus()

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)

		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		select {
		case <-tp.Updates():
		case <-time.After(2 * time.Second):
			t.Fatal("expected the fallback shell to produce output")
		}
	})

	t.Run("renders every underline style", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		for _, sgr := range []string{"4", "4:2", "4:3", "4:4", "4:5"} {
			_, err := tp.Emulator().Write(
				fmt.Appendf(nil, "\x1b[%sma\x1b[0m", sgr),
			)
			assert.NoError(t, err)
		}

		assert.NotPanics(t, func() { _ = m.View() })
	})

	t.Run("focused click forwards to the shell", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)

		termID := e.Tree().Focus()
		tp, ok := e.Tree().Get(termID).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		area := tp.Area()

		_, err := tp.Emulator().Write([]byte("\x1b[?1000h"))
		assert.NoError(t, err)
		assert.True(t, tp.MouseEnabled())

		m2, _ := m.Update(tea.MouseClickMsg{
			X: area.X, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		_ = m.View()

		// the click was consumed by the shell, not the normal focus/select
		// logic, so the terminal pane stays focused and mouse mode stays on
		assert.Equal(t, termID, e.Tree().Focus())
		assert.True(t, tp.MouseEnabled())
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

		tp.IngestOutput([]byte("\x1b]0;MYTITLE\x07"))
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

		waitForResize(t, tp)
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

		waitForResize(t, tp)
		writeScrollbackLines(t, tp, 50)

		assert.True(t, tp.SearchScrollback("line 3"))
		assert.Positive(t, tp.ScrollOffset())
		assert.False(t, tp.SearchScrollback("does-not-exist"))
	})

	t.Run("Ctrl-backslash detaches without closing", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		_, ok = e.VSplit(doc.ID())
		assert.True(t, ok)
		focus := e.Tree().Focus()

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: '\\'})
		m = m2.(ui.Model)
		_ = m.View()

		assert.NotEqual(t, focus, e.Tree().Focus())
		_, ok = e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
	})

	t.Run("wheel down scrolls toward live output", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		area := tp.Area()
		writeScrollbackLines(t, tp, 50)
		tp.ScrollLines(50)
		before := tp.ScrollOffset()
		assert.Positive(t, before)

		m2, _ := m.Update(tea.MouseWheelMsg{
			X: area.X, Y: area.Y, Button: tea.MouseWheelDown,
		})
		m = m2.(ui.Model)
		_ = m.View()

		assert.Less(t, tp.ScrollOffset(), before)
	})

	t.Run("release and motion forward when enabled", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		area := tp.Area()

		_, err := tp.Emulator().Write([]byte("\x1b[?1000h"))
		assert.NoError(t, err)
		assert.True(t, tp.MouseEnabled())

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: area.X, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseMotionMsg{
			X: area.X, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		_ = m.View()

		assert.Equal(t, tp.ID(), e.Tree().Get(e.Tree().Focus()).ID())
	})

	t.Run("click below content area is dropped", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		area := tp.Area()

		_, err := tp.Emulator().Write([]byte("\x1b[?1000h"))
		assert.NoError(t, err)

		statusRow := area.Y + area.Height - 1
		m2, _ := m.Update(tea.MouseClickMsg{
			X: area.X, Y: statusRow, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		_ = m.View()

		assert.Equal(t, tp.ID(), e.Tree().Get(e.Tree().Focus()).ID())
	})

	t.Run("focused click is a no-op untracked", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		area := tp.Area()
		assert.False(t, tp.MouseEnabled())

		m2, _ := m.Update(tea.MouseClickMsg{
			X: area.X, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		_ = m.View()

		assert.Equal(t, tp.ID(), e.Tree().Get(e.Tree().Focus()).ID())
	})

	t.Run("polling restores the pane on shell exit", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)
		focus := e.Tree().Focus()

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)

		assert.NoError(t, tp.Close())
		<-tp.Closed()

		batch, ok := m.Init()().(tea.BatchMsg)
		assert.True(t, ok)
		for _, cmd := range batch {
			if msg, ok := runWithTimeout(cmd, time.Second); ok {
				m2, _ := m.Update(msg)
				m = m2.(ui.Model)
			}
		}

		v, ok := e.FocusedView()
		assert.True(t, ok)
		assert.Equal(t, focus, v.ID())
	})

	t.Run("closing all panes kills their shells", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)

		ui.CloseAllTerminalPanes(e)

		<-tp.Closed()
	})

	t.Run("Ctrl-w x isn't bound while focused", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		focus := e.Tree().Focus()
		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		waitForResize(t, tp)

		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'w'})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
		m = m2.(ui.Model)
		_ = m.View()

		tp2, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		assert.Same(t, tp, tp2)
	})

	t.Run("Ctrl-w p pastes the clipboard register", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		waitForResize(t, tp)

		e.WriteRegister(view.RegisterClipboard, []string{"pasted-text"})

		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'w'})
		m = m2.(ui.Model)
		_ = sendKey(m, 'p')

		assert.Eventually(t, func() bool {
			return strings.Contains(tp.Emulator().String(), "pasted-text")
		}, time.Second, 5*time.Millisecond)
	})

	t.Run("OSC 52 syncs nested clipboard writes", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		payload := base64.StdEncoding.EncodeToString([]byte("nested-copy"))
		tp.IngestOutput(fmt.Appendf(nil, "\x1b]52;c;%s\x07", payload))

		assert.Eventually(t, func() bool {
			return clip.System == "nested-copy"
		}, time.Second, 5*time.Millisecond)
	})

	t.Run("OSC 52 query is ignored, not written", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })

		tp.IngestOutput([]byte("\x1b]52;c;?\x07"))
		time.Sleep(20 * time.Millisecond)

		assert.Empty(t, clip.System)
	})

	t.Run("bell marks status until focused", func(t *testing.T) {
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

		e.FocusNextView()
		assert.NotEqual(t, termID, e.Tree().Focus())

		tp.IngestOutput([]byte("\x07"))
		assert.Eventually(t, func() bool {
			return strings.Contains(m.View().Content, "TRM*")
		}, time.Second, 5*time.Millisecond)

		e.Tree().SetFocus(termID)
		content := m.View().Content
		assert.NotContains(t, content, "TRM*")
	})

	t.Run("click-drag copies selected text", func(t *testing.T) {
		t.Setenv("SHELL", "/bin/cat")
		e := editorWithText(t, "hello toe")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		waitForResize(t, tp)

		_, err := tp.Emulator().Write([]byte("COPYME"))
		assert.NoError(t, err)
		assert.False(t, tp.MouseEnabled())

		area := tp.Area()
		m2, _ := m.Update(tea.MouseClickMsg{
			X: area.X, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseMotionMsg{
			X: area.X + 5, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseReleaseMsg{
			X: area.X + 5, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		_ = m.View()

		assert.Equal(t, "COPYME", clip.System)
	})

	t.Run("click-drag selects while scrolled back", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		waitForResize(t, tp)
		writeScrollbackLines(t, tp, 50)

		// pick a scrollback line and scroll so it lands on the top content
		// row, using the same window math the renderer uses
		sb := tp.Emulator().Scrollback()
		target := sb.Len() - 5
		area := tp.Area()
		contentH := area.Height - 1
		total := sb.Len() + tp.Emulator().Height()
		tp.ScrollLines(total - contentH - target)
		assert.Positive(t, tp.ScrollOffset())
		want := strings.TrimRight(sb.Line(target).String(), " ")

		m2, _ := m.Update(tea.MouseClickMsg{
			X: area.X, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseReleaseMsg{
			X: area.X + area.Width - 1, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		_ = m.View()

		assert.Equal(t, want, clip.System)
	})

	t.Run("drag past the top edge auto-scrolls scrollback", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		waitForResize(t, tp)
		writeScrollbackLines(t, tp, 50)
		assert.Equal(t, 0, tp.ScrollOffset())

		area := tp.Area()
		m2, cmd := m.Update(tea.MouseClickMsg{
			X: area.X, Y: area.Y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.Nil(t, cmd)

		m2, cmd = m.Update(tea.MouseMotionMsg{
			X: area.X, Y: area.Y - 1, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.NotNil(t, cmd)

		msg, ok := runWithTimeout(cmd, time.Second)
		assert.True(t, ok)
		m2, _ = m.Update(msg)
		_ = m2.(ui.Model)

		assert.Positive(t, tp.ScrollOffset())
	})

	t.Run("Ctrl-w / jumps to a scrollback match", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Close() })
		waitForResize(t, tp)
		writeScrollbackLines(t, tp, 50)

		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'w'})
		m = m2.(ui.Model)
		m = sendKey(m, '/')
		for _, ch := range "line 3" {
			m = sendKey(m, ch)
		}
		_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

		assert.Positive(t, tp.ScrollOffset())
	})
}

// runWithTimeout runs cmd and reports its message, or ok=false if it hasn't
// fired within d — used to skip Init's long-lived, event-driven commands
func runWithTimeout(cmd tea.Cmd, d time.Duration) (tea.Msg, bool) {
	if cmd == nil {
		return nil, false
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		return msg, true
	case <-time.After(d):
		return nil, false
	}
}

// waitForResize blocks until the pane's debounced PTY/emulator resize has
// applied, so a test's writes land at the pane's real dimensions
func waitForResize(t *testing.T, tp *ui.TerminalPane) {
	t.Helper()
	area := tp.Area()
	w, h := max(area.Width, 1), max(area.Height-1, 1)
	assert.Eventually(t, func() bool {
		return tp.Emulator().Width() == w && tp.Emulator().Height() == h
	}, time.Second, 5*time.Millisecond)
}

func writeScrollbackLines(t *testing.T, tp *ui.TerminalPane, n int) {
	t.Helper()
	for i := range n {
		_, err := tp.Emulator().Write(fmt.Appendf(nil, "line %d\r\n", i))
		assert.NoError(t, err)
	}
}
