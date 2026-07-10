package ui

func listIndexAt(b Bounds, scroll, x, y int) (int, bool) {
	if !b.contains(x, y) {
		return 0, false
	}
	return scroll + (y - b.y), true
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
