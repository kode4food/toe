package ui

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	act "github.com/kode4food/toe/internal/view/action"
)

type completionComponent struct {
	ec         *EditorComponent
	all        []view.CompletionItem
	items      []view.CompletionItem
	anchor     completionAnchor
	opts       CompletionOptions
	cursor     int
	scroll     int
	bounds     bounds
	listBounds bounds
	refreshGen int
	manual     bool
	incomplete bool
}

type completionAnchor struct {
	docID  view.DocumentId
	viewID view.Id
	rev    int
	pos    int
}

type completionMatch struct {
	item  view.CompletionItem
	score int
	order int
}

type completionRowParts struct {
	icon  string
	label string
	info  string
}

type completionRefreshMsg struct {
	layer *completionComponent
	gen   int
	rev   int
	pos   int
	res   view.CompletionResult
	err   error
}

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
	completionMinWidth        = 1
	completionMaxRows         = 10
	completionPageRows        = completionMaxRows - 1
	completionScrollGap       = 1
	completionPreviewMaxWidth = 40
)

func (c *completionComponent) HandleEvent(
	msg tea.Msg, cx *Context,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case completionRefreshMsg:
		return c.handleRefreshMsg(msg, cx)
	case tea.MouseClickMsg:
		return c.handleMouseClick(msg, cx), nil
	case tea.MouseWheelMsg:
		return c.handleMouseWheel(msg, cx), nil
	case tea.KeyPressMsg:
		return c.handleKeyPress(msg, cx)
	default:
		return ignored(), nil
	}
}

