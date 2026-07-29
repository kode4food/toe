package ui

import (
	"fmt"
	"path/filepath"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
)

const imageSizeTimes = "\u00d7" // '×' - multiplication sign

func (r *renderPass) renderImagePane(
	buf *tui.Buffer, pane *ImagePane, y0 int, focused bool,
) {
	if pane.Image() == nil {
		return
	}
	a := pane.Area()
	contentH := max(a.Height-1, 0)
	r.paintImage(buf, pane, geom.Area{
		Point: geom.Point{X: a.X, Y: y0 + a.Y},
		Size:  geom.Size{Width: a.Width, Height: contentH},
	})
	r.renderImageStatus(renderImageStatusArgs{
		buf:     buf,
		pane:    pane,
		at:      geom.Point{X: a.X, Y: y0 + a.Y + contentH},
		width:   a.Width,
		focused: focused,
	})
}

type renderImageStatusArgs struct {
	buf     *tui.Buffer
	pane    *ImagePane
	at      geom.Point
	width   int
	focused bool
}

func (r *renderPass) renderImageStatus(args renderImageStatusArgs) {
	pane := args.pane
	th := r.activeTheme()
	baseTUI := th.Get("ui.statusline.inactive")
	modeSt := baseTUI
	if args.focused {
		baseTUI = th.Get("ui.statusline")
		modeSt = th.Get("ui.statusline." + pane.Mode().Scope())
	}

	pixels := pane.Image().Size()
	right := []statusElem{{
		text: fmt.Sprintf("%d%s%d %d%%",
			pixels.Width, imageSizeTimes, pixels.Height, pane.Zoom(),
		),
		style: baseTUI,
	}}
	right = r.withMaximizedStatus(right)
	renderStatusElems(renderStatusElemsArgs{
		buf:       args.buf,
		at:        args.at,
		width:     args.width,
		baseStyle: baseTUI,
		left: []statusElem{
			statusBadge("IMG", modeSt),
			{text: filepath.Base(pane.Path()), style: baseTUI},
		},
		right: right,
	})
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
	bg := r.activeTheme().Get("ui.background").BgColor()
	style := tui.Style{}.
		Fg(tui.ImageColor(id)).
		UlColor(tui.ImageColor(imagePlacementID(cells))).
		Bg(bg)
	// show a centered window into the grid; pan scrolls it so a zoomed-in image
	// exposes its clipped edges instead of pinning to the top-left
	visW := min(cells.Width, area.Width)
	visH := min(cells.Height, area.Height)
	screen := geom.Point{
		X: area.X + (area.Width-visW)/2,
		Y: area.Y + (area.Height-visH)/2,
	}
	pan := pane.Pan()
	grid := geom.Point{
		X: min(max((cells.Width-visW)/2+pan.X, 0), cells.Width-visW),
		Y: min(max((cells.Height-visH)/2+pan.Y, 0), cells.Height-visH),
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
	style := r.activeTheme().Get("ui.text")
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
		maxCells: args.maxCells,
		pixels:   args.pixels,
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
