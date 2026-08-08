package view

import (
	"cmp"
	"slices"

	"github.com/kode4food/toe/internal/geom"
)

type (
	// Tree manages the spatial layout of views as a split tree
	Tree struct {
		root      Id
		focus     Id
		maximized Id
		area      geom.Area
		nodes     map[Id]*treeNode
		nextID    Id
		focusSeq  int
		redraw    func()
	}

	// Pane is the interface implemented by every split tree leaf
	Pane interface {
		ID() Id
		Path() string
		SetID(Id)
		Area() geom.Area
		SetArea(geom.Area)
		MarkDirty()
		Mode() Mode
		SaveSession(*SessionWriter)
		Split() (Pane, error)
		Close()
		Discard()
		Shutdown()
	}

	// AsyncRenderer marks a pane that mutates outside the event loop; the tree
	// hands it a redraw hook on insertion so it can wake the render loop
	AsyncRenderer interface {
		SetRedraw(func())
	}

	// Displaceable marks a pane that frees heavy resources while stashed behind
	// another pane and reacquires them when reverted back into view
	Displaceable interface {
		OnDisplace()
		OnRevert()
	}

	treeContainer struct {
		layout   Layout
		children []Id
		area     geom.Area
		ratios   []float64
	}

	treeNode struct {
		parent    Id
		pane      Pane
		history   []Pane
		container *treeContainer
		focusSeq  int
	}

	// Layout describes how child panes are arranged within a split container
	Layout bool

	// Direction is used to navigate between splits
	Direction int
)

