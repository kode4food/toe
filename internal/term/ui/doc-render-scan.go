package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type linePrefixArgs struct {
	rev       int
	lineNum   int
	lineStart int
	lineEnd   int
	tabWidth  int
	horzOff   int
	text      core.Rope
}

func scanLinePrefix(args linePrefixArgs) linePrefixScan {
	pos := args.lineStart
	col := 0
	indentCol := 0
	indentDone := false
	found := false
	args.text.ForEachSegment(core.Span{
		From: args.lineStart,
		To:   args.lineEnd,
	}, func(seg string) {
		if found || col >= args.horzOff {
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
			if uint32(ch)-0x20 < 0x5f {
				w = 1
			} else {
				w = view.RuneWidth(ch, core.TabStop{
					Column:   col,
					TabWidth: args.tabWidth,
				})
			}
			if col+w > args.horzOff {
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

type visualColOfArgs struct {
	line     string
	charOff  int
	tabWidth int
}

// visualColOf returns the visual column of the character at CharOffset within
// the line, expanding tabs to TabWidth-wide stops
func visualColOf(args visualColOfArgs) int {
	col, charIdx := 0, 0
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
