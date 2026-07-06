package view

import "slices"

// Next returns the id of the view after the focused one in DFS order
func (t *Tree) Next() Id {
	views := t.Traverse()
	for i, v := range views {
		if v.id == t.focus {
			return views[(i+1)%len(views)].id
		}
	}
	if len(views) > 0 {
		return views[0].id
	}
	return t.focus
}

// Prev returns the id of the view before the focused one in DFS order
func (t *Tree) Prev() Id {
	views := t.Traverse()
	for i, v := range views {
		if v.id == t.focus {
			return views[(i+len(views)-1)%len(views)].id
		}
	}
	if len(views) > 0 {
		return views[len(views)-1].id
	}
	return t.focus
}

// Transpose flips the layout of the container holding the focused view
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

// SwapSplitInDirection swaps the focused view with the nearest view in the
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
		fv := t.nodes[focus].view
		tv := t.nodes[target].view
		fv.area, tv.area = tv.area, fv.area
	} else {
		fc := t.nodes[focusParent].container
		tc := t.nodes[targetParent].container
		fi := slices.Index(fc.children, focus)
		ti := slices.Index(tc.children, target)
		fc.children[fi], tc.children[ti] = tc.children[ti], fc.children[fi]
		t.nodes[focus].parent = targetParent
		t.nodes[target].parent = focusParent
		fv := t.nodes[focus].view
		tv := t.nodes[target].view
		fv.area, tv.area = tv.area, fv.area
	}
	return true
}
