package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type linePrefixRequest struct {
	rev       int
	lineNum   int
	lineStart int
	tabWidth  int
	horzOff   int
	text      string
}

func scanLinePrefix(req *linePrefixRequest) linePrefixScan {
	pos := req.lineStart
	col := 0
	indentCol := 0
	indentDone := false
	windowByte := len(req.text)
	for i, ch := range req.text {
		if !indentDone {
			switch ch {
			case runeTab, runeSpace, runeNbsp, runeNnbsp:
			default:
				indentDone = true
				indentCol = col
			}
		}
		var w int
		if uint32(ch)-0x20 < 0x5f {
			w = 1
		} else {
			w = view.RuneWidth(ch, core.TabStop{
				Column:   col,
				TabWidth: req.tabWidth,
			})
		}
		if col+w > req.horzOff {
			windowByte = i
			break
		}
		col += w
		pos++
	}

	if !indentDone {
		indentCol = col
	}
	return linePrefixScan{
		indentCol:  indentCol,
		windowPos:  pos,
		windowCol:  col,
		windowByte: windowByte,
	}
}

type visualColOfArgs struct {
	line     string
	charOff  int
	tabWidth int
}

func visualColOf(args visualColOfArgs) int {
	col := 0
	charIdx := 0
	for _, ch := range args.line {
		if charIdx >= args.charOff {
			break
		}
		charIdx++
		if ch == runeTab {
			col += args.tabWidth - col%args.tabWidth
		} else {
			col++
		}
	}
	return col
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

func lineString(text core.Rope, span core.Span) string {
	if span.From >= span.To {
		return ""
	}
	if s, err := text.SliceString(span); err == nil {
		return s
	}
	return ""
}

func runePrefix(text string, count int) string {
	for i := range text {
		if count <= 0 {
			return text[:i]
		}
		count--
	}
	return text
}
