package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type (
	completionComponent struct {
		overlayBuf
		ec         *EditorComponent
		all        []view.CompletionItem
		items      []view.CompletionItem
		anchor     completionAnchor
		cursor     int
		scroll     int
		bounds     geom.Area
		listBounds geom.Area
		refreshGen int
		manual     bool
		incomplete bool
		nerd       bool
	}

	completionAnchor struct {
		docID  view.DocumentId
		viewID view.Id
		rev    int
		pos    int
	}

	completionRefreshMsg struct {
		layer *completionComponent
		gen   int
		rev   int
		pos   int
		res   view.CompletionResult
		err   error
	}
)

const (
	// CompletionMode is the keymap mode used while the completion popup is
	// focused
	CompletionMode = "COM"

	// CompletionAcceptAction accepts the selected completion item
	CompletionAcceptAction = "completion_accept"

	// CompletionCancelAction dismisses the completion popup
	CompletionCancelAction = "completion_cancel"

	// CompletionPreviousAction selects the previous completion item
	CompletionPreviousAction = "completion_previous"

	// CompletionNextAction selects the next completion item
	CompletionNextAction = "completion_next"

	// CompletionPageUpAction moves selection up by one completion page
	CompletionPageUpAction = "completion_page_up"

	// CompletionPageDownAction moves selection down by one completion page
	CompletionPageDownAction = "completion_page_down"

	// CompletionFirstAction selects the first completion item
	CompletionFirstAction = "completion_first"

	// CompletionLastAction selects the last completion item
	CompletionLastAction = "completion_last"
)

const (
	completionMinWidth  = 1
	completionMaxRows   = 10
	completionPageRows  = completionMaxRows - 1
	completionScrollGap = 1
)

func (c *completionComponent) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case completionRefreshMsg:
		return c.handleRefreshMsg(cx, msg)
	case tea.MouseClickMsg:
		return c.handleMouseClick(cx, msg), nil
	case tea.MouseWheelMsg:
		return c.handleMouseWheel(cx, msg), nil
	case tea.KeyPressMsg:
		return c.handleKeyPress(cx, msg)
	default:
		return ignored(), nil
	}
}

func (c *completionComponent) lookupAction(
	cx *Context, k command.KeyEvent,
) (string, bool) {
	name, found, _ := cx.Keymaps.LookupCommand(
		CompletionMode, []command.KeyEvent{k},
	)
	return name, found
}

func (c *completionComponent) handleAction(
	cx *Context, name string,
) EventResult {
	switch name {
	case CompletionAcceptAction:
		c.accept(cx)
		return consumedWith(popLayer)
	case CompletionCancelAction:
		return consumedWith(popLayer)
	case CompletionPreviousAction:
		c.moveBy(-1)
		return consumed()
	case CompletionNextAction:
		c.moveBy(1)
		return consumed()
	case CompletionPageUpAction:
		c.moveBy(-completionPageRows)
		return consumed()
	case CompletionPageDownAction:
		c.moveBy(completionPageRows)
		return consumed()
	case CompletionFirstAction:
		c.moveTo(0)
		return consumed()
	case CompletionLastAction:
		c.moveTo(len(c.items) - 1)
		return consumed()
	default:
		return ignored()
	}
}

func (c *completionComponent) Cursor(*Context, geom.Size) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (c *completionComponent) Layout(
	cx *Context, screen geom.Size,
) (geom.Area, bool) {
	if !c.valid(cx) || len(c.items) == 0 {
		return geom.Area{}, false
	}
	c.nerd = cx.Editor.Options().NerdFonts
	at := c.popupPos(cx, screen.Height)
	w := c.width()
	rows := min(len(c.items), completionMaxRows)
	h := rows + 2
	return fitPopup(geom.Area{
		Point: at,
		Size:  geom.Size{Width: w, Height: h},
	}, screen), true
}

func (c *completionComponent) PaintBuffer(
	cx *Context, pl geom.Area,
) *tui.Buffer {
	return c.maybePaint(cx, pl.Size, func(buf *tui.Buffer) {
		c.paint(cx, buf, pl)
	})
}

