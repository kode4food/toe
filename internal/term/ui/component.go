package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type (
	// Callback lets a component push, pop, or mutate compositor layers
	// without direct coupling — the compositor executes it after event
	// propagation completes
	Callback func(*Context, *Compositor) tea.Cmd

	// EventResult is returned by every Component.HandleEvent call
	EventResult struct {
		Consumed bool
		Callback Callback
	}

	// Component is the interface every compositor layer must implement
	Component interface {
		HandleEvent(*Context, tea.Msg) (EventResult, tea.Cmd)
		Cursor(*Context, geom.Size) (tea.Cursor, bool)
	}

	// BufferRenderer exposes the raw cell buffer a base component rendered
	// into, so overlay layers can draw directly onto it
	BufferRenderer interface {
		Component
		Render(*Context, geom.Size) *tui.Buffer
	}

	// BufferOverlayComponent extends Component for overlay layers that own
	// their own cell buffer instead of drawing into the shared one
	BufferOverlayComponent interface {
		Component
		Layout(*Context, geom.Size) (geom.Area, bool)
		PaintBuffer(*Context, geom.Area) *tui.Buffer
	}

	// PaneInput is a pane that handles bubbletea key and mouse events itself.
	// It receives the event first; an unconsumed event (handled=false) falls
	// through to the editor's default keymap/document handling
	PaneInput interface {
		HandleEvent(*Context, tea.Msg) (EventResult, bool)
	}

	// PaneCursor is a pane that positions its own cursor
	PaneCursor interface {
		Cursor(*Context) (tea.Cursor, bool)
	}

	// Pasteable is a pane that consumes a paste itself instead of the
	// document/selection paste
	Pasteable interface {
		Paste(text string)
	}

	// Draggable is a pane that handles mouse drags itself. Drags span several
	// events with cross-event state, so they stay separate from PaneInput
	Draggable interface {
		BeginDrag(*Context, geom.Point, tea.KeyMod) bool
		ContinueDrag(*Context, geom.Point) tea.Cmd
		EndDrag(*Context, geom.Point) tea.Cmd
		CancelDrag()
		DragTick(cx *Context, gen int, toTop bool) tea.Cmd
	}

	// overlayBuf is embedded by BufferOverlayComponent implementers to
	// reuse their paint buffer across frames instead of reallocating it
	overlayBuf struct {
		buf      *tui.Buffer
		dirty    bool
		styleGen int
	}
)

// scrollbarThumb is the marker drawn in a list's scrollbar column
const scrollbarThumb = "▌"

func (o *overlayBuf) maybePaint(
	cx *Context, size geom.Size, paint func(buf *tui.Buffer),
) *tui.Buffer {
	gen := cx.StyleGen()
	resized := o.buf == nil || o.buf.Size != size
	repaint := resized || o.dirty || o.styleGen != gen
	if resized {
		o.buf = tui.NewBuffer(size)
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

func popLayer(_ *Context, c *Compositor) tea.Cmd {
	c.Pop()
	return nil
}
