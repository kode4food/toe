package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/tui"
)

type (
	pickerBoxFrame struct {
		border       lipgloss.Border
		borderStyle  tui.Style
		contentStyle tui.Style
	}

	pickerBoxAreas struct {
		left  popupArea
		right popupArea
	}
)

func (f pickerBoxFrame) drawSplit(
	buf *tui.Buffer, x, y, w, h, lw, cutY int,
) pickerBoxAreas {
	rw := max(w-lw-3, 0)
	for dy := range h {
		buf.FillRange(x, y+dy, w, f.contentStyle)
	}
	if w < 2 || h < 2 {
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
	buf.SetString(x, y, top, f.borderStyle)
	buf.SetString(x, y+h-1, bot, f.borderStyle)
	for i := 0; i < h-2; i++ {
		ry := y + 1 + i
		if cutY > 0 && i == cutY-1 {
			cut := f.border.MiddleLeft +
				strings.Repeat(f.border.Top, lw) +
				f.border.MiddleRight
			buf.SetString(x, ry, cut, f.borderStyle)
			buf.SetString(x+w-1, ry, f.border.Right, f.borderStyle)
		} else {
			buf.SetString(x, ry, f.border.Left, f.borderStyle)
			buf.SetString(x+1+lw, ry, f.border.Left, f.borderStyle)
			buf.SetString(x+w-1, ry, f.border.Right, f.borderStyle)
		}
	}
	return pickerBoxAreas{
		left:  popupArea{x: x + 1, y: y + 1, w: lw, h: h - 2},
		right: popupArea{x: x + 2 + lw, y: y + 1, w: rw, h: h - 2},
	}
}

func (f pickerBoxFrame) drawSingle(
	buf *tui.Buffer, x, y, w, h, cutY int,
) popupArea {
	innerW := max(w-2, 0)
	for dy := range h {
		buf.FillRange(x, y+dy, w, f.contentStyle)
	}
	if w < 2 || h < 2 {
		return popupArea{}
	}
	top := f.border.TopLeft +
		strings.Repeat(f.border.Top, innerW) +
		f.border.TopRight
	bot := f.border.BottomLeft +
		strings.Repeat(f.border.Bottom, innerW) +
		f.border.BottomRight
	buf.SetString(x, y, top, f.borderStyle)
	buf.SetString(x, y+h-1, bot, f.borderStyle)
	for i := 0; i < h-2; i++ {
		ry := y + 1 + i
		if cutY > 0 && i == cutY-1 {
			cut := f.border.MiddleLeft +
				strings.Repeat(f.border.Top, innerW) +
				f.border.MiddleRight
			buf.SetString(x, ry, cut, f.borderStyle)
		} else {
			buf.SetString(x, ry, f.border.Left, f.borderStyle)
			buf.SetString(x+w-1, ry, f.border.Right, f.borderStyle)
		}
	}
	return popupArea{x: x + 1, y: y + 1, w: innerW, h: h - 2}
}
