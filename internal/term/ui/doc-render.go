package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

func digitCount(n int) int {
	if n <= 0 {
		return 1
	}
	d := 0
	for n > 0 {
		d++
		n /= 10
	}
	return d
}

func lineNumberDigits(text core.Rope) int {
	nLines := text.LenLines()
	if hasTrailingEmptyLine(text) {
		nLines--
	}
	return digitCount(nLines)
}

func gutterWidthFor(text core.Rope, g view.Gutter) int {
	layout := g.GutterLayout()
	lineNumW := gutterLineNumberWidth(text, g, layout)
	return gutterLayoutWidth(layout, lineNumW)
}

func bufferlineVisible(cx *Context) bool {
	switch cx.Editor.Options().BufferLine {
	case view.BufferLineAlways:
		return true
	case view.BufferLineMultiple:
		return len(cx.Editor.AllDocuments()) > 1
	default:
		return false
	}
}

func hasTrailingEmptyLine(text core.Rope) bool {
	n := text.LenLines()
	if n == 0 {
		return false
	}
	from, err := text.LineToChar(n - 1)
	return err == nil && from == text.LenChars()
}
