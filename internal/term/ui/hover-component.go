package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	hoverComponent struct {
		overlayBuf
		ec     *EditorComponent
		anchor hoverAnchor
		text   string
		lines  []popupLine
	}

	hoverAnchor struct {
		docID  view.DocumentId
		viewID view.Id
		pos    int
	}
)

var _ BufferOverlayComponent = (*hoverComponent)(nil)

func newHoverComponent(
	ec *EditorComponent, anchor hoverAnchor, text string,
) *hoverComponent {
	return &hoverComponent{ec: ec, anchor: anchor, text: text}
}

func (h *hoverComponent) HandleEvent(
	_ *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return ignored(), nil
	}
	switch key.Code {
	case tea.KeyEscape, tea.KeyEnter:
		return consumedWith(popLayer), nil
	default:
		return ignoredWith(popLayer), nil
	}
}

func (h *hoverComponent) Cursor(*Context, geom.Size) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (h *hoverComponent) Layout(
	cx *Context, screen geom.Size,
) (geom.Area, bool) {
	if !h.valid(cx) {
		return geom.Area{}, false
	}
	x, y := 0, 0
	if cur, ok := h.ec.Cursor(cx, screen); ok {
		x, y = cur.X+1, cur.Y+1
	}
	maxW := max(screen.Width-x, 30)
	maxH := min(screen.Height-y, 15)
	lines, size := measureTextPopup(
		geom.Size{Width: maxW, Height: maxH}, h.text,
	)
	if x+size.Width > screen.Width {
		x = max(screen.Width-size.Width, 0)
	}
	if y+size.Height > screen.Height {
		y = max(screen.Height-size.Height, 0)
	}
	h.lines = lines
	return geom.Area{
		Point: geom.Point{X: x, Y: y},
		Size:  size,
	}, true
}

func (h *hoverComponent) PaintBuffer(cx *Context, pl geom.Area) *tui.Buffer {
	return h.maybePaint(cx, pl.Size, func(buf *tui.Buffer) {
		paintTextPopup(cx, buf, h.lines)
	})
}

func (h *hoverComponent) valid(cx *Context) bool {
	v, ok := cx.Editor.FocusedView()
	if !ok || v.ID() != h.anchor.viewID || v.DocID() != h.anchor.docID {
		return false
	}
	doc, ok := cx.Editor.FocusedDocument()
	if !ok || doc.ID() != h.anchor.docID {
		return false
	}
	sel := doc.SelectionFor(v.ID())
	return sel.Primary().Cursor(doc.Text()) == h.anchor.pos
}

func newHoverAnchor(doc *view.Document, v *view.View) hoverAnchor {
	sel := doc.SelectionFor(v.ID())
	return hoverAnchor{
		docID:  doc.ID(),
		viewID: v.ID(),
		pos:    sel.Primary().Cursor(doc.Text()),
	}
}
