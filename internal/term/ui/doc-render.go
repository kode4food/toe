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

// lineNumberDigits returns the digit width for the largest line number that
// will be drawn. A trailing empty line produced by a final newline is not
// counted
func lineNumberDigits(text core.Rope) int {
	nLines := text.LenLines()
	lastDrawn := nLines
	if nLines > 0 {
		if last, err := text.Line(nLines - 1); err == nil &&
			last.LenChars() == 0 {
			lastDrawn = nLines - 1
		}
	}
	return digitCount(lastDrawn)
}

// gutterWidthFor returns the total configured gutter width
func gutterWidthFor(text core.Rope, g view.Gutter) int {
	layout := g.GutterLayout()
	lineNumberW := gutterLineNumberWidth(text, g, layout)
	return gutterLayoutWidth(layout, lineNumberW)
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
