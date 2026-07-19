package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view"
)

func TestSeparatorAt(t *testing.T) {
	t.Run("vertical separator detected", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.VSplitNew()
		tree := e.Tree()

		// separator is between the two panes at left.X + left.Width
		vs := e.Views()
		sepX := vs[0].View.Area().X + vs[0].View.Area().Width

		res, ok := tree.SeparatorAt(geom.Point{X: sepX})
		assert.True(t, ok)
		assert.Equal(t, view.LayoutVertical, res.Layout)
	})

	t.Run("no hit inside pane content", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.VSplitNew()
		tree := e.Tree()

		// column 0 is inside the left pane, not a separator
		_, ok := tree.SeparatorAt(geom.Point{})
		assert.False(t, ok)
	})

	t.Run("horizontal separator detected", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.HSplitNew()
		tree := e.Tree()

		// separator is the 1-row gap after the top pane
		vs := e.Views()
		sepY := vs[0].View.Area().Y + vs[0].View.Area().Height

		res, ok := tree.SeparatorAt(geom.Point{Y: sepY})
		assert.True(t, ok)
		assert.Equal(t, view.LayoutHorizontal, res.Layout)
	})

	t.Run("no separator when single pane", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		tree := e.Tree()

		_, ok := tree.SeparatorAt(geom.Point{})
		assert.False(t, ok)
	})
}

func TestMoveSeparator(t *testing.T) {
	t.Run("vertical split resizes panes", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.VSplitNew()
		tree := e.Tree()

		vs := e.Views()
		origLeft := vs[0].View.Area().Width
		origRight := vs[1].View.Area().Width
		sepX := vs[0].View.Area().X + origLeft

		res, _ := tree.SeparatorAt(geom.Point{X: sepX})

		// move separator 10 columns to the left
		tree.MoveSeparator(
			res.ContainerID, res.ChildIdx, res.Layout, sepX-10,
		)

		vs2 := e.Views()
		assert.Less(t, vs2[0].View.Area().Width, origLeft)
		assert.Greater(t, vs2[1].View.Area().Width, origRight)
		assert.Equal(t, 79, vs2[0].View.Area().Width+vs2[1].View.Area().Width)
	})

	t.Run("horizontal split resizes panes", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.HSplitNew()
		tree := e.Tree()

		vs := e.Views()
		origTop := vs[0].View.Area().Height
		origBot := vs[1].View.Area().Height
		sepY := vs[0].View.Area().Y + origTop // gap row after top pane

		res, _ := tree.SeparatorAt(geom.Point{Y: sepY})

		// move separator 3 rows up
		tree.MoveSeparator(
			res.ContainerID, res.ChildIdx, res.Layout, sepY-3,
		)

		vs2 := e.Views()
		assert.Less(t, vs2[0].View.Area().Height, origTop)
		assert.Greater(t, vs2[1].View.Area().Height, origBot)
		// pane heights + 1 gap row = 24
		assert.Equal(t, 23, vs2[0].View.Area().Height+vs2[1].View.Area().Height)
	})

	t.Run("enforces minimum pane width", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.VSplitNew()
		tree := e.Tree()

		vs := e.Views()
		sepX := vs[0].View.Area().X + vs[0].View.Area().Width
		res, _ := tree.SeparatorAt(geom.Point{X: sepX})

		// try to collapse left pane to 0
		tree.MoveSeparator(res.ContainerID, res.ChildIdx, res.Layout, -100)

		vs2 := e.Views()
		assert.GreaterOrEqual(t, vs2[0].View.Area().Width, 10)
		assert.GreaterOrEqual(t, vs2[1].View.Area().Width, 10)
	})

	t.Run("enforces minimum pane height", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.HSplitNew()
		tree := e.Tree()

		vs := e.Views()
		sepY := vs[0].View.Area().Y + vs[0].View.Area().Height
		res, _ := tree.SeparatorAt(geom.Point{Y: sepY})

		// try to collapse top pane to 0
		tree.MoveSeparator(res.ContainerID, res.ChildIdx, res.Layout, -100)

		vs2 := e.Views()
		assert.GreaterOrEqual(t, vs2[0].View.Area().Height, 4)
		assert.GreaterOrEqual(t, vs2[1].View.Area().Height, 4)
	})
}

