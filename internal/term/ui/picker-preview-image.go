package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
)

func (p *previewImageEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, x, y int,
) {
	style := lipglossToTUIStyle(ctx.th.Get("ui.text"))
	if ctx.images == nil || !ctx.images.graphics {
		msg := i18n.Text(i18n.StatusImageUnsupported)
		renderImageMessage(buf, x, y, ctx.w, ctx.h, msg, style)
		return
	}
	w, h := p.image.Size()
	cols, rows := imageCellSize(ctx.w, ctx.h, w, h)
	if !ctx.images.isReady(p.id, cols, rows) {
		msg := i18n.Text(i18n.StatusImageLoading)
		renderImageMessage(buf, x, y, ctx.w, ctx.h, msg, style)
		return
	}
	sx := x + max((ctx.w-cols)/2, 0)
	sy := y + max((ctx.h-rows)/2, 0)
	bg := lipglossToTUIStyle(ctx.th.Get("ui.popup")).BgColor()
	style = tui.Style{}.
		Fg(tui.ImageColor(p.id)).
		Bg(bg)
	for row := range rows {
		for col := range cols {
			sym := ctx.images.placeholder(cols, rows, row, col)
			buf.Set(sx+col, sy+row, tui.Cell{Symbol: sym, Style: style})
		}
	}
}

func (p *PickerComponent) previewImageCmd(
	cx *Context, screenW, screenH int,
) tea.Cmd {
	res, ok := p.previewImage(cx, screenW, screenH)
	if !ok {
		return nil
	}
	return cx.images.display(displayArgs{
		img:  res.entry.image,
		path: res.entry.path,
		id:   res.entry.id,
		cols: res.cols,
		rows: res.rows,
	})
}

func (p *PickerComponent) hasPreviewImage(
	cx *Context, screenW, screenH int,
) bool {
	_, ok := p.previewImage(cx, screenW, screenH)
	return ok
}

type previewImageRes struct {
	entry      *previewImageEntry
	cols, rows int
}

func (p *PickerComponent) previewImage(
	cx *Context, screenW, screenH int,
) (previewImageRes, bool) {
	item := p.state.selection()
	if item == nil || item.Location.Target.Path == "" {
		return previewImageRes{}, false
	}
	entry, ok := p.state.previewCache.path(
		cx.Syntax, item.Location.Target.Path,
	).(*previewImageEntry)
	if !ok {
		return previewImageRes{}, false
	}
	w, h := p.previewImageSize(cx, screenW, screenH)
	imgW, imgH := entry.image.Size()
	cols, rows := imageCellSize(w, h, imgW, imgH)
	if cols == 0 || rows == 0 {
		return previewImageRes{}, false
	}
	return previewImageRes{entry: entry, cols: cols, rows: rows}, true
}

func (p *PickerComponent) previewImageSize(
	cx *Context, screenW, screenH int,
) (int, int) {
	w, h := pickerOverlaySize(screenW, screenH)
	if w <= pickerMinPreviewArea || !previewEnabled(p.state.source) {
		return 0, 0
	}
	lw := pickerSplitLeftWidth(
		w, cx.pickerLayout.SplitRatioFor(p.state.source.ID()),
	)
	return max(w-lw-3-2*pickerPadX, 1), max(h-2, 0)
}
