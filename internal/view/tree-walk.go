package view

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
