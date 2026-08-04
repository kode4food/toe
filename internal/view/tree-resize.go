package view

import (
	"slices"

	"github.com/kode4food/toe/internal/geom"
)

// horizontalParent locates the nearest horizontal-split ancestor of the focus:
// the container to resize within and the child branch holding the focus
type horizontalParent struct {
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

	if layout == LayoutVertical {
		t.moveSepVertical(parent, boundary, a.X+a.Width+sign*delta)
	} else {
		t.moveSepHorizontal(parent, boundary, a.Y+a.Height+sign*delta)
	}
	return true
}

// GrowFocusedWidth widens the focused pane by moving the nearest vertical
// split, constrained by the minimum width of its sibling
func (t *Tree) GrowFocusedWidth(delta int) bool {
	if delta <= 0 || t.IsEmpty() {
		return false
	}
	if t.Maximized() {
		return false
	}
	branch := t.focus
	parent := t.nodes[branch].parent
	for {
		c := t.nodes[parent].container
		if c.layout == LayoutVertical && len(c.children) > 1 {
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
	if idx < len(c.children)-1 {
		a := t.areaOf(branch)
		t.moveSepVertical(parent, idx, a.X+a.Width+delta)
		return true
	}
	a := t.areaOf(branch)
	t.moveSepVertical(parent, idx-1, a.X-delta-1)
	return true
}

// GrowFocusedHeight grows the focused pane by moving the nearest horizontal
// split, constrained by the minimum height of its sibling
func (t *Tree) GrowFocusedHeight(delta int) bool {
	if delta <= 0 || t.IsEmpty() {
		return false
	}
	if t.Maximized() {
		return false
	}
	hp, ok := t.focusedHorizontalParent()
	if !ok {
		return false
	}

	c := t.nodes[hp.container].container
	idx := slices.Index(c.children, hp.branch)
	if idx < len(c.children)-1 {
		a := t.areaOf(hp.branch)
		t.moveSepHorizontal(hp.container, idx, a.Y+a.Height+delta)
		return true
	}
	a := t.areaOf(hp.branch)
	t.moveSepHorizontal(hp.container, idx-1, a.Y-delta-1)
	return true
}

// FocusedParentHeight reports the height of the nearest horizontal-split
// container above the focused pane, the pool its vertical size is drawn from
func (t *Tree) FocusedParentHeight() (int, bool) {
	hp, ok := t.focusedHorizontalParent()
	if !ok {
		return 0, false
	}
	return t.nodes[hp.container].container.area.Height, true
}

// MoveSeparator adjusts the split between children[childIdx] and
// children[childIdx+1] in containerID, in tree coordinates
func (t *Tree) MoveSeparator(
	containerID Id, childIdx int, layout Layout, newPos int,
) {
	t.Unmaximize()
	if layout == LayoutVertical {
		t.moveSepVertical(containerID, childIdx, newPos)
	} else {
		t.moveSepHorizontal(containerID, childIdx, newPos)
	}
}

// focusedHorizontalParent walks up from the focus to the nearest
// horizontal-split ancestor
func (t *Tree) focusedHorizontalParent() (horizontalParent, bool) {
	branch := t.focus
	parent := t.nodes[branch].parent
	for {
		c := t.nodes[parent].container
		if c.layout == LayoutHorizontal && len(c.children) > 1 {
			return horizontalParent{container: parent, branch: branch}, true
		}
		if parent == t.root {
			return horizontalParent{}, false
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

func (t *Tree) moveSepVertical(containerID Id, childIdx, newX int) {
	n := t.nodes[containerID]
	if n == nil || n.container == nil || n.container.layout != LayoutVertical {
		return
	}
	c := n.container
	ln := len(c.children)
	if childIdx < 0 || childIdx >= ln-1 {
		return
	}
	innerGap := 1
	usable := max(c.area.Width-(ln-1)*innerGap, 0)
	if usable == 0 {
		return
	}
	if c.ratios == nil {
		c.ratios = make([]float64, ln)
		for i, child := range c.children {
			c.ratios[i] = float64(t.widthOf(child)) / float64(usable)
		}
	}

	leftStart := c.area.X
	for i := range childIdx {
		leftStart += max(ratioCells(usable, c.ratios[i]), minPaneWidth) + innerGap
	}

	minRatio := float64(minPaneWidth) / float64(usable)
	total := c.ratios[childIdx] + c.ratios[childIdx+1]
	leftRatio := float64(newX-leftStart) / float64(usable)
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

func (t *Tree) moveSepHorizontal(containerID Id, childIdx, newY int) {
	n := t.nodes[containerID]
	if n == nil || n.container == nil ||
		n.container.layout != LayoutHorizontal {
		return
	}
	c := n.container
	ln := len(c.children)
	if childIdx < 0 || childIdx >= ln-1 {
		return
	}
	innerGap := 1
	usable := max(c.area.Height-(ln-1)*innerGap, 0)
	if usable == 0 {
		return
	}
	if c.ratios == nil {
		c.ratios = make([]float64, ln)
		for i, child := range c.children {
			c.ratios[i] = float64(t.heightOf(child)) / float64(usable)
		}
	}

	topStart := c.area.Y
	for i := range childIdx {
		topStart += max(ratioCells(usable, c.ratios[i]), minPaneHeight) + innerGap
	}

	minRatio := float64(minPaneHeight) / float64(usable) // min rows per pane
	total := c.ratios[childIdx] + c.ratios[childIdx+1]
	// newY is the gap row after children[childIdx]; height = newY-topStart
	leftRatio := float64(newY-topStart) / float64(usable)
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
