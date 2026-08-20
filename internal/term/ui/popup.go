package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type popup struct {
	borderStyle  tui.Style
	contentStyle tui.Style
	padX         int
}

const (
	overlayPadX = 1

	// the bottom row a centered overlay stays clear of: the bottom right pane's
	// statusline, which carries maximized, register, and macro state
	overlayKeepClear = 1
)

func (p popup) drawInto(buf *tui.Buffer, area geom.Area) geom.Area {
	for dy := range area.Height {
		buf.FillRange(
			area.Add(geom.Point{Y: dy}), area.Width, p.contentStyle,
		)
	}
	if area.Width >= 2 && area.Height >= 2 {
		rule := strings.Repeat(tui.BorderH, area.Width-2)
		top := tui.BorderTL + rule + tui.BorderTR
		bot := tui.BorderBL + rule + tui.BorderBR
		buf.SetString(area.Point, top, p.borderStyle)
		buf.SetString(geom.Point{
			X: area.X,
			Y: area.Bottom(),
		}, bot, p.borderStyle)
		for y := 1; y < area.Height-1; y++ {
			buf.SetString(
				area.Add(geom.Point{Y: y}), tui.BorderV, p.borderStyle,
			)

			buf.SetString(geom.Point{
				X: area.Right(),
				Y: area.Y + y,
			}, tui.BorderV, p.borderStyle)
		}
	}
	return area.Inset(geom.Size{Width: 1 + p.padX, Height: 1})
}

func (p popup) divider(buf *tui.Buffer, area geom.Area, dx int) {
	x := area.X + dx
	buf.SetString(geom.Point{X: x, Y: area.Y}, tui.BorderMT, p.borderStyle)
	buf.SetString(
		geom.Point{X: x, Y: area.Bottom()}, tui.BorderMB, p.borderStyle,
	)
	for y := area.Y + 1; y < area.Bottom(); y++ {
		buf.SetString(geom.Point{X: x, Y: y}, tui.BorderV, p.borderStyle)
	}
}

func drawPopupTitle(
	buf *tui.Buffer, area geom.Area, title string, style tui.Style,
) {
	if title == "" {
		return
	}
	maxW := area.Width - 2
	if maxW < 2 {
		return
	}
	text := " " + runewidth.Truncate(title, maxW-2, "") + " "
	buf.SetString(area.Add(geom.Point{X: 1}), text, style)
}

type drawPopupRuleArgs struct {
	buf   *tui.Buffer
	at    geom.Point
	width int
	style tui.Style
}

func drawPopupRule(args drawPopupRuleArgs) {
	inner := max(args.width-2, 0)
	line := tui.BorderML + strings.Repeat(tui.BorderH, inner) + tui.BorderMR
	args.buf.SetString(args.at, line, args.style)
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
