package ui

import (
	"strings"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type (
	pickerBoxFrame struct {
		borderStyle  tui.Style
		contentStyle tui.Style
	}

	pickerBoxAreas struct {
		left  geom.Area
		right geom.Area
	}
)

func (f pickerBoxFrame) drawSplit(
	buf *tui.Buffer, area geom.Area, lw, cutY int,
) pickerBoxAreas {
	rw := max(area.Width-lw-3, 0)
	for dy := range area.Height {
		buf.FillRange(
			area.Point.Add(geom.Point{Y: dy}), area.Width, f.contentStyle,
		)
	}
	if area.Width < 2 || area.Height < 2 {
		return pickerBoxAreas{}
	}
	top := borderTL + strings.Repeat(borderH, lw) + borderMT +
		strings.Repeat(borderH, rw) + borderTR
	bot := borderBL + strings.Repeat(borderH, lw) + borderMB +
		strings.Repeat(borderH, rw) + borderBR
	buf.SetString(area.Point, top, f.borderStyle)
	buf.SetString(geom.Point{
		X: area.X,
		Y: area.Bottom(),
	}, bot, f.borderStyle)
	for i := 0; i < area.Height-2; i++ {
		ry := area.Y + 1 + i
		if cutY > 0 && i == cutY-1 {
			cut := borderML + strings.Repeat(borderH, lw) + borderMR
			buf.SetString(geom.Point{X: area.X, Y: ry}, cut, f.borderStyle)
			buf.SetString(geom.Point{
				X: area.Right(),
				Y: ry,
			}, borderV, f.borderStyle)
		} else {
			buf.SetString(
				geom.Point{X: area.X, Y: ry}, borderV, f.borderStyle,
			)
			buf.SetString(geom.Point{
				X: area.X + 1 + lw,
				Y: ry,
			}, borderV, f.borderStyle)
			buf.SetString(geom.Point{
				X: area.Right(),
				Y: ry,
			}, borderV, f.borderStyle)
		}
	}
	return pickerBoxAreas{
		left: geom.Area{
			Point: area.Point.Add(geom.Point{X: 1, Y: 1}),
			Size:  geom.Size{Width: lw, Height: area.Height - 2},
		},
		right: geom.Area{
			Point: area.Point.Add(geom.Point{X: 2 + lw, Y: 1}),
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
