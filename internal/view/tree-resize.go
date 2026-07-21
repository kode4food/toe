package view

import (
	"slices"

	"github.com/kode4food/toe/internal/geom"
)

// ResizeFocused pushes a border of the focused pane's split by delta cells in
// dir, falling back to its other border if it has none on that side. False if
// no ancestor splits along that axis
func (t *Tree) ResizeFocused(dir Direction, delta int) bool {
	if delta <= 0 || t.IsEmpty() {
		return false
	}
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

	switch layout {
	case LayoutVertical:
		t.moveSepVertical(parent, boundary, a.X+a.Width+sign*delta)
	case LayoutHorizontal:
		t.moveSepHorizontal(parent, boundary, a.Y+a.Height+sign*delta)
	}
	return true
}

// MoveSeparator adjusts the split between children[childIdx] and
// children[childIdx+1] in containerID, in tree coordinates
func (t *Tree) MoveSeparator(
	containerID Id, childIdx int, layout Layout, newPos int,
) {
	switch layout {
	case LayoutVertical:
		t.moveSepVertical(containerID, childIdx, newPos)
	case LayoutHorizontal:
		t.moveSepHorizontal(containerID, childIdx, newPos)
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
