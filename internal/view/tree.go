package view

type (
	// Tree manages the spatial layout of views as a split tree
	Tree struct {
		root   Id
		focus  Id
		area   Area
		nodes  map[Id]*treeNode
		nextID Id
	}

	treeContainer struct {
		layout   Layout
		children []Id
		area     Area
	}

	treeNode struct {
		parent    Id
		view      *View
		container *treeContainer
	}

	treeWork struct {
		id   Id
		area Area
	}
)

func newTree(width, height int) *Tree {
	t := &Tree{
		nodes: map[Id]*treeNode{},
	}
	t.area = Area{Width: width, Height: height}
	// root is always a container node
	t.nextID++
	rootID := t.nextID
	t.nodes[rootID] = &treeNode{
		container: &treeContainer{layout: LayoutVertical},
	}
	t.nodes[rootID].parent = rootID
	t.root = rootID
	t.focus = rootID
	return t
}

func (t *Tree) allocID() Id {
	t.nextID++
	return t.nextID
}

// Insert adds a view as the next sibling after the currently focused view
func (t *Tree) Insert(v *View) Id {
	focus := t.focus
	parent := t.nodes[focus].parent

	id := t.allocID()
	v.id = id
	t.nodes[id] = &treeNode{parent: parent, view: v}

	c := t.nodes[parent].container
	if len(c.children) == 0 {
		c.children = []Id{id}
	} else {
		pos := indexInSlice(c.children, focus)
		c.children = insert(c.children, pos+1, id)
	}
	t.focus = id
	t.recalculate()
	return id
}

// Split creates a new view alongside the focused view using the given layout.
// If the focused view's parent container already uses the same layout, the new
// view is added as a sibling. Otherwise a new sub-container is created
func (t *Tree) Split(v *View, layout Layout) Id {
	focus := t.focus
	parent := t.nodes[focus].parent

	id := t.allocID()
	v.id = id
	t.nodes[id] = &treeNode{view: v}

	parentC := t.nodes[parent].container
	if parentC.layout == layout {
		pos := indexInSlice(parentC.children, focus)
		parentC.children = insert(parentC.children, pos+1, id)
		t.nodes[id].parent = parent
	} else {
		// wrap focus and new view in a new sub-container
		subID := t.allocID()
		t.nodes[subID] = &treeNode{
			parent: parent,
			container: &treeContainer{
				layout:   layout,
				children: []Id{focus, id},
			},
		}
		t.nodes[focus].parent = subID
		t.nodes[id].parent = subID

		pos := indexInSlice(parentC.children, focus)
		parentC.children[pos] = subID
	}

	t.focus = id
	t.recalculate()
	return id
}

func (t *Tree) removeOrReplace(child Id, replacement Id) {
	parent := t.nodes[child].parent
	delete(t.nodes, child)

	c := t.nodes[parent].container
	pos := indexInSlice(c.children, child)
	if replacement == 0 {
		c.children = append(c.children[:pos], c.children[pos+1:]...)
	} else {
		c.children[pos] = replacement
		t.nodes[replacement].parent = parent
	}
}

// Remove removes a view from the tree. Focus is moved to the previous view
// before removal. Empty containers are collapsed
func (t *Tree) Remove(id Id) {
	if t.focus == id {
		t.focus = t.Prev()
	}

	parent := t.nodes[id].parent
	parentIsRoot := parent == t.root

	t.removeOrReplace(id, 0)

	c := t.nodes[parent].container
	if len(c.children) == 1 && !parentIsRoot {
		sibling := c.children[0]
		c.children = nil
		t.removeOrReplace(parent, sibling)
	}

	t.recalculate()
}

// Get returns the view with the given id
func (t *Tree) Get(id Id) *View {
	if n, ok := t.nodes[id]; ok && n.view != nil {
		return n.view
	}
	return nil
}

// Focus returns the currently focused view id
func (t *Tree) Focus() Id { return t.focus }

// SetFocus moves focus to the given view id
func (t *Tree) SetFocus(id Id) { t.focus = id }

// IsEmpty reports whether the tree has no views
func (t *Tree) IsEmpty() bool {
	return len(t.nodes[t.root].container.children) == 0
}

// Resize updates the total area and recalculates view areas. Returns true if
// the area changed
func (t *Tree) Resize(width, height int) bool {
	a := Area{Width: width, Height: height}
	if t.area == a {
		return false
	}
	t.area = a
	t.recalculate()
	return true
}

// recalculate distributes the tree area to all view nodes using the same
func (t *Tree) recalculate() {
	if t.IsEmpty() {
		t.focus = t.root
		return
	}

	stack := []treeWork{{t.root, t.area}}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		n := t.nodes[item.id]
		if n.view != nil {
			n.view.SetArea(item.area)
			continue
		}

		c := n.container
		c.area = item.area
		a := item.area
		switch c.layout {
		case LayoutHorizontal:
			// children stacked (hsplit): distribute height evenly
			ln := len(c.children)
			h := a.Height / ln
			childY := a.Y
			for i, child := range c.children {
				childH := h
				if i == ln-1 {
					childH = a.Y + a.Height - childY
				}
				area := Area{a.X, childY, a.Width, childH}
				stack = append(stack, treeWork{child, area})
				childY += h
			}
		case LayoutVertical:
			// children side by side (vsplit): distribute width evenly, 1px gap
			ln := len(c.children)
			lnu := ln
			innerGap := 1
			totalGap := innerGap * max(lnu-2, 0)
			usedArea := max(a.Width-totalGap, 0)
			w := usedArea / lnu
			childX := a.X
			for i, child := range c.children {
				childW := w
				if i == ln-1 {
					childW = a.X + a.Width - childX
				}
				area := Area{childX, a.Y, childW, a.Height}
				stack = append(stack, treeWork{child, area})
				childX += w + innerGap
			}
		}
	}
}

