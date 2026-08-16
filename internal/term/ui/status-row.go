package ui

import (
	"slices"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type (
	// statusRow lays out a single row of widgets, filling from both edges and
	// shedding unpinned ones from the middle when the row runs out of space
	statusRow struct {
		at        geom.Point
		width     int
		baseStyle tui.Style
		left      []statusElem
		right     []statusElem
	}

	// statusElem is a single rendered piece of a status row
	statusElem struct {
		text    string
		style   tui.Style
		pinned  bool
		compact bool
	}
)

func (r statusRow) contentWidth() int {
	return max(r.width-statusElemsWidth(r.right), 0)
}

func (r statusRow) paint(buf *tui.Buffer) {
	left, right := r.left, r.right
	for statusElemsWidth(left)+statusElemsWidth(right) > r.width {
		var ok bool
		if right, ok = dropUnpinned(right, false); ok {
			continue
		}
		if left, ok = dropUnpinned(left, true); !ok {
			break
		}
	}
	buf.FillRange(r.at, r.width, r.baseStyle)
	r.writeElems(buf, left, r.at.X)
	r.writeElems(buf, right, r.at.X+r.width-statusElemsWidth(right))
}

func (r statusRow) writeElems(buf *tui.Buffer, elems []statusElem, x int) {
	for _, e := range elems {
		if !e.compact {
			buf.SetString(geom.Point{X: x, Y: r.at.Y}, " ", r.baseStyle)
			x++
		}
		buf.SetString(geom.Point{X: x, Y: r.at.Y}, e.text, e.style)
		x += runewidth.StringWidth(e.text)
		if !e.compact {
			buf.SetString(geom.Point{X: x, Y: r.at.Y}, " ", r.baseStyle)
			x++
		}
	}
}

func statusBadge(text string, style tui.Style) statusElem {
	return statusElem{
		text:    " " + text + " ",
		style:   style,
		pinned:  true,
		compact: true,
	}
}

func statusElemsWidth(elems []statusElem) int {
	w := 0
	for _, e := range elems {
		w += runewidth.StringWidth(e.text)
		if !e.compact {
			w += 2
		}
	}
	return w
}

func dropUnpinned(elems []statusElem, fromEnd bool) ([]statusElem, bool) {
	for n, i := len(elems), 0; i < n; i++ {
		idx := i
		if fromEnd {
			idx = n - 1 - i
		}
		if !elems[idx].pinned {
			return slices.Delete(elems, idx, idx+1), true
		}
	}
	return elems, false
}
