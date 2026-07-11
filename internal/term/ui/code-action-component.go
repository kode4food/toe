package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

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
	bounds     Bounds
	listBounds Bounds
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
	msg tea.Msg, cx *Context,
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
		return m.handleMouseWheel(msg, cx), nil
	}
	return ignored(), nil
}

func (m *codeActionMenu) Cursor(int, int, *Context) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (m *codeActionMenu) Layout(
	screenW, screenH int, cx *Context,
) (Bounds, bool) {
	if len(m.actions) == 0 || !m.valid(cx) {
		return Bounds{}, false
	}
	x, y := m.popupPos(screenH, cx)
	w := m.width()
	h := min(len(m.actions), codeActionMaxRows) + 2
	if x+w > screenW {
		x = max(screenW-w, 0)
	}
	if y+h > screenH {
		y = max(y-h-1, 0)
	}
	return Bounds{x: x, y: y, w: w, h: h}, true
}

func (m *codeActionMenu) PaintBuffer(pl Bounds, cx *Context) *tui.Buffer {
	return m.maybePaint(pl.w, pl.h, cx, func(buf *tui.Buffer) {
		m.paint(buf, pl, cx)
	})
}

func (m *codeActionMenu) paint(buf *tui.Buffer, pl Bounds, cx *Context) {
	w, h := pl.w, pl.h
	m.bounds = pl
	menu, selected := promptCompletionStyles(cx)
	pop := popup{
		border: lipgloss.RoundedBorder(),
		borderStyle: lipglossToTUIStyle(
			menu.Foreground(pickerFrameStyle(cx).GetForeground()),
		),
		contentStyle: lipglossToTUIStyle(menu),
	}
	area := pop.drawInto(buf, 0, 0, w, h)
	m.listBounds = area.translate(pl.x, pl.y)
	base := lipglossToTUIStyle(menu)
	sel := lipglossToTUIStyle(selected)
	m.scroll = listClampScroll(m.scroll, len(m.actions), area.h)
	overflow := len(m.actions) > area.h
	listW := area.w
	if overflow {
		listW = max(area.w-completionScrollGap, 0)
	}
	for i := 0; i < area.h && m.scroll+i < len(m.actions); i++ {
		idx := m.scroll + i
		style := base
		if idx == m.cursor {
			style = sel
		}
		text := clipPad(" "+m.actions[idx].Title, listW)
		buf.SetString(area.x, area.y+i, text, style)
	}
	if overflow {
		m.renderScroll(buf, w-1, area.y, area.h, base)
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

func (m *codeActionMenu) popupPos(screenH int, cx *Context) (int, int) {
	return m.ec.popupAnchorBelowCaret(screenH, cx, codeActionMaxRows)
}

func (m *codeActionMenu) move(n int) {
	m.markDirty()
	m.cursor = (m.cursor + n + len(m.actions)) % len(m.actions)
	m.scroll = listEnsureCursorVisible(
		m.scroll, m.cursor, len(m.actions), m.visibleRows(),
	)
}

func (m *codeActionMenu) visibleRows() int {
	if m.listBounds.h > 0 {
		return m.listBounds.h
	}
	return codeActionMaxRows
}

func (m *codeActionMenu) apply(comp *Compositor, cx *Context) tea.Cmd {
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
	if !m.bounds.contains(msg.X, msg.Y) {
		return ignoredWith(popLayer)
	}
	if idx, ok := listIndexAt(m.listBounds, m.scroll, msg.X, msg.Y); ok {
		if idx >= 0 && idx < len(m.actions) {
			m.markDirty()
			m.cursor = idx
			return consumedWith(m.apply)
		}
	}
	return consumed()
}

func (m *codeActionMenu) handleMouseWheel(
	msg tea.MouseWheelMsg, cx *Context,
) EventResult {
	if !m.listBounds.contains(msg.X, msg.Y) {
		return ignoredWith(popLayer)
	}
	step := cx.Editor.Options().ScrollLines
	m.markDirty()
	switch msg.Button {
	case tea.MouseWheelUp:
		m.scroll = listScrollBy(m.scroll, len(m.actions), m.visibleRows(), -step)
	case tea.MouseWheelDown:
		m.scroll = listScrollBy(m.scroll, len(m.actions), m.visibleRows(), step)
	}
	return consumed()
}

func (m *codeActionMenu) renderScroll(
	buf *tui.Buffer, x, y, rows int, style tui.Style,
) {
	if rows <= 0 || len(m.actions) <= rows {
		return
	}
	scrollH := min((rows*rows+len(m.actions)-1)/len(m.actions), rows)
	scrollY := (rows - scrollH) * m.scroll / (len(m.actions) - rows)
	for i := range scrollH {
		buf.SetString(x, y+scrollY+i, "▌", style)
	}
}
