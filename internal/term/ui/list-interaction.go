package ui

import "github.com/kode4food/toe/internal/geom"

func listIndexAt(b geom.Area, scroll int, at geom.Point) (int, bool) {
	if !b.Contains(at) {
		return 0, false
	}
	return scroll + (at.Y - b.Y), true
}

func listScrollBy(scroll, count, rows, delta int) int {
	return listClampScroll(scroll+delta, count, rows)
}

func listEnsureCursorVisible(scroll, cursor, count, rows int) int {
	if rows <= 0 {
		return listClampScroll(scroll, count, rows)
	}
	if cursor < scroll {
		scroll = cursor
	} else if cursor >= scroll+rows {
		scroll = cursor - rows + 1
	}
	return listClampScroll(scroll, count, rows)
}

func listClampScroll(scroll, count, rows int) int {
	if rows <= 0 || count <= rows {
		return 0
	}
	scroll = min(scroll, count-rows)
	return max(scroll, 0)
}