func (c *completionComponent) Render(int, int, *Context) string {
	return ""
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
	name string, cx *Context,
) EventResult {
	switch name {
	case CompletionAcceptAction:
		c.accept(cx)
		return consumedWith(popLayer)
	case CompletionCancelAction:
		return consumedWith(popLayer)
	case CompletionPreviousAction:
		c.move(-1)
		return consumed()
	case CompletionNextAction:
		c.move(1)
		return consumed()
	case CompletionPageUpAction:
		c.move(-completionPageRows)
		return consumed()
	case CompletionPageDownAction:
		c.move(completionPageRows)
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

func (c *completionComponent) Cursor(int, int, *Context) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (c *completionComponent) RenderOverBuffer(buf *tui.Buffer, cx *Context) {
	if !c.valid(cx) {
		return
	}
	if len(c.items) == 0 {
		return
	}
	query, _ := c.query(cx)
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
		border: lipgloss.RoundedBorder(),
		borderStyle: lipglossToTUIStyle(
			menu.Foreground(pickerFrameStyle(cx).GetForeground()),
		),
		contentStyle: lipglossToTUIStyle(menu),
	}
	area := pop.drawInto(buf, x, y, w, h)
	c.bounds = bounds{x: x, y: y, w: w, h: h}
	c.listBounds = bounds(area)
	base := lipglossToTUIStyle(menu)
	sel := lipglossToTUIStyle(selected)
	match := lipglossToTUIStyle(pickerMatchStyle(cx))
	selMatch := lipglossToTUIStyle(pickerSelMatchStyle(cx))
	info := lipglossToTUIStyle(completionInfoStyle(cx, false))
	selInfo := lipglossToTUIStyle(completionInfoStyle(cx, true))
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
		c.renderRow(
			buf, area.x, area.y+i, area.w, listW,
			renderCompletionRowArgs{
				item: item, selected: selected, query: query,
				base: style, match: matchStyle,
				icon: iconStyle, info: infoStyle,
			},
		)
	}
	if overflow {
		c.renderScroll(buf, x+w-1, area.y, area.h, base)
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

func (c *completionComponent) move(n int) {
	c.cursor = (c.cursor + n + len(c.items)) % len(c.items)
	c.manual = true
	c.ensureCursorVisible(c.visibleRows())
}

func (c *completionComponent) moveTo(idx int) {
	c.cursor = min(max(idx, 0), len(c.items)-1)
	c.manual = true
	c.ensureCursorVisible(c.visibleRows())
}

func (c *completionComponent) handleKeyPress(
	msg tea.KeyPressMsg, cx *Context,
) (EventResult, tea.Cmd) {
	k := FromTeaKey(msg)
	if name, ok := c.lookupAction(cx, k); ok {
		return c.handleAction(name, cx), nil
	}
	if cx.Editor.Mode() == view.ModeInsert && k.IsTypable() {
		act.InsertChar(cx.Editor, k.Code.Char)
		return consumedWith(c.refresh), nil
	}
	return ignoredWith(popLayer), nil
}

func (c *completionComponent) handleMouseClick(
	msg tea.MouseClickMsg, _ *Context,
) EventResult {
	if !c.bounds.contains(msg.X, msg.Y) {
		return ignoredWith(popLayer)
	}
	if idx, ok := listIndexAt(c.listBounds, c.scroll, msg.X, msg.Y); ok {
		if idx >= 0 && idx < len(c.items) {
			c.moveTo(idx)
		}
	}
	return consumed()
}

func (c *completionComponent) handleMouseWheel(
	msg tea.MouseWheelMsg, cx *Context,
) EventResult {
	if !c.listBounds.contains(msg.X, msg.Y) {
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
		if c.incomplete {
			return c.refreshCmd(cx)
		}
		comp.Pop()
		return nil
	}
	if c.manual {
		c.restoreCursor(selected)
		return c.refreshCmd(cx)
	}
	c.resetCursor()
	return c.refreshCmd(cx)
}

func (c *completionComponent) handleRefreshMsg(
	msg completionRefreshMsg, cx *Context,
) (EventResult, tea.Cmd) {
	if msg.layer != c {
		return ignored(), nil
	}
	if msg.gen != c.refreshGen || !c.refreshValid(cx, msg) {
		return consumed(), nil
	}
	if msg.err != nil {
		cx.Editor.SetStatusMsg(msg.err.Error())
		return consumed(), nil
	}
	c.all = msg.res.Items
	c.incomplete = msg.res.Incomplete
	query, ok := c.query(cx)
	if !ok {
		return consumedWith(popLayer), nil
	}
	selected := c.selectedKey()
	c.items = filterCompletionItems(c.all, query)
	if len(c.items) == 0 {
		return consumedWith(popLayer), nil
	}
	if c.manual {
		c.restoreCursor(selected)
	} else {
		c.resetCursor()
	}
	return consumed(), nil
}

func (c *completionComponent) refreshCmd(cx *Context) tea.Cmd {
	if !c.incomplete {
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
	pos := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
	c.refreshGen++
	gen := c.refreshGen
	rev := doc.Revision()
	return func() tea.Msg {
		res, err := ls.Completions(doc, v.ID())
		return completionRefreshMsg{
			layer: c,
			gen:   gen,
			rev:   rev,
			pos:   pos,
			res:   res,
			err:   err,
		}
	}
}

func (c *completionComponent) refreshValid(
	cx *Context, msg completionRefreshMsg,
) bool {
	if !c.valid(cx) {
		return false
	}
	doc, ok := cx.Editor.FocusedDocument()
	if !ok || doc.Revision() != msg.rev {
		return false
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return false
	}
	pos := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
	return pos == msg.pos
}

func (c *completionComponent) query(cx *Context) (string, bool) {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return "", false
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return "", false
	}
	pos := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
	query, err := doc.Text().SliceString(c.anchor.pos, pos)
	if err != nil {
		return "", false
	}
	return query, true
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
		w = max(w, c.rowWidth(item, true)+2)
	}
	if len(c.items) > completionMaxRows {
		w += completionScrollGap
	}
	return w + 2
}

func (c *completionComponent) rowWidth(
	item view.CompletionItem, selected bool,
) int {
	return runewidth.StringWidth(c.rowLeft(item, selected))
}

type renderCompletionRowArgs struct {
	item     view.CompletionItem
	selected bool
	query    string
	base     tui.Style
	match    tui.Style
	icon     tui.Style
	info     tui.Style
}

func (c *completionComponent) renderRow(
	buf *tui.Buffer, x, y, w, listW int, args renderCompletionRowArgs,
) {
	buf.SetString(x, y, clipPad("", w), args.base)
	parts := c.rowParts(args.item, args.selected)
	labelX := x
	budget := listW
	if parts.icon != "" {
		next := writeCompletionPart(
			buf, labelX, y, budget, parts.icon, args.icon,
		)
		budget -= next - labelX
		labelX = next
		if budget > 0 {
			buf.SetString(labelX, y, " ", args.base)
			labelX++
			budget--
		}
	}
	if budget <= 0 {
		return
	}
	writePickerMatched(buf, writePickerMatchedArgs{
		x: labelX, y: y, maxW: budget, text: parts.label,
		indices: completionLabelMatchIndices(parts.label, args.query),
		base:    args.base, match: args.match,
	})
	used := min(runewidth.StringWidth(parts.label), budget)
	labelX += used
	budget -= used
	if parts.info == "" || budget <= 1 {
		return
	}
	buf.SetString(labelX, y, " ", args.base)
	labelX++
	budget--
	writeCompletionPart(buf, labelX, y, budget, parts.info, args.info)
}

func (c *completionComponent) rowLeft(
	item view.CompletionItem, selected bool,
) string {
	return completionRowText(c.rowParts(item, selected))
}

func (c *completionComponent) rowParts(
	item view.CompletionItem, selected bool,
) completionRowParts {
	return completionRowPartsFor(item, c.opts.Icons, selected)
}

func (c *completionComponent) popupPos(
	buf *tui.Buffer, cx *Context,
) (int, int) {
	if cur, ok := c.ec.Cursor(buf.Width, buf.Height, cx); ok {
		return cur.X, cur.Y + 1
	}
	return 0, max(buf.Height-completionMaxRows-2, 0)
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
	c.ensureCursorVisible(c.visibleRows())
}

func (c *completionComponent) restoreCursor(selected string) {
	if selected != "" {
		for i, item := range c.items {
			if completionItemKey(item) == selected {
				c.cursor = i
				c.ensureCursorVisible(c.visibleRows())
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

func (c *completionComponent) clampScroll(rows int) {
	c.scroll = listClampScroll(c.scroll, len(c.items), rows)
}

func (c *completionComponent) scrollBy(delta int) {
	c.scroll = listScrollBy(
		c.scroll, len(c.items), c.visibleRows(), delta,
	)
}

func (c *completionComponent) ensureCursorVisible(rows int) {
	c.scroll = listEnsureCursorVisible(
		c.scroll, c.cursor, len(c.items), rows,
	)
}

func (c *completionComponent) visibleRows() int {
	if c.listBounds.h > 0 {
		return c.listBounds.h
	}
	return completionMaxRows
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

func filterCompletionItems(
	items []view.CompletionItem, query string,
) []view.CompletionItem {
	if query == "" {
		return append([]view.CompletionItem(nil), items...)
	}
	matches := make([]completionMatch, 0, len(items))
	for i, item := range items {
		if score, ok := completionMatchScore(item, query); ok {
			matches = append(matches, completionMatch{
				item:  item,
				score: score,
				order: i,
			})
		}
	}
	slices.SortStableFunc(matches, func(a, b completionMatch) int {
		if a.score > b.score {
			return -1
		}
		if a.score < b.score {
			return 1
		}
		return a.order - b.order
	})
	out := make([]view.CompletionItem, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.item)
	}
	return out
}

func completionMatchScore(item view.CompletionItem, query string) (int, bool) {
	text := item.Filter
	if text == "" {
		text = item.Label
	}
	return fuzzyCompletionScore(text, query)
}

func completionItemKey(item view.CompletionItem) string {
	if item.ID != "" {
		return item.ID
	}
	return item.Label + "\x00" + item.Insert + "\x00" + item.Kind
}

func fuzzyCompletionScore(text, query string) (int, bool) {
	text = strings.TrimLeftFunc(text, unicode.IsSpace)
	if query == "" {
		return 0, true
	}
	if strings.HasPrefix(strings.ToLower(text), strings.ToLower(query)) {
		return 100000 - runewidth.StringWidth(text), true
	}
	rs := []rune(text)
	score := 0
	gaps := 0
	last := -1
	from := 0
	for _, q := range query {
		found := -1
		q = unicode.ToLower(q)
		for i := from; i < len(rs); i++ {
			if unicode.ToLower(rs[i]) == q {
				found = i
				break
			}
		}
		if found < 0 {
			return 0, false
		}
		if last >= 0 {
			gaps += found - last - 1
			if found == last+1 {
				score += 3
			}
		}
		if completionBoundary(rs, found) {
			score += 5
		}
		score += 10
		last = found
		from = found + 1
	}
	score -= gaps * 2
	score -= runewidth.StringWidth(text)
	return score, true
}

func completionBoundary(rs []rune, idx int) bool {
	if idx == 0 {
		return true
	}
	prev := rs[idx-1]
	return !unicode.IsLetter(prev) && !unicode.IsNumber(prev)
}

func completionRowPartsFor(
	item view.CompletionItem, icons CompletionIconMode, selected bool,
) completionRowParts {
	parts := completionRowParts{
		icon:  completionKindMarker(item.Kind, icons),
		label: item.Label,
	}
	if selected {
		parts.label += strings.Join(strings.Fields(item.LabelDetail), " ")
		var info []string
		if detail := completionRowDetail(item); detail != "" {
			info = append(info, detail)
		}
		desc := strings.Join(strings.Fields(item.LabelDescription), " ")
		if desc != "" {
			info = append(info, desc)
		}
		if item.Deprecated {
			info = append(info, "deprecated")
		}
		parts.info = completionPreview(strings.Join(info, " "))
	}
	return parts
}

func completionRowText(parts completionRowParts) string {
	out := parts.label
	if parts.icon != "" {
		out = parts.icon + " " + out
	}
	if parts.info != "" {
		out += " " + parts.info
	}
	return out
}

func completionRowDetail(item view.CompletionItem) string {
	detail := strings.Join(strings.Fields(item.Detail), " ")
	labelDetail := strings.Join(strings.Fields(item.LabelDetail), " ")
	if detail == "" || detail == labelDetail {
		return ""
	}
	return detail
}

func completionPreview(s string) string {
	if s == "" {
		return ""
	}
	return runewidth.Truncate(s, completionPreviewMaxWidth, "...")
}

func writeCompletionPart(
	buf *tui.Buffer, x, y, maxW int, text string, st tui.Style,
) int {
	if maxW <= 0 || text == "" {
		return x
	}
	text = runewidth.Truncate(text, maxW, "")
	buf.SetString(x, y, text, st)
	return x + runewidth.StringWidth(text)
}

func completionLabelMatchIndices(label, query string) []int {
	if query == "" {
		return nil
	}
	rs := []rune(label)
	if strings.HasPrefix(strings.ToLower(label), strings.ToLower(query)) {
		n := min(utf8.RuneCountInString(query), len(rs))
		indices := make([]int, n)
		for i := range n {
			indices[i] = i
		}
		return indices
	}
	indices := make([]int, 0, utf8.RuneCountInString(query))
	from := 0
	for _, q := range query {
		q = unicode.ToLower(q)
		found := -1
		for i := from; i < len(rs); i++ {
			if unicode.ToLower(rs[i]) == q {
				found = i
				break
			}
		}
		if found < 0 {
			return nil
		}
		indices = append(indices, found)
		from = found + 1
	}
	return indices
}

func completionKindMarker(kind string, mode CompletionIconMode) string {
	if kind == "" || mode == CompletionIconsNone {
		return ""
	}
	var icon string
	if mode == CompletionIconsASCII {
		icon = completionKindASCIIIcon(kind)
	} else {
		icon = completionKindCodicon(kind)
	}
	if icon == "" {
		return "?"
	}
	return icon
}

func completionKindCodicon(kind string) string {
	switch kind {
	case "text":
		return ""
	case "function", "method", "constructor":
		return ""
	case "field":
		return ""
	case "variable":
		return ""
	case "class":
		return ""
	case "interface":
		return ""
	case "module":
		return ""
	case "property":
		return ""
	case "unit":
		return ""
	case "value", "enum":
		return ""
	case "keyword":
		return ""
	case "snippet":
		return ""
	case "color":
		return ""
	case "file":
		return ""
	case "reference":
		return ""
	case "folder":
		return ""
	case "constant":
		return ""
	case "struct":
		return ""
	case "event":
		return ""
	case "operator":
		return ""
	case "type_param":
		return ""
	case "enum_member":
		return ""
	default:
		return ""
	}
}

func completionKindASCIIIcon(kind string) string {
	switch kind {
	case "function", "method":
		return "fn"
	case "constructor":
		return "+"
	case "field", "property":
		return "."
	case "variable":
		return "v"
	case "class":
		return "C"
	case "interface":
		return "I"
	case "module":
		return "M"
	case "keyword":
		return "K"
	case "snippet":
		return "S"
	case "file":
		return "F"
	case "folder":
		return "D"
	case "constant":
		return "k"
	case "struct":
		return "S"
	case "enum":
		return "E"
	case "enum_member":
		return "e"
	case "type_param":
		return "T"
	default:
		return ""
	}
}
