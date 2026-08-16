package ui

func (c *completionComponent) moveBy(n int) {
	c.syncList()
	c.list.moveBy(n)
	c.manual = true
	c.markDirty()
}

func (c *completionComponent) moveTo(idx int) {
	c.syncList()
	c.list.moveTo(idx)
	c.manual = true
	c.markDirty()
}

func (c *completionComponent) resetCursor() {
	cursor := 0
	for i, item := range c.items {
		if item.Preselect {
			cursor = i
			break
		}
	}
	c.list.scroll = 0
	c.syncList()
	c.list.moveTo(cursor)
}

func (c *completionComponent) restoreCursor(selected completionItemKey) {
	if selected != (completionItemKey{}) {
		for i := range c.items {
			if keyOfCompletionItem(c.items[i]) == selected {
				c.syncList()
				c.list.moveTo(i)
				return
			}
		}
	}
	c.resetCursor()
}

func (c *completionComponent) selectedKey() completionItemKey {
	if c.list.cursor < 0 || c.list.cursor >= len(c.items) {
		return completionItemKey{}
	}
	return keyOfCompletionItem(c.items[c.list.cursor])
}

func (c *completionComponent) syncList() {
	c.list.resize(
		len(c.items), visibleRows(c.listBounds, completionMaxRows),
	)
}
