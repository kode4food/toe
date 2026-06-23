package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestSeparatorAt(t *testing.T) {
	t.Run("vertical separator detected", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		tree := e.Tree()

		// separator is between the two panes at left.X + left.Width
		vs := tree.Views()
		sepX := vs[0].View.Area().X + vs[0].View.Area().Width

		_, _, layout, ok := tree.SeparatorAt(sepX, 0)
		assert.True(t, ok)
		assert.Equal(t, view.LayoutVertical, layout)
	})

	t.Run("no hit inside pane content", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		tree := e.Tree()

		// column 0 is inside the left pane, not a separator
		_, _, _, ok := tree.SeparatorAt(0, 0)
		assert.False(t, ok)
	})

	t.Run("horizontal separator detected", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.HSplitNew()
		tree := e.Tree()

		// separator is the 1-row gap after the top pane
		vs := tree.Views()
		sepY := vs[0].View.Area().Y + vs[0].View.Area().Height

		_, _, layout, ok := tree.SeparatorAt(0, sepY)
		assert.True(t, ok)
		assert.Equal(t, view.LayoutHorizontal, layout)
	})

	t.Run("no separator when single pane", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		tree := e.Tree()

		_, _, _, ok := tree.SeparatorAt(0, 0)
		assert.False(t, ok)
	})
}

func TestMoveSeparator(t *testing.T) {
	t.Run("vertical split resizes panes", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		tree := e.Tree()

		vs := tree.Views()
		origLeft := vs[0].View.Area().Width
		origRight := vs[1].View.Area().Width
		sepX := vs[0].View.Area().X + origLeft

		cID, idx, layout, _ := tree.SeparatorAt(sepX, 0)

		// move separator 10 columns to the left
		tree.MoveSeparator(cID, idx, layout, sepX-10)

		vs2 := tree.Views()
		assert.Less(t, vs2[0].View.Area().Width, origLeft)
		assert.Greater(t, vs2[1].View.Area().Width, origRight)
		assert.Equal(t, 79, vs2[0].View.Area().Width+vs2[1].View.Area().Width)
	})

	t.Run("horizontal split resizes panes", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.HSplitNew()
		tree := e.Tree()

		vs := tree.Views()
		origTop := vs[0].View.Area().Height
		origBot := vs[1].View.Area().Height
		sepY := vs[0].View.Area().Y + origTop // gap row after top pane

		cID, idx, layout, _ := tree.SeparatorAt(0, sepY)

		// move separator 3 rows up
		tree.MoveSeparator(cID, idx, layout, sepY-3)

		vs2 := tree.Views()
		assert.Less(t, vs2[0].View.Area().Height, origTop)
		assert.Greater(t, vs2[1].View.Area().Height, origBot)
		// pane heights + 1 gap row = 24
		assert.Equal(t, 23, vs2[0].View.Area().Height+vs2[1].View.Area().Height)
	})

	t.Run("enforces minimum pane width", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		tree := e.Tree()

		vs := tree.Views()
		sepX := vs[0].View.Area().X + vs[0].View.Area().Width
		cID, idx, layout, _ := tree.SeparatorAt(sepX, 0)

		// try to collapse left pane to 0
		tree.MoveSeparator(cID, idx, layout, -100)

		vs2 := tree.Views()
		assert.GreaterOrEqual(t, vs2[0].View.Area().Width, 10)
		assert.GreaterOrEqual(t, vs2[1].View.Area().Width, 10)
	})

	t.Run("enforces minimum pane height", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.HSplitNew()
		tree := e.Tree()

		vs := tree.Views()
		sepY := vs[0].View.Area().Y + vs[0].View.Area().Height
		cID, idx, layout, _ := tree.SeparatorAt(0, sepY)

		// try to collapse top pane to 0
		tree.MoveSeparator(cID, idx, layout, -100)

		vs2 := tree.Views()
		assert.GreaterOrEqual(t, vs2[0].View.Area().Height, 4)
		assert.GreaterOrEqual(t, vs2[1].View.Area().Height, 4)
	})
}

func TestCanSplit(t *testing.T) {
	t.Run("wide pane allows vertical", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		assert.True(t, e.Tree().CanSplit(view.LayoutVertical))
	})

	t.Run("tall pane allows horizontal", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		assert.True(t, e.Tree().CanSplit(view.LayoutHorizontal))
	})

	t.Run("too narrow rejects vertical", func(t *testing.T) {
		// 2*minPaneWidth+1 = 21; width of 20 can't fit two minimum-width panes
		e := view.NewEditor("/tmp")
		e.ResizeTree(20, 24)
		assert.False(t, e.Tree().CanSplit(view.LayoutVertical))
	})

	t.Run("too short rejects horizontal", func(t *testing.T) {
		// 2*minPaneHeight+1 = 9; height of 8 can't fit two minimum-height panes
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 8)
		assert.False(t, e.Tree().CanSplit(view.LayoutHorizontal))
	})

	t.Run("crowded siblings reject vertical", func(t *testing.T) {
		// 80 wide with 7 siblings: (80-6)/8 = 9 < 10 (minPaneWidth)
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		for range 6 {
			e.VSplitNew()
		}
		assert.False(t, e.Tree().CanSplit(view.LayoutVertical))
	})
}

func TestWalkSeparatorsLayouts(t *testing.T) {
	t.Run("vertical layout and dimensions", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()

		var seps []view.Separator
		e.Tree().WalkSeparators(func(s view.Separator) {
			seps = append(seps, s)
		})
		assert.Equal(t, 1, len(seps))
		assert.Equal(t, view.LayoutVertical, seps[0].Layout)
		assert.Equal(t, 1, seps[0].W)
		assert.Equal(t, 24, seps[0].H)
	})

	t.Run("horizontal layout and gap row position", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.HSplitNew()

		var seps []view.Separator
		e.Tree().WalkSeparators(func(s view.Separator) {
			seps = append(seps, s)
		})
		assert.Equal(t, 1, len(seps))
		assert.Equal(t, view.LayoutHorizontal, seps[0].Layout)
		assert.Equal(t, 80, seps[0].W)
		assert.Equal(t, 1, seps[0].H)
		// gap row is at top pane's Y + Height (one row after the pane)
		vs := e.Tree().Views()
		topArea := vs[0].View.Area()
		assert.Equal(t, topArea.Y+topArea.Height, seps[0].Y)
	})
}
