package view_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view"
)

func TestNewView(t *testing.T) {
	t.Run("creates view with unique id", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v1 := e.FocusedView()
		assert.NotEqual(t, view.InvalidViewId, v1.ID())
	})

	t.Run("default mode is normal", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		assert.Equal(t, view.ModeNormal, v.Mode())
	})
}

func TestViewMode(t *testing.T) {
	t.Run("mode can be changed", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		v.SetMode(view.ModeInsert)
		assert.Equal(t, view.ModeInsert, v.Mode())
		v.SetMode(view.ModeSelect)
		assert.Equal(t, view.ModeSelect, v.Mode())
	})
}

func TestViewOffset(t *testing.T) {
	t.Run("default offset is zero", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		assert.Equal(t, view.Position{}, v.Offset())
	})

	t.Run("set offset", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		p := view.Position{Anchor: 5, HorizontalOffset: 2}
		v.SetOffset(p)
		assert.Equal(t, p, v.Offset())
	})
}

func TestViewJumpList(t *testing.T) {
	t.Run("backward on empty list returns false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		_, _, ok := v.JumpBackward()
		assert.False(t, ok)
	})

	t.Run("push and backward retrieves latest", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		d := e.FocusedDocument()
		v.PushJump(d.ID(), 0, core.PointSelection(0))
		v.PushJump(d.ID(), 10, core.PointSelection(10))
		docID, anchor, ok := v.JumpBackward()
		assert.True(t, ok)
		assert.Equal(t, d.ID(), docID)
		assert.Equal(t, 10, anchor)
	})

	t.Run("backward twice walks the history", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		d := e.FocusedDocument()
		v.PushJump(d.ID(), 0, core.PointSelection(0))
		v.PushJump(d.ID(), 10, core.PointSelection(10))
		v.JumpBackward()
		_, anchor, ok := v.JumpBackward()
		assert.True(t, ok)
		assert.Equal(t, 0, anchor)
	})

	t.Run("forward after backward returns start", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		d := e.FocusedDocument()
		v.PushJump(d.ID(), 0, core.PointSelection(0))
		v.PushJump(d.ID(), 10, core.PointSelection(10))
		_, back, ok := v.JumpBackward()
		assert.True(t, ok)
		assert.Equal(t, 10, back)
		docID, anchor, ok := v.JumpForward()
		assert.True(t, ok)
		assert.Equal(t, d.ID(), docID)
		// the position held when JumpBackward ran, recorded automatically
		assert.Equal(t, 0, anchor)
	})

	t.Run("new jump truncates forward history", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		d := e.FocusedDocument()
		v.PushJump(d.ID(), 0, core.PointSelection(0))
		v.PushJump(d.ID(), 10, core.PointSelection(10))
		v.JumpBackward()
		v.JumpBackward()
		v.PushJump(d.ID(), 20, core.PointSelection(20))
		_, _, ok := v.JumpForward()
		assert.False(t, ok)
	})

	t.Run("forward on empty returns false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		_, _, ok := v.JumpForward()
		assert.False(t, ok)
	})
}

