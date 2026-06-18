package view_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

func TestNewView(t *testing.T) {
	t.Run("creates view with unique id", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v1, _ := e.FocusedView()
		assert.NotEqual(t, view.InvalidViewId, v1.ID())
	})

	t.Run("default mode is normal", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		assert.Equal(t, view.ModeNormal, v.Mode())
	})
}

func TestViewMode(t *testing.T) {
	t.Run("mode can be changed", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		v.SetMode(view.ModeInsert)
		assert.Equal(t, view.ModeInsert, v.Mode())
		v.SetMode(view.ModeSelect)
		assert.Equal(t, view.ModeSelect, v.Mode())
	})
}

func TestViewOffset(t *testing.T) {
	t.Run("default offset is zero", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		assert.Equal(t, view.Position{}, v.Offset())
	})

	t.Run("set offset", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		p := view.Position{Anchor: 5, HorizontalOffset: 2}
		v.SetOffset(p)
		assert.Equal(t, p, v.Offset())
	})
}

func TestViewJumpList(t *testing.T) {
	t.Run("backward on empty list returns false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		_, _, ok := v.JumpBackward()
		assert.False(t, ok)
	})

	t.Run("push and backward retrieves previous", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		v.PushJump(d.ID(), 0)
		v.PushJump(d.ID(), 10)
		docID, anchor, ok := v.JumpBackward()
		assert.True(t, ok)
		assert.Equal(t, d.ID(), docID)
		assert.Equal(t, 0, anchor)
	})

	t.Run("forward after backward returns next", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		v.PushJump(d.ID(), 0)
		v.PushJump(d.ID(), 10)
		v.JumpBackward()
		docID, anchor, ok := v.JumpForward()
		assert.True(t, ok)
		assert.Equal(t, d.ID(), docID)
		assert.Equal(t, 10, anchor)
	})

	t.Run("forward on empty returns false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		_, _, ok := v.JumpForward()
		assert.False(t, ok)
	})
}

func TestViewEnsureCursorVisible(t *testing.T) {
	t.Run("scrolls when cursor below visible area", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope("line1\nline2\nline3\nline4\nline5\n")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(24)}, 0)
		v.EnsureCursorVisible(doc, sel, 2, 0, nil)
		assert.Greater(t, v.Offset().Anchor, 0)
	})

	t.Run("no scroll when cursor already visible", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope("line1\nline2\n")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		v.EnsureCursorVisible(doc, sel, 10, 5, nil)
		assert.Equal(t, 0, v.Offset().Anchor)
	})

	t.Run("zero height is a no-op", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope("abc")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		v.EnsureCursorVisible(doc, sel, 0, 5, nil)
		assert.Equal(t, 0, v.Offset().Anchor)
	})

	t.Run("mid-viewport cursor does not scroll", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		// line 1 is a long unbreakable run that wraps to 8 visual rows; the
		// cursor sits on line 2, well within the viewport in visual rows even
		// though its text-line index is adjacent to the anchor line
		long := strings.Repeat("x", 80)
		doc := core.NewRope("a\n" + long + "\nb\nc\nd\ne\nf\ng")
		line1Start, _ := doc.LineToChar(1)
		line2Start, _ := doc.LineToChar(2)
		v.SetOffset(view.Position{Anchor: line1Start})
		sel, _ := core.NewSelection(
			[]core.Range{core.PointRange(line2Start)}, 0,
		)
		vf := &core.VisualMoveFormat{
			ViewportWidth: 10, TabWidth: 4, MaxWrap: 2,
		}
		// the text-line fallback would wrongly scroll up here; the visual path
		// recognizes the cursor is already in range and leaves the anchor put
		v.EnsureCursorVisible(doc, sel, 12, 3, vf)
		assert.Equal(t, line1Start, v.Offset().Anchor)
	})

	t.Run("scrolls within a tall line", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		// a single line of 80 cells wraps to 8 visual rows at width 10; with a
		// 4-row viewport the cursor near the end cannot be shown by anchoring at
		// the line start, so the view must scroll into the line itself
		doc := core.NewRope(strings.Repeat("x", 80))
		sel, _ := core.NewSelection([]core.Range{core.PointRange(79)}, 0)
		vf := &core.VisualMoveFormat{
			ViewportWidth: 10, TabWidth: 4, MaxWrap: 2,
		}
		v.EnsureCursorVisible(doc, sel, 4, 0, vf)
		// anchor stays at the line start, but the view is scrolled 4 visual rows
		// into the line so the cursor (row 7) sits on the bottom viewport row
		assert.Equal(t, 0, v.Offset().Anchor)
		assert.Equal(t, 4, v.Offset().VerticalOffset)
	})
}

