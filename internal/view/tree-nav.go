package view

import "slices"

// Next returns the id of the pane after the focused one in DFS order
func (t *Tree) Next() Id {
	var first, next Id
	afterFocus := false
	t.Range(func(p Pane) bool {
		id := p.ID()
		if first == 0 {
			first = id
		}
		if afterFocus {
			next = id
			return false
		}
		afterFocus = id == t.focus
		return true
	})
	if next != 0 {
		return next
	}
	if first != 0 {
		return first
	}
	return t.focus
}

// Prev returns the id of the pane before the focused one in DFS order
func (t *Tree) Prev() Id {
	var prev, last, beforeFocus Id
	foundFocus := false
	t.Range(func(p Pane) bool {
		id := p.ID()
		if id == t.focus {
			foundFocus = true
			beforeFocus = prev
		}
		prev, last = id, id
		return true
	})
	switch {
	case beforeFocus != 0:
		return beforeFocus
	case foundFocus, last != 0:
		return last
	default:
		return t.focus
	}
}

// Transpose flips the layout of the container holding the focused pane
func (t *Tree) Transpose() {
	parent := t.nodes[t.focus].parent
	if c := t.nodes[parent].container; c != nil {
		if c.layout == LayoutVertical {
			c.layout = LayoutHorizontal
		} else {
			c.layout = LayoutVertical
		}
		t.recalculate()
	}
}

// FindSplitInDirection finds the nearest split in the given direction from id,
func (t *Tree) FindSplitInDirection(id Id, dir Direction) (Id, bool) {
	parent := t.nodes[id].parent
	if parent == id {
		return 0, false
	}
	c := t.nodes[parent].container

	switch {
	case (dir == DirectionUp || dir == DirectionDown) &&
		c.layout == LayoutVertical,
		(dir == DirectionLeft || dir == DirectionRight) &&
			c.layout == LayoutHorizontal:
		// direction is perpendicular to container layout — search up
		return t.FindSplitInDirection(parent, dir)

	default:
		// direction is parallel to container layout — search within children
		if child, ok := t.findChild(id, c.children, dir); ok {
			return child, true
		}
		return t.FindSplitInDirection(parent, dir)
	}
}

// SwapSplitInDirection swaps the focused pane with the nearest pane in the
// given direction
func (t *Tree) SwapSplitInDirection(dir Direction) bool {
	target, ok := t.FindSplitInDirection(t.focus, dir)
	if !ok {
		return false
	}
	focus := t.focus
	focusParent := t.nodes[focus].parent
	targetParent := t.nodes[target].parent

	if focusParent == targetParent {
		c := t.nodes[focusParent].container
		fi := slices.Index(c.children, focus)
		ti := slices.Index(c.children, target)
		c.children[fi], c.children[ti] = c.children[ti], c.children[fi]
		swapPaneAreas(t.nodes[focus].pane, t.nodes[target].pane)
	} else {
		fc := t.nodes[focusParent].container
		tc := t.nodes[targetParent].container
		fi := slices.Index(fc.children, focus)
		ti := slices.Index(tc.children, target)
		fc.children[fi], tc.children[ti] = tc.children[ti], fc.children[fi]
		t.nodes[focus].parent = targetParent
		t.nodes[target].parent = focusParent
		swapPaneAreas(t.nodes[focus].pane, t.nodes[target].pane)
	}
	return true
}

func swapPaneAreas(a, b Pane) {
	aArea, bArea := a.Area(), b.Area()
	a.SetArea(bArea)
	b.SetArea(aArea)
}
