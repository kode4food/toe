package ui

import (
	"encoding/binary"
	"hash"
	"hash/fnv"
	"image"
	"image/color"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	scrollbarImageState struct {
		image    *Image
		styles   *styles
		size     geom.Size
		thumb    core.Span
		rev      uint64
		imageRev uint64
		styleRev uint64
	}

	scrollbarPlacement struct {
		images *imageRegistry
		redraw *view.Tree
		id     uint32
	}

	scrollbarRowScale struct {
		height int
		slots  int
	}
)

const (
	scrollbarImageSalt      = 0x5C401B
	previewScrollbarSurface = 0xB17E5C
	scrollbarRevChunk       = 256
)

func (m Model) scrollbarImageCmd() tea.Cmd {
	cx := m.context
	cache := m.component.cache
	if cache == nil || !cx.images.graphics {
		return nil
	}
	var cmds []tea.Cmd
	for id, bar := range cache.viewScrollbars {
		cmds = append(cmds, bar.img.displayCmd(
			cx.images, &bar.bar, scrollbarImageID(uint32(id)),
		))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (r *renderPass) drawScrollbar(
	bar *scrollbar, buf *tui.Buffer, id view.Id,
) {
	vs, ok := r.editor.cache.viewScrollbars[id]
	if !ok {
		return
	}
	r.scrollbarPlacement(id).draw(bar, &vs.img, buf)
}

func (r *renderPass) scrollbarPlacement(id view.Id) scrollbarPlacement {
	return scrollbarPlacement{
		images: r.context.images,
		redraw: r.context.Editor.Tree(),
		id:     scrollbarImageID(uint32(id)),
	}
}

func (p *previewCtx) scrollbarPlacement() scrollbarPlacement {
	return scrollbarPlacement{
		images: p.images,
		redraw: p.editor.Tree(),
		id:     scrollbarImageID(previewScrollbarSurface),
	}
}

func (s scrollbarRowScale) slotsAt(y int) core.Span {
	return core.Span{
		From: y * s.slots / s.height,
		To:   max((y+1)*s.slots/s.height, y*s.slots/s.height+1),
	}
}

func (s *scrollbar) rowKind(at core.Span) scrollMarkKind {
	kind := scrollMarkNone
	for slot := at.From; slot < min(at.To, len(s.marks)); slot++ {
		kind = max(kind, s.marks[slot])
	}
	return kind
}

func (s scrollbarPlacement) active() bool {
	return s.images != nil && s.images.graphics
}

func (s scrollbarPlacement) draw(
	bar *scrollbar, st *scrollbarImageState, buf *tui.Buffer,
) {
	if s.active() {
		cell := s.images.cell
		size := geom.Size{
			Width: cell.Width, Height: bar.geom.rows * cell.Height,
		}
		if st.seal(bar, size); st.rev != st.imageRev && s.redraw != nil {
			// this pass's transmit was built before it drew
			s.redraw.Redraw()
		}
	}
	cells := geom.Size{Width: 1, Height: bar.geom.rows}
	if !s.active() || !s.images.isReady(s.id, cells) {
		bar.draw(buf)
		return
	}
	style := scrollbarImageStyle(s.id, bar.styles.scrollTrack[0].BgColor())
	for row := range bar.geom.rows {
		buf.Set(bar.at.Add(geom.Point{Y: row}), tui.Cell{
			Symbol: s.images.placeholder(cells, geom.Point{Y: row}),
			Style:  style,
		})
	}
}

func (s *scrollbarImageState) seal(bar *scrollbar, size geom.Size) {
	g := bar.geom
	s.size = size
	s.thumb = core.Span{
		From: g.slotAt(bar.topLine),
		To:   g.slotEnd(bar.topLine + g.rows - 1),
	}
	if s.styles != bar.styles {
		s.styles = bar.styles
		s.styleRev++
	}
	h := fnv.New64a()
	writeMarksRev(h, bar.marks)
	writeScrollbarRev(h,
		s.thumb.From, s.thumb.To, len(bar.marks),
		size.Width, size.Height, int(s.styleRev),
	)
	s.rev = h.Sum64()
}

func (s *scrollbarImageState) ensure(bar *scrollbar) (*Image, uint64, bool) {
	if s.rev == 0 || bar.styles == nil || s.size.IsEmpty() {
		return nil, 0, false
	}
	if s.image == nil || s.imageRev != s.rev {
		s.image = renderScrollbarImage(bar, s.thumb, s.size)
		s.imageRev = s.rev
	}
	return s.image, s.rev, true
}

func (s *scrollbarImageState) displayCmd(
	images *imageRegistry, bar *scrollbar, id uint32,
) tea.Cmd {
	if !images.graphics {
		return nil
	}
	img, rev, ok := s.ensure(bar)
	if !ok {
		return nil
	}
	return images.display(displayArgs{
		img:   img,
		id:    id,
		cells: geom.Size{Width: 1, Height: bar.geom.rows},
		rev:   rev,
	})
}

func scrollbarImageID(surface uint32) uint32 {
	return kittyImageID(kittyImageIDArgs{
		content: scrollbarImageSalt,
		surface: surface,
	})
}

func renderScrollbarImage(
	marks *scrollbar, thumb core.Span, size geom.Size,
) *Image {
	img := image.NewRGBA(image.Rect(0, 0, size.Width, size.Height))
	scale := scrollbarRowScale{
		height: size.Height,
		slots:  marks.geom.slots(),
	}
	// a lone line must not shrink to a hairline
	thick := max(size.Height/(marks.geom.rows*8), 1)
	held := scrollMarkNone
	pending := 0
	for y := range size.Height {
		at := scale.slotsAt(y)
		kind := marks.rowKind(at)
		switch {
		case kind != scrollMarkNone:
			held = kind
			pending = thick - 1
		case pending > 0:
			kind = held
			pending--
		}
		style := marks.styles.scrollTrack[kind]
		if at.From >= thumb.From && at.From < thumb.To {
			style = marks.styles.scrollThumb[kind]
		}
		paint := style.FgColor()
		if kind == scrollMarkNone {
			paint = style.BgColor()
		}
		paintScrollbarRow(img, y, paint)
	}
	return &Image{Image: img}
}

func paintScrollbarRow(img *image.RGBA, y int, c color.Color) {
	r, g, b, a := c.RGBA()
	at := img.PixOffset(0, y)
	row := img.Pix[at : at+img.Bounds().Dx()*4]
	for x := 0; x < len(row); x += 4 {
		row[x] = uint8(r >> 8)
		row[x+1] = uint8(g >> 8)
		row[x+2] = uint8(b >> 8)
		row[x+3] = uint8(a >> 8)
	}
}

func scrollbarImageStyle(id uint32, bg tui.Color) tui.Style {
	return tui.Style{}.
		Fg(tui.ImageColor(id)).
		UlColor(tui.ImageColor(imagePlacementID(id))).
		Bg(bg)
}

func writeMarksRev(h hash.Hash64, marks []scrollMarkKind) {
	var buf [scrollbarRevChunk]byte
	n := 0
	for _, mark := range marks {
		buf[n] = byte(mark)
		if n++; n == len(buf) {
			_, _ = h.Write(buf[:n])
			n = 0
		}
	}
	_, _ = h.Write(buf[:n])
}

func writeScrollbarRev(h hash.Hash64, vs ...int) {
	var b [8]byte
	for _, n := range vs {
		binary.LittleEndian.PutUint64(b[:], uint64(n))
		_, _ = h.Write(b[:])
	}
}
