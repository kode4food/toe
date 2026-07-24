package ui

import (
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

func (p *PickerComponent) drawPickerBox(
	cx *Context, buf *tui.Buffer, area geom.Area, lw int,
) {
	ps := p.state
	innerH := area.Height - 2

	cols := ps.source.Columns()
	showHeader := pickerHasHeader(cols) && len(ps.list.matched) > 0
	headerH := 0
	if showHeader {
		headerH = 1
	}
	ps.list.height = max(innerH-2-headerH, 1)

	frame := pickerBoxFrame{
		borderStyle:  pickerFrameStyle(cx),
		contentStyle: pickerContentStyle(cx),
	}
	areas := frame.drawSplit(buf, area, lw, 2)

	writePickerPromptRow(cx, buf, areas.left, ps)
	itemY := areas.left.Y + 2 // row 1 is the cut-separator, skip it
	if showHeader {
		writePickerHeader(cx, buf, geom.Area{
			Point: geom.Point{X: areas.left.X, Y: itemY},
			Size:  geom.Size{Width: areas.left.Width, Height: 1},
		}, ps)
		itemY++
	}
	ps.clampScroll()
	for i := range ps.list.height {
		idx := ps.list.scroll + i
		if idx >= len(ps.list.matched) {
			break
		}
		writePickerItem(
			buf, geom.Point{X: areas.left.X, Y: itemY + i},
			&pickerItemRender{
				p: ps, match: ps.list.matched[idx], w: areas.left.Width,
				selected: idx == ps.list.cursor, cx: cx,
			},
		)
	}
	if len(ps.list.matched) == 0 {
		writePickerCenteredHint(cx, buf, geom.Area{
			Point: geom.Point{X: areas.left.X, Y: itemY},
			Size: geom.Size{
				Width:  areas.left.Width,
				Height: ps.list.height,
			},
		}, pickerEmptyHint(ps))
	}

	p.drawPreviewInto(cx, buf, areas.right)
}

func (p *PickerComponent) drawPickerPane(
	cx *Context, buf *tui.Buffer, area geom.Area,
) {
	ps := p.state
	innerH := area.Height - 2

	cols := ps.source.Columns()
	showHeader := pickerHasHeader(cols) && len(ps.list.matched) > 0
	headerH := 0
	if showHeader {
		headerH = 1
	}
	ps.list.height = max(innerH-2-headerH, 1)

	frame := pickerBoxFrame{
		borderStyle:  pickerFrameStyle(cx),
		contentStyle: pickerContentStyle(cx),
	}
	area = frame.drawSingle(buf, area, 2)

	writePickerPromptRow(cx, buf, area, ps)
	itemY := area.Y + 2 // row 1 is the cut-separator, skip it
	if showHeader {
		writePickerHeader(cx, buf, geom.Area{
			Point: geom.Point{X: area.X, Y: itemY},
			Size:  geom.Size{Width: area.Width, Height: 1},
		}, ps)
		itemY++
	}
	ps.clampScroll()
	for i := 0; ps.list.scroll+i < len(ps.list.matched) &&
		i < ps.list.height; i++ {
		idx := ps.list.scroll + i
		writePickerItem(buf,
			geom.Point{X: area.X, Y: itemY + i},
			&pickerItemRender{
				p: ps, match: ps.list.matched[idx], w: area.Width,
				selected: idx == ps.list.cursor, cx: cx,
			},
		)
	}
	if len(ps.list.matched) == 0 {
		writePickerCenteredHint(cx, buf, geom.Area{
			Point: geom.Point{X: area.X, Y: itemY},
			Size: geom.Size{
				Width:  area.Width,
				Height: ps.list.height,
			},
		}, pickerEmptyHint(ps))
	}
}

func (p *PickerComponent) drawPreviewInto(
	cx *Context, buf *tui.Buffer, area geom.Area,
) {
	ps := p.state
	p.previewBounds = area
	if ps.list.cursor != ps.preview.scrollFor {
		ps.preview.scroll = 0
		ps.preview.scrollFor = ps.list.cursor
	}
	item := ps.selection()
	if item == nil {
		return
	}
	innerW := max(area.Width-2*pickerPadX, 1)
	ctx := previewCtx{
		picker: ps,
		item:   item,
		editor: cx.Editor,
		syntax: cx.Syntax,
		images: cx.images,
		size:   geom.Size{Width: innerW, Height: area.Height},
		th:     cx.Theme(),
		hlFrom: -1,
	}
	if lr := item.Location.Lines; lr != nil {
		ctx.hlFrom = lr.From
		ctx.hlTo = lr.To
	}
	ctx.renderInto(buf, area.Point.Add(geom.Point{X: pickerPadX}))
}
