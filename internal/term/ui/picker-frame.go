package ui

import (
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
	pop := f.popup()
	pop.drawInto(args.buffer, area)
	if area.Width < 2 || area.Height < 2 {
		return pickerBoxAreas{}
	}
	drawPopupTitle(args.buffer, area, f.title, f.borderStyle)
	pop.divider(args.buffer, area, 1+args.leftWidth)
	if args.cutY > 0 {
		drawPopupRule(drawPopupRuleArgs{
			buf:   args.buffer,
			at:    geom.Point{X: area.X, Y: area.Y + args.cutY},
			width: args.leftWidth + 2,
			style: f.borderStyle,
		})
	}
	return pickerBoxAreas{
		left: geom.Area{
			Point: area.Add(geom.Point{X: 1, Y: 1}),
			Size:  geom.Size{Width: args.leftWidth, Height: area.Height - 2},
		},
		right: geom.Area{
			Point: area.Add(geom.Point{X: 2 + args.leftWidth, Y: 1}),
			Size:  geom.Size{Width: rw, Height: area.Height - 2},
		},
	}
}

func (f pickerBoxFrame) drawSingle(
	buf *tui.Buffer, area geom.Area, cutY int,
) geom.Area {
	inner := f.popup().drawInto(buf, area)
	if area.Width < 2 || area.Height < 2 {
		return geom.Area{}
	}
	drawPopupTitle(buf, area, f.title, f.borderStyle)
	if cutY > 0 {
		drawPopupRule(drawPopupRuleArgs{
			buf:   buf,
			at:    geom.Point{X: area.X, Y: area.Y + cutY},
			width: area.Width,
			style: f.borderStyle,
		})
	}
	return inner
}

func (f pickerBoxFrame) popup() popup {
	return popup{borderStyle: f.borderStyle, contentStyle: f.contentStyle}
}
