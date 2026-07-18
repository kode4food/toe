package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

type (
	terminalDragScrollMsg struct {
		dc    Draggable
		gen   int
		toTop bool
	}
	terminalPollMsg struct{}
)

const terminalPollInterval = 40 * time.Millisecond

// CloseAllTerminalPanes kills every open terminal's shell, including ones
// stashed behind a replacement, so the process doesn't orphan them on exit
func CloseAllTerminalPanes(e *view.Editor) {
	e.Tree().Range(func(p view.Pane) bool {
		p.Shutdown()
		return true
	})
}

// HandleEvent routes key and mouse events to the shell
func (t *TerminalPane) HandleEvent(
	msg tea.Msg, cx *Context,
) (EventResult, bool) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		return t.handleKey(key, cx)
	}
	return t.handleMouse(msg, cx)
}

// handleKey forwards msg to the shell, unless it is one of the leaders that
// detach or close the pane
func (t *TerminalPane) handleKey(
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

// handleMouse scrolls into scrollback on wheel when the shell hasn't
// requested mouse tracking, or otherwise forwards the event to it
func (t *TerminalPane) handleMouse(
	msg tea.Msg, cx *Context,
) (EventResult, bool) {
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

// BeginDrag starts a selection if the shell hasn't grabbed mouse tracking, or
// forwards the click to it otherwise
func (t *TerminalPane) BeginDrag(cx *Context, x, y int, mod tea.KeyMod) bool {
	m, ok := t.localMouse(cx, x, y)
	if !ok {
		return false
	}
	if t.MouseEnabled() {
		t.SendMouse(uv.MouseClickEvent(uv.Mouse{
			X: m.X, Y: m.Y, Button: tea.MouseLeft, Mod: mod,
		}))
		return false
	}
	t.beginSelection(uv.Position{X: m.X, Y: m.Y})
	return true
}

// ContinueDrag extends the selection to (x, y), auto-scrolling and
// scheduling further ticks if the drag has crossed the pane's top or
// bottom edge
func (t *TerminalPane) ContinueDrag(cx *Context, x, y int) tea.Cmd {
	yOff := 0
	if bufferlineVisible(cx) {
		yOff = 1
	}
	a := t.Area()
	contentH := max(a.Height-1, 0)
	scrollOff := cx.Editor.Options().ScrollOff
	atTop, atBottom, clampedY := t.drag.update(
		y-yOff, a.Y, a.Y+contentH-1, autoScrollMargin(contentH, scrollOff),
	)
	localX := min(max(x-a.X, 0), max(a.Width-1, 0))
	t.extendSelection(uv.Position{X: localX, Y: clampedY - a.Y})
	return t.drag.trigger(atTop, atBottom, localX, t.scheduleDragTick)
}

// EndDrag finalizes the selection at (x, y), copying it to the clipboard
func (t *TerminalPane) EndDrag(cx *Context, x, y int) tea.Cmd {
	t.drag.stop()
	m := t.clampedMouse(cx, x, y)
	if text := t.endSelection(uv.Position{X: m.X, Y: m.Y}); text != "" {
		cx.Editor.WriteRegister(view.RegisterClipboard, []string{text})
		cx.Editor.SetStatusMsg(i18n.Text(i18n.StatusClipboardCopied))
	}
	return nil
}

// CancelDrag stops any pending auto-scroll tick, without side effects
func (t *TerminalPane) CancelDrag() {
	t.drag.stop()
}

// DragTick continues scrolling toward toTop if gen still matches the
// scheduling tick, or is a no-op if a newer drag has since superseded it
func (t *TerminalPane) DragTick(_ *Context, gen int, toTop bool) tea.Cmd {
	if gen != t.drag.gen {
		return nil
	}
	if toTop {
		t.ScrollLines(1)
	} else {
		t.ScrollLines(-1)
	}
	contentH := max(t.Area().Height-1, 0)
	edgeY := contentH - 1
	if toTop {
		edgeY = 0
	}
	t.extendSelection(uv.Position{X: t.drag.fixed, Y: edgeY})
	return t.drag.tick(toTop, t.scheduleDragTick)
}

func (t *TerminalPane) scheduleDragTick(
	toTop bool, gen int, interval time.Duration,
) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return terminalDragScrollMsg{dc: t, gen: gen, toTop: toTop}
	})
}

func (t *TerminalPane) clampedMouse(cx *Context, x, y int) uv.Mouse {
	yOff := 0
	if bufferlineVisible(cx) {
		yOff = 1
	}
	a := t.area
	localX := min(max(x-a.X, 0), max(a.Width-1, 0))
	contentH := max(a.Height-1, 0)
	localY := min(max((y-yOff)-a.Y, 0), max(contentH-1, 0))
	return uv.Mouse{X: localX, Y: localY}
}

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

// the Ctrl-w prefix must reach the keymap even while a pane has raw input
// focus, so it can't be treated as a normal keystroke
func (e *EditorComponent) inWindowChord(msg tea.KeyPressMsg) bool {
	if len(e.pending) > 0 {
		return true
	}
	k := msg.Key()
	return k.Mod&tea.ModCtrl != 0 && k.Code == 'w'
}

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
	// defer closes until after the scan, since closing mutates the tree
	for _, tp := range closing {
		closeTerminal(cx.Editor, tp)
	}
}

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

func closeTerminal(e *view.Editor, tp *TerminalPane) {
	_ = tp.Stop()
	if tp.restore != nil {
		e.ReplacePane(tp.ID(), tp.restore)
		return
	}
	e.ClosePane(tp.ID())
}

func mouseFields(msg tea.Msg) (
	x, y int, btn tea.MouseButton, mod tea.KeyMod, ok bool,
) {
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
