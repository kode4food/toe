package view

import (
	"math"

	"github.com/kode4food/toe/internal/geom"
)

type treeWork struct {
	id   Id
	area geom.Area
}

func (t *Tree) recalculate() {
	if t.IsEmpty() {
		t.focus = t.root
		return
	}

	stack := []treeWork{{id: t.root, area: t.area}}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		n := t.nodes[item.id]
		if n.pane != nil {
			n.pane.SetArea(item.area)
			continue
		}

		c := n.container
		c.area = item.area
		a := item.area
		switch c.layout {
		case LayoutHorizontal:
			// children stacked (hsplit): distribute height evenly, 1px gap
			ln := len(c.children)
			innerGap := 1
			usable := max(a.Height-(ln-1)*innerGap, 0)
			h := usable / ln
			childY := a.Y
			for i, child := range c.children {
				var childH int
				switch {
				case i == ln-1:
					childH = max(a.Y+a.Height-childY, 2)
				case c.ratios != nil && i < len(c.ratios):
					childH = max(ratioCells(usable, c.ratios[i]), minPaneHeight)
				default:
					childH = h
				}
				area := geom.Area{
					Point: geom.Point{X: a.X, Y: childY},
					Size:  geom.Size{Width: a.Width, Height: childH},
				}
				stack = append(stack, treeWork{id: child, area: area})
				childY += childH + innerGap
			}
		case LayoutVertical:
			// children side by side (vsplit): distribute width evenly, 1px gap
			ln := len(c.children)
			innerGap := 1
			usable := max(a.Width-(ln-1)*innerGap, 0)
			w := usable / ln
			childX := a.X
			for i, child := range c.children {
				var childW int
				switch {
				case i == ln-1:
					childW = max(a.X+a.Width-childX, 1)
				case c.ratios != nil && i < len(c.ratios):
					childW = max(ratioCells(usable, c.ratios[i]), minPaneWidth)
				default:
					childW = w
				}
				area := geom.Area{
					Point: geom.Point{X: childX, Y: a.Y},
					Size:  geom.Size{Width: childW, Height: a.Height},
				}
				stack = append(stack, treeWork{id: child, area: area})
				childX += childW + innerGap
			}
		}
	}
}

// rounds rather than truncates: incremental resizes round-trip through ratios
// repeatedly, and truncation would silently lose a cell each time
func ratioCells(usable int, ratio float64) int {
	return int(math.Round(float64(usable) * ratio))
}
