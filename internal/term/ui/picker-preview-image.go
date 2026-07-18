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
	entry, cols, rows, ok := p.previewImage(cx, screenW, screenH)
	if !ok {
		return nil
	}
	return cx.images.display(entry.id, entry.image, cols, rows)
}

func (p *PickerComponent) hasPreviewImage(
	cx *Context, screenW, screenH int,
) bool {
	_, _, _, ok := p.previewImage(cx, screenW, screenH)
	return ok
}

func (p *PickerComponent) previewImage(
	cx *Context, screenW, screenH int,
) (*previewImageEntry, int, int, bool) {
	item := p.state.selection()
	if item == nil || item.Location.Target.Path == "" {
		return nil, 0, 0, false
	}
	entry, ok := p.state.previewCache.path(
		cx.Syntax, item.Location.Target.Path,
	).(*previewImageEntry)
	if !ok {
		return nil, 0, 0, false
	}
	w, h := p.previewImageSize(cx, screenW, screenH)
	imgW, imgH := entry.image.Size()
	cols, rows := imageCellSize(w, h, imgW, imgH)
	if cols == 0 || rows == 0 {
		return nil, 0, 0, false
	}
	return entry, cols, rows, true
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
