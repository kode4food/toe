package ui

import "github.com/kode4food/toe/internal/geom"

// listScroll is the scroll state of a fixed-height list viewport: a cursor and
// a scroll offset over count items, rows of which are visible
type listScroll struct {
	scroll int
	cursor int
	count  int
	rows   int
}

func (l listScroll) scrollBy(delta int) int {
	return l.clampTo(l.scroll + delta)
}

// ensureCursorVisible scrolls the minimum amount needed to bring the cursor
// row into view
func (l listScroll) ensureCursorVisible() int {
	switch {
	case l.rows <= 0:
		return l.clamped()
	case l.cursor < l.scroll:
		return l.clampTo(l.cursor)
	case l.cursor >= l.scroll+l.rows:
		return l.clampTo(l.cursor - l.rows + 1)
	}
	return l.clamped()
}

func (l listScroll) clamped() int {
	return l.clampTo(l.scroll)
}

func (l listScroll) clampTo(scroll int) int {
	if l.rows <= 0 || l.count <= l.rows {
		return 0
	}
	return max(min(scroll, l.count-l.rows), 0)
}

func listIndexAt(b geom.Area, scroll int, at geom.Point) (int, bool) {
	if !b.Contains(at) {
		return 0, false
	}
	return scroll + (at.Y - b.Y), true
}
