package view

import (
	"cmp"
	"slices"

	"github.com/kode4food/toe/internal/geom"
)

type resizeParent struct {
	container Id
	branch    Id
}

// ResizeFocused pushes a border of the focused pane's split by delta cells in
// dir, falling back to its other border if it has none on that side. False if
// no ancestor splits along that axis
func (t *Tree) ResizeFocused(dir Direction, delta int) bool {
	if delta <= 0 || t.IsEmpty() {
		return false
	}
	t.Unmaximize()
	layout := LayoutVertical
	if dir == DirectionUp || dir == DirectionDown {
		layout = LayoutHorizontal
	}

	branch := t.focus
	parent := t.nodes[branch].parent
	for {
		c := t.nodes[parent].container
		if c.layout == layout && len(c.children) > 1 {
			break
		}
		if parent == t.root {
			return false
		}
		branch = parent
		parent = t.nodes[parent].parent
	}

	c := t.nodes[parent].container
	idx := slices.Index(c.children, branch)
	last := len(c.children) - 1
	var boundary, sign int
	switch dir {
	case DirectionRight, DirectionDown:
		sign = 1
		boundary = idx
		if idx == last {
			boundary = idx - 1
		}
	default: // DirectionLeft, DirectionUp
		sign = -1
		boundary = idx - 1
		if idx == 0 {
			boundary = idx
		}
	}
	a := t.areaOf(c.children[boundary])
	start := a.X
	size := a.Width
	if layout == LayoutHorizontal {
		start = a.Y
		size = a.Height
	}
	t.moveSeparatorAt(parent, boundary, layout, start+size+sign*delta)
	return true
}

// GrowFocusedWidth widens the focused pane by moving the nearest vertical
// split, constrained by the minimum width of its sibling
func (t *Tree) GrowFocusedWidth(delta int) bool {
	return t.growFocused(LayoutVertical, delta)
}

// MoveSeparator adjusts the split between children[childIdx] and
// children[childIdx+1] in containerID, in tree coordinates
func (t *Tree) MoveSeparator(
	containerID Id, childIdx int, layout Layout, newPos int,
) {
	t.Unmaximize()
	t.moveSeparatorAt(containerID, childIdx, layout, newPos)
}

func (t *Tree) growFocused(layout Layout, delta int) bool {
	if delta <= 0 || t.IsEmpty() || t.Maximized() {
		return false
	}
	p, ok := t.focusedParent(layout)
	if !ok {
		return false
	}
	c := t.nodes[p.container].container
	usable := t.ensureRatios(c, layout)
	if usable == 0 {
		return false
	}
	minSize := minPaneWidth
	if layout == LayoutHorizontal {
		minSize = minPaneHeight
	}
	idx := slices.Index(c.children, p.branch)
	got := t.takeFromSiblings(takeFromSiblingsArgs{
		container: c,
		childIdx:  idx,
		wantRatio: float64(delta) / float64(usable),
		minRatio:  float64(minSize) / float64(usable),
	})
	if got <= 0 {
		return false
	}
	c.ratios[idx] += got
	t.recalculate()
	return true
}

type takeFromSiblingsArgs struct {
	container *treeContainer
	childIdx  int
	wantRatio float64
	minRatio  float64
}

func (t *Tree) takeFromSiblings(args takeFromSiblingsArgs) float64 {
	c := args.container
	order := make([]int, 0, len(c.children)-1)
	for i := range c.children {
		if i != args.childIdx {
			order = append(order, i)
		}
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(
			t.nodes[c.children[a]].focusSeq, t.nodes[c.children[b]].focusSeq,
		)
	})
	var got float64
	for _, i := range order {
		if got >= args.wantRatio {
			break
		}
		take := min(args.wantRatio-got, max(c.ratios[i]-args.minRatio, 0))
		c.ratios[i] -= take
		got += take
	}
	return got
}

func (t *Tree) ensureRatios(c *treeContainer, layout Layout) int {
	ln := len(c.children)
	if ln == 0 {
		return 0
	}
	size, sizeOf := c.area.Width, t.widthOf
	if layout == LayoutHorizontal {
		size, sizeOf = c.area.Height, t.heightOf
	}
	usable := max(size-(ln-1), 0)
	if usable == 0 {
		return 0
	}
	if c.ratios == nil {
		c.ratios = make([]float64, ln)
		for i, child := range c.children {
			c.ratios[i] = float64(sizeOf(child)) / float64(usable)
		}
	}
	return usable
}

func (t *Tree) focusedParent(layout Layout) (resizeParent, bool) {
	branch := t.focus
	parent := t.nodes[branch].parent
	for {
		c := t.nodes[parent].container
		if c.layout == layout && len(c.children) > 1 {
			return resizeParent{container: parent, branch: branch}, true
		}
		if parent == t.root {
			return resizeParent{}, false
		}
		branch = parent
		parent = t.nodes[parent].parent
	}
}

func (t *Tree) areaOf(id Id) geom.Area {
	n := t.nodes[id]
	if n.pane != nil {
		return n.pane.Area()
	}
	return n.container.area
}

func (t *Tree) moveSeparatorAt(
	containerID Id, childIdx int, layout Layout, newPos int,
) {
	n := t.nodes[containerID]
	if n == nil || n.container == nil || n.container.layout != layout {
		return
	}
	c := n.container
	ln := len(c.children)
	if childIdx < 0 || childIdx >= ln-1 {
		return
	}
	start := c.area.X
	minSize := minPaneWidth
	if layout == LayoutHorizontal {
		start = c.area.Y
		minSize = minPaneHeight
	}
	usable := t.ensureRatios(c, layout)
	if usable == 0 {
		return
	}
	for i := range childIdx {
		start += max(ratioCells(usable, c.ratios[i]), minSize) + 1
	}
	minRatio := float64(minSize) / float64(usable)
	total := c.ratios[childIdx] + c.ratios[childIdx+1]
	leftRatio := float64(newPos-start) / float64(usable)
	if leftRatio < minRatio {
		leftRatio = minRatio
	}
	rightRatio := total - leftRatio
	if rightRatio < minRatio {
		rightRatio = minRatio
		leftRatio = total - rightRatio
	}
	c.ratios[childIdx] = leftRatio
	c.ratios[childIdx+1] = rightRatio
	t.recalculate()
}

func (t *Tree) widthOf(id Id) int {
	n := t.nodes[id]
	if n == nil {
		return 0
	}
	if n.pane != nil {
		return n.pane.Area().Width
	}
	if n.container != nil {
		return n.container.area.Width
	}
	return 0
}

func (t *Tree) heightOf(id Id) int {
	n := t.nodes[id]
	if n == nil {
		return 0
	}
	if n.pane != nil {
		return n.pane.Area().Height
	}
	if n.container != nil {
		return n.container.area.Height
	}
	return 0
}
