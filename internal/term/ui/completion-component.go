package ui

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	act "github.com/kode4food/toe/internal/view/action"
)

type completionComponent struct {
	ec       *EditorComponent
	all      []view.CompletionItem
	items    []view.CompletionItem
	anchor   completionAnchor
	cursor   int
	scroll   int
	resolved map[string]bool
}

type completionAnchor struct {
	docID  view.DocumentId
	viewID view.Id
	pos    int
}

const (
	completionMinWidth  = 1
	completionMaxRows   = 10
	completionScrollGap = 1
)

func newCompletionComponent(
	ec *EditorComponent, items []view.CompletionItem, anchor completionAnchor,
) *completionComponent {
	c := &completionComponent{
		ec:       ec,
		all:      items,
		items:    items,
		anchor:   anchor,
		resolved: map[string]bool{},
	}
	c.resetCursor()
	return c
}

func (c *completionComponent) HandleEvent(
	msg tea.Msg, cx *Context,
) (EventResult, tea.Cmd) {
	switch msg.(type) {
	case tea.MouseClickMsg, tea.MouseReleaseMsg, tea.MouseMotionMsg:
		return ignoredWith(popLayer), nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return ignored(), nil
	}
	switch key.Code {
	case tea.KeyEscape:
		return consumedWith(popLayer), nil
	case tea.KeyUp:
		c.move(-1)
		return consumed(), nil
	case tea.KeyDown:
		c.move(1)
		return consumed(), nil
	case tea.KeyEnter:
		c.accept(cx)
		return consumedWith(popLayer), nil
	default:
		k := FromTeaKey(key)
		if cx.Editor.Mode() == view.ModeInsert && k.IsTypable() {
			act.InsertChar(cx.Editor, k.Code.Char)
			return consumedWith(c.refresh), nil
		}
		return ignoredWith(popLayer), nil
	}
}

func (c *completionComponent) Render(int, int, *Context) string {
	return ""
}

func (c *completionComponent) Cursor(
	int, int, *Context,
) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (c *completionComponent) RenderOverBuffer(buf *tui.Buffer, cx *Context) {
	if !c.valid(cx) {
		return
	}
	if len(c.items) == 0 {
		return
	}
	x, y := c.popupPos(buf, cx)
	w := c.width()
	rows := min(len(c.items), completionMaxRows)
	h := rows + 2
	if x+w > buf.Width {
		x = max(buf.Width-w, 0)
	}
	if y+h > buf.Height {
		y = max(y-h-1, 0)
	}
	menu, selected := promptCompletionStyles(cx)
	pop := popup{
		borderStyle: lipglossToTUIStyle(
			menu.Foreground(pickerFrameStyle(cx).GetForeground()),
		),
		contentStyle: lipglossToTUIStyle(menu),
	}
	area := pop.drawInto(buf, x, y, w, h)
	base := lipglossToTUIStyle(menu)
	sel := lipglossToTUIStyle(selected)
	c.clampScroll(area.h)
	overflow := len(c.items) > area.h
	listW := area.w
	if overflow {
		listW = max(area.w-completionScrollGap, 0)
	}
	for i := 0; i < area.h && c.scroll+i < len(c.items); i++ {
		idx := c.scroll + i
		item := c.items[idx]
		style := base
		if idx == c.cursor {
			style = sel
		}
		c.renderRow(buf, area.x, area.y+i, area.w, listW, item, style)
	}
	if overflow {
		c.renderScroll(buf, x+w-1, area.y, area.h, base)
	}
	c.resolveSelected(cx)
	c.renderDocs(buf, x, y, w, h, cx)
}

