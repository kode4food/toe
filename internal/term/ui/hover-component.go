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
	ec     *EditorComponent
	anchor hoverAnchor
	text   string
}

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

func (h *hoverComponent) Render(int, int, *Context) string {
	return ""
}

func (h *hoverComponent) Cursor(int, int, *Context) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (h *hoverComponent) RenderOverBuffer(buf *tui.Buffer, cx *Context) {
	if !h.valid(cx) {
		return
	}
	x, y := 0, 0
	if cur, ok := h.ec.Cursor(buf.Width, buf.Height, cx); ok {
		x, y = cur.X+1, cur.Y+1
	}
	drawTextPopup(buf, x, y, max(buf.Width-x, 30), min(buf.Height-y, 15),
		h.text, cx)
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