func TestCanSplit(t *testing.T) {
	t.Run("empty tree allows split", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		e.CloseView(v.ID())

		assert.True(t, e.Tree().CanSplit(view.LayoutVertical))
	})

	t.Run("wide pane allows vertical", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		assert.True(t, e.Tree().CanSplit(view.LayoutVertical))
	})

	t.Run("tall pane allows horizontal", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		assert.True(t, e.Tree().CanSplit(view.LayoutHorizontal))
	})

	t.Run("too narrow rejects vertical", func(t *testing.T) {
		// 2*minPaneWidth+1 = 21; width of 20 can't fit two minimum-width panes
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 20, Height: 24})
		assert.False(t, e.Tree().CanSplit(view.LayoutVertical))
	})

	t.Run("too short rejects horizontal", func(t *testing.T) {
		// 2*minPaneHeight+1 = 9; height of 8 can't fit two minimum-height panes
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 8})
		assert.False(t, e.Tree().CanSplit(view.LayoutHorizontal))
	})

	t.Run("crowded siblings reject vertical", func(t *testing.T) {
		// 80 wide with 7 siblings: (80-6)/8 = 9 < 10 (minPaneWidth)
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		for range 6 {
			e.VSplitNew()
		}
		assert.False(t, e.Tree().CanSplit(view.LayoutVertical))
	})

	t.Run("crowded siblings reject horizontal", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		for range 5 {
			e.HSplitNew()
		}
		assert.False(t, e.Tree().CanSplit(view.LayoutHorizontal))
	})
}

func TestTreeSplitEdges(t *testing.T) {
	t.Run("same layout inserts sibling", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 120, Height: 24})

		v2 := e.VSplitNew()
		assert.NotNil(t, v2)
		v3 := e.VSplitNew()
		assert.NotNil(t, v3)

		views := e.Views()
		assert.Equal(t, 3, len(views))
		assert.Equal(t, v3.ID(), e.Tree().Focus())
	})

	t.Run("removing nested split collapses parent", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 120, Height: 40})
		left, ok := e.FocusedView()
		assert.True(t, ok)
		right, ok := e.VSplit(left.DocID())
		assert.True(t, ok)
		bottom, ok := e.HSplit(right.DocID())
		assert.True(t, ok)

		e.CloseView(bottom.ID())

		views := e.Views()
		assert.Equal(t, 2, len(views))
		_, ok = e.Tree().ContainerLayoutAt(right.ID())
		assert.True(t, ok)
	})
}

func TestWalkSeparatorsLayouts(t *testing.T) {
	t.Run("vertical layout and dimensions", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.VSplitNew()

		var seps []view.Separator
		e.Tree().WalkSeparators(func(s view.Separator) {
			seps = append(seps, s)
		})
		assert.Equal(t, 1, len(seps))
		assert.Equal(t, view.LayoutVertical, seps[0].Layout)
		assert.Equal(t, 1, seps[0].Width)
		assert.Equal(t, 24, seps[0].Height)
	})

	t.Run("horizontal layout and gap row position", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.HSplitNew()

		var seps []view.Separator
		e.Tree().WalkSeparators(func(s view.Separator) {
			seps = append(seps, s)
		})
		assert.Equal(t, 1, len(seps))
		assert.Equal(t, view.LayoutHorizontal, seps[0].Layout)
		assert.Equal(t, 80, seps[0].Width)
		assert.Equal(t, 1, seps[0].Height)
		// gap row is at top pane's Y + Height (one row after the pane)
		vs := e.Views()
		topArea := vs[0].View.Area()
		assert.Equal(t, topArea.Y+topArea.Height, seps[0].Y)
	})
}

func TestMoveSeparatorEdges(t *testing.T) {
	t.Run("wrong layout leaves panes unchanged", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.VSplitNew()
		tree := e.Tree()
		vs := e.Views()
		before := vs[0].View.Area().Width
		sepX := vs[0].View.Area().X + before
		res, ok := tree.SeparatorAt(geom.Point{X: sepX})
		assert.True(t, ok)

		tree.MoveSeparator(
			res.ContainerID, res.ChildIdx, view.LayoutHorizontal, sepX-5,
		)

		after := e.Views()[0].View.Area().Width
		assert.Equal(t, before, after)
	})

	t.Run("invalid index leaves panes unchanged", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.HSplitNew()
		tree := e.Tree()
		vs := e.Views()
		before := vs[0].View.Area().Height
		sepY := vs[0].View.Area().Y + before
		res, ok := tree.SeparatorAt(geom.Point{Y: sepY})
		assert.True(t, ok)

		tree.MoveSeparator(res.ContainerID, 5, res.Layout, sepY-2)

		after := e.Views()[0].View.Area().Height
		assert.Equal(t, before, after)
	})
}

