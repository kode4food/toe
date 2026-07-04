package view

type (
	// Separator describes the position and extent of the gap between two
	// adjacent split panes in a container
	Separator struct {
		Layout Layout
		X, Y   int
		W, H   int
	}

	sepVisitor func(cID Id, childIdx int, sep Separator) bool
)

// WalkSeparators calls fn for each separator between adjacent panes. Vertical
// seps have W=1 and span the container height; horizontal seps have H=1 and
// span the container width
func (t *Tree) WalkSeparators(fn func(Separator)) {
	if !t.IsEmpty() {
		t.walkSep(t.root, fn)
	}
}

func (t *Tree) findChild(id Id, children []Id, dir Direction) (Id, bool) {
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
				if d := abs(t.leftOf(cc) - curX); d < bestDist {
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
				if d := abs(t.topOf(cc) - curY); d < bestDist {
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

// SeparatorAt returns the container ID, left-child index, and layout of the
// separator hit by the click at tree column x, tree row y (bufferline excluded)
func (t *Tree) SeparatorAt(
	x, y int,
) (containerID Id, childIdx int, layout Layout, ok bool) {
	if t.IsEmpty() {
		return
	}
	t.walkSepWithID(t.root,
		func(cID Id, idx int, s Separator) bool {
			if x >= s.X && x < s.X+s.W && y >= s.Y && y < s.Y+s.H {
				containerID, childIdx, layout, ok = cID, idx, s.Layout, true
				return false
			}
			return true
		},
	)
	return
}

func (t *Tree) walkSepWithID(id Id, fn sepVisitor) bool {
	n := t.nodes[id]
	if n.view != nil {
		return true
	}
	c := n.container
	for i, child := range c.children {
		if !t.walkSepWithID(child, fn) {
			return false
		}
		if i >= len(c.children)-1 {
			continue
		}
		cn := t.nodes[child]
		var a Area
		if cn.view != nil {
			a = cn.view.Area()
		} else {
			a = cn.container.area
		}
		switch c.layout {
		case LayoutVertical:
			// 1-column gap between panes; separator is that gap column
			s := Separator{
				Layout: LayoutVertical,
				X:      a.X + a.Width,
				Y:      c.area.Y,
				W:      1,
				H:      c.area.Height,
			}
			if !fn(id, i, s) {
				return false
			}
		case LayoutHorizontal:
			// 1-row gap after child[i]; separator is that gap row
			s := Separator{
				Layout: LayoutHorizontal,
				X:      c.area.X,
				Y:      a.Y + a.Height,
				W:      c.area.Width,
				H:      1,
			}
			if !fn(id, i, s) {
				return false
			}
		}
	}
	return true
}

func (t *Tree) walkSep(id Id, fn func(Separator)) {
	t.walkSepWithID(id, func(_ Id, _ int, s Separator) bool {
		fn(s)
		return true
	})
}

func abs(n int) int {
	return max(-n, n)
}
