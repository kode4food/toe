package ui

import tea "charm.land/bubbletea/v2"

func (c *completionComponent) refresh(cx *Context, comp *Compositor) tea.Cmd {
	c.markDirty()
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
	cx *Context, msg completionRefreshMsg,
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
	c.markDirty()
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
