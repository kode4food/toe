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
		HandleEvent(msg tea.Msg, cx *Context) (EventResult, tea.Cmd)
		Cursor(width, height int, cx *Context) (cur tea.Cursor, ok bool)
	}

	// BufferRenderer is implemented by a base (layer 0) component that can
	// expose the cell buffer it rendered into, instead of only a serialized
	// ANSI string. This lets overlay layers above it draw directly onto the
	// same buffer, skipping an ANSI round-trip
	BufferRenderer interface {
		Component
		Render(width, height int, cx *Context) *tui.Buffer
	}

	// BufferOverlayComponent extends Component for overlay layers that own
	// their own cell buffer instead of drawing into the shared one
	BufferOverlayComponent interface {
		Component
		Layout(screenW, screenH int, cx *Context) (pl Bounds, ok bool)
		PaintBuffer(pl Bounds, cx *Context) *tui.Buffer
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
