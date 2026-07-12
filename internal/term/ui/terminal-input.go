package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/kode4food/toe/internal/view"
)

type terminalPollMsg struct{}

const terminalPollInterval = 40 * time.Millisecond

// CloseAllTerminalPanes kills every open terminal's shell, including ones
// stashed behind a replacement, so the process doesn't orphan them on exit
func CloseAllTerminalPanes(e *view.Editor) {
	e.Tree().Range(func(p view.Pane) bool {
		closeTerminalChain(p)
		return true
	})
}

var _ PaneInput = (*TerminalPane)(nil)

// HandleKey forwards msg to the shell, unless it is one of the leaders that
// detach or close the pane
func (t *TerminalPane) HandleKey(
	msg tea.KeyPressMsg, cx *Context,
) (EventResult, bool) {
	k := msg.Key()
	if k.Mod&tea.ModCtrl != 0 {
		switch k.Code {
		case '\\':
			cx.Editor.FocusNextView()
			return consumed(), true
		case ']':
			closeTerminal(cx.Editor, t)
			return consumed(), true
		}
	}
	t.SendKey(uv.KeyPressEvent(uv.Key{
		Text: k.Text, Mod: k.Mod, Code: k.Code,
		ShiftedCode: k.ShiftedCode, BaseCode: k.BaseCode, IsRepeat: k.IsRepeat,
	}))
	return consumed(), true
}

// HandleMouse scrolls into scrollback on wheel when the shell hasn't
// requested mouse tracking, or otherwise forwards the event to it
func (t *TerminalPane) HandleMouse(msg tea.Msg, cx *Context) (EventResult, bool) {
	wheel, isWheel := msg.(tea.MouseWheelMsg)
	if isWheel && !t.MouseEnabled() {
		n := cx.Editor.Options().ScrollLines
		switch wheel.Button {
		case tea.MouseWheelUp:
			t.ScrollLines(n)
		case tea.MouseWheelDown:
			t.ScrollLines(-n)
		}
		return consumed(), true
	}
	if !t.MouseEnabled() {
		return ignored(), false
	}
	x, y, btn, mod, ok := mouseFields(msg)
	if !ok {
		return ignored(), false
	}
	m, ok := t.localMouse(cx, x, y)
	if !ok {
		return consumed(), true
	}
	m.Button, m.Mod = btn, mod
	t.SendMouse(wrapMouseEvent(msg, m))
	return consumed(), true
}

// inWindowChord reports whether msg starts or continues the Ctrl-w prefix,
// which must reach the keymap even while a pane has raw input focus
func (e *EditorComponent) inWindowChord(msg tea.KeyPressMsg) bool {
	if len(e.pending) > 0 {
		return true
	}
	k := msg.Key()
	return k.Mod&tea.ModCtrl != 0 && k.Code == 'w'
}

// pollTerminals closes any terminal pane whose shell process has exited,
// deferring closes (which mutate the tree) until after the scan completes
func (e *EditorComponent) pollTerminals(cx *Context) {
	var closing []*TerminalPane
	cx.Editor.Tree().Range(func(p view.Pane) bool {
		if tp, ok := p.(*TerminalPane); ok {
			select {
			case <-tp.Closed():
				closing = append(closing, tp)
			default:
			}
		}
		return true
	})
	for _, tp := range closing {
		closeTerminal(cx.Editor, tp)
	}
}

// paneAt returns the leaf pane whose area contains screen point (x, y)
func paneAt(cx *Context, x, y int) (view.Pane, bool) {
	yOff := 0
	if bufferlineVisible(cx) {
		yOff = 1
	}
	cy := y - yOff
	var found view.Pane
	cx.Editor.Tree().Range(func(p view.Pane) bool {
		a := p.Area()
		if x >= a.X && x < a.X+a.Width && cy >= a.Y && cy < a.Y+a.Height {
			found = p
			return false
		}
		return true
	})
	return found, found != nil
}

func terminalPollCmd() tea.Cmd {
	return tea.Tick(terminalPollInterval, func(time.Time) tea.Msg {
		return terminalPollMsg{}
	})
}

// closeTerminal kills tp's shell and puts back whatever pane it replaced —
// falling back to a scratch buffer if it wasn't opened via replacement
func closeTerminal(e *view.Editor, tp *TerminalPane) {
	_ = tp.Close()
	if tp.restore != nil {
		e.ReplacePane(tp.ID(), tp.restore)
		return
	}
	e.ClosePane(tp.ID())
}

// closeTerminalChain closes p's shell and walks p.restore, in case a
// terminal was itself stashed behind a later one — a no-op for a *View
func closeTerminalChain(p view.Pane) {
	for {
		tp, ok := p.(*TerminalPane)
		if !ok {
			return
		}
		_ = tp.Close()
		p = tp.restore
	}
}

// localMouse translates screen point (x, y) to t's content-local
// coordinates, reporting false if the point falls outside its content area
func (t *TerminalPane) localMouse(cx *Context, x, y int) (uv.Mouse, bool) {
	yOff := 0
	if bufferlineVisible(cx) {
		yOff = 1
	}
	a := t.Area()
	localX, localY := x-a.X, (y-yOff)-a.Y
	contentH := max(a.Height-1, 0)
	if localX < 0 || localX >= a.Width || localY < 0 || localY >= contentH {
		return uv.Mouse{}, false
	}
	return uv.Mouse{X: localX, Y: localY}, true
}

// mouseFields extracts the fields common to every mouse message kind
func mouseFields(msg tea.Msg) (x, y int, btn tea.MouseButton, mod tea.KeyMod, ok bool) {
	switch m := msg.(type) {
	case tea.MouseClickMsg:
		return m.X, m.Y, m.Button, m.Mod, true
	case tea.MouseReleaseMsg:
		return m.X, m.Y, m.Button, m.Mod, true
	case tea.MouseMotionMsg:
		return m.X, m.Y, m.Button, m.Mod, true
	case tea.MouseWheelMsg:
		return m.X, m.Y, m.Button, m.Mod, true
	}
	return 0, 0, 0, 0, false
}

// wrapMouseEvent rebuilds msg's concrete uv event kind around m, once its
// coordinates and button have been translated to the pane's local space
func wrapMouseEvent(msg tea.Msg, m uv.Mouse) uv.MouseEvent {
	switch msg.(type) {
	case tea.MouseReleaseMsg:
		return uv.MouseReleaseEvent(m)
	case tea.MouseMotionMsg:
		return uv.MouseMotionEvent(m)
	case tea.MouseWheelMsg:
		return uv.MouseWheelEvent(m)
	default:
		return uv.MouseClickEvent(m)
	}
}
