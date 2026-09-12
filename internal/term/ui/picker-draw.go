package ui

import (
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
)

type pickerRender struct {
	component *PickerComponent
	context   *Context
	buffer    *tui.Buffer
	theme     *theme.Theme

	columns     []string
	widths      []int
	width       int
	matchColumn int
	header      bool
	frame       pickerBoxFrame
	itemStyle   tui.Style
	matchStyle  tui.Style
	selectStyle tui.Style
	selectMatch tui.Style
	section     tui.Style
	count       tui.Style
}

func newPickerRender(
	p *PickerComponent, cx *Context, buf *tui.Buffer,
) *pickerRender {
	ps := p.state
	cols := ps.source.Columns()
	return &pickerRender{
		component:   p,
		context:     cx,
		buffer:      buf,
		theme:       cx.Theme(),
		columns:     cols,
		matchColumn: ps.source.MatchColumn(),
		header:      pickerHasHeader(cols) && len(ps.list.matched) > 0,
		frame: pickerBoxFrame{
			borderStyle:  pickerFrameStyle(cx),
			contentStyle: pickerContentStyle(cx),
			title:        ps.source.Title(),
		},
		itemStyle:   pickerItemStyle(cx),
		matchStyle:  pickerMatchStyle(cx),
		selectStyle: pickerSelStyle(cx),
		selectMatch: pickerSelMatchStyle(cx),
		section:     pickerSectionStyle(cx),
		count:       pickerCountStyle(cx),
	}
}

func (p *pickerRender) drawBox(area geom.Area, lw int) {
	areas := p.frame.drawSplit(drawSplitArgs{
		buffer:    p.buffer,
		area:      area,
		leftWidth: lw,
		cutY:      2,
	})
	p.drawList(areas.left)
	p.drawPreview(areas.right)
}

func (p *pickerRender) drawPane(area geom.Area) {
	p.drawList(p.frame.drawSingle(p.buffer, area, 2))
}

func (p *pickerRender) drawList(area geom.Area) {
	ps := p.component.state
	headerH := 0
	if p.header {
		headerH = 1
	}
	ps.list.rows = max(area.Height-2-headerH, 1)
	p.width = area.Width
	if len(p.columns) > 1 {
		p.widths = pickerColumnWidths(ps, max(p.width-pickerMarkerW-1, 0))
	}
	p.component.caret = p.writePromptRow(area)
	itemY := area.Y + 2 // row 1 is the cut-separator, skip it
	if p.header {
		p.writeHeader(geom.Area{
			X:      area.X,
			Y:      itemY,
			Width:  area.Width,
			Height: 1,
		})
		itemY++
	}
	ps.clampScroll()
	for i := range ps.list.rows {
		idx := ps.list.scroll + i
		if idx >= len(ps.list.matched) {
			break
		}
		p.writeItem(
			geom.Point{X: area.X, Y: itemY + i},
			ps.list.matched[idx], idx == ps.list.cursor,
		)
	}
	if len(ps.list.matched) == 0 {
		p.writeCenteredHint(geom.Area{
			X:      area.X,
			Y:      itemY,
			Width:  area.Width,
			Height: ps.list.rows,
		}, pickerEmptyHint(ps))
	}
}

func (p *pickerRender) drawPreview(area geom.Area) {
	cx := p.context
	comp := p.component
	ps := comp.state

	comp.previewBounds = area
	if ps.list.cursor != ps.preview.scrollFor {
		ps.preview.vScroll = 0
		ps.preview.hScroll = 0
		ps.preview.scrollFor = ps.list.cursor
	}
	item := ps.selection()
	if item == nil {
		return
	}
	innerW := max(area.Width-2*overlayPadX, 1)
	if comp.previewTheme != p.theme {
		comp.previewTheme = p.theme
		comp.previewStyle = previewHighlighter(p.theme)
	}
	ctx := previewCtx{
		highlight: comp.previewStyle,
		picker:    ps,
		item:      item,
		editor:    ps.editor,
		syntax:    cx.Syntax,
		images:    cx.images,
		size:      geom.Size{Width: innerW, Height: area.Height},
		wrap:      comp.previewWrapWidth(innerW),
		theme:     p.theme,
		styles:    comp.styles,
		hlFrom:    -1,
	}
	if lr := item.TargetLines(); lr != nil {
		ctx.hlFrom = lr.From
		ctx.hlTo = lr.To
	}
	ctx.renderInto(p.buffer, area.Point.Add(geom.Point{X: overlayPadX}))
}