const (
	// LayoutVertical places splits side by side
	LayoutVertical Layout = false
	// LayoutHorizontal stacks splits one above the other
	LayoutHorizontal Layout = true
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

// maxPaneHistory caps how many displaced panes a slot remembers, so a long
// chain of replacements can't pin documents in memory indefinitely
const maxPaneHistory = 10

func newTree(size geom.Size) *Tree {
	t := &Tree{
		nodes: map[Id]*treeNode{},
	}
	t.area = geom.Area{Size: size}
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

// SetRedraw installs the hook the tree hands to [AsyncRenderer] panes on
// insertion, wiring any that are already present
func (t *Tree) SetRedraw(fn func()) {
	t.redraw = fn
	for _, n := range t.nodes {
		if ar, ok := n.pane.(AsyncRenderer); ok {
			ar.SetRedraw(fn)
		}
	}
}

// Redraw wakes the renderer after asynchronous editor state changes
func (t *Tree) Redraw() {
	if t.redraw != nil {
		t.redraw()
	}
}

// Insert adds a pane as the next sibling after the currently focused pane
func (t *Tree) Insert(p Pane) Id {
	focus := t.focus
	parent := t.nodes[focus].parent

	id := t.allocID()
	t.attach(p, id)
	t.nodes[id] = &treeNode{parent: parent, pane: p}

	c := t.nodes[parent].container
	if len(c.children) == 0 {
		c.children = []Id{id}
	} else {
		pos := slices.Index(c.children, focus)
		c.children = slices.Insert(c.children, pos+1, id)
	}
	c.ratios = nil
	t.setFocus(id)
	t.recalculate()
	return id
}

// ReplacePane swaps the pane at id for p, keeping its tree position and area
func (t *Tree) ReplacePane(id Id, p Pane) {
	n, ok := t.nodes[id]
	if !ok || n.pane == nil {
		return
	}
	t.attach(p, id)
	p.SetArea(n.pane.Area())
	n.pane = p
	if t.focus == id {
		p.MarkDirty()
	}
}

// DisplacePane swaps the pane at id for p, stashing the displaced pane on the
// node so RevertPane can bring it back when p closes. The oldest entry is
// discarded once the stack exceeds maxPaneHistory
func (t *Tree) DisplacePane(id Id, p Pane) {
	n, ok := t.nodes[id]
	if !ok || n.pane == nil {
		return
	}
	displaced := n.pane
	t.ReplacePane(id, p)
	stashPane(displaced)
	n.history = append(n.history, displaced)
	if len(n.history) > maxPaneHistory {
		n.history[0].Discard()
		n.history = slices.Delete(n.history, 0, 1)
	}
}

// RevertPane restores the most recently displaced pane at id, reporting whether
// one was available
func (t *Tree) RevertPane(id Id) bool {
	n, ok := t.nodes[id]
	if !ok || len(n.history) == 0 {
		return false
	}
	last := len(n.history) - 1
	prev := n.history[last]
	n.history = n.history[:last]
	if d, ok := prev.(Displaceable); ok {
		d.OnRevert()
	}
	t.ReplacePane(id, prev)
	t.recalculate()
	return true
}

// DiscardHistory discards every pane stashed behind id, for when the slot is
// vacated without reverting
func (t *Tree) DiscardHistory(id Id) {
	n, ok := t.nodes[id]
	if !ok {
		return
	}
	for _, p := range n.history {
		p.Discard()
	}
	n.history = nil
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
	a := t.nodes[focus].pane.Area()
	switch layout {
	case LayoutVertical:
		return a.Width >= 2*minPaneWidth+1
	case LayoutHorizontal:
		return a.Height >= 2*minPaneHeight+1
	}
	return false
}

// Split creates a new pane alongside the focused pane using the given layout.
// If the focused pane's parent container already uses the same layout, the
// new pane is added as a sibling. Otherwise a new sub-container is created
func (t *Tree) Split(p Pane, layout Layout) Id {
	focus := t.focus
	parent := t.nodes[focus].parent

	id := t.allocID()
	t.attach(p, id)
	t.nodes[id] = &treeNode{pane: p}

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

	t.setFocus(id)
	t.recalculate()
	return id
}

// Remove removes a view from the tree. Focus is moved to the previous view
// before removal. Empty containers are collapsed
func (t *Tree) Remove(id Id) {
	if t.focus == id {
		t.setFocus(t.Prev())
	}

	parent := t.nodes[id].parent
	parentIsRoot := parent == t.root

	t.removeOrReplace(removeOrReplaceArgs{child: id})

	c := t.nodes[parent].container
	if len(c.children) == 1 && !parentIsRoot {
		sibling := c.children[0]
		c.children = nil
		t.removeOrReplace(removeOrReplaceArgs{
			child:       parent,
			replacement: sibling,
		})
	}

	if t.Count() < 2 {
		t.maximized = 0
	}
	t.recalculate()
}

// Get returns the pane at id, or nil if id is not a leaf node
func (t *Tree) Get(id Id) Pane {
	if n, ok := t.nodes[id]; ok {
		return n.pane
	}
	return nil
}

// Focus returns the currently focused view id
func (t *Tree) Focus() Id {
	return t.focus
}

// SetFocus moves focus to the given view id
func (t *Tree) SetFocus(id Id) {
	maximized := t.Maximized()
	if !t.setFocus(id) {
		return
	}
	if maximized {
		t.recalculate()
	}
}

// Maximized reports whether one pane temporarily occupies the full tree area
func (t *Tree) Maximized() bool {
	return t.maximized != 0
}

// ToggleMaximized maximizes the focused pane or restores the split layout
func (t *Tree) ToggleMaximized() {
	if t.maximized != 0 {
		t.Unmaximize()
		return
	}
	if t.Count() > 1 {
		t.maximized = t.focus
	}
	t.recalculate()
}

// Unmaximize restores the preserved split layout
func (t *Tree) Unmaximize() {
	if t.maximized == 0 {
		return
	}
	t.maximized = 0
	t.recalculate()
}

// IsEmpty reports whether the tree has no views
func (t *Tree) IsEmpty() bool {
	return len(t.nodes[t.root].container.children) == 0
}

// Resize updates the total area and recalculates view areas. Returns true if
// the area changed
func (t *Tree) Resize(size geom.Size) bool {
	a := geom.Area{Size: size}
	if t.area == a {
		return false
	}
	t.area = a
	t.recalculate()
	return true
}

// Range calls fn for each leaf pane in DFS order (left-to-right,
// top-to-bottom), stopping early if fn returns false. It does not allocate
func (t *Tree) Range(fn func(Pane) bool) {
	t.rangePane(t.root, fn)
}

// RangeVisible calls fn for each pane currently visible in the layout
func (t *Tree) RangeVisible(fn func(Pane) bool) {
	if t.maximized != 0 {
		fn(t.nodes[t.maximized].pane)
		return
	}
	t.Range(fn)
}

// Any reports whether any leaf pane satisfies pred, without allocating
func (t *Tree) Any(pred func(Pane) bool) bool {
	found := false
	t.Range(func(p Pane) bool {
		found = pred(p)
		return !found
	})
	return found
}

// Count returns the number of leaf panes, without allocating
func (t *Tree) Count() int {
	n := 0
	t.Range(func(Pane) bool {
		n++
		return true
	})
	return n
}

// Traverse returns all leaf panes in DFS order (left-to-right, top-to-bottom).
// Prefer [Tree.Range] when a slice isn't actually needed
func (t *Tree) Traverse() []Pane {
	out := make([]Pane, 0, t.Count())
	t.Range(func(p Pane) bool {
		out = append(out, p)
		return true
	})
	return out
}

// ContainerLayoutAt returns the layout of the container that holds id
func (t *Tree) ContainerLayoutAt(id Id) (Layout, bool) {
	n, ok := t.nodes[id]
	if !ok {
		return LayoutVertical, false
	}
	parent := n.parent
	pn, ok := t.nodes[parent]
	if !ok || pn.container == nil {
		return LayoutVertical, false
	}
	return pn.container.layout, true
}

func (t *Tree) setFocus(id Id) bool {
	if id == t.focus {
		return false
	}
	if old, ok := t.nodes[t.focus]; ok && old.pane != nil {
		old.pane.MarkDirty()
	}
	if n, ok := t.nodes[id]; ok && n.pane != nil {
		n.pane.MarkDirty()
	}
	t.focus = id
	t.maximized = 0
	t.advanceFocusSeq(id)
	return true
}

func (t *Tree) advanceFocusSeq(id Id) {
	t.focusSeq++
	for {
		n, ok := t.nodes[id]
		if !ok {
			return
		}
		n.focusSeq = t.focusSeq
		if id == t.root {
			return
		}
		id = n.parent
	}
}

func (t *Tree) leafWithLatestFocus() (Id, bool) {
	var best Id
	var seq int
	for id, n := range t.nodes {
		if n.pane == nil {
			continue
		}
		// only a corrupt session ties; lowest id keeps the pick deterministic
		tie := n.focusSeq == seq && (best == 0 || id < best)
		if n.focusSeq > seq || tie {
			best, seq = id, n.focusSeq
		}
	}
	return best, seq > 0
}

func (t *Tree) compactFocusSeq() {
	ids := make([]Id, 0, len(t.nodes))
	for id := range t.nodes {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b Id) int {
		if c := cmp.Compare(t.nodes[a].focusSeq, t.nodes[b].focusSeq); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})
	for i, id := range ids {
		t.nodes[id].focusSeq = i + 1
	}
	t.focusSeq = len(ids)
}

func (t *Tree) allocID() Id {
	t.nextID++
	return t.nextID
}

type removeOrReplaceArgs struct {
	child       Id
	replacement Id
}

func (t *Tree) removeOrReplace(args removeOrReplaceArgs) {
	child := args.child
	replacement := args.replacement
	for _, p := range t.nodes[child].history {
		p.Discard()
	}
	parent := t.nodes[child].parent
	delete(t.nodes, child)

	c := t.nodes[parent].container
	pos := slices.Index(c.children, child)
	if replacement == 0 {
		c.children = slices.Delete(c.children, pos, pos+1)
		c.ratios = nil
	} else {
		c.children[pos] = replacement
		t.nodes[replacement].parent = parent
	}
}

// rangePane visits id in DFS order, returning false as soon as fn does, so
// the caller stops walking sibling subtrees too
func (t *Tree) rangePane(id Id, fn func(Pane) bool) bool {
	n := t.nodes[id]
	if n.pane != nil {
		return fn(n.pane)
	}
	for _, child := range n.container.children {
		if !t.rangePane(child, fn) {
			return false
		}
	}
	return true
}

// wires the pane id and, for async panes, its redraw hook on insertion
func (t *Tree) attach(p Pane, id Id) {
	p.SetID(id)
	if t.redraw == nil {
		return
	}
	if ar, ok := p.(AsyncRenderer); ok {
		ar.SetRedraw(t.redraw)
	}
}

// stashPane lets a pane release heavy resources once it is hidden in history
func stashPane(p Pane) {
	if d, ok := p.(Displaceable); ok {
		d.OnDisplace()
	}
}
