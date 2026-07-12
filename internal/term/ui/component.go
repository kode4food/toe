package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/tui"
)

type (
	// Callback lets a component push, pop, or mutate compositor layers
	// without direct coupling — the compositor executes it after event
	// propagation completes
	Callback func(*Compositor, *Context) tea.Cmd

	// EventResult is returned by every Component.HandleEvent call
	EventResult struct {
		Consumed bool
		Callback Callback
	}

	// Component is the interface every compositor layer must implement
	Component interface {
		HandleEvent(tea.Msg, *Context) (EventResult, tea.Cmd)
		Cursor(width, height int, cx *Context) (tea.Cursor, bool)
	}

	// BufferRenderer exposes the raw cell buffer a base component rendered
	// into, so overlay layers can draw directly onto it
	BufferRenderer interface {
		Component
		Render(width, height int, cx *Context) *tui.Buffer
	}

	// BufferOverlayComponent extends Component for overlay layers that own
	// their own cell buffer instead of drawing into the shared one
	BufferOverlayComponent interface {
		Component
		Layout(screenW, screenH int, cx *Context) (Bounds, bool)
		PaintBuffer(Bounds, *Context) *tui.Buffer
	}

	// PaneInput is implemented by a [view.Pane] that wants to handle its own
	// raw key and mouse events, instead of the normal keymap/document mouse
	// logic. ok is false to fall through to that normal handling
	PaneInput interface {
		HandleKey(tea.KeyPressMsg, *Context) (EventResult, bool)
		HandleMouse(tea.Msg, *Context) (EventResult, bool)
	}

	// PaneCursor is implemented by a [view.Pane] that draws its own cursor,
	// instead of the normal document cursor-shape logic
	PaneCursor interface {
		Cursor(*Context) (tea.Cursor, bool)
	}

	// Bounds is a screen-space rectangle
	Bounds struct{ x, y, w, h int }

	// overlayBuf is embedded by BufferOverlayComponent implementers to
	// reuse their paint buffer across frames instead of reallocating it
	overlayBuf struct {
		buf      *tui.Buffer
		dirty    bool
		styleGen int
	}
)

func (o *overlayBuf) maybePaint(
	w, h int, cx *Context, paint func(buf *tui.Buffer),
) *tui.Buffer {
	gen := cx.StyleGen()
	resized := o.buf == nil || o.buf.Width != w || o.buf.Height != h
	repaint := resized || o.dirty || o.styleGen != gen
	if resized {
		o.buf = tui.NewBuffer(w, h)
	} else if repaint {
		o.buf.Clear()
	}
	o.dirty = false
	o.styleGen = gen
	if repaint {
		paint(o.buf)
	}
	return o.buf
}

func (o *overlayBuf) markDirty() {
	o.dirty = true
}

func (b Bounds) contains(x, y int) bool {
	return x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h
}

func (b Bounds) translate(dx, dy int) Bounds {
	return Bounds{x: b.x + dx, y: b.y + dy, w: b.w, h: b.h}
}

func consumed() EventResult {
	return EventResult{Consumed: true}
}

func consumedWith(cb Callback) EventResult {
	return EventResult{Consumed: true, Callback: cb}
}

func ignored() EventResult {
	return EventResult{}
}

func ignoredWith(cb Callback) EventResult {
	return EventResult{Callback: cb}
}

func popLayer(c *Compositor, _ *Context) tea.Cmd {
	c.Pop()
	return nil
}