func (c *completionComponent) valid(cx *Context) bool {
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

func (c *completionComponent) move(n int) {
	c.cursor = (c.cursor + n + len(c.items)) % len(c.items)
}

func (c *completionComponent) refresh(comp *Compositor, cx *Context) tea.Cmd {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		comp.Pop()
		return nil
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		comp.Pop()
		return nil
	}
	sel := doc.SelectionFor(v.ID())
	pos := sel.Primary().Cursor(doc.Text())
	if pos < c.anchor.pos {
		comp.Pop()
		return nil
	}
	query, err := doc.Text().SliceString(c.anchor.pos, pos)
	if err != nil {
		comp.Pop()
		return nil
	}
	selected := c.selectedKey()
	c.items = filterCompletionItems(c.all, query)
	if len(c.items) == 0 {
		comp.Pop()
		return nil
	}
	c.restoreCursor(selected)
	return nil
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

func (c *completionComponent) width() int {
	w := completionMinWidth
	for _, item := range c.items {
		w = max(w, c.rowWidth(item)+2)
	}
	if len(c.items) > completionMaxRows {
		w += completionScrollGap
	}
	return w + 2
}

func (c *completionComponent) rowWidth(item view.CompletionItem) int {
	if item.Kind == "" {
		return runewidth.StringWidth(item.Label)
	}
	return runewidth.StringWidth(item.Label) + 2 +
		runewidth.StringWidth(item.Kind)
}

func (c *completionComponent) renderRow(
	buf *tui.Buffer, x, y, w, listW int, item view.CompletionItem,
	style tui.Style,
) {
	buf.SetString(x, y, clipPad("", w), style)
	if item.Kind == "" {
		buf.SetString(x, y, clipPad(item.Label, listW), style)
		return
	}
	kindW := runewidth.StringWidth(item.Kind)
	if kindW > listW {
		buf.SetString(x, y, clipPad(item.Label, listW), style)
		return
	}
	labelW := max(listW-kindW-2, 0)
	buf.SetString(x, y, clipPad(item.Label, labelW), style)
	buf.SetString(x+listW-kindW, y, item.Kind, style)
}

func (c *completionComponent) popupPos(
	buf *tui.Buffer, cx *Context,
) (int, int) {
	if cur, ok := c.ec.Cursor(buf.Width, buf.Height, cx); ok {
		return cur.X, cur.Y + 1
	}
	return 0, max(buf.Height-completionMaxRows-2, 0)
}

func (c *completionComponent) resolveSelected(cx *Context) {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return
	}
	item := c.items[c.cursor]
	if item.ID == "" || c.resolved[item.ID] {
		return
	}
	c.resolved[item.ID] = true
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
	resolved, err := ls.ResolveCompletion(doc, v.ID(), item)
	if err != nil {
		cx.Editor.SetStatusMsg(err.Error())
		return
	}
	c.items[c.cursor] = resolved
	for i, item := range c.all {
		if item.ID == resolved.ID {
			c.all[i] = resolved
			return
		}
	}
}

func (c *completionComponent) renderDocs(
	buf *tui.Buffer, x, y, w, h int, cx *Context,
) {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return
	}
	text := c.items[c.cursor].Docs
	if text == "" {
		return
	}
	rightW := buf.Width - (x + w)
	if rightW > 30 {
		drawTextPopup(buf, x+w, y, rightW, buf.Height-y, text, cx)
		return
	}
	curY := y
	if cur, ok := c.ec.Cursor(buf.Width, buf.Height, cx); ok {
		curY = cur.Y
	}
	above := min(curY, y) - 1
	below := buf.Height - max(curY, y+h) - 1
	if below >= above && below > 1 {
		drawTextPopup(buf, 0, buf.Height-below, buf.Width, below, text, cx)
		return
	}
	if above > 1 {
		drawTextPopup(buf, 0, 0, buf.Width, min(above, 15), text, cx)
	}
}

func (c *completionComponent) resetCursor() {
	c.cursor = 0
	for i, item := range c.items {
		if item.Preselect {
			c.cursor = i
			break
		}
	}
	c.scroll = 0
}

func (c *completionComponent) restoreCursor(selected string) {
	if selected != "" {
		for i, item := range c.items {
			if completionItemKey(item) == selected {
				c.cursor = i
				return
			}
		}
	}
	c.resetCursor()
}

func (c *completionComponent) selectedKey() string {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return ""
	}
	return completionItemKey(c.items[c.cursor])
}

func filterCompletionItems(
	items []view.CompletionItem, query string,
) []view.CompletionItem {
	if query == "" {
		return append([]view.CompletionItem(nil), items...)
	}
	out := make([]view.CompletionItem, 0, len(items))
	for _, item := range items {
		if completionMatches(item, query) {
			out = append(out, item)
		}
	}
	return out
}

func completionMatches(item view.CompletionItem, query string) bool {
	text := item.Filter
	if text == "" {
		text = item.Label
	}
	return completionStartsWith(text, query)
}

func completionItemKey(item view.CompletionItem) string {
	if item.ID != "" {
		return item.ID
	}
	return item.Label + "\x00" + item.Insert + "\x00" + item.Kind
}

func completionStartsWith(text, query string) bool {
	text = strings.TrimLeftFunc(text, unicode.IsSpace)
	return strings.HasPrefix(strings.ToLower(text), strings.ToLower(query))
}

func (c *completionComponent) clampScroll(rows int) {
	if rows <= 0 || len(c.items) <= rows {
		c.scroll = 0
		return
	}
	if c.cursor < c.scroll {
		c.scroll = c.cursor
	}
	if c.cursor >= c.scroll+rows {
		c.scroll = c.cursor - rows + 1
	}
	c.scroll = min(c.scroll, len(c.items)-rows)
	c.scroll = max(c.scroll, 0)
}

func (c *completionComponent) renderScroll(
	buf *tui.Buffer, x, y, rows int, style tui.Style,
) {
	if rows <= 0 || len(c.items) <= rows {
		return
	}
	scrollH := min((rows*rows+len(c.items)-1)/len(c.items), rows)
	scrollY := 0
	if len(c.items) > rows {
		scrollY = (rows - scrollH) * c.scroll / (len(c.items) - rows)
	}
	for i := range scrollH {
		buf.SetString(x, y+scrollY+i, "▌", style)
	}
}

func popLayer(c *Compositor, _ *Context) tea.Cmd {
	c.Pop()
	return nil
}
