package ui

func (c *completionComponent) moveBy(n int) {
	c.cursor = (c.cursor + n + len(c.items)) % len(c.items)
	c.manual = true
	c.ensureCursorVisible(c.visibleRows())
	c.markDirty()
}

func (c *completionComponent) moveTo(idx int) {
	c.cursor = min(max(idx, 0), len(c.items)-1)
	c.manual = true
	c.ensureCursorVisible(c.visibleRows())
	c.markDirty()
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

func (c *completionComponent) restoreCursor(selected completionItemKey) {
	if selected != (completionItemKey{}) {
		for i, item := range c.items {
			if keyOfCompletionItem(item) == selected {
				c.cursor = i
				c.ensureCursorVisible(c.visibleRows())
				return
			}
		}
	}
	c.resetCursor()
}

func (c *completionComponent) selectedKey() completionItemKey {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return completionItemKey{}
	}
	return keyOfCompletionItem(c.items[c.cursor])
}

func (c *completionComponent) clampScroll(rows int) {
	c.scroll = listClampScroll(c.scroll, len(c.items), rows)
}

func (c *completionComponent) scrollBy(delta int) {
	c.markDirty()
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
	if c.listBounds.Height > 0 {
		return c.listBounds.Height
	}
	return completionMaxRows
}
