package view

import "slices"

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
		ratios   []float64
	}

	treeNode struct {
		parent    Id
		view      *View
		container *treeContainer
	}

	// Layout describes how child views are arranged within a split container
	Layout int

	// Direction is used to navigate between splits
	Direction int
)

const (
	// LayoutVertical places splits side by side
	LayoutVertical Layout = iota
	// LayoutHorizontal stacks splits one above the other
	LayoutHorizontal
)

const (
	DirectionUp Direction = iota
	DirectionDown
	DirectionLeft
	DirectionRight
)

const (
	minPaneWidth  = 16
	minPaneHeight = 4
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
		pos := slices.Index(c.children, focus)
		c.children = slices.Insert(c.children, pos+1, id)
	}
	c.ratios = nil
	t.focus = id
	t.recalculate()
	return id
}

// CanSplit reports whether there is enough room to split the focused pane in
// the given layout while keeping all resulting panes at or above the min size
func (t *Tree) CanSplit(layout Layout) bool {
	if t.IsEmpty() {
		return true
	}
	focus := t.focus
	c := t.nodes[t.nodes[focus].parent].container
	if c.layout == layout {
		// one sibling is added to the existing container; gains one more gap
		ln := len(c.children)
		switch layout {
		case LayoutVertical:
			return max(c.area.Width-ln, 0)/(ln+1) >= minPaneWidth
		case LayoutHorizontal:
			return max(c.area.Height-ln, 0)/(ln+1) >= minPaneHeight
		}
	}
	// focus is wrapped in a new 2-child sub-container with one gap
	a := t.nodes[focus].view.area
	switch layout {
	case LayoutVertical:
		return a.Width >= 2*minPaneWidth+1
	case LayoutHorizontal:
		return a.Height >= 2*minPaneHeight+1
	}
	return false
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
		pos := slices.Index(parentC.children, focus)
		parentC.children = slices.Insert(parentC.children, pos+1, id)
		t.nodes[id].parent = parent
		parentC.ratios = nil
	} else {
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

		pos := slices.Index(parentC.children, focus)
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
	if id == t.focus {
		return
	}
	if old, ok := t.nodes[t.focus]; ok {
		old.view.MarkDirty()
	}
	if n, ok := t.nodes[id]; ok {
		n.view.MarkDirty()
	}
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
	pos := slices.Index(c.children, child)
	if replacement == 0 {
		c.children = append(c.children[:pos], c.children[pos+1:]...)
		c.ratios = nil
	} else {
		c.children[pos] = replacement
		t.nodes[replacement].parent = parent
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
