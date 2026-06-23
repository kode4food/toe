package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// scanLinePrefix walks the rope from lineStart, returning indentCol (the visual
// column where indentation ends), windowPos (the char offset of the first char
// at or after column hStart), and windowCol (that char's visual column, which
// may be < hStart when a tab straddles the window boundary).
//
// For printable ASCII the inner loop uses a direct width-1 assignment instead
// of calling view.RuneWidth, so common code files need no per-char function
// call overhead in the prefix
func scanLinePrefix(
	text core.Rope, lineStart, lineEnd, tabW, hStart int,
) (indentCol, windowPos, windowCol int) {
	pos := lineStart
	col := 0
	indentDone := false
	found := false
	text.ForEachSegment(lineStart, lineEnd, func(seg string) {
		if found || col >= hStart {
			return
		}
		for _, ch := range seg {
			if !indentDone {
				switch ch {
				case view.RuneTab, view.RuneSpace, view.RuneNbsp,
					view.RuneNnbsp:
				default:
					indentDone = true
					indentCol = col
				}
			}
			var w int
			if uint32(ch)-0x20 < 0x5f { // printable ASCII
				w = 1
			} else {
				w = view.RuneWidth(ch, col, tabW)
			}
			if col+w > hStart {
				found = true
				return
			}
			col += w
			pos++
		}
	})
	if !indentDone {
		indentCol = col
	}
	windowPos = pos
	windowCol = col
	return
}

func cursorCols(
	selSpans []selectionSpan, lStr string,
	lineStart, lineEnd, tabW, colStart int,
) (primary, secondary map[int]bool) {
	for _, sp := range selSpans {
		if sp.cur < lineStart || sp.cur > lineEnd {
			continue
		}
		vcol := colStart
		offset := sp.cur - lineStart
		charIdx := 0
		for _, ch := range lStr {
			if charIdx >= offset {
				break
			}
			charIdx++
			if ch == view.RuneTab {
				vcol += tabW - vcol%tabW
			} else {
				vcol++
			}
		}
		if sp.primary {
			if primary == nil {
				primary = make(map[int]bool)
			}
			primary[vcol] = true
		} else {
			if secondary == nil {
				secondary = make(map[int]bool)
			}
			secondary[vcol] = true
		}
	}
	return
}

func indentWidth(lineStr string, tabW int) int {
	col := 0
	for _, ch := range lineStr {
		switch ch {
		case view.RuneTab:
			col += tabW - col%tabW
		case view.RuneSpace, view.RuneNbsp, view.RuneNnbsp:
			col++
		default:
			return col
		}
	}
	return col
}

func lineString(text core.Rope, from, to int) string {
	if from >= to {
		return ""
	}
	s, err := text.SliceString(from, to)
	if err != nil {
		return ""
	}
	return s
}
