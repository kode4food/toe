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
		Render(width, height int, cx *Context) string
		Cursor(width, height int, cx *Context) (cur tea.Cursor, ok bool)
	}

	// OverlayComponent extends Component for layers that composite over
	// the content rendered by layers beneath them
	OverlayComponent interface {
		Component
		RenderOver(width, height int, base string, cx *Context) string
	}

	// BufferRenderer is implemented by a base (layer 0) component that can
	// expose the cell buffer it rendered into, instead of only a serialized
	// ANSI string. This lets overlay layers above it draw directly onto the
	// same buffer, skipping an ANSI round-trip
	BufferRenderer interface {
		Component
		RenderBuffer(width, height int, cx *Context) *tui.Buffer
	}

	// BufferOverlayComponent extends Component for overlay layers that draw
	// directly onto the shared cell buffer rather than compositing ANSI
	// strings. The compositor only takes this path when every layer above
	// the base supports it; otherwise it falls back to OverlayComponent
	BufferOverlayComponent interface {
		Component
		RenderOverBuffer(buf *tui.Buffer, cx *Context)
	}

	boundedOverlay interface {
		Component
		lastBounds() bounds
	}
)

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