func TestViewEnsureCursorVisible(t *testing.T) {
	t.Run("scrolls when cursor below visible area", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("line1\nline2\nline3\nline4\nline5\n")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(24)}, 0)
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    2,
			ScrollOff: 0,
		})
		assert.Greater(t, v.Offset().Anchor, 0)
	})

	t.Run("no scroll when cursor already visible", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("line1\nline2\n")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    10,
			ScrollOff: 5,
		})
		assert.Equal(t, 0, v.Offset().Anchor)
	})

	t.Run("zero height is a no-op", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("abc")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    0,
			ScrollOff: 5,
		})
		assert.Equal(t, 0, v.Offset().Anchor)
	})

	t.Run("mid-viewport cursor does not scroll", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		// line 1 is a long unbreakable run that wraps to 8 visual rows. The
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
		// the text-line fallback would wrongly scroll up here. The visual path
		// recognizes the cursor is already in range and leaves the anchor put
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    12,
			ScrollOff: 3,
			Visual:    vf,
		})
		assert.Equal(t, line1Start, v.Offset().Anchor)
	})

	t.Run("scrolls within a tall line", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		// a single line of 80 cells wraps to 8 visual rows at width 10. With a
		// 4-row viewport cannot show the cursor near the end by anchoring at
		// the line start, so the view must scroll into the line itself
		doc := core.NewRope(strings.Repeat("x", 80))
		sel, _ := core.NewSelection([]core.Range{core.PointRange(79)}, 0)
		vf := &core.VisualMoveFormat{
			ViewportWidth: 10, TabWidth: 4, MaxWrap: 2,
		}
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    4,
			ScrollOff: 0,
			Visual:    vf,
		})
		// anchor stays at the line start, but the view scrolls 4 visual rows
		// into the line so the cursor sits on the bottom viewport row
		assert.Equal(t, 0, v.Offset().Anchor)
		assert.Equal(t, 4, v.Offset().VerticalOffset)
	})

	t.Run("visual scrolls up above anchor", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("a\nb\nc\nd\n")
		line2, _ := doc.LineToChar(2)
		v.SetOffset(view.Position{Anchor: line2})
		sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		vf := &core.VisualMoveFormat{
			ViewportWidth: 10, TabWidth: 4, MaxWrap: 2,
		}
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    4,
			ScrollOff: 1,
			Visual:    vf,
		})
		assert.Equal(t, 0, v.Offset().Anchor)
	})

	t.Run("visual scrolls below wrapped lines", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		long := strings.Repeat("x", 40)
		doc := core.NewRope(long + "\n" + long + "\nend")
		last, _ := doc.LineToChar(2)
		sel, _ := core.NewSelection([]core.Range{core.PointRange(last)}, 0)
		vf := &core.VisualMoveFormat{
			ViewportWidth: 10, TabWidth: 4, MaxWrap: 2,
		}
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    3,
			ScrollOff: 0,
			Visual:    vf,
		})
		assert.Greater(t, v.Offset().Anchor, 0)
	})
}

func TestViewEnsureCursorVisibleHorizontal(t *testing.T) {
	// width 10, no scrolloff, tab width 4 throughout unless noted
	t.Run("scrolls right past right edge", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(15)}, 0)
		v.EnsureCursorVisibleHorizontal(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Width:     10,
			TabWidth:  4,
			ScrollOff: 0,
		})
		// cursor at column 15, width 10 -> offset = 15 - 10 + 1 = 6
		assert.Equal(t, 6, v.Offset().HorizontalOffset)
	})

	t.Run("scrolls back to zero at line start", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		v.SetOffset(view.Position{HorizontalOffset: 6})
		sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		v.EnsureCursorVisibleHorizontal(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Width:     10,
			TabWidth:  4,
			ScrollOff: 0,
		})
		assert.Equal(t, 0, v.Offset().HorizontalOffset)
	})

	t.Run("no scroll when cursor already visible", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(5)}, 0)
		v.EnsureCursorVisibleHorizontal(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Width:     10,
			TabWidth:  4,
			ScrollOff: 0,
		})
		assert.Equal(t, 0, v.Offset().HorizontalOffset)
	})

	t.Run("non-positive width resets offset", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		v.SetOffset(view.Position{HorizontalOffset: 6})
		sel, _ := core.NewSelection([]core.Range{core.PointRange(15)}, 0)
		v.EnsureCursorVisibleHorizontal(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Width:     0,
			TabWidth:  4,
			ScrollOff: 0,
		})
		assert.Equal(t, 0, v.Offset().HorizontalOffset)
	})

	t.Run("tabs expand into the visual column", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		// three tabs (tab width 4) -> visual column 12 at char offset 3
		doc := core.NewRope("\t\t\tx")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(3)}, 0)
		v.EnsureCursorVisibleHorizontal(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Width:     10,
			TabWidth:  4,
			ScrollOff: 0,
		})
		// visual col 12, width 10 -> offset = 12 - 10 + 1 = 3
		assert.Equal(t, 3, v.Offset().HorizontalOffset)
	})

	t.Run("scrolloff keeps a left margin", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("0123456789abcdefghij")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(15)}, 0)
		v.EnsureCursorVisibleHorizontal(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Width:     10,
			TabWidth:  4,
			ScrollOff: 2,
		})
		// scrolloff 2: offset = 15 - 10 + 1 + 2 = 8
		assert.Equal(t, 8, v.Offset().HorizontalOffset)
	})
}

