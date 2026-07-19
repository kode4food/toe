package view

import "github.com/kode4food/toe/internal/geom"

type (
	// Separator describes the position and extent of the gap between two
	// adjacent split panes in a container
	Separator struct {
		Layout Layout
		geom.Area
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
	focusedPane := t.nodes[t.focus].pane
	var curX, curY int
	if focusedPane != nil {
		a := focusedPane.Area()
		curX = a.X
		curY = a.Y
	}

	for t.nodes[childID].pane == nil {
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
	if n.pane != nil {
		return n.pane.Area().X
	}
	// container: area is not tracked separately; use first child
	if len(n.container.children) > 0 {
		return t.leftOf(n.container.children[0])
	}
	return 0
}

func (t *Tree) topOf(id Id) int {
	n := t.nodes[id]
	if n.pane != nil {
		return n.pane.Area().Y
	}
	if len(n.container.children) > 0 {
		return t.topOf(n.container.children[0])
	}
	return 0
}

type SeparatorAtRes struct {
	ContainerID Id
	ChildIdx    int
	Layout      Layout
}

// SeparatorAt returns the container ID, left-child index, and layout of the
// separator hit by the click at tree column x, tree row y (bufferline excluded)
// SeparatorAtRes identifies a separator and its owning child
func (t *Tree) SeparatorAt(at geom.Point) (SeparatorAtRes, bool) {
	if t.IsEmpty() {
		return SeparatorAtRes{}, false
	}
	var res SeparatorAtRes
	ok := false
	t.walkSepWithID(t.root,
		func(cID Id, idx int, s Separator) bool {
			if s.Contains(at) {
				res = SeparatorAtRes{
					ContainerID: cID,
					ChildIdx:    idx,
					Layout:      s.Layout,
				}
				ok = true
				return false
			}
			return true
		},
	)
	return res, ok
}

func (t *Tree) walkSepWithID(id Id, fn sepVisitor) bool {
	n := t.nodes[id]
	if n.pane != nil {
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
		var a geom.Area
		if cn.pane != nil {
			a = cn.pane.Area()
		} else {
			a = cn.container.area
		}
		switch c.layout {
		case LayoutVertical:
			// 1-column gap between panes; separator is that gap column
			s := Separator{
				Layout: LayoutVertical,
				Area: geom.Area{
					Point: geom.Point{X: a.X + a.Width, Y: c.area.Y},
					Size:  geom.Size{Width: 1, Height: c.area.Height},
				},
			}
			if !fn(id, i, s) {
				return false
			}
		case LayoutHorizontal:
			// 1-row gap after child[i]; separator is that gap row
			s := Separator{
				Layout: LayoutHorizontal,
				Area: geom.Area{
					Point: geom.Point{X: c.area.X, Y: a.Y + a.Height},
					Size:  geom.Size{Width: c.area.Width, Height: 1},
				},
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