func TestMoveSeparatorMore(t *testing.T) {
	t.Run("wrong layout on H container", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.HSplitNew()
		tree := e.Tree()
		vs := e.Views()
		sepY := vs[0].View.Area().Y + vs[0].View.Area().Height
		res, ok := tree.SeparatorAt(geom.Point{Y: sepY})
		assert.True(t, ok)
		before := e.Views()[0].View.Area().Height
		tree.MoveSeparator(
			res.ContainerID, res.ChildIdx, view.LayoutVertical, 0,
		)
		assert.Equal(t, before, e.Views()[0].View.Area().Height)
	})

	t.Run("out of bounds idx vertical", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.VSplitNew()
		tree := e.Tree()
		vs := e.Views()
		sepX := vs[0].View.Area().X + vs[0].View.Area().Width
		res, ok := tree.SeparatorAt(geom.Point{X: sepX})
		assert.True(t, ok)
		before := e.Views()[0].View.Area().Width
		tree.MoveSeparator(res.ContainerID, 99, view.LayoutVertical, sepX)
		assert.Equal(t, before, e.Views()[0].View.Area().Width)
	})

	t.Run("usable zero vertical", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.VSplitNew()
		e.ResizeTree(geom.Size{Width: 1, Height: 24})
		tree := e.Tree()
		res, ok := tree.SeparatorAt(geom.Point{})
		if ok {
			tree.MoveSeparator(
				res.ContainerID, res.ChildIdx, res.Layout, 0,
			)
		}
		assert.Equal(t, 2, len(e.Views()))
	})

	t.Run("usable zero horizontal", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.HSplitNew()
		e.ResizeTree(geom.Size{Width: 80, Height: 1})
		tree := e.Tree()
		res, ok := tree.SeparatorAt(geom.Point{})
		if ok {
			tree.MoveSeparator(
				res.ContainerID, res.ChildIdx, res.Layout, 0,
			)
		}
		assert.Equal(t, 2, len(e.Views()))
	})

	t.Run("childIdx=1 runs leftStart loop V", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 120, Height: 24})
		e.VSplitNew()
		e.VSplitNew()
		tree := e.Tree()
		vs := e.Views()
		v2Area := vs[1].View.Area()
		sepX := v2Area.X + v2Area.Width
		res, ok := tree.SeparatorAt(geom.Point{X: sepX})
		assert.True(t, ok)
		assert.Equal(t, 1, res.ChildIdx)
		tree.MoveSeparator(
			res.ContainerID, res.ChildIdx, res.Layout, sepX-5,
		)
		assert.Equal(t, 3, len(e.Views()))
	})

	t.Run("childIdx=1 runs topStart loop H", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 48})
		e.HSplitNew()
		e.HSplitNew()
		tree := e.Tree()
		vs := e.Views()
		v2Area := vs[1].View.Area()
		sepY := v2Area.Y + v2Area.Height
		res, ok := tree.SeparatorAt(geom.Point{Y: sepY})
		assert.True(t, ok)
		assert.Equal(t, 1, res.ChildIdx)
		tree.MoveSeparator(
			res.ContainerID, res.ChildIdx, res.Layout, sepY-2,
		)
		assert.Equal(t, 3, len(e.Views()))
	})

	t.Run("drag far right clamps right pane V", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.VSplitNew()
		tree := e.Tree()
		vs := e.Views()
		sepX := vs[0].View.Area().X + vs[0].View.Area().Width
		res, _ := tree.SeparatorAt(geom.Point{X: sepX})
		tree.MoveSeparator(res.ContainerID, res.ChildIdx, res.Layout, 76)
		assert.GreaterOrEqual(t, e.Views()[1].View.Area().Width, 1)
	})

	t.Run("drag far down clamps bottom pane H", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		e.HSplitNew()
		tree := e.Tree()
		vs := e.Views()
		sepY := vs[0].View.Area().Y + vs[0].View.Area().Height
		res, _ := tree.SeparatorAt(geom.Point{Y: sepY})
		tree.MoveSeparator(res.ContainerID, res.ChildIdx, res.Layout, 22)
		assert.GreaterOrEqual(t, e.Views()[1].View.Area().Height, 1)
	})

	t.Run("nested V sep init widthOf container", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 120, Height: 40})
		v1ID := e.Tree().Focus()
		e.VSplitNew()
		e.Tree().SetFocus(v1ID)
		e.HSplitNew()
		tree := e.Tree()
		var vSep view.Separator
		tree.WalkSeparators(func(s view.Separator) {
			if s.Layout == view.LayoutVertical {
				vSep = s
			}
		})
		res, ok := tree.SeparatorAt(vSep.Point)
		assert.True(t, ok)
		tree.MoveSeparator(
			res.ContainerID, res.ChildIdx, res.Layout, vSep.X-5,
		)
		assert.Equal(t, 3, len(e.Views()))
	})

	t.Run("nested H sep init heightOf container", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 120, Height: 60})
		v1ID := e.Tree().Focus()
		e.HSplitNew()
		e.Tree().SetFocus(v1ID)
		e.VSplitNew()
		tree := e.Tree()
		var hSep view.Separator
		tree.WalkSeparators(func(s view.Separator) {
			if s.Layout == view.LayoutHorizontal {
				hSep = s
			}
		})
		res, ok := tree.SeparatorAt(hSep.Point)
		assert.True(t, ok)
		tree.MoveSeparator(
			res.ContainerID, res.ChildIdx, res.Layout, hSep.Y-2,
		)
		assert.Equal(t, 3, len(e.Views()))
	})
}

