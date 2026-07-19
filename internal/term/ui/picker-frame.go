package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type (
	pickerBoxFrame struct {
		border       lipgloss.Border
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
			geom.Point{X: area.X, Y: area.Y + dy}, area.Width, f.contentStyle,
		)
	}
	if area.Width < 2 || area.Height < 2 {
		return pickerBoxAreas{}
	}
	top := f.border.TopLeft +
		strings.Repeat(f.border.Top, lw) +
		f.border.MiddleTop +
		strings.Repeat(f.border.Top, rw) +
		f.border.TopRight
	bot := f.border.BottomLeft +
		strings.Repeat(f.border.Bottom, lw) +
		f.border.MiddleBottom +
		strings.Repeat(f.border.Bottom, rw) +
		f.border.BottomRight
	buf.SetString(area.Point, top, f.borderStyle)
	buf.SetString(geom.Point{
		X: area.X,
		Y: area.Y + area.Height - 1,
	}, bot, f.borderStyle)
	for i := 0; i < area.Height-2; i++ {
		ry := area.Y + 1 + i
		if cutY > 0 && i == cutY-1 {
			cut := f.border.MiddleLeft +
				strings.Repeat(f.border.Top, lw) +
				f.border.MiddleRight
			buf.SetString(geom.Point{X: area.X, Y: ry}, cut, f.borderStyle)
			buf.SetString(geom.Point{
				X: area.X + area.Width - 1,
				Y: ry,
			}, f.border.Right, f.borderStyle)
		} else {
			buf.SetString(
				geom.Point{X: area.X, Y: ry}, f.border.Left, f.borderStyle,
			)
			buf.SetString(geom.Point{
				X: area.X + 1 + lw,
				Y: ry,
			}, f.border.Left, f.borderStyle)
			buf.SetString(geom.Point{
				X: area.X + area.Width - 1,
				Y: ry,
			}, f.border.Right, f.borderStyle)
		}
	}
	return pickerBoxAreas{
		left: geom.Area{
			Point: geom.Point{X: area.X + 1, Y: area.Y + 1},
			Size:  geom.Size{Width: lw, Height: area.Height - 2},
		},
		right: geom.Area{
			Point: geom.Point{X: area.X + 2 + lw, Y: area.Y + 1},
			Size:  geom.Size{Width: rw, Height: area.Height - 2},
		},
	}
}

func (f pickerBoxFrame) drawSingle(
	buf *tui.Buffer, area geom.Area, cutY int,
) geom.Area {
	innerW := max(area.Width-2, 0)
	for dy := range area.Height {
		buf.FillRange(geom.Point{X: area.X, Y: area.Y + dy},
			area.Width, f.contentStyle)
	}
	if area.Width < 2 || area.Height < 2 {
		return geom.Area{}
	}
	top := f.border.TopLeft +
		strings.Repeat(f.border.Top, innerW) +
		f.border.TopRight
	bot := f.border.BottomLeft +
		strings.Repeat(f.border.Bottom, innerW) +
		f.border.BottomRight
	buf.SetString(area.Point, top, f.borderStyle)
	buf.SetString(geom.Point{
		X: area.X,
		Y: area.Y + area.Height - 1,
	}, bot, f.borderStyle)
	for i := 0; i < area.Height-2; i++ {
		ry := area.Y + 1 + i
		if cutY > 0 && i == cutY-1 {
			cut := f.border.MiddleLeft +
				strings.Repeat(f.border.Top, innerW) +
				f.border.MiddleRight
			buf.SetString(geom.Point{X: area.X, Y: ry}, cut, f.borderStyle)
		} else {
			buf.SetString(
				geom.Point{X: area.X, Y: ry}, f.border.Left, f.borderStyle,
			)
			buf.SetString(geom.Point{
				X: area.X + area.Width - 1,
				Y: ry,
			}, f.border.Right, f.borderStyle)
		}
	}
	return geom.Area{
		Point: geom.Point{X: area.X + 1, Y: area.Y + 1},
		Size:  geom.Size{Width: innerW, Height: area.Height - 2},
	}
}
