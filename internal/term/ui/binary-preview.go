package ui

import (
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
)

func (p *previewBinaryEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, at geom.Point,
) {
	offsetWidth := binaryOffsetWidth(p.size)
	width := binaryBytesPerRow(binaryBytesPerRowArgs{
		width:       ctx.size.Width,
		offsetWidth: offsetWidth,
	})
	rows := (p.size + int64(width) - 1) / int64(width)
	maxScroll := max(int(rows)-ctx.size.Height, 0)
	scroll := min(max(ctx.picker.preview.scroll, 0), maxScroll)
	ctx.picker.preview.scroll = scroll
	offset := int64(scroll * width)
	data, err := readBinaryRange(
		p.path, offset, width*ctx.size.Height,
	)
	area := geom.Area{Point: at, Size: ctx.size}
	style := tui.Style{}.Bg(ctx.th.Get("ui.popup").BgColor())
	if err != nil {
		renderCenteredMessage(buf, area, i18n.ErrorText(err), style)
		return
	}
	styles := binaryStyles(ctx.th, style)
	for row := range ctx.size.Height {
		start := row * width
		if start >= len(data) {
			break
		}
		end := min(start+width, len(data))
		renderBinaryRow(renderBinaryRowArgs{
			buf:         buf,
			at:          at.Add(geom.Point{Y: row}),
			offset:      offset + int64(start),
			data:        data[start:end],
			bytesPerRow: width,
			offsetWidth: offsetWidth,
			maxWidth:    ctx.size.Width,
			styles:      styles,
		})
	}
}
