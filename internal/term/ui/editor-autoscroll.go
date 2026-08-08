package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/view"
)

type (
	mouseAutoScrollAxis struct {
		axisTicker
		scroll mouseAxisScrollFunc
		pos    mouseAxisPosFunc
	}

	// axisTicker is the drag-edge-detection and repeating-tick state shared by
	// every kind of mouse-drag auto-scroll (doc view or terminal pane)
	axisTicker struct {
		last, fixed   int
		gen           int
		active, toLow bool
		interval      time.Duration
	}

	// dragBounds is a drag position and the range it is clamped within
	dragBounds struct {
		pos               int
		lowEdge, highEdge int
		margin            int
	}

	// edgeState reports which edge a drag is pinned against, and the
	// position clamped into range
	edgeState struct {
		atLow   bool
		atHigh  bool
		clamped int
	}

	mouseAxisScrollMsg struct {
		gen   int
		axis  *mouseAutoScrollAxis
		toLow bool
	}

	axisTickSchedule func(toLow bool, gen int, interval time.Duration) tea.Cmd

	mouseAxisScrollFunc func(e *view.Editor, v *view.View, toLow bool)

	mouseAxisPosFunc func(
		r *renderPass, doc *view.Document, v *view.View, fixed int, toLow bool,
	) (int, bool)
)

const (
	mouseAutoScrollMaxInterval = 400 * time.Millisecond
	mouseAutoScrollMinInterval = 50 * time.Millisecond
)

func (e *EditorComponent) continueAxisScroll(
	cx *Context, axis *mouseAutoScrollAxis, toLow bool,
) tea.Cmd {
	doc := cx.Editor.FocusedDocument()
	if doc == nil {
		return nil
	}
	v := cx.Editor.FocusedView()
	if v == nil {
		return nil
	}
	axis.scroll(cx.Editor, v, toLow)

	r := &renderPass{editor: e, context: cx, size: e.size}
	if pos, ok := axis.pos(r, doc, v, axis.fixed, toLow); ok {
		extendSelectionTo(cx, doc, v, pos)
	}
	return axis.tick(toLow, axis.schedule)
}

func (a *mouseAutoScrollAxis) schedule(
	toLow bool, gen int, interval time.Duration,
) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return mouseAxisScrollMsg{gen: gen, axis: a, toLow: toLow}
	})
}

// tick starts ticking toward toLow, scheduling the next tick via schedule
func (a *axisTicker) tick(toLow bool, schedule axisTickSchedule) tea.Cmd {
	a.gen++
	a.active = true
	a.toLow = toLow
	interval := a.interval
	if interval <= 0 {
		interval = mouseAutoScrollMaxInterval
	}
	return schedule(toLow, a.gen, interval)
}

func (a *axisTicker) stop() {
	if !a.active {
		return
	}
	a.gen++
	a.active = false
}

func (a *axisTicker) update(bounds dragBounds) edgeState {
	edge := dragEdge(dragEdgeArgs{
		bounds: bounds,
		last:   a.last,
		onLow:  a.active && a.toLow,
		onHigh: a.active && !a.toLow,
	})
	a.last = bounds.pos
	a.interval = autoScrollInterval(bounds, edge)
	return edge
}

// trigger starts or continues ticking toward whichever edge was crossed, or
// stops ticking if neither has
func (a *axisTicker) trigger(
	edge edgeState, fixed int, schedule axisTickSchedule,
) tea.Cmd {
	if !edge.atLow && !edge.atHigh {
		a.stop()
		return nil
	}
	a.fixed = fixed
	if a.active && a.toLow == edge.atLow {
		return nil
	}
	return a.tick(edge.atLow, schedule)
}

type dragEdgeArgs struct {
	bounds        dragBounds
	last          int
	onLow, onHigh bool
}

func dragEdge(args dragEdgeArgs) edgeState {
	b := args.bounds
	towardLow := b.pos < args.last
	towardHigh := b.pos > args.last
	stuckLow := args.onLow && b.pos <= b.lowEdge
	stuckHigh := args.onHigh && b.pos >= b.highEdge
	return edgeState{
		atLow:   b.pos <= b.lowEdge+b.margin && (towardLow || stuckLow),
		atHigh:  b.pos >= b.highEdge-b.margin && (towardHigh || stuckHigh),
		clamped: min(max(b.pos, b.lowEdge), b.highEdge),
	}
}

func autoScrollInterval(b dragBounds, edge edgeState) time.Duration {
	if b.margin <= 0 {
		return mouseAutoScrollMinInterval
	}
	var depth int
	switch {
	case edge.atLow:
		depth = b.lowEdge + b.margin - b.pos
	case edge.atHigh:
		depth = b.pos - (b.highEdge - b.margin)
	default:
		return mouseAutoScrollMaxInterval
	}
	depth = min(max(depth, 0), b.margin)
	t := float64(depth) / float64(b.margin)
	t *= t // ease in: stays slow through most of the margin, then drops fast
	span := mouseAutoScrollMaxInterval - mouseAutoScrollMinInterval
	return mouseAutoScrollMaxInterval - time.Duration(t*float64(span))
}