// Traverse returns all view nodes in DFS order (left-to-right, top-to-bottom)
func (t *Tree) Traverse() []*View {
	return t.traverse(t.root, nil)
}

func (t *Tree) traverse(id Id, out []*View) []*View {
	n := t.nodes[id]
	if n.view != nil {
		return append(out, n.view)
	}
	for _, child := range n.container.children {
		out = t.traverse(child, out)
	}
	return out
}

// Views returns all views in DFS order with a focused flag
func (t *Tree) Views() []struct {
	View    *View
	Focused bool
} {
	all := t.Traverse()
	out := make([]struct {
		View    *View
		Focused bool
	}, len(all))
	for i, v := range all {
		out[i] = struct {
			View    *View
			Focused bool
		}{v, v.id == t.focus}
	}
	return out
}

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

func (t *Tree) findChild(
	id Id, children []Id, dir Direction,
) (Id, bool) {
	var childID Id
	switch dir {
	case DirectionUp, DirectionLeft:
		for i := len(children) - 1; i >= 0; i-- {
			if children[i] == id && i > 0 {
				childID = children[i-1]
				goto found
			}
		}
		return 0, false
	default:
		for i, c := range children {
			if c == id && i < len(children)-1 {
				childID = children[i+1]
				goto found
			}
		}
		return 0, false
	}

found:
	focusedView := t.nodes[t.focus].view
	var curX, curY int
	if focusedView != nil {
		curX = focusedView.area.X
		curY = focusedView.area.Y
	}

	for t.nodes[childID].view == nil {
		c := t.nodes[childID].container
		if c.layout == LayoutVertical {
			// find closest by X
			best := c.children[0]
			bestDist := abs(t.leftOf(best) - curX)
			for _, cc := range c.children[1:] {
				d := abs(t.leftOf(cc) - curX)
				if d < bestDist {
					bestDist = d
					best = cc
				}
			}
			childID = best
		} else {
			// find closest by Y
			best := c.children[0]
			bestDist := abs(t.topOf(best) - curY)
			for _, cc := range c.children[1:] {
				d := abs(t.topOf(cc) - curY)
				if d < bestDist {
					bestDist = d
					best = cc
				}
			}
			childID = best
		}
	}
	return childID, true
}

func (t *Tree) leftOf(id Id) int {
	n := t.nodes[id]
	if n.view != nil {
		return n.view.area.X
	}
	// container: area is not tracked separately; use first child
	if len(n.container.children) > 0 {
		return t.leftOf(n.container.children[0])
	}
	return 0
}

func (t *Tree) topOf(id Id) int {
	n := t.nodes[id]
	if n.view != nil {
		return n.view.area.Y
	}
	if len(n.container.children) > 0 {
		return t.topOf(n.container.children[0])
	}
	return 0
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
		fi := indexInSlice(c.children, focus)
		ti := indexInSlice(c.children, target)
		c.children[fi], c.children[ti] = c.children[ti], c.children[fi]
		// swap areas
		fv := t.nodes[focus].view
		tv := t.nodes[target].view
		fv.area, tv.area = tv.area, fv.area
	} else {
		fc := t.nodes[focusParent].container
		tc := t.nodes[targetParent].container
		fi := indexInSlice(fc.children, focus)
		ti := indexInSlice(tc.children, target)
		fc.children[fi], tc.children[ti] = tc.children[ti], fc.children[fi]
		t.nodes[focus].parent = targetParent
		t.nodes[target].parent = focusParent
		// swap areas
		fv := t.nodes[focus].view
		tv := t.nodes[target].view
		fv.area, tv.area = tv.area, fv.area
	}
	return true
}

// NodeID returns the ViewId of the treeNode that holds the given view id,
// which is the same as the view id for leaf nodes
func (t *Tree) NodeID(viewID Id) Id { return viewID }

// ContainerLayoutAt returns the layout of the container that holds viewID
func (t *Tree) ContainerLayoutAt(viewID Id) (Layout, bool) {
	n, ok := t.nodes[viewID]
	if !ok {
		return 0, false
	}
	parent := n.parent
	pn, ok := t.nodes[parent]
	if !ok || pn.container == nil {
		return 0, false
	}
	return pn.container.layout, true
}

// WalkSeparators calls fn for each vertical separator column in the layout. x
// is the buffer column, y is the top row in tree coordinates, height is the
// number of rows
func (t *Tree) WalkSeparators(fn func(x, y, height int)) {
	if !t.IsEmpty() {
		t.walkSep(t.root, fn)
	}
}

func (t *Tree) walkSep(id Id, fn func(x, y, height int)) {
	n := t.nodes[id]
	if n.view != nil {
		return
	}
	c := n.container
	for i, child := range c.children {
		t.walkSep(child, fn)
		if i < len(c.children)-1 && c.layout == LayoutVertical {
			cn := t.nodes[child]
			var a Area
			if cn.view != nil {
				a = cn.view.Area()
			} else {
				a = cn.container.area
			}
			fn(a.X+a.Width, c.area.Y, c.area.Height)
		}
	}
}

func indexInSlice(s []Id, id Id) int {
	for i, v := range s {
		if v == id {
			return i
		}
	}
	return -1
}

func insert(s []Id, pos int, id Id) []Id {
	s = append(s, 0)
	copy(s[pos+1:], s[pos:])
	s[pos] = id
	return s
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
