package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
)

// renderImagePane draws Kitty placeholder cells and an image status line
func (r *renderPass) renderImagePane(
	buf *tui.Buffer, pane *ImagePane, y0 int, focused bool,
) {
	a := pane.Area()
	contentH := max(a.Height-1, 0)
	r.paintImage(buf, pane, geom.Area{
		Point: geom.Point{X: a.X, Y: y0 + a.Y},
		Size:  geom.Size{Width: a.Width, Height: contentH},
	})
	r.renderImageStatus(
		buf, pane, geom.Point{X: a.X, Y: y0 + a.Y + contentH},
		a.Width, focused,
	)
}

func (r *renderPass) renderImageStatus(
	buf *tui.Buffer, pane *ImagePane, at geom.Point, width int, focused bool,
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
	buf.SetString(at, strings.Repeat(" ", width), baseTUI)

	cx := at.X
	mode := " IMG "
	buf.SetString(geom.Point{X: cx, Y: at.Y}, mode, modeSt)
	cx += runewidth.StringWidth(mode)
	buf.SetString(
		geom.Point{X: cx, Y: at.Y},
		" "+filepath.Base(pane.Path()), baseTUI,
	)

	pixels := img.Size()
	info := fmt.Sprintf("%d×%d %d%% ", pixels.Width, pixels.Height, pane.Zoom())
	buf.SetString(geom.Point{
		X: at.X + width - runewidth.StringWidth(info),
		Y: at.Y,
	}, info, baseTUI)
}

// paintImage fills a width by height cell region with centered kitty Unicode
// placeholder cells; the terminal paints the transmitted image over them
func (r *renderPass) paintImage(
	buf *tui.Buffer, pane *ImagePane, area geom.Area,
) {
	if !r.cx.images.graphics {
		r.renderImageMessage(buf, area, i18n.StatusImageUnsupported)
		return
	}
	img := pane.Image()
	id := kittyImageID(img.ContentID(), uint32(pane.ID()), false)
	// Draw at the put size, not the live zoom, so the grid, the placement's
	// c=/r=, and the centering box cannot drift apart while a zoom settles
	cells, ok := r.cx.images.readySize(id)
	if !ok {
		r.renderImageLoading(buf, area)
		return
	}
	// editor background shows through transparent pixels
	bg := lipglossToTUIStyle(r.activeTheme().Get("ui.background")).BgColor()
	style := tui.Style{}.
		Fg(tui.ImageColor(id)).
		Bg(bg)
	// show the centered window of the grid, so a zoomed-in image crops
	// symmetrically instead of pinning to the top-left
	visW := min(cells.Width, area.Width)
	visH := min(cells.Height, area.Height)
	screen := geom.Point{
		X: area.X + (area.Width-visW)/2,
		Y: area.Y + (area.Height-visH)/2,
	}
	grid := geom.Point{
		X: (cells.Width - visW) / 2,
		Y: (cells.Height - visH) / 2,
	}
	for row := range visH {
		for col := range visW {
			sym := r.cx.images.placeholder(
				cells, geom.Point{X: grid.X + col, Y: grid.Y + row},
			)
			buf.Set(geom.Point{X: screen.X + col, Y: screen.Y + row},
				tui.Cell{Symbol: sym, Style: style})
		}
	}
}

func (r *renderPass) renderImageLoading(
	buf *tui.Buffer, area geom.Area,
) {
	r.renderImageMessage(buf, area, i18n.StatusImageLoading)
}

func (r *renderPass) renderImageMessage(
	buf *tui.Buffer, area geom.Area, key i18n.Key,
) {
	style := lipglossToTUIStyle(r.activeTheme().Get("ui.text"))
	renderImageMessage(buf, area, i18n.Text(key), style)
}

func renderImageMessage(
	buf *tui.Buffer, area geom.Area, msg string, style tui.Style,
) {
	if area.Empty() {
		return
	}
	mw := runewidth.StringWidth(msg)
	buf.SetString(geom.Point{
		X: area.X + max((area.Width-mw)/2, 0),
		Y: area.Y + area.Height/2,
	}, msg, style)
}

type imagePaneCellSizeArgs struct {
	pane     *ImagePane
	maxCells geom.Size
	pixels   geom.Size
}

func imagePaneCellSize(args imagePaneCellSizeArgs) geom.Size {
	cells := imageCellSize(imageCellSizeArgs{
		maxCells: args.maxCells, pixels: args.pixels,
	})
	if cells.Empty() {
		return geom.Size{}
	}
	zoom := args.pane.Zoom()
	return geom.Size{
		Width:  max(cells.Width*zoom/100, 1),
		Height: max(cells.Height*zoom/100, 1),
	}
}
