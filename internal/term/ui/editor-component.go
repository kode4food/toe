package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	act "github.com/kode4food/toe/internal/view/action"
)

type (
	EditorComponent struct {
		w               int
		h               int
		buf             *tui.Buffer
		pending         []command.KeyEvent
		status          string
		hint            string
		continuation    command.Continuation
		cmdMsg          string
		saveSlot        *saveGenSlot
		cache           *renderCache
		macroSlot       *macroSlot
		nextLayer       layerFunc
		bufferlineShown bool
		focused         bool
		infoTitle       string
		infoItems       []command.KeyHint
		mouseDownRange  *core.Range
	}

	saveGenSlot struct{ gen int }

	autoSaveMsg struct{ gen int }
)

func (e *EditorComponent) HandleEvent(
	msg tea.Msg, cx *Context,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.w = msg.Width
		e.h = msg.Height
		e.resize(cx)
		return consumed(), nil

	case tea.KeyPressMsg:
		result, cmd := e.handleKeyPress(msg, cx)
		if shown := bufferlineVisible(cx); shown != e.bufferlineShown {
			e.bufferlineShown = shown
			e.resize(cx)
		}
		return result, tea.Batch(cmd, e.autoSaveCmd(cx))

	case tea.FocusMsg:
		e.focused = true
		return ignored(), nil

	case tea.BlurMsg:
		e.focused = false
		if cx.Editor.Options().AutoSaveFocusLost {
			cx.Editor.SaveAll()
		}
		return ignored(), nil

	case autoSaveMsg:
		if msg.gen == e.saveSlot.gen {
			cx.Editor.SaveAll()
		}
		return consumed(), nil

	case tea.MouseClickMsg:
		e.cancelPending(cx)
		if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
			r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
			r.handleMouseClick(msg.X, msg.Y, msg.Mod)
		}
		return consumed(), nil

	case tea.MouseMotionMsg:
		if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
			r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
			r.handleMouseDrag(msg.X, msg.Y)
		}
		return consumed(), nil

	case tea.MouseReleaseMsg:
		if !cx.Editor.Options().Mouse {
			return consumed(), nil
		}
		switch msg.Button {
		case tea.MouseLeft:
			e.handleMouseLeftRelease(cx)
		case tea.MouseMiddle:
			if cx.Editor.Options().MiddleClickPaste {
				r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
				r.handleMouseMiddleRelease(msg.X, msg.Y, msg.Mod)
			}
		}
		return consumed(), nil

	case tea.MouseWheelMsg:
		e.cancelPending(cx)
		if !cx.Editor.Options().Mouse {
			return consumed(), nil
		}
		up := msg.Button == tea.MouseWheelUp
		r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
		v, ok := r.contentViewAt(msg.X, msg.Y)
		if !ok {
			return consumed(), nil
		}
		act.ScrollViewLines(cx.Editor, v, cx.Editor.Options().ScrollLines, up)
		return consumed(), nil
	}
	return ignored(), nil
}

func (e *EditorComponent) Render(w, h int, cx *Context) string {
	return e.RenderBuffer(w, h, cx).RenderToANSI()
}

// RenderBuffer lets an overlay layer draw on top of the editor buffer
// without an ANSI round-trip
func (e *EditorComponent) RenderBuffer(w, h int, cx *Context) *tui.Buffer {
	if e.buf == nil || e.buf.Width != w || e.buf.Height != h {
		e.buf = tui.NewBuffer(w, h)
	} else {
		e.buf.Clear()
	}
	r := &renderPass{ec: e, cx: cx, w: w, h: h}
	r.renderEditorContent(e.buf)
	return e.buf
}

func (e *EditorComponent) Cursor(w, h int, cx *Context) (tea.Cursor, bool) {
	r := &renderPass{ec: e, cx: cx, w: w, h: h}
	return r.editorCursor()
}

func newEditorComponent() *EditorComponent {
	return &EditorComponent{
		saveSlot:  &saveGenSlot{},
		cache:     newRenderCache(),
		macroSlot: &macroSlot{macros: map[rune][]command.KeyEvent{}},
		focused:   true,
	}
}

func (e *EditorComponent) cancelPending(cx *Context) {
	e.pending = nil
	e.status = ""
	e.infoTitle = ""
	e.infoItems = nil
	e.continuation = nil
	e.hint = ""
	cx.Editor.ResetCount()
}

func (e *EditorComponent) syncEditorMessages(cx *Context) {
	if h := cx.Editor.TakeHint(); h != "" {
		e.hint = h
	}
	if m := cx.Editor.TakeStatusMsg(); m != "" {
		e.cmdMsg = m
	}
}

func (e *EditorComponent) resize(cx *Context) {
	overhead := 1
	if bufferlineVisible(cx) {
		overhead++
	}
	cx.Editor.SetViewHeight(e.h - overhead)
	th := e.h - 1
	if bufferlineVisible(cx) {
		th--
	}
	cx.Editor.ResizeTree(e.w, max(th, 0))
}

func (e *EditorComponent) handleMouseLeftRelease(cx *Context) {
	if e.mouseDownRange == nil {
		return
	}
	down := *e.mouseDownRange
	e.mouseDownRange = nil
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return
	}
	cur := doc.SelectionFor(v.ID()).Primary()
	if cur.Anchor != down.Anchor || cur.Head != down.Head {
		act.YankToPrimaryClipboard(cx.Editor)
	}
}

func (e *EditorComponent) autoSaveCmd(cx *Context) tea.Cmd {
	opts := cx.Editor.Options()
	if !opts.AutoSaveAfterDelay {
		return nil
	}
	e.saveSlot.gen++
	gen := e.saveSlot.gen
	d := time.Duration(opts.AutoSaveDelayTimeout) * time.Millisecond
	return tea.Tick(d, func(time.Time) tea.Msg {
		return autoSaveMsg{gen: gen}
	})
}
