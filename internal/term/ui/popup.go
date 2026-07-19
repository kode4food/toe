package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type (
	popup struct {
		border       lipgloss.Border
		borderStyle  tui.Style
		contentStyle tui.Style
		padX         int
	}
)

// drawInto fills and borders the popup rectangle, returning its inner bounds
// in buffer coordinates
func (p popup) drawInto(buf *tui.Buffer, area geom.Area) geom.Area {
	for dy := range area.Height {
		buf.FillRange(
			area.Point.Add(geom.Point{Y: dy}), area.Width, p.contentStyle,
		)
	}
	if area.Width >= 2 && area.Height >= 2 {
		top := p.border.TopLeft +
			strings.Repeat(p.border.Top, area.Width-2) +
			p.border.TopRight
		bot := p.border.BottomLeft +
			strings.Repeat(p.border.Bottom, area.Width-2) +
			p.border.BottomRight
		buf.SetString(area.Point, top, p.borderStyle)
		buf.SetString(geom.Point{
			X: area.X,
			Y: area.Bottom(),
		}, bot, p.borderStyle)
		for y := 1; y < area.Height-1; y++ {
			buf.SetString(
				area.Point.Add(geom.Point{Y: y}), p.border.Left, p.borderStyle,
			)

			buf.SetString(geom.Point{
				X: area.Right(),
				Y: area.Y + y,
			}, p.border.Right, p.borderStyle)
		}
	}
	return area.Inset(geom.Size{Width: 1 + p.padX, Height: 1})
}

func fitPopup(area geom.Area, screen geom.Size) geom.Area {
	if area.X+area.Width > screen.Width {
		area.X = max(screen.Width-area.Width, 0)
	}
	if area.Y+area.Height > screen.Height {
		area.Y = max(area.Y-area.Height-1, 0)
	}
	return area
}