func (c *completionComponent) paint(
	cx *Context, buf *tui.Buffer, pl geom.Area,
) {
	c.nerd = cx.Editor.Options().NerdFonts
	query, _ := c.query(cx)
	w := pl.Width
	c.bounds = pl
	menu, selected := promptCompletionStyles(cx)
	pop := popup{
		border: lipgloss.RoundedBorder(),
		borderStyle: lipglossToTUIStyle(
			menu.Foreground(pickerFrameStyle(cx).GetForeground()),
		),
		contentStyle: lipglossToTUIStyle(menu),
	}
	area := pop.drawInto(buf, geom.Area{Size: pl.Size})
	c.listBounds = area.Translate(pl.Point)
	base := lipglossToTUIStyle(menu)
	sel := lipglossToTUIStyle(selected)
	match := lipglossToTUIStyle(pickerMatchStyle(cx))
	selMatch := lipglossToTUIStyle(pickerSelMatchStyle(cx))
	info := lipglossToTUIStyle(completionInfoStyle(cx, false))
	selInfo := lipglossToTUIStyle(completionInfoStyle(cx, true))
	c.clampScroll(area.Height)
	overflow := len(c.items) > area.Height
	listW := area.Width
	if overflow {
		listW = max(area.Width-completionScrollGap, 0)
	}
	for i := 0; i < area.Height && c.scroll+i < len(c.items); i++ {
		idx := c.scroll + i
		item := c.items[idx]
		style := base
		matchStyle := match
		infoStyle := info
		selected := idx == c.cursor
		iconStyle := lipglossToTUIStyle(
			completionIconStyle(cx, item.Kind, selected),
		)
		if selected {
			style = sel
			matchStyle = selMatch
			infoStyle = selInfo
		}
		c.renderRow(renderCompletionRowArgs{
			buf:      buf,
			at:       area.Point.Add(geom.Point{Y: i}),
			width:    area.Width,
			listW:    listW,
			item:     item,
			selected: selected,
			query:    query,
			base:     style,
			match:    matchStyle,
			icon:     iconStyle,
			info:     infoStyle,
		})
	}
	if overflow {
		c.renderScroll(
			buf, geom.Point{X: w - 1, Y: area.Y}, area.Height, base,
		)
	}
}

func (c *completionComponent) valid(cx *Context) bool {
	if cx.Editor.Mode() != view.ModeInsert {
		return false
	}
	doc, ok := cx.Editor.FocusedDocument()
	if !ok || doc.ID() != c.anchor.docID {
		return false
	}
	v, ok := cx.Editor.FocusedView()
	if !ok || v.ID() != c.anchor.viewID {
		return false
	}
	pos := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
	return pos >= c.anchor.pos
}

func (c *completionComponent) handleKeyPress(
	cx *Context, msg tea.KeyPressMsg,
) (EventResult, tea.Cmd) {
	k := FromTeaKey(msg)
	if name, ok := c.lookupAction(cx, k); ok {
		return c.handleAction(cx, name), nil
	}
	if cx.Editor.Mode() == view.ModeInsert && k.IsTypable() {
		action.InsertChar(cx.Editor, k.Code.Char)
		return consumedWith(c.refresh), nil
	}
	return ignoredWith(popLayer), nil
}

func (c *completionComponent) handleMouseClick(
	_ *Context, msg tea.MouseClickMsg,
) EventResult {
	if !c.bounds.Contains(geom.Point{X: msg.X, Y: msg.Y}) {
		return ignoredWith(popLayer)
	}
	at := geom.Point{X: msg.X, Y: msg.Y}
	if idx, ok := listIndexAt(c.listBounds, c.scroll, at); ok {
		if idx >= 0 && idx < len(c.items) {
			c.moveTo(idx)
		}
	}
	return consumed()
}

func (c *completionComponent) handleMouseWheel(
	cx *Context, msg tea.MouseWheelMsg,
) EventResult {
	if !c.listBounds.Contains(geom.Point{X: msg.X, Y: msg.Y}) {
		return ignoredWith(popLayer)
	}
	step := cx.Editor.Options().ScrollLines
	switch msg.Button {
	case tea.MouseWheelUp:
		c.scrollBy(-step)
	case tea.MouseWheelDown:
		c.scrollBy(step)
	}
	return consumed()
}

func (c *completionComponent) accept(cx *Context) {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return
	}
	item := c.items[c.cursor]
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return
	}
	ls := cx.Editor.LanguageServerController()
	if ls == nil {
		return
	}
	if err := ls.ApplyCompletion(doc, v.ID(), item); err != nil {
		cx.Editor.SetStatusMsg(err.Error())
	}
}

func (c *completionComponent) popupPos(
	cx *Context, screenH int,
) geom.Point {
	return c.ec.popupAnchorBelowCaret(cx, screenH, completionMaxRows)
}