func TestViewArea(t *testing.T) {
	t.Run("area is set by layout engine", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		v := e.FocusedView()
		a := v.Area()
		assert.Equal(t, 80, a.Width)
		assert.Equal(t, 24, a.Height)
	})
}

func TestViewConsumeDirty(t *testing.T) {
	t.Run("new view starts dirty", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		assert.True(t, v.ConsumeDirty())
	})

	t.Run("consuming clears the flag", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		v.ConsumeDirty()
		assert.False(t, v.ConsumeDirty())
	})

	t.Run("setting area to same value stays clean", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		v.ConsumeDirty()
		v.SetArea(v.Area())
		assert.False(t, v.ConsumeDirty())
	})

	t.Run("setting area to new value marks dirty", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		v.ConsumeDirty()
		v.SetArea(geom.Area{Size: geom.Size{Width: 10, Height: 5}})
		assert.True(t, v.ConsumeDirty())
	})

	t.Run("mark dirty forces dirty on next consume", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		v.ConsumeDirty()
		v.MarkDirty()
		assert.True(t, v.ConsumeDirty())
		assert.False(t, v.ConsumeDirty())
	})
}

func TestViewFreeScroll(t *testing.T) {
	t.Run("begin and end round-trips", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		assert.False(t, v.FreeScroll())
		v.BeginFreeScroll(0, core.PointSelection(0))
		assert.True(t, v.FreeScroll())
		v.EndFreeScroll()
		assert.False(t, v.FreeScroll())
	})

	t.Run("sync keeps unchanged state", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		sel := core.PointSelection(0)
		v.BeginFreeScroll(1, sel)
		assert.True(t, v.SyncFreeScroll(1, sel))
		assert.True(t, v.FreeScroll())
	})

	t.Run("sync ends on selection change", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		v.BeginFreeScroll(1, core.PointSelection(0))
		assert.False(t, v.SyncFreeScroll(1, core.PointSelection(3)))
		assert.False(t, v.FreeScroll())
	})

	t.Run("sync ends on revision change", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		sel := core.PointSelection(0)
		v.BeginFreeScroll(1, sel)
		assert.False(t, v.SyncFreeScroll(2, sel))
		assert.False(t, v.FreeScroll())
	})

	t.Run("sync inactive reports false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		assert.False(t, v.SyncFreeScroll(0, core.PointSelection(0)))
	})
}

func TestViewJumps(t *testing.T) {
	t.Run("entries includes pushed jumps", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		d := e.FocusedDocument()
		v.PushJump(d.ID(), 0, core.PointSelection(0))
		v.PushJump(d.ID(), 5, core.PointSelection(5))
		entries := v.Jumps()
		assert.Equal(t, 2, len(entries))
		assert.Equal(t, 0, entries[0].Anchor)
		assert.Equal(t, 5, entries[1].Anchor)
	})

	t.Run("deduplicates adjacent jumps", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		d := e.FocusedDocument()
		v.PushJump(d.ID(), 0, core.PointSelection(0))
		v.PushJump(d.ID(), 0, core.PointSelection(0))
		entries := v.Jumps()
		assert.Equal(t, 1, len(entries))
		assert.Equal(t, 0, entries[0].Anchor)
	})

	t.Run("empty jumps returns empty slice", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		entries := v.Jumps()
		assert.Empty(t, entries)
	})
}

