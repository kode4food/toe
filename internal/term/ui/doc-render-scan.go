package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type linePrefixArgs struct {
	rev, lineNum, lineStart, lineEnd, tabW, hOff int
	text                                         core.Rope
}

// scanLinePrefix walks the rope from args.lineStart to compute the indent
// column and the horizontal-scroll window start described by
// linePrefixScan. rev and lineNum are unused here — they exist only for
// docRenderCache's cache key in ensureLinePrefix, which forwards args as-is
//
// For printable ASCII the inner loop uses a direct width-1 assignment instead
// of calling view.RuneWidth, so common code files need no per-char function
// call overhead in the prefix
func scanLinePrefix(args linePrefixArgs) linePrefixScan {
	pos := args.lineStart
	col := 0
	indentCol := 0
	indentDone := false
	found := false
	args.text.ForEachSegment(args.lineStart, args.lineEnd, func(seg string) {
		if found || col >= args.hOff {
			return
		}
		for _, ch := range seg {
			if !indentDone {
				switch ch {
				case runeTab, runeSpace, runeNbsp, runeNnbsp:
				default:
					indentDone = true
					indentCol = col
				}
			}
			var w int
			if uint32(ch)-0x20 < 0x5f { // printable ASCII
				w = 1
			} else {
				w = view.RuneWidth(ch, col, args.tabW)
			}
			if col+w > args.hOff {
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
	return linePrefixScan{
		indentCol: indentCol,
		windowPos: pos,
		windowCol: col,
	}
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
			if ch == runeTab {
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
		case runeTab:
			col += tabW - col%tabW
		case runeSpace, runeNbsp, runeNnbsp:
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
