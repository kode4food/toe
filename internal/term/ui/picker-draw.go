package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/tui"
)

func (p *PickerComponent) drawPickerBox(
	buf *tui.Buffer, x, y, w, h, lw int, cx *Context,
) {
	ps := p.state
	innerH := h - 2

	cols := ps.source.Columns()
	showHeader := pickerHasHeader(cols) && len(ps.matched) > 0
	headerH := 0
	if showHeader {
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
	if showHeader {
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
	if len(ps.matched) == 0 {
		writePickerCenteredHint(buf, areas.left.x, itemY, areas.left.w,
			ps.listHeight, pickerEmptyHint(ps), cx)
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
	showHeader := pickerHasHeader(cols) && len(ps.matched) > 0
	headerH := 0
	if showHeader {
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
	if showHeader {
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
	if len(ps.matched) == 0 {
		writePickerCenteredHint(buf, area.x, itemY, area.w,
			ps.listHeight, pickerEmptyHint(ps), cx)
	}
}

func (p *PickerComponent) drawPreviewInto(
	buf *tui.Buffer, x, y, w, h int, cx *Context,
) {
	ps := p.state
	p.previewBounds = Bounds{x: x, y: y, w: w, h: h}
	if ps.cursor != ps.previewScrollFor {
		ps.previewScroll = 0
		ps.previewScrollFor = ps.cursor
	}
	item := ps.selection()
	if item == nil {
		return
	}
	innerW := max(w-2*pickerPadX, 1)
	ctx := previewCtx{
		picker: ps,
		item:   item,
		editor: cx.Editor,
		syntax: cx.Syntax,
		images: cx.images,
		w:      innerW,
		h:      h,
		th:     cx.Theme(),
		hlFrom: -1,
	}
	if lr := item.Location.Lines; lr != nil {
		ctx.hlFrom = lr.From
		ctx.hlTo = lr.To
	}
	ctx.renderInto(buf, x+pickerPadX, y)
}
