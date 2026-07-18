package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
)

// renderImagePane draws Kitty placeholder cells and an image status line
func (r *renderPass) renderImagePane(
	buf *tui.Buffer, pane *ImagePane, y0 int, focused bool,
) {
	a := pane.Area()
	contentH := max(a.Height-1, 0)
	r.paintImage(buf, pane, a.X, y0+a.Y, a.Width, contentH)
	r.renderImageStatus(
		buf, pane, a.X, y0+a.Y+contentH, a.Width, focused,
	)
}

func (r *renderPass) renderImageStatus(
	buf *tui.Buffer, pane *ImagePane, x, y, width int, focused bool,
) {
	img := pane.Image()
	th := r.activeTheme()
	statusKey := "ui.statusline"
	if !focused {
		statusKey = "ui.statusline.inactive"
	}
	baseTUI := lipglossToTUIStyle(th.Get(statusKey))
	modeSt := baseTUI
	if focused {
		modeSt = lipglossToTUIStyle(th.Get("ui.statusline.normal"))
	}
	buf.SetString(x, y, strings.Repeat(" ", width), baseTUI)

	cx := x
	mode := " IMG "
	buf.SetString(cx, y, mode, modeSt)
	cx += runewidth.StringWidth(mode)
	buf.SetString(cx, y, " "+filepath.Base(pane.Path()), baseTUI)

	w, h := img.Size()
	info := fmt.Sprintf("%d×%d %d%% ", w, h, pane.Zoom())
	buf.SetString(
		x+width-runewidth.StringWidth(info), y, info, baseTUI,
	)
}

// paintImage fills a width by height cell region with centered kitty Unicode
// placeholder cells; the terminal paints the transmitted image over them
func (r *renderPass) paintImage(
	buf *tui.Buffer, pane *ImagePane, x, y, width, height int,
) {
	if !r.cx.images.graphics {
		r.renderImageMessage(
			buf, x, y, width, height, i18n.StatusImageUnsupported,
		)
		return
	}
	img := pane.Image()
	w, h := img.Size()
	cols, rows := imagePaneCellSize(pane, width, height, w, h)
	id := kittyImageID(img.ContentID(), uint32(pane.ID()), false)
	if !r.cx.images.isReady(id, cols, rows) {
		if size, ok := r.cx.images.readySize(id); ok {
			cols, rows = size[0], size[1]
		} else {
			r.renderImageLoading(buf, x, y, width, height)
			return
		}
	}
	sx := x + (width-cols)/2
	sy := y + (height-rows)/2
	// The cell background shows through transparent image pixels, so use the
	// editor background rather than letting the terminal default bleed through
	bg := lipglossToTUIStyle(r.activeTheme().Get("ui.background")).BgColor()
	style := tui.Style{}.
		Fg(tui.ImageColor(id)).
		Bg(bg)
	for row := range rows {
		py := sy + row
		if py < y || py >= y+height {
			continue
		}
		for col := range cols {
			px := sx + col
			if px < x || px >= x+width {
				continue
			}
			sym := r.cx.images.placeholder(cols, rows, row, col)
			buf.Set(px, py, tui.Cell{Symbol: sym, Style: style})
		}
	}
}

func (r *renderPass) renderImageLoading(
	buf *tui.Buffer, x, y, width, height int,
) {
	r.renderImageMessage(buf, x, y, width, height, i18n.StatusImageLoading)
}

func (r *renderPass) renderImageMessage(
	buf *tui.Buffer, x, y, width, height int, key i18n.Key,
) {
	style := lipglossToTUIStyle(r.activeTheme().Get("ui.text"))
	renderImageMessage(buf, x, y, width, height, i18n.Text(key), style)
}

func renderImageMessage(
	buf *tui.Buffer, x, y, width, height int, msg string, style tui.Style,
) {
	if width <= 0 || height <= 0 {
		return
	}
	mw := runewidth.StringWidth(msg)
	buf.SetString(x+max((width-mw)/2, 0), y+height/2, msg, style)
}

func imagePaneCellSize(
	pane *ImagePane, maxCols, maxRows, imgW, imgH int,
) (int, int) {
	cols, rows := imageCellSize(maxCols, maxRows, imgW, imgH)
	if cols == 0 || rows == 0 {
		return 0, 0
	}
	zoom := pane.Zoom()
	return max(cols*zoom/100, 1), max(rows*zoom/100, 1)
}
