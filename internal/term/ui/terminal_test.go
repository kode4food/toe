package ui_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

func TestTerminalPane(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: spawns a real pty per subtest")
	}
	t.Run("supports pane split", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		m := renderedModel(e)
		m.TerminalAction(e)
		t.Cleanup(func() { ui.CloseAllTerminalPanes(e) })

		_ = e.SplitFocused(view.LayoutVertical)

		assert.Equal(t, 2, e.Tree().Count())
		_, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
	})

	t.Run("dims background when unfocused", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		e.Options().InactiveDim = 50
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		docID := e.Tree().Focus()
		_ = e.SplitFocused(view.LayoutVertical)
		m.TerminalAction(e)
		t.Cleanup(func() { ui.CloseAllTerminalPanes(e) })
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		waitForShellOutput(t, tp)
		e.FocusPane(docID)

		_ = m.View()

		// mocha ui.background (30,30,46) darkened 50%
		assert.Equal(t,
			tui.ColorRGB(15, 15, 23), tp.Emulator().BackgroundColor(),
		)
	})

	t.Run("second invocation is a no-op", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		focus := e.Tree().Focus()
		m.TerminalAction(e)

		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

		m.TerminalAction(e)

		tp2, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		assert.Same(t, tp, tp2)
	})

	t.Run("mouse click focuses it", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		split := e.VSplit(doc.ID())
		assert.NotNil(t, split)

		m.TerminalAction(e)

		termID := e.Tree().Focus()
		tp, ok := e.Tree().Get(termID).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

		m.TerminalAction(e)

		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

		waitForShellOutput(t, tp)
	})

	t.Run("renders every underline style", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

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

		m.TerminalAction(e)

		termID := e.Tree().Focus()
		tp, ok := e.Tree().Get(termID).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		termDir := filepath.Join(dir, "term")
		assert.NoError(t, os.Mkdir(termDir, 0o755))
		tp.IngestOutput(
			[]byte("\x1b]7;file://localhost" + termDir + "\x07"),
		)

		sessionPath := filepath.Join(dir, "session.json")
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
		_ = ui.New(next, command.NewKeymaps()) // registers pane restorers
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		focus := next.Tree().Focus()
		reopened, ok := next.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = reopened.Stop() })
		assert.Equal(t, termDir, reopened.Path())
	})

	t.Run("OSC title updates the status label", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

		assert.Equal(t, "", tp.Title())

		tp.IngestOutput([]byte("\x1b]0;MYTITLE\x07"))
		assert.Equal(t, "MYTITLE", tp.Title())

		assert.Contains(t, m.View().Content, "MYTITLE")
	})

	t.Run("path starts at editor cwd", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

		assert.Equal(t, e.Cwd(), tp.Path())
	})

	t.Run("OSC 7 updates path", func(t *testing.T) {
		dir := t.TempDir()
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

		tp.IngestOutput([]byte("\x1b]7;file://localhost" + dir + "\x07"))

		assert.Equal(t, dir, tp.Path())
	})

	t.Run("OSC 7 unescapes path", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

		tp.IngestOutput([]byte("\x1b]7;file://localhost/tmp/a%20b\x07"))

		assert.Equal(t, "/tmp/a b", tp.Path())
	})

	t.Run("OSC 9 queues a notification", func(t *testing.T) {
		tp := terminalPane(t)

		tp.IngestOutput([]byte("\x1b]9;build finished\x07"))

		assert.Equal(t, []string{"build finished"}, tp.ConsumeNotifications())
		assert.Empty(t, tp.ConsumeNotifications())
	})

	t.Run("OSC 9 ignores ConEmu progress reports", func(t *testing.T) {
		tp := terminalPane(t)

		tp.IngestOutput([]byte("\x1b]9;4;1;50\x07"))

		assert.Empty(t, tp.ConsumeNotifications())
	})

	t.Run("OSC 777 joins title and body", func(t *testing.T) {
		tp := terminalPane(t)

		tp.IngestOutput([]byte("\x1b]777;notify;tests;all green\x07"))

		assert.Equal(t,
			[]string{"tests: all green"}, tp.ConsumeNotifications(),
		)
	})

	t.Run("OSC 777 accepts a title alone", func(t *testing.T) {
		tp := terminalPane(t)

		tp.IngestOutput([]byte("\x1b]777;notify;done\x07"))

		assert.Equal(t, []string{"done"}, tp.ConsumeNotifications())
	})

	t.Run("OSC 777 ignores other subcommands", func(t *testing.T) {
		tp := terminalPane(t)

		tp.IngestOutput([]byte("\x1b]777;precmd;whatever\x07"))

		assert.Empty(t, tp.ConsumeNotifications())
	})

	t.Run("pending notifications are capped", func(t *testing.T) {
		tp := terminalPane(t)

		for i := range 20 {
			tp.IngestOutput(fmt.Appendf(nil, "\x1b]9;note %d\x07", i))
		}

		assert.Len(t, tp.ConsumeNotifications(), 8)
	})

	t.Run("notification becomes a toast", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		m.SetAnimation(false)
		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		_ = m.View()

		batch, ok := m.Init()().(tea.BatchMsg)
		assert.True(t, ok)
		var redraw tea.Cmd
		for _, cmd := range batch {
			if msg, ok := runWithTimeout(cmd, 20*time.Millisecond); ok {
				next, cmd := m.Update(msg)
				m = next.(ui.Model)
				redraw = cmd
			}
		}
		if !assert.NotNil(t, redraw) {
			return
		}

		tp.IngestOutput([]byte("\x1b]9;build finished\x07"))
		msg, ok := runWithTimeout(redraw, time.Second)
		if !assert.True(t, ok) {
			return
		}
		next, _ := m.Update(msg)
		m = next.(ui.Model)

		assert.Contains(t, stripANSI(m.View().Content), "build finished")
		assert.Equal(t,
			highlight.LogTerminal+highlight.LogSeparator+"build finished\n",
			e.MessagesDocument().Text().String(),
		)
	})

	t.Run("mouse wheel scrolls into scrollback", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

		waitForResize(t, tp)
		writeScrollbackLines(t, tp, 50)

		assert.True(t, tp.SearchScrollback("line 3"))
		assert.Positive(t, tp.ScrollOffset())
		assert.False(t, tp.SearchScrollback("does-not-exist"))
	})

	t.Run("wheel down scrolls toward live output", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)

		assert.NoError(t, tp.Stop())
		<-tp.Closed()

		batch, ok := m.Init()().(tea.BatchMsg)
		assert.True(t, ok)
		for _, cmd := range batch {
			if msg, ok := runWithTimeout(cmd, time.Second); ok {
				m2, _ := m.Update(msg)
				m = m2.(ui.Model)
			}
		}

		v := e.FocusedView()
		assert.NotNil(t, v)
		assert.Equal(t, focus, v.ID())
	})

	t.Run("closing all panes kills their shells", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		m.TerminalAction(e)
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

		m.TerminalAction(e)
		focus := e.Tree().Focus()
		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

	t.Run("resize mode captures keys, not the shell", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		leftID := e.Tree().Focus()
		_ = e.SplitFocused(view.LayoutVertical)
		e.Tree().SetFocus(leftID)

		m.TerminalAction(e)
		focus := e.Tree().Focus()
		tp, ok := e.Tree().Get(focus).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		waitForResize(t, tp)

		before := tp.Area().Width

		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'w'})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
		m = m2.(ui.Model)
		_ = m.View()

		assert.Equal(t, before+1, tp.Area().Width)
	})

	t.Run("Ctrl-backslash p pastes clipboard", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		waitForResize(t, tp)

		e.WriteRegister(view.RegisterClipboard, []string{"pasted-text"})

		// the shell owns a bare Space, so the terminal's leader is Ctrl-\
		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: '\\'})
		m = m2.(ui.Model)
		_ = sendKey(m, 'p')

		assert.Eventually(t, func() bool {
			return strings.Contains(tp.Emulator().String(), "pasted-text")
		}, time.Second, 5*time.Millisecond)
	})

	t.Run("bracketed paste reaches shell", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		waitForResize(t, tp)

		_, _ = m.Update(tea.PasteMsg{Content: "pasted-text"})

		assert.Eventually(t, func() bool {
			return strings.Contains(tp.Emulator().String(), "pasted-text")
		}, time.Second, 5*time.Millisecond)
	})

	t.Run("terminal menu uses space trie", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)

		space := func(ch rune) []command.KeyEvent {
			return []command.KeyEvent{
				{Code: command.KeyCode{Char: ' '}},
				{Code: command.KeyCode{Char: ch}},
			}
		}
		// Terminal has the same canonical trie, filtered by terminal mode. Raw
		// Space bypasses keymap dispatch while Ctrl-\ aliases the Space node
		for _, ch := range []rune{'f', 'b'} {
			nor, ok := km.Lookup(view.ModeNormal, space(ch))
			assert.True(t, ok)
			trm, ok := km.Lookup(view.ModeTerminal, space(ch))
			assert.True(t, ok)
			assert.Equal(t, nor.Name, trm.Name)
		}
	})

	t.Run("Ctrl-backslash aliases document space", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)

		before := len(userDocuments(e))
		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: '\\'})
		m = m2.(ui.Model)
		m = sendKey(m, 'w')
		_ = sendKey(m, 'n')

		assert.Len(t, userDocuments(e), before+1)
	})

	t.Run("ctrl-backslash opens space menu", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 100, 30)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		waitForResize(t, tp)

		// Ctrl-\ is the terminal's Space leader, since the shell needs a Space
		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: '\\'})
		m = m2.(ui.Model)
		m = sendKey(m, 'f')
		m = sendSpecial(m, tea.KeyEnter)
		_ = m.View()

		// picking the file replaces the terminal, so its shell is gone
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		want, _ := filepath.EvalSymlinks(path)
		got, _ := filepath.EvalSymlinks(doc.Path())
		assert.Equal(t, want, got)
		_, stillTerm := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.False(t, stillTerm)
	})

	t.Run("OSC 52 syncs nested clipboard writes", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		m := renderedModel(e)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

		tp.IngestOutput([]byte("\x1b]52;c;?\x07"))
		time.Sleep(20 * time.Millisecond)

		assert.Empty(t, clip.System)
	})

	t.Run("bell marks status until focused", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		split := e.VSplit(doc.ID())
		assert.NotNil(t, split)

		m.TerminalAction(e)
		termID := e.Tree().Focus()
		tp, ok := e.Tree().Get(termID).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

	t.Run("top-edge drag scrolls back", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := renderedModel(e)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
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

	t.Run("resize preserves silent content", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		e.Options().Mouse = true
		m := ui.New(e, command.NewKeymaps())
		m = resize(m, 80, 24)

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		split := e.VSplit(doc.ID())
		assert.NotNil(t, split)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		waitForResize(t, tp)

		tp.IngestOutput([]byte("RESIZESURVIVOR\r\n$ "))
		assert.Contains(t, m.View().Content, "RESIZESURVIVOR")

		var sepX int
		e.Tree().WalkSeparators(func(s view.Separator) {
			if s.Layout == view.LayoutVertical {
				sepX = s.X
			}
		})

		// drag the separator to resize the terminal pane, then let the
		// debounced emulator resize apply while the shell stays silent
		m2, _ := m.Update(tea.MouseClickMsg{
			X: sepX, Y: 5, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		var cmd tea.Cmd
		for x := sepX; x <= 65; x += 3 {
			m2, cmd = m.Update(tea.MouseMotionMsg{
				X: x, Y: 5, Button: tea.MouseLeft,
			})
			m = m2.(ui.Model)
		}
		m2, _ = m.Update(tea.MouseReleaseMsg{
			X: 65, Y: 5, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m = feedCmds(m, cmd)
		waitForResize(t, tp)

		// content must remain visible without any further shell output
		assert.Contains(t, m.View().Content, "RESIZESURVIVOR")
	})

	t.Run("Ctrl-w / jumps to a scrollback match", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		waitForResize(t, tp)
		writeScrollbackLines(t, tp, 50)

		m2, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'w'})
		m = m2.(ui.Model)
		m = sendKey(m, '/')
		assert.Contains(t, stripANSI(m.View().Content), " Search scrollback ")
		for _, ch := range "line 3" {
			m = sendKey(m, ch)
		}
		_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

		assert.Positive(t, tp.ScrollOffset())
	})
}

func TestTerminalResize(t *testing.T) {
	t.Run("reflows the emulator synchronously", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		tp, err := ui.NewTerminalPane(
			e, "cat", geom.Size{Width: 20, Height: 10},
		)
		if !assert.NoError(t, err) {
			return
		}
		t.Cleanup(func() { _ = tp.Stop() })

		tp.SetArea(geom.Area{Width: 30, Height: 12})

		// no debounce: the emulator matches the new area at once, reserving
		// the bottom row for the status line
		assert.Equal(t, 30, tp.Emulator().Width())
		assert.Equal(t, 11, tp.Emulator().Height())
	})

	t.Run("holds the shell during a drag", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.NotNil(t, e.VSplit(doc.ID()))
		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		waitForResize(t, tp)
		before := tp.Emulator().Width()

		var sepX int
		e.Tree().WalkSeparators(func(s view.Separator) {
			if s.Layout == view.LayoutVertical {
				sepX = s.X
			}
		})
		m2, _ := m.Update(tea.MouseClickMsg{
			X: sepX, Y: 5, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		var cmd tea.Cmd
		for x := sepX; x <= 60; x += 3 {
			m2, cmd = m.Update(tea.MouseMotionMsg{
				X: x, Y: 5, Button: tea.MouseLeft,
			})
			m = m2.(ui.Model)
		}

		// the pane narrows, but the shell keeps its size until the drag
		// settles
		assert.Less(t, tp.Area().Width, before)
		assert.Equal(t, before, tp.Emulator().Width())
		m2, _ = m.Update(tea.MouseReleaseMsg{
			X: 60, Y: 5, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		feedCmds(m, cmd)
		waitForResize(t, tp)
	})

	t.Run("holds the shell until resize settles", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		m.TerminalAction(e)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })
		waitForResize(t, tp)
		before := tp.Emulator().Width()

		var cmd tea.Cmd
		for w := 78; w >= 60; w -= 2 {
			var m2 tea.Model
			m2, cmd = m.Update(tea.WindowSizeMsg{Width: w, Height: 24})
			m = m2.(ui.Model)
		}

		assert.Equal(t, before, tp.Emulator().Width())
		feedCmds(m, cmd)
		waitForResize(t, tp)
		assert.Equal(t, 60, tp.Emulator().Width())
	})
}

func terminalPane(t *testing.T) *ui.TerminalPane {
	t.Helper()
	e := editorWithText(t, "hello toe")
	m := renderedModel(e)
	m.TerminalAction(e)
	tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
	assert.True(t, ok)
	t.Cleanup(func() { _ = tp.Stop() })
	return tp
}

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

func waitForShellOutput(t *testing.T, tp *ui.TerminalPane) {
	t.Helper()
	emu := tp.Emulator()
	assert.Eventually(t, func() bool {
		if pos := emu.CursorPosition(); pos.X != 0 || pos.Y != 0 {
			return true
		}
		for x := range emu.Width() {
			if c := emu.CellAt(x, 0); c != nil && c.Content != "" &&
				c.Content != " " {
				return true
			}
		}
		return false
	}, 5*time.Second, 5*time.Millisecond)
}

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
