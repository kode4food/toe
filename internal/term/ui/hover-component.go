package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type hoverAnchor struct {
	docID  view.DocumentId
	viewID view.Id
	pos    int
}

type hoverComponent struct {
	overlayBuf
	ec     *EditorComponent
	anchor hoverAnchor
	text   string
	lines  []popupLine
}

var _ BufferOverlayComponent = (*hoverComponent)(nil)

func newHoverComponent(
	ec *EditorComponent, anchor hoverAnchor, text string,
) *hoverComponent {
	return &hoverComponent{ec: ec, anchor: anchor, text: text}
}

func (h *hoverComponent) HandleEvent(
	msg tea.Msg, _ *Context,
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

func (h *hoverComponent) Cursor(int, int, *Context) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (h *hoverComponent) Layout(
	screenW, screenH int, cx *Context,
) (Bounds, bool) {
	if !h.valid(cx) {
		return Bounds{}, false
	}
	x, y := 0, 0
	if cur, ok := h.ec.Cursor(screenW, screenH, cx); ok {
		x, y = cur.X+1, cur.Y+1
	}
	maxW := max(screenW-x, 30)
	maxH := min(screenH-y, 15)
	lines, w, hh := measureTextPopup(maxW, maxH, h.text)
	if x+w > screenW {
		x = max(screenW-w, 0)
	}
	if y+hh > screenH {
		y = max(screenH-hh, 0)
	}
	h.lines = lines
	return Bounds{x: x, y: y, w: w, h: hh}, true
}

func (h *hoverComponent) PaintBuffer(pl Bounds, cx *Context) *tui.Buffer {
	buf := h.get(pl.w, pl.h)
	paintTextPopup(buf, h.lines, cx)
	return buf
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