func TestViewEnsureCursorVisibleScrollOff(t *testing.T) {
	t.Run("scrolloff clamped when height is small", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope(
			"line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\n",
		)
		sel, _ := core.NewSelection([]core.Range{core.PointRange(36)}, 0)
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    3,
			ScrollOff: 10,
		})
		assert.Greater(t, v.Offset().Anchor, 0)
	})

	t.Run("scrolls up when cursor above visible", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		doc := core.NewRope("line1\nline2\nline3\nline4\nline5\n")
		sel, _ := core.NewSelection([]core.Range{core.PointRange(24)}, 0)
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    2,
			ScrollOff: 0,
		})
		offset := v.Offset().Anchor
		sel2, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel2,
			Height:    10,
			ScrollOff: 0,
		})
		assert.LessOrEqual(t, v.Offset().Anchor, offset)
	})

	t.Run("keeps bottom margin past EOF", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v := e.FocusedView()
		// lines 0..29, cursor on the last line, height 10, scrolloff 5
		doc := core.NewRope(strings.Repeat("x\n", 30))
		last, _ := doc.LineToChar(29)
		sel, _ := core.NewSelection([]core.Range{core.PointRange(last)}, 0)
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       doc,
			Selection: sel,
			Height:    10,
			ScrollOff: 5,
		})
		// Anchor puts the cursor at height-soBottom-1=4, leaving the full
		// bottom margin of blank rows below EOF (anchor = 29-4 = line 25)
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
	e.ResizeTree(geom.Size{Width: 80, Height: 24})
	v1, err := e.OpenFile(path)
	assert.NoError(t, err)
	// Create a second view into the same document via vertical split
	v2 := e.VSplit(v1.DocID())
	assert.NotNil(t, v2)
	doc := e.Document(v1.DocID())
	assert.NotNil(t, doc)

	doc.SetSelectionFor(v1.ID(), core.PointSelection(1))
	doc.SetSelectionFor(v2.ID(), core.PointSelection(4))

	assert.Equal(t, 1, doc.SelectionFor(v1.ID()).Primary().Head)
	assert.Equal(t, 4, doc.SelectionFor(v2.ID()).Primary().Head)
}

func TestScrollOffsetPerDocument(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.txt")
	b := filepath.Join(tmp, "b.txt")
	assert.NoError(t, os.WriteFile(a, []byte("aaa\nbbb\nccc\n"), 0o644))
	assert.NoError(t, os.WriteFile(b, []byte("xxx\nyyy\n"), 0o644))
	revisited := view.Position{Anchor: 4, VerticalOffset: 2}
	current := view.Position{Anchor: 4, VerticalOffset: 1}
	tests := []struct {
		name        string
		initialPath string
		position    view.Position
		openPaths   []string
		wantOffsets []view.Position
	}{
		{
			name:        "restores revisited offset",
			initialPath: a,
			position:    revisited,
			openPaths:   []string{b, a},
			wantOffsets: []view.Position{{}, revisited},
		},
		{
			name:        "keeps current offset",
			initialPath: b,
			position:    current,
			openPaths:   []string{b},
			wantOffsets: []view.Position{current},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := view.NewEditor(tmp)
			e.ResizeTree(geom.Size{Width: 80, Height: 24})
			v, err := e.OpenFile(tt.initialPath)
			assert.NoError(t, err)
			v.SetOffset(tt.position)
			for i, path := range tt.openPaths {
				_, err = e.OpenFile(path)
				assert.NoError(t, err)
				assert.Equal(t,
					tt.wantOffsets[i], e.FocusedView().Offset(),
				)
			}
		})
	}
}
