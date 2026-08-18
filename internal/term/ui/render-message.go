package ui

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

func renderCenteredMessage(
	buf *tui.Buffer, area geom.Area, msg string, style tui.Style,
) {
	if area.IsEmpty() {
		return
	}
	width := runewidth.StringWidth(msg)
	buf.SetString(geom.Point{
		X: area.X + max((area.Width-width)/2, 0),
		Y: area.Y + area.Height/2,
	}, msg, style)
}
