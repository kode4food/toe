package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type codeActionMenu struct {
	overlayBuf
	ec         *EditorComponent
	docID      view.DocumentId
	viewID     view.Id
	actions    []view.CodeAction
	cursor     int
	scroll     int
	bounds     geom.Area
	listBounds geom.Area
}

const (
	codeActionMaxRows  = 10
	codeActionMinWidth = 16
)

var _ BufferOverlayComponent = (*codeActionMenu)(nil)

func newCodeActionMenu(
	ec *EditorComponent, docID view.DocumentId, viewID view.Id,
	actions []view.CodeAction,
) *codeActionMenu {
	cursor := 0
	for i, a := range actions {
		if a.Preferred {
			cursor = i
			break
		}
	}
	return &codeActionMenu{
		ec: ec, docID: docID, viewID: viewID, actions: actions, cursor: cursor,
	}
}

func (m *codeActionMenu) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyEscape:
			return consumedWith(popLayer), nil
		case tea.KeyUp:
			m.move(-1)
			return consumed(), nil
		case tea.KeyDown:
			m.move(1)
			return consumed(), nil
		case tea.KeyEnter:
			return consumedWith(m.apply), nil
		}
		// any other key dismisses the menu and passes through
		return ignoredWith(popLayer), nil
	case tea.MouseClickMsg:
		return m.handleMouseClick(msg), nil
	case tea.MouseWheelMsg:
		return m.handleMouseWheel(cx, msg), nil
	}
	return ignored(), nil
}

func (m *codeActionMenu) Cursor(*Context, geom.Size) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (m *codeActionMenu) Layout(
	cx *Context, screen geom.Size,
) (geom.Area, bool) {
	if len(m.actions) == 0 || !m.valid(cx) {
		return geom.Area{}, false
	}
	at := m.popupPos(cx, screen.Height)
	w := m.width()
	h := min(len(m.actions), codeActionMaxRows) + 2
	if at.X+w > screen.Width {
		at.X = max(screen.Width-w, 0)
	}
	if at.Y+h > screen.Height {
		at.Y = max(at.Y-h-1, 0)
	}
	return geom.Area{
		Point: at,
		Size:  geom.Size{Width: w, Height: h},
	}, true
}

func (m *codeActionMenu) PaintBuffer(cx *Context, pl geom.Area) *tui.Buffer {
	return m.maybePaint(cx, pl.Size, func(buf *tui.Buffer) {
		m.paint(cx, buf, pl)
	})
}

func (m *codeActionMenu) paint(cx *Context, buf *tui.Buffer, pl geom.Area) {
	w, h := pl.Width, pl.Height
	m.bounds = pl
	menu, selected := promptCompletionStyles(cx)
	pop := popup{
		border: lipgloss.RoundedBorder(),
		borderStyle: lipglossToTUIStyle(
			menu.Foreground(pickerFrameStyle(cx).GetForeground()),
		),
		contentStyle: lipglossToTUIStyle(menu),
	}
	area := pop.drawInto(buf, geom.Area{Size: geom.Size{Width: w, Height: h}})
	m.listBounds = area.Translate(pl.Point)
	base := lipglossToTUIStyle(menu)
	sel := lipglossToTUIStyle(selected)
	m.scroll = listClampScroll(m.scroll, len(m.actions), area.Height)
	overflow := len(m.actions) > area.Height
	listW := area.Width
	if overflow {
		listW = max(area.Width-completionScrollGap, 0)
	}
	for i := 0; i < area.Height && m.scroll+i < len(m.actions); i++ {
		idx := m.scroll + i
		style := base
		if idx == m.cursor {
			style = sel
		}
		text := clipPad(" "+m.actions[idx].Title, listW)
		buf.SetString(geom.Point{
			X: area.X,
			Y: area.Y + i,
		}, text, style)
	}
	if overflow {
		m.renderScroll(
			buf, geom.Point{X: w - 1, Y: area.Y}, area.Height, base,
		)
	}
}

func (m *codeActionMenu) width() int {
	w := codeActionMinWidth
	for _, a := range m.actions {
		w = max(w, runewidth.StringWidth(a.Title)+2)
	}
	if len(m.actions) > codeActionMaxRows {
		w += completionScrollGap
	}
	return w + 2
}

func (m *codeActionMenu) popupPos(
	cx *Context, screenH int,
) geom.Point {
	return m.ec.popupAnchorBelowCaret(cx, screenH, codeActionMaxRows)
}

func (m *codeActionMenu) move(n int) {
	m.markDirty()
	m.cursor = (m.cursor + n + len(m.actions)) % len(m.actions)
	m.scroll = listEnsureCursorVisible(
		m.scroll, m.cursor, len(m.actions), m.visibleRows(),
	)
}

func (m *codeActionMenu) visibleRows() int {
	if m.listBounds.Height > 0 {
		return m.listBounds.Height
	}
	return codeActionMaxRows
}

func (m *codeActionMenu) apply(cx *Context, comp *Compositor) tea.Cmd {
	comp.Pop()
	if m.cursor < 0 || m.cursor >= len(m.actions) {
		return nil
	}
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return nil
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return nil
	}
	ls := cx.Editor.LanguageServerController()
	if ls == nil {
		return nil
	}
	if err := ls.ApplyCodeAction(doc, v.ID(), m.actions[m.cursor]); err != nil {
		cx.Editor.SetStatusMsg(err.Error())
	}
	return nil
}

func (m *codeActionMenu) valid(cx *Context) bool {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok || doc.ID() != m.docID {
		return false
	}
	v, ok := cx.Editor.FocusedView()
	return ok && v.ID() == m.viewID
}

func (m *codeActionMenu) handleMouseClick(msg tea.MouseClickMsg) EventResult {
	if !m.bounds.Contains(geom.Point{X: msg.X, Y: msg.Y}) {
		return ignoredWith(popLayer)
	}
	at := geom.Point{X: msg.X, Y: msg.Y}
	if idx, ok := listIndexAt(m.listBounds, m.scroll, at); ok {
		if idx >= 0 && idx < len(m.actions) {
			m.markDirty()
			m.cursor = idx
			return consumedWith(m.apply)
		}
	}
	return consumed()
}

func (m *codeActionMenu) handleMouseWheel(
	cx *Context, msg tea.MouseWheelMsg,
) EventResult {
	if !m.listBounds.Contains(geom.Point{X: msg.X, Y: msg.Y}) {
		return ignoredWith(popLayer)
	}
	step := cx.Editor.Options().ScrollLines
	m.markDirty()
	switch msg.Button {
	case tea.MouseWheelUp:
		m.scroll = listScrollBy(
			m.scroll, len(m.actions), m.visibleRows(), -step,
		)
	case tea.MouseWheelDown:
		m.scroll = listScrollBy(m.scroll, len(m.actions), m.visibleRows(), step)
	}
	return consumed()
}

func (m *codeActionMenu) renderScroll(
	buf *tui.Buffer, at geom.Point, rows int, style tui.Style,
) {
	if rows <= 0 || len(m.actions) <= rows {
		return
	}
	scrollH := min((rows*rows+len(m.actions)-1)/len(m.actions), rows)
	scrollY := (rows - scrollH) * m.scroll / (len(m.actions) - rows)
	for i := range scrollH {
		buf.SetString(geom.Point{
			X: at.X,
			Y: at.Y + scrollY + i,
		}, scrollbarThumb, style)
	}
}
