package ui

import (
	"fmt"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
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
	th := r.context.ThemeFor(focused)
	r.paintImage(buf, pane, geom.Area{
		Point: geom.Point{X: a.X, Y: y0 + a.Y},
		Size:  geom.Size{Width: a.Width, Height: contentH},
	}, th)
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
	th := r.context.Theme()
	baseTUI := th.Get("ui.statusline.inactive")
	modeSt := baseTUI
	if args.focused {
		baseTUI = th.Get("ui.statusline")
		modeSt = th.Get("ui.statusline." + args.pane.Mode().Scope())
	}

	pixels := args.pane.Image().Size()
	right := []statusElem{{
		text: fmt.Sprintf("%d%s%d %d%%",
			pixels.Width, imageSizeTimes, pixels.Height, args.pane.Zoom(),
		),
		style: baseTUI,
	}}
	right = r.withMaximizedStatus(right)
	name := view.DocumentRelativeName(view.DocumentRelativeNameArgs{
		Path:    args.pane.Path(),
		BaseDir: r.context.Editor.Cwd(),
	})
	statusRow{
		at:        args.at,
		width:     args.width,
		baseStyle: baseTUI,
		left: []statusElem{
			statusBadge(args.pane.Mode().String(), modeSt),
			{text: name, style: baseTUI},
		},
		right: right,
	}.paint(args.buf)
}

func (r *renderPass) paintImage(
	buf *tui.Buffer, pane *ImagePane, area geom.Area, th *theme.Theme,
) {
	bg := th.Get("ui.background").BgColor()
	if !r.context.images.graphics {
		r.renderImageMessage(buf, area, i18n.StatusImageUnsupported, th)
		return
	}
	img := pane.Image()
	id := kittyImageID(kittyImageIDArgs{
		content: img.ContentID(),
		surface: uint32(pane.ID()),
	})
	// Draw at the put size, not the live zoom, so the grid, the placement's
	// c=/r=, and the centering box cannot drift apart while a zoom settles
	cells, ok := r.context.images.readySize(id)
	if !ok {
		r.renderImageLoading(buf, area, th)
		return
	}
	style := tui.Style{}.
		Fg(tui.ImageColor(id)).
		UlColor(tui.ImageColor(imagePlacementID(id))).
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
			sym := r.context.images.placeholder(
				cells, geom.Point{X: grid.X + col, Y: grid.Y + row},
			)
			buf.Set(geom.Point{X: screen.X + col, Y: screen.Y + row},
				tui.Cell{Symbol: sym, Style: style})
		}
	}
}

func (r *renderPass) renderImageLoading(
	buf *tui.Buffer, area geom.Area, th *theme.Theme,
) {
	r.renderImageMessage(buf, area, i18n.StatusImageLoading, th)
}

func (r *renderPass) renderImageMessage(
	buf *tui.Buffer, area geom.Area, key i18n.Key, th *theme.Theme,
) {
	style := th.Get("ui.text")
	renderCenteredMessage(buf, area, i18n.Text(key), style)
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
