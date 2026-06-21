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
func (t *Tree) Focus() Id {
	return t.focus
}

// SetFocus moves focus to the given view id
func (t *Tree) SetFocus(id Id) {
	t.focus = id
}

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

// Traverse returns all view nodes in DFS order (left-to-right, top-to-bottom)
func (t *Tree) Traverse() []*View {
	return t.traverse(t.root, nil)
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

// NodeID returns the ViewId of the treeNode that holds the given view id,
// which is the same as the view id for leaf nodes
func (t *Tree) NodeID(viewID Id) Id {
	return viewID
}

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
