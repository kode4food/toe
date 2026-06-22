package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/tui"
)

func (p *PickerComponent) drawPickerBox(
	buf *tui.Buffer, x, y, w, h int, cx *Context,
) {
	ps := p.state
	lw := w/2 - 1
	innerH := h - 2

	cols := ps.source.Columns()
	headerH := 0
	if len(cols) > 1 {
		headerH = 1
	}
	ps.listHeight = max(innerH-2-headerH, 1)

	frame := pickerBoxFrame{
		border:       lipgloss.RoundedBorder(),
		borderStyle:  lipglossToTUIStyle(pickerFrameStyle(cx)),
		contentStyle: lipglossToTUIStyle(pickerContentStyle(cx)),
	}
	areas := frame.drawSplit(buf, x, y, w, h, lw, 2)

	writePickerPromptRow(
		buf, areas.left.x, areas.left.y, areas.left.w, ps, cx,
	)
	itemY := areas.left.y + 2 // row 1 is the cut-separator, skip it
	if len(cols) > 1 {
		writePickerHeader(buf, areas.left.x, itemY, areas.left.w, ps, cx)
		itemY++
	}
	ps.clampScroll()
	for i := range ps.listHeight {
		idx := ps.listScroll + i
		if idx >= len(ps.matched) {
			break
		}
		writePickerItem(buf, areas.left.x, itemY+i, areas.left.w,
			&pickerItemRender{
				p: ps, match: ps.matched[idx], w: areas.left.w,
				selected: idx == ps.cursor, cx: cx,
			},
		)
	}

	p.drawPreviewInto(buf, areas.right.x, areas.right.y, areas.right.w,
		areas.right.h, cx)
}

func (p *PickerComponent) drawPickerPane(
	buf *tui.Buffer, x, y, w, h int, cx *Context,
) {
	ps := p.state
	innerH := h - 2

	cols := ps.source.Columns()
	headerH := 0
	if len(cols) > 1 {
		headerH = 1
	}
	ps.listHeight = max(innerH-2-headerH, 1)

	frame := pickerBoxFrame{
		border:       lipgloss.RoundedBorder(),
		borderStyle:  lipglossToTUIStyle(pickerFrameStyle(cx)),
		contentStyle: lipglossToTUIStyle(pickerContentStyle(cx)),
	}
	area := frame.drawSingle(buf, x, y, w, h, 2)

	writePickerPromptRow(buf, area.x, area.y, area.w, ps, cx)
	itemY := area.y + 2 // row 1 is the cut-separator, skip it
	if len(cols) > 1 {
		writePickerHeader(buf, area.x, itemY, area.w, ps, cx)
		itemY++
	}
	ps.clampScroll()
	for i := 0; ps.listScroll+i < len(ps.matched) && i < ps.listHeight; i++ {
		idx := ps.listScroll + i
		writePickerItem(buf, area.x, itemY+i, area.w, &pickerItemRender{
			p: ps, match: ps.matched[idx], w: area.w,
			selected: idx == ps.cursor, cx: cx,
		})
	}
}

func (p *PickerComponent) drawPreviewInto(
	buf *tui.Buffer, x0, y0, w, h int, cx *Context,
) {
	ps := p.state
	p.previewBounds = bounds{x: x0, y: y0, w: w, h: h}
	if ps.cursor != ps.previewScrollFor {
		ps.previewScroll = 0
		ps.previewScrollFor = ps.cursor
	}
	item := ps.selection()
	if item == nil {
		return
	}
	innerW := max(w-2*pickerPadX, 1)
	from, to, ok := item.Location.lineRange()
	ctx := previewCtx{
		picker: ps,
		item:   item,
		editor: cx.Editor,
		w:      innerW,
		h:      h,
		th:     cx.Theme(),
		hlFrom: -1,
	}
	if ok {
		ctx.hlFrom = from
		ctx.hlTo = to
	}
	ctx.renderInto(buf, x0+pickerPadX, y0)
}

func (b bounds) contains(x, y int) bool {
	return x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h
}
