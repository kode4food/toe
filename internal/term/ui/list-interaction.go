package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
)

// listScroll is the scroll state of a fixed-height list viewport: a cursor and
// a scroll offset over count items, rows of which are visible
type listScroll struct {
	scroll int
	cursor int
	count  int
	rows   int
}

func (l *listScroll) moveBy(n int) {
	if l.count == 0 {
		return
	}
	l.moveTo(((l.cursor+n)%l.count + l.count) % l.count)
}

func (l *listScroll) moveTo(idx int) {
	l.cursor = min(max(idx, 0), max(l.count-1, 0))
	l.scroll = l.ensureCursorVisible()
}

func (l *listScroll) resize(count, rows int) {
	l.count = count
	l.rows = rows
	l.cursor = min(l.cursor, max(count-1, 0))
	l.scroll = l.clamped()
}

func (l *listScroll) wheel(button tea.MouseButton, step int) {
	l.scroll = l.scrollWheel(button, step)
}

func (l *listScroll) scrollWheel(button tea.MouseButton, step int) int {
	switch button {
	case tea.MouseWheelUp:
		return l.clampTo(l.scroll - step)
	case tea.MouseWheelDown:
		return l.clampTo(l.scroll + step)
	default:
		return l.scroll
	}
}

func (l *listScroll) ensureCursorVisible() int {
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

func (l *listScroll) clamped() int {
	return l.clampTo(l.scroll)
}

func (l *listScroll) clampTo(scroll int) int {
	if l.rows <= 0 || l.count <= l.rows {
		return 0
	}
	return max(min(scroll, l.count-l.rows), 0)
}

func (l *listScroll) indexAt(b geom.Area, at geom.Point) (int, bool) {
	if !b.Contains(at) {
		return 0, false
	}
	idx := l.scroll + (at.Y - b.Y)
	return idx, idx >= 0 && idx < l.count
}

func visibleRows(bounds geom.Area, fallback int) int {
	if bounds.Height > 0 {
		return bounds.Height
	}
	return fallback
}
