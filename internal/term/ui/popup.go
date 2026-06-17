package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/tui"
)

type (
	// popup is a bordered window that owns its background. The render methods
	// fill every cell of the box with contentStyle first, then draw the border
	// on top. Callers may write per-cell-styled content into the returned inner
	// area without worrying about ANSI nesting resets, since every cell already
	// has its final style and content writes simply overwrite cells in place
	popup struct {
		border       lipgloss.Border
		borderStyle  tui.Style
		contentStyle tui.Style
		padX         int
	}

	// popupArea is the inner content rectangle returned by popup.draw and
	// popup.drawInto — the popup owns every cell outside this rectangle
	popupArea struct {
		x, y, w, h int
	}
)

// draw allocates a buffer of the given outer size, fills it with the popup's
// contentStyle, draws the border, and returns the inner content rectangle
func (p popup) draw(w, h int) (*tui.Buffer, popupArea) {
	buf := tui.NewBuffer(w, h)
	area := p.drawInto(buf, 0, 0, w, h)
	return buf, area
}

// drawInto fills the rectangle (ox, oy, w, h) of buf with the popup's
// contentStyle, draws the border on its edges, and returns the inner content
// rectangle in buf's coordinates
func (p popup) drawInto(buf *tui.Buffer, ox, oy, w, h int) popupArea {
	for dy := range h {
		buf.FillRange(ox, oy+dy, w, p.contentStyle)
	}
	if w >= 2 && h >= 2 {
		top := p.border.TopLeft +
			strings.Repeat(p.border.Top, w-2) +
			p.border.TopRight
		bot := p.border.BottomLeft +
			strings.Repeat(p.border.Bottom, w-2) +
			p.border.BottomRight
		buf.SetString(ox, oy, top, p.borderStyle)
		buf.SetString(ox, oy+h-1, bot, p.borderStyle)
		for y := 1; y < h-1; y++ {
			buf.SetString(ox, oy+y, p.border.Left, p.borderStyle)
			buf.SetString(ox+w-1, oy+y, p.border.Right, p.borderStyle)
		}
	}
	return popupArea{
		x: ox + 1 + p.padX,
		y: oy + 1,
		w: max(w-2-2*p.padX, 0),
		h: max(h-2, 0),
	}
}
