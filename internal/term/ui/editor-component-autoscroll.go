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
		last, fixed int
		gen         int
		on, toLo    bool
		interval    time.Duration
	}

	mouseAxisScrollMsg struct {
		gen  int
		axis *mouseAutoScrollAxis
		toLo bool
	}

	axisTickSchedule func(toLo bool, gen int, interval time.Duration) tea.Cmd

	mouseAxisScrollFunc func(e *view.Editor, v *view.View, toLo bool)

	mouseAxisPosFunc func(
		r *renderPass, doc *view.Document, v *view.View, fixed int, toLo bool,
	) (int, bool)
)

const (
	mouseAutoScrollMaxInterval = 400 * time.Millisecond
	mouseAutoScrollMinInterval = 50 * time.Millisecond
)

func (e *EditorComponent) continueAxisScroll(
	cx *Context, axis *mouseAutoScrollAxis, toLo bool,
) tea.Cmd {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return nil
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return nil
	}
	axis.scroll(cx.Editor, v, toLo)

	r := &renderPass{ec: e, cx: cx, size: e.size}
	if pos, ok := axis.pos(r, doc, v, axis.fixed, toLo); ok {
		extendSelectionTo(cx, doc, v, pos)
	}
	return axis.tick(toLo, axis.schedule)
}

func (a *mouseAutoScrollAxis) schedule(
	toLo bool, gen int, interval time.Duration,
) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return mouseAxisScrollMsg{gen: gen, axis: a, toLo: toLo}
	})
}

// tick starts ticking toward toLo, scheduling the next tick via schedule
func (a *axisTicker) tick(toLo bool, schedule axisTickSchedule) tea.Cmd {
	a.gen++
	a.on = true
	a.toLo = toLo
	interval := a.interval
	if interval <= 0 {
		interval = mouseAutoScrollMaxInterval
	}
	return schedule(toLo, a.gen, interval)
}

func (a *axisTicker) stop() {
	if !a.on {
		return
	}
	a.gen++
	a.on = false
}

func (a *axisTicker) update(
	pos, lo, hi, margin int,
) (atLo, atHi bool, clamped int) {
	atLo, atHi, clamped = dragEdge(dragEdgeArgs{
		pos: pos, last: a.last, lo: lo, hi: hi, margin: margin,
		onLo: a.on && a.toLo, onHi: a.on && !a.toLo,
	})
	a.last = pos
	a.interval = autoScrollInterval(pos, lo, hi, margin, atLo, atHi)
	return atLo, atHi, clamped
}

// trigger starts or continues ticking toward whichever edge atLo/atHi
// crossed, or stops ticking if neither has
func (a *axisTicker) trigger(
	atLo, atHi bool, fixed int, schedule axisTickSchedule,
) tea.Cmd {
	if !atLo && !atHi {
		a.stop()
		return nil
	}
	a.fixed = fixed
	if a.on && a.toLo == atLo {
		return nil
	}
	return a.tick(atLo, schedule)
}

type dragEdgeArgs struct {
	pos, last  int
	lo, hi     int
	margin     int
	onLo, onHi bool
}

func dragEdge(a dragEdgeArgs) (atLo, atHi bool, clamped int) {
	towardLo := a.pos < a.last
	towardHi := a.pos > a.last
	stuckLo := a.onLo && a.pos <= a.lo
	stuckHi := a.onHi && a.pos >= a.hi
	atLo = a.pos <= a.lo+a.margin && (towardLo || stuckLo)
	atHi = a.pos >= a.hi-a.margin && (towardHi || stuckHi)
	clamped = min(max(a.pos, a.lo), a.hi)
	return atLo, atHi, clamped
}

func autoScrollInterval(
	pos, lo, hi, margin int, atLo, atHi bool,
) time.Duration {
	if margin <= 0 {
		return mouseAutoScrollMinInterval
	}
	var depth int
	switch {
	case atLo:
		depth = lo + margin - pos
	case atHi:
		depth = pos - (hi - margin)
	default:
		return mouseAutoScrollMaxInterval
	}
	depth = min(max(depth, 0), margin)
	t := float64(depth) / float64(margin)
	t *= t // ease in: stays slow through most of the margin, then drops fast
	span := mouseAutoScrollMaxInterval - mouseAutoScrollMinInterval
	return mouseAutoScrollMaxInterval - time.Duration(t*float64(span))
}