func TestTreeEdges(t *testing.T) {
	t.Run("resize reports changed area", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		tree := e.Tree()

		assert.True(t, tree.Resize(geom.Size{Width: 80, Height: 24}))
		assert.False(t, tree.Resize(geom.Size{Width: 80, Height: 24}))
		assert.True(t, tree.Resize(geom.Size{Width: 100, Height: 24}))
	})

	t.Run("invalid ids are ignored", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		tree := e.Tree()

		assert.Nil(t, tree.Get(view.InvalidViewId))
		_, ok := tree.ContainerLayoutAt(view.InvalidViewId)
		assert.False(t, ok)
	})

	t.Run("set focus changes focused view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		first, ok := e.FocusedView()
		assert.True(t, ok)
		second := e.VSplitNew()
		assert.NotNil(t, second)

		e.Tree().SetFocus(first.ID())

		assert.Equal(t, first.ID(), e.Tree().Focus())
		views := e.Views()
		assert.Equal(t, 2, len(views))
		assert.True(t, views[0].Focused)
		assert.False(t, views[1].Focused)
	})

	t.Run("set focus marks old and new view dirty", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		first, ok := e.FocusedView()
		assert.True(t, ok)
		second := e.VSplitNew()
		assert.NotNil(t, second)
		first.ConsumeDirty()
		second.ConsumeDirty()

		e.Tree().SetFocus(first.ID())

		assert.True(t, first.ConsumeDirty())
		assert.True(t, second.ConsumeDirty())
	})

	t.Run("set focus to same id is a no-op", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		v.ConsumeDirty()

		e.Tree().SetFocus(v.ID())

		assert.False(t, v.ConsumeDirty())
	})

	t.Run("remove last view empties tree", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, ok := e.FocusedView()
		assert.True(t, ok)

		e.Tree().Remove(v.ID())

		assert.True(t, e.Tree().IsEmpty())
		assert.Empty(t, e.Tree().Traverse())
	})
}

func TestTreeReplacePane(t *testing.T) {
	t.Run("keeps position and marks pane dirty", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		v, ok := e.FocusedView()
		assert.True(t, ok)
		id := v.ID()
		area := v.Area()
		replacement := &fakePane{}

		e.Tree().ReplacePane(id, replacement)

		assert.Equal(t, id, replacement.ID())
		assert.Equal(t, area, replacement.Area())
		assert.True(t, replacement.ConsumeDirty())
	})

	t.Run("does not mark an unfocused pane dirty", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		first, ok := e.FocusedView()
		assert.True(t, ok)
		e.VSplitNew()
		replacement := &fakePane{}

		e.Tree().ReplacePane(first.ID(), replacement)

		assert.False(t, replacement.ConsumeDirty())
	})

	t.Run("is a no-op for an unknown id", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.NotPanics(t, func() {
			e.Tree().ReplacePane(view.InvalidViewId, &fakePane{})
		})
	})
}

func TestViewEdges(t *testing.T) {
	t.Run("jump list keeps newest entries", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		for i := range 70 {
			v.PushJump(view.DocumentId(i+1), i, core.PointSelection(i))
		}

		entries := v.Jumps()

		assert.Equal(t, 64, len(entries))
		assert.Equal(t, view.DocumentId(7), entries[0].DocID)
		assert.Equal(t, 69, entries[len(entries)-1].Anchor)
	})

	t.Run("wide rune width uses fallback", func(t *testing.T) {
		assert.Equal(t, 2, view.RuneWidth('界', 0, 4))
	})

	t.Run("horizontal width zero resets offset", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		v.SetOffset(view.Position{HorizontalOffset: 8})

		v.EnsureCursorVisibleHorizontal(
			core.NewRope("abc"), core.PointSelection(0), 0, 4, 1,
		)

		assert.Equal(t, 0, v.Offset().HorizontalOffset)
	})

	t.Run("cursor below viewport scrolls by line", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc := core.NewRope("a\nb\nc\nd\ne\n")
		v.EnsureCursorVisible(doc, core.PointSelection(8), 3, 1, nil)

		assert.Greater(t, v.Offset().Anchor, 0)
		assert.Equal(t, 0, v.Offset().VerticalOffset)
	})
}