func TestViewEnsureCursorVisibleHorizontal(t *testing.T) {
	// width 10, no scrolloff, tab width 4 throughout unless noted
	t.Run("scrolls right when cursor past right edge", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(15)}, 0)
		v.EnsureCursorVisibleHorizontal(doc, sel, 10, 4, 0)
		// cursor at column 15, width 10 -> offset = 15 - 10 + 1 = 6
		assert.Equal(t, 6, v.Offset().HorizontalOffset)
	})

	t.Run("scrolls back to zero at line start", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		v.SetOffset(view.Position{HorizontalOffset: 6})
		sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		v.EnsureCursorVisibleHorizontal(doc, sel, 10, 4, 0)
		assert.Equal(t, 0, v.Offset().HorizontalOffset)
	})

	t.Run("no scroll when cursor already visible", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(5)}, 0)
		v.EnsureCursorVisibleHorizontal(doc, sel, 10, 4, 0)
		assert.Equal(t, 0, v.Offset().HorizontalOffset)
	})

	t.Run("non-positive width disables and resets offset", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		v.SetOffset(view.Position{HorizontalOffset: 6})
		sel, _ := core.NewSelection([]core.Range{core.PointRange(15)}, 0)
		v.EnsureCursorVisibleHorizontal(doc, sel, 0, 4, 0)
		assert.Equal(t, 0, v.Offset().HorizontalOffset)
	})

	t.Run("tabs expand into the visual column", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		// three tabs (tab width 4) -> visual column 12 at char offset 3
		doc := core.NewRope("\t\t\tx")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(3)}, 0)
		v.EnsureCursorVisibleHorizontal(doc, sel, 10, 4, 0)
		// visual col 12, width 10 -> offset = 12 - 10 + 1 = 3
		assert.Equal(t, 3, v.Offset().HorizontalOffset)
	})

	t.Run("scrolloff keeps a left margin", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(15)}, 0)
		v.EnsureCursorVisibleHorizontal(doc, sel, 10, 4, 2)
		// scrolloff 2: offset = 15 - 10 + 1 + 2 = 8
		assert.Equal(t, 8, v.Offset().HorizontalOffset)
	})
}

func TestViewArea(t *testing.T) {
	t.Run("area is set by layout engine", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		v, _ := e.FocusedView()
		a := v.Area()
		assert.Equal(t, 80, a.Width)
		assert.Equal(t, 24, a.Height)
	})
}

func TestViewJumps(t *testing.T) {
	t.Run("entries includes pushed jumps", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		v.PushJump(d.ID(), 0)
		v.PushJump(d.ID(), 5)
		entries := v.Jumps()
		assert.Equal(t, 2, len(entries))
		assert.Equal(t, 0, entries[0].Anchor)
		assert.Equal(t, 5, entries[1].Anchor)
	})

	t.Run("empty jumps returns empty slice", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		entries := v.Jumps()
		assert.Empty(t, entries)
	})
}

func TestViewEnsureCursorVisibleScrolloff(t *testing.T) {
	t.Run("scrolloff clamped when height is small", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope(
			"line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\n",
		)
		sel, _ := core.NewSelection([]core.Range{core.PointRange(36)}, 0)
		v.EnsureCursorVisible(doc, sel, 3, 10, nil)
		assert.Greater(t, v.Offset().Anchor, 0)
	})

	t.Run("scrolls up when cursor above visible area", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc := core.NewRope("line1\nline2\nline3\nline4\nline5\n")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(24)}, 0)
		v.EnsureCursorVisible(doc, sel, 2, 0, nil)
		offset := v.Offset().Anchor
		sel2, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		v.EnsureCursorVisible(doc, sel2, 10, 0, nil)
		assert.LessOrEqual(t, v.Offset().Anchor, offset)
	})

	t.Run("keeps bottom margin past EOF", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		// lines 0..29; cursor on the last line, height 10, scrolloff 5
		doc := core.NewRope(strings.Repeat("x\n", 30))
		last, _ := doc.LineToChar(29)
		sel, _ := core.NewSelection([]core.Range{core.PointRange(last)}, 0)
		v.EnsureCursorVisible(doc, sel, 10, 5, nil)
		// Helix anchors so the cursor sits at height-soBottom-1=4, leaving the
		// full bottom margin of blank rows below EOF (anchor = 29-4 = line 25)
		anchorLine, _ := doc.CharToLine(v.Offset().Anchor)
		assert.Equal(t, 25, anchorLine)
	})
}

func TestSelectionPerView(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "shared.txt")
	err := os.WriteFile(path, []byte("abcdef"), 0o644)
	assert.NoError(t, err)

	e := view.NewEditor(tmp)
	v1, err := e.OpenFile(path)
	assert.NoError(t, err)
	// Create a second view into the same document via vertical split
	v2, ok := e.VSplit(v1.DocID())
	assert.True(t, ok)
	doc, ok := e.Document(v1.DocID())
	assert.True(t, ok)

	doc.SetSelectionFor(v1.ID(), core.PointSelection(1))
	doc.SetSelectionFor(v2.ID(), core.PointSelection(4))

	assert.Equal(t, 1, doc.SelectionFor(v1.ID()).Primary().Head)
	assert.Equal(t, 4, doc.SelectionFor(v2.ID()).Primary().Head)
}
