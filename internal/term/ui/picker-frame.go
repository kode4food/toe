package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type (
	pickerBoxFrame struct {
		borderStyle  tui.Style
		contentStyle tui.Style
		title        string
	}

	pickerBoxAreas struct {
		left  geom.Area
		right geom.Area
	}
)

type drawSplitArgs struct {
	buffer    *tui.Buffer
	area      geom.Area
	leftWidth int
	cutY      int
}

func (f pickerBoxFrame) drawSplit(args drawSplitArgs) pickerBoxAreas {
	area := args.area
	rw := max(area.Width-args.leftWidth-3, 0)
	for dy := range area.Height {
		args.buffer.FillRange(
			area.Point.Add(geom.Point{Y: dy}), area.Width, f.contentStyle,
		)
	}
	if area.Width < 2 || area.Height < 2 {
		return pickerBoxAreas{}
	}
	top := borderTL + strings.Repeat(borderH, args.leftWidth) + borderMT +
		strings.Repeat(borderH, rw) + borderTR
	bot := borderBL + strings.Repeat(borderH, args.leftWidth) + borderMB +
		strings.Repeat(borderH, rw) + borderBR
	args.buffer.SetString(area.Point, top, f.borderStyle)
	f.drawTitle(args.buffer, area)
	args.buffer.SetString(geom.Point{
		X: area.X,
		Y: area.Bottom(),
	}, bot, f.borderStyle)
	for i := 0; i < area.Height-2; i++ {
		ry := area.Y + 1 + i
		if args.cutY > 0 && i == args.cutY-1 {
			cut := borderML + strings.Repeat(borderH, args.leftWidth) + borderMR
			args.buffer.SetString(
				geom.Point{X: area.X, Y: ry}, cut, f.borderStyle,
			)
			args.buffer.SetString(geom.Point{
				X: area.Right(),
				Y: ry,
			}, borderV, f.borderStyle)
		} else {
			args.buffer.SetString(
				geom.Point{X: area.X, Y: ry}, borderV, f.borderStyle,
			)
			args.buffer.SetString(geom.Point{
				X: area.X + 1 + args.leftWidth,
				Y: ry,
			}, borderV, f.borderStyle)
			args.buffer.SetString(geom.Point{
				X: area.Right(),
				Y: ry,
			}, borderV, f.borderStyle)
		}
	}
	return pickerBoxAreas{
		left: geom.Area{
			Point: area.Point.Add(geom.Point{X: 1, Y: 1}),
			Size:  geom.Size{Width: args.leftWidth, Height: area.Height - 2},
		},
		right: geom.Area{
			Point: area.Point.Add(geom.Point{X: 2 + args.leftWidth, Y: 1}),
			Size:  geom.Size{Width: rw, Height: area.Height - 2},
		},
	}
}

func (f pickerBoxFrame) drawSingle(
	buf *tui.Buffer, area geom.Area, cutY int,
) geom.Area {
	innerW := max(area.Width-2, 0)
	for dy := range area.Height {
		buf.FillRange(area.Point.Add(geom.Point{Y: dy}),
			area.Width, f.contentStyle)
	}
	if area.Width < 2 || area.Height < 2 {
		return geom.Area{}
	}
	top := borderTL + strings.Repeat(borderH, innerW) + borderTR
	bot := borderBL + strings.Repeat(borderH, innerW) + borderBR
	buf.SetString(area.Point, top, f.borderStyle)
	f.drawTitle(buf, area)
	buf.SetString(geom.Point{
		X: area.X,
		Y: area.Bottom(),
	}, bot, f.borderStyle)
	for i := 0; i < area.Height-2; i++ {
		ry := area.Y + 1 + i
		if cutY > 0 && i == cutY-1 {
			cut := borderML + strings.Repeat(borderH, innerW) + borderMR
			buf.SetString(geom.Point{X: area.X, Y: ry}, cut, f.borderStyle)
		} else {
			buf.SetString(geom.Point{X: area.X, Y: ry}, borderV, f.borderStyle)
			buf.SetString(geom.Point{
				X: area.Right(),
				Y: ry,
			}, borderV, f.borderStyle)
		}
	}
	return area.Inset(geom.Size{Width: 1, Height: 1})
}

// drawTitle overwrites the top-left of the border with the frame's title,
// truncated to fit inside the corners
func (f pickerBoxFrame) drawTitle(buf *tui.Buffer, area geom.Area) {
	if f.title == "" {
		return
	}
	maxW := area.Width - 2
	if maxW < 2 {
		return
	}
	title := " " + runewidth.Truncate(f.title, maxW-2, "") + " "
	buf.SetString(
		area.Point.Add(geom.Point{X: 1}), title, f.borderStyle,
	)
}
