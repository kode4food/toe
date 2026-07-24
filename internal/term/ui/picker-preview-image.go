package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
)

func (p *previewImageEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, at geom.Point,
) {
	area := geom.Area{Point: at, Size: ctx.size}
	style := ctx.th.Get("ui.text")
	if ctx.images == nil || !ctx.images.graphics {
		msg := i18n.Text(i18n.StatusImageUnsupported)
		renderImageMessage(buf, area, msg, style)
		return
	}
	pixels := p.image.Size()
	cells := imageCellSize(imageCellSizeArgs{
		maxCells: ctx.size,
		pixels:   pixels,
	})
	if !ctx.images.isReady(p.id, cells) {
		msg := i18n.Text(i18n.StatusImageLoading)
		renderImageMessage(buf, area, msg, style)
		return
	}
	start := area.Center(cells)
	bg := ctx.th.Get("ui.popup").BgColor()
	style = tui.Style{}.
		Fg(tui.ImageColor(p.id)).
		UlColor(tui.ImageColor(imagePlacementID(cells))).
		Bg(bg)
	for row := range cells.Height {
		for col := range cells.Width {
			cell := geom.Point{X: col, Y: row}
			sym := ctx.images.placeholder(cells, cell)
			buf.Set(start.Add(cell), tui.Cell{Symbol: sym, Style: style})
		}
	}
}

func (p *PickerComponent) previewImageCmd(
	cx *Context, screen geom.Size,
) tea.Cmd {
	res, ok := p.previewImage(cx, screen)
	if !ok {
		return nil
	}
	return cx.images.display(displayArgs{
		img:   res.entry.image,
		path:  res.entry.path,
		id:    res.entry.id,
		cells: res.cells,
	})
}

func (p *PickerComponent) hasPreviewImage(
	cx *Context, screen geom.Size,
) bool {
	_, ok := p.previewImage(cx, screen)
	return ok
}

type previewImageRes struct {
	entry *previewImageEntry
	cells geom.Size
}

func (p *PickerComponent) previewImage(
	cx *Context, screen geom.Size,
) (previewImageRes, bool) {
	item := p.state.selection()
	if item == nil || item.Location.Target.Path == "" {
		return previewImageRes{}, false
	}
	entry, ok := p.state.preview.cache.path(
		cx.Syntax, item.Location.Target.Path,
	).(*previewImageEntry)
	if !ok {
		return previewImageRes{}, false
	}
	size := p.previewImageSize(cx, screen)
	pixels := entry.image.Size()
	cells := imageCellSize(imageCellSizeArgs{
		maxCells: size,
		pixels:   pixels,
	})
	if cells.Empty() {
		return previewImageRes{}, false
	}
	return previewImageRes{entry: entry, cells: cells}, true
}

func (p *PickerComponent) previewImageSize(
	cx *Context, screen geom.Size,
) geom.Size {
	size := pickerOverlaySize(screen)
	if size.Width <= pickerMinPreviewArea || !previewEnabled(p.state.source) {
		return geom.Size{}
	}
	lw := pickerSplitLeftWidth(
		size.Width, cx.pickerLayout.SplitRatioFor(p.state.source.ID()),
	)
	return geom.Size{
		Width:  max(size.Width-lw-3-2*pickerPadX, 1),
		Height: max(size.Height-2, 0),
	}
}
