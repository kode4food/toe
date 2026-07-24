package view

// FocusedView returns the currently focused view
func (e *Editor) FocusedView() (*View, bool) {
	v, ok := e.panes.tree.Get(e.panes.tree.Focus()).(*View)
	return v, ok
}

// FocusedPane returns the currently focused pane
func (e *Editor) FocusedPane() Pane {
	return e.panes.tree.Get(e.panes.tree.Focus())
}

// FocusView moves focus to the given view
func (e *Editor) FocusView(vid Id) {
	if _, ok := e.panes.tree.Get(vid).(*View); ok {
		e.FocusPane(vid)
	}
}

// FocusPane moves focus to the given pane
func (e *Editor) FocusPane(id Id) {
	if e.panes.tree.Get(id) == nil {
		return
	}
	e.recordLeavingDoc()
	e.panes.tree.SetFocus(id)
	e.markDocAccessed()
}

// FocusNextView moves focus to the next view in DFS order
func (e *Editor) FocusNextView() {
	e.recordLeavingDoc()
	e.panes.tree.SetFocus(e.panes.tree.Next())
	e.markDocAccessed()
}

// FocusPrevView moves focus to the previous view in DFS order
func (e *Editor) FocusPrevView() {
	e.recordLeavingDoc()
	e.panes.tree.SetFocus(e.panes.tree.Prev())
	e.markDocAccessed()
}

// FocusDirection moves focus to the nearest split in the given direction
func (e *Editor) FocusDirection(dir Direction) {
	if id, ok := e.panes.tree.FindSplitInDirection(
		e.panes.tree.Focus(), dir,
	); ok {
		e.recordLeavingDoc()
		e.panes.tree.SetFocus(id)
		e.markDocAccessed()
	}
}

// SwapSplitInDirection swaps focus with the nearest split in the direction
func (e *Editor) SwapSplitInDirection(dir Direction) {
	e.panes.tree.SwapSplitInDirection(dir)
}

// Transpose flips the layout of the container holding the focused view
func (e *Editor) Transpose() {
	e.panes.tree.Transpose()
}

// ResizeFocusedSplit pushes the border on the given side of the focused split
// by delta cells, screen-direction style (see [Tree.ResizeFocused])
func (e *Editor) ResizeFocusedSplit(dir Direction, delta int) {
	e.panes.tree.ResizeFocused(dir, delta)
}

// hasView reports whether any view in the tree satisfies pred
func (e *Editor) hasView(pred func(*View) bool) bool {
	return e.panes.tree.Any(func(p Pane) bool {
		v, ok := p.(*View)
		return ok && pred(v)
	})
}
