package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

// wordTest encodes one word-motion test case
type wordTest struct {
	text     string
	count    int
	inRange  core.Range
	outRange core.Range
}

func TestMoveHorizontally(t *testing.T) {
	t.Run("moves forward by count graphemes", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		r := core.PointRange(0)
		r = r.MoveHorizontally(doc, core.DirectionForward, 3, core.MovementMove)
		assert.Equal(t, 3, r.Head)
		assert.Equal(t, 3, r.Anchor)
	})

	t.Run("moves backward by count graphemes", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		r := core.PointRange(5)
		r = r.MoveHorizontally(doc, core.DirectionBackward, 2, core.MovementMove)
		assert.Equal(t, 3, r.Head)
		assert.Equal(t, 3, r.Anchor)
	})

	t.Run("extend keeps anchor fixed", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		r := core.PointRange(0)
		r = r.MoveHorizontally(doc, core.DirectionForward, 3, core.MovementExtend)
		assert.Equal(t, 0, r.Anchor)
		assert.Equal(t, 4, r.Head) // next grapheme boundary after pos 3
	})

	t.Run("clamps at document start", func(t *testing.T) {
		doc := core.NewRope("abc")
		r := core.PointRange(0)
		r = r.MoveHorizontally(doc, core.DirectionBackward, 999, core.MovementMove)
		assert.Equal(t, 0, r.Head)
	})

	t.Run("clamps at document end", func(t *testing.T) {
		doc := core.NewRope("abc")
		r := core.PointRange(3)
		r = r.MoveHorizontally(doc, core.DirectionForward, 999, core.MovementMove)
		assert.Equal(t, 3, r.Head)
	})
}

func TestRangeSliceAndFragment(t *testing.T) {
	t.Run("slice returns correct sub-rope", func(t *testing.T) {
		doc := core.NewRope("hello world")
		r := core.NewRange(6, 11)
		s, err := r.Slice(doc)
		assert.NoError(t, err)
		assert.Equal(t, "world", s.String())
	})

	t.Run("fragment returns string", func(t *testing.T) {
		doc := core.NewRope("hello world")
		r := core.NewRange(0, 5)
		f, err := r.Fragment(doc)
		assert.NoError(t, err)
		assert.Equal(t, "hello", f)
	})

	t.Run("grapheme aligned snaps to boundaries", func(t *testing.T) {
		doc := core.NewRope("abc")
		r := core.NewRange(1, 2)
		aligned := r.GraphemeAligned(doc)
		assert.Equal(t, 1, aligned.Anchor)
		assert.Equal(t, 2, aligned.Head)
	})

	t.Run("expands empty range by one grapheme", func(t *testing.T) {
		doc := core.NewRope("abc")
		r := core.PointRange(1)
		r2 := r.MinWidth1(doc)
		assert.Equal(t, 1, r2.Anchor)
		assert.Equal(t, 2, r2.Head)
	})

	t.Run("min_width_1 non-empty range unchanged", func(t *testing.T) {
		doc := core.NewRope("abc")
		r := core.NewRange(1, 3)
		r2 := r.MinWidth1(doc)
		assert.Equal(t, r, r2)
	})
}

func TestRangeCursorAndPutCursor(t *testing.T) {
	t.Run("cursor of point range is head", func(t *testing.T) {
		doc := core.NewRope("abcde")
		r := core.PointRange(2)
		assert.Equal(t, 2, r.Cursor(doc))
	})

	t.Run("forward range cursor before head", func(t *testing.T) {
		doc := core.NewRope("abcde")
		r := core.NewRange(1, 3)
		assert.Equal(t, 2, r.Cursor(doc))
	})

	t.Run("put_cursor move collapses to point", func(t *testing.T) {
		doc := core.NewRope("abcde")
		r := core.NewRange(1, 3)
		r2 := r.PutCursor(doc, 2, false)
		assert.Equal(t, 2, r2.Anchor)
		assert.Equal(t, 2, r2.Head)
	})

	t.Run("put_cursor extend expands selection", func(t *testing.T) {
		doc := core.NewRope("abcde")
		r := core.PointRange(1)
		r2 := r.PutCursor(doc, 3, true)
		assert.Equal(t, 1, r2.Anchor)
		assert.True(t, r2.Head > r2.Anchor)
	})

	t.Run("cursor_line returns correct line", func(t *testing.T) {
		doc := core.NewRope("abc\ndef")
		r := core.PointRange(5)
		line, err := r.CursorLine(doc)
		assert.NoError(t, err)
		assert.Equal(t, 1, line)
	})
}

func TestMoveNextWordStart(t *testing.T) {
	// test_behaviour_when_moving_to_start_of_next_words
	tests := []wordTest{
		{
			"Basic forward motion stops at the first space",
			1, core.NewRange(0, 0), core.NewRange(0, 6),
		},
		{
			" Starting from a boundary advances the anchor",
			1, core.NewRange(0, 0), core.NewRange(1, 10),
		},
		{
			"Long       whitespace gap is bridged by the head",
			1, core.NewRange(0, 0), core.NewRange(0, 11),
		},
		{
			"Previous anchor is irrelevant for forward motions",
			1, core.NewRange(12, 0), core.NewRange(0, 9),
		},
		{
			"    Starting from whitespace moves to last space in sequence",
			1, core.NewRange(0, 0), core.NewRange(0, 4),
		},
		{
			"Starting from mid-word leaves anchor at start position and moves head",
			1, core.NewRange(3, 3), core.NewRange(3, 9),
		},
		{
			"Identifiers_with_underscores are considered a single word",
			1, core.NewRange(0, 0), core.NewRange(0, 29),
		},
		{
			"Multiple motions at once resolve correctly",
			3, core.NewRange(0, 0), core.NewRange(17, 20),
		},
		{
			"",
			1, core.NewRange(0, 0), core.NewRange(0, 0),
		},
		{
			"ヒーリクス multibyte characters behave as normal characters",
			1, core.NewRange(0, 0), core.NewRange(0, 6),
		},
	}

	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MoveNextWordStart(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMovePrevWordStart(t *testing.T) {
	// test_behaviour_when_moving_to_start_of_previous_words
	tests := []wordTest{
		{
			"Basic backward motion from the middle of a word",
			1, core.NewRange(3, 3), core.NewRange(4, 0),
		},
		{
			"    Jump to start of a word preceded by whitespace",
			1, core.NewRange(5, 5), core.NewRange(6, 4),
		},
		{
			"Identifiers_with_underscores are considered a single word",
			1, core.NewRange(0, 20), core.NewRange(20, 0),
		},
		{
			"",
			1, core.NewRange(0, 0), core.NewRange(0, 0),
		},
		{
			"\n\n\n\n\n",
			1, core.NewRange(5, 5), core.NewRange(0, 0),
		},
		{
			"Multiple motions at once resolve correctly",
			3, core.NewRange(18, 18), core.NewRange(9, 0),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MovePrevWordStart(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMoveNextWordEnd(t *testing.T) {
	tests := []wordTest{
		{
			"Basic forward motion from the start of a word to the end of it",
			1, core.NewRange(0, 0), core.NewRange(0, 5),
		},
		{
			"Basic forward motion from the end of a word to the end of the next",
			1, core.NewRange(0, 5), core.NewRange(5, 13),
		},
		{
			"Identifiers_with_underscores are considered a single word",
			1, core.NewRange(0, 0), core.NewRange(0, 28),
		},
		{
			"",
			1, core.NewRange(0, 0), core.NewRange(0, 0),
		},
		{
			"Multiple motions at once resolve correctly",
			3, core.NewRange(0, 0), core.NewRange(16, 19),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MoveNextWordEnd(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMoveNextLongWordStart(t *testing.T) {
	tests := []wordTest{
		{
			"Basic forward motion stops at the first space",
			1, core.NewRange(0, 0), core.NewRange(0, 6),
		},
		{
			"alphanumeric.!,and.?=punctuation are not treated any " +
				"differently than alphanumerics",
			1, core.NewRange(0, 0), core.NewRange(0, 33),
		},
		{
			"",
			1, core.NewRange(0, 0), core.NewRange(0, 0),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MoveNextLongWordStart(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMovePrevLongWordStart(t *testing.T) {
	tests := []wordTest{
		{
			"Basic backward motion from the middle of a word",
			1, core.NewRange(3, 3), core.NewRange(4, 0),
		},
		{
			"",
			1, core.NewRange(0, 0), core.NewRange(0, 0),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MovePrevLongWordStart(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMoveNextSubWordStart(t *testing.T) {
	tests := []wordTest{
		{
			"NextSubwordStart",
			1, core.NewRange(0, 0), core.NewRange(0, 4),
		},
		{
			"NextSubwordStart",
			1, core.NewRange(4, 4), core.NewRange(4, 11),
		},
		{
			"next_subword_start",
			1, core.NewRange(0, 0), core.NewRange(0, 5),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MoveNextSubWordStart(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMoveNextSubWordEnd(t *testing.T) {
	tests := []wordTest{
		{
			"NextSubwordEnd",
			1, core.NewRange(0, 0), core.NewRange(0, 4),
		},
		{
			"NextSubwordEnd",
			1, core.NewRange(4, 4), core.NewRange(4, 11),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MoveNextSubWordEnd(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMovePrevSubWordStart(t *testing.T) {
	tests := []wordTest{
		{
			"PrevSubwordEnd",
			1, core.NewRange(13, 13), core.NewRange(14, 11),
		},
		{
			"PrevSubwordEnd",
			1, core.NewRange(11, 11), core.NewRange(11, 4),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MovePrevSubWordStart(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMovePrevSubWordEnd(t *testing.T) {
	tests := []wordTest{
		{
			"PrevSubwordEnd",
			1, core.NewRange(13, 13), core.NewRange(14, 11),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MovePrevSubWordEnd(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMovePrevWordEnd(t *testing.T) {
	tests := []wordTest{
		{
			"Basic backward word end motion",
			1, core.NewRange(4, 4), core.NewRange(5, 0),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MovePrevWordEnd(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMoveNextLongWordEnd(t *testing.T) {
	tests := []wordTest{
		{
			"Basic forward motion stops at end",
			1, core.NewRange(0, 0), core.NewRange(0, 5),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MoveNextLongWordEnd(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMovePrevLongWordEnd(t *testing.T) {
	tests := []wordTest{
		// Starting from a space, PrevLongWordEnd moves to just before the word
		// and the boundary anchor is snapped
		{
			"Basic backward long word end",
			1, core.NewRange(5, 5), core.NewRange(5, 0),
		},
	}
	for _, tc := range tests {
		doc := core.NewRope(tc.text)
		got := core.MovePrevLongWordEnd(doc, tc.inRange, tc.count)
		assert.Equal(t, tc.outRange, got)
	}
}

func TestMoveVertically(t *testing.T) {
	doc := core.NewRope("one\ntwo\nthree\n")

	t.Run("down from first line", func(t *testing.T) {
		r := core.PointRange(0)
		got := r.MoveVertically(doc, core.DirectionForward, 1, core.MovementMove)
		assert.Equal(t, 4, got.Head)
	})

	t.Run("down preserves column", func(t *testing.T) {
		r := core.PointRange(2) // col 2 of "one"
		got := r.MoveVertically(doc, core.DirectionForward, 1, core.MovementMove)
		assert.Equal(t, 6, got.Head) // col 2 of "two"
	})

	t.Run("down clamps to shorter line", func(t *testing.T) {
		r := core.PointRange(10) // col 2 of "three"
		// move down to the empty last line
		got := r.MoveVertically(doc, core.DirectionForward, 1, core.MovementMove)
		// last line after trailing \n is empty, cursor at start
		assert.Equal(t, doc.LenChars(), got.Head)
	})

	t.Run("up from second line", func(t *testing.T) {
		r := core.PointRange(4) // start of "two"
		got := r.MoveVertically(doc, core.DirectionBackward, 1, core.MovementMove)
		assert.Equal(t, 0, got.Head)
	})

	t.Run("up from first line stays", func(t *testing.T) {
		r := core.PointRange(1)
		got := r.MoveVertically(doc, core.DirectionBackward, 1, core.MovementMove)
		// column preserved at col 1 on line 0
		assert.Equal(t, 1, got.Head)
	})

	t.Run("down multi count", func(t *testing.T) {
		r := core.PointRange(0)
		got := r.MoveVertically(doc, core.DirectionForward, 2, core.MovementMove)
		assert.Equal(t, 8, got.Head) // start of "three"
	})
}

func TestMoveVerticallyVisual(t *testing.T) {
	// viewport 10 wide, tab 4, no wrap indicator, no indent retain
	vf := &core.VisualMoveFormat{
		ViewportWidth:    10,
		TabWidth:         4,
		MaxIndentRetain:  0,
		WrapIndicatorLen: 0,
	}
	// "0123456789ab\ncd" — first line wraps at col 10, "ab" is row 1
	// row 0: "0123456789"  (chars 0-9)
	// row 1: "ab"           (chars 10-11), \n at 12
	// line 1: "cd"          (chars 13-14)
	t.Run("text-line fallback when SoftWrap off", func(t *testing.T) {
		doc := core.NewRope("aaa\nbbb\nccc")
		r := core.PointRange(0)
		got := (*core.VisualMoveFormat)(nil).MoveVerticallyVisual(
			doc, r, core.DirectionForward, 1,
		)
		assert.Equal(t, 4, got.Head) // start of "bbb"
	})

	t.Run("moves within wrapped line", func(t *testing.T) {
		doc := core.NewRope("0123456789ab\ncd")
		r := core.PointRange(0) // col 0, row 0 of line 0
		got := vf.MoveVerticallyVisual(doc, r, core.DirectionForward, 1)
		assert.Equal(t, 10, got.Head) // row 1 of line 0, col 0
	})

	t.Run("moves from wrapped row to next text line", func(t *testing.T) {
		doc := core.NewRope("0123456789ab\ncd")
		r := core.PointRange(10) // row 1 of line 0
		got := vf.MoveVerticallyVisual(doc, r, core.DirectionForward, 1)
		assert.Equal(t, 13, got.Head) // "c" in line 1
	})

	t.Run("moves up within wrapped line", func(t *testing.T) {
		doc := core.NewRope("0123456789ab\ncd")
		r := core.PointRange(10) // row 1 of line 0
		got := vf.MoveVerticallyVisual(doc, r, core.DirectionBackward, 1)
		assert.Equal(t, 0, got.Head) // back to row 0 of line 0
	})

	t.Run("moves up to previous wrapped row", func(t *testing.T) {
		doc := core.NewRope("0123456789ab\ncd")
		r := core.PointRange(13) // "c" in line 1
		got := vf.MoveVerticallyVisual(doc, r, core.DirectionBackward, 1)
		// line 0 last row is row 1 (chars 10-11), so col 0 → char 10
		assert.Equal(t, 10, got.Head)
	})

	t.Run("clamps at first line", func(t *testing.T) {
		doc := core.NewRope("abc\ndef")
		vf2 := &core.VisualMoveFormat{ViewportWidth: 80, TabWidth: 4}
		r := core.PointRange(0)
		got := vf2.MoveVerticallyVisual(doc, r, core.DirectionBackward, 5)
		assert.Equal(t, 0, got.Head)
	})

	t.Run("extend movement preserves anchor", func(t *testing.T) {
		doc := core.NewRope("aaa\nbbb\nccc")
		vf2 := &core.VisualMoveFormat{ViewportWidth: 80, TabWidth: 4}
		r := core.NewRange(0, 0)
		got := vf2.ExtendVerticallyVisual(doc, r, core.DirectionForward, 1)
		assert.Equal(t, 0, got.Anchor)
		// PutCursor with extend sets Head=charIdx+1 for forward ranges
		assert.Equal(t, 5, got.Head)
		assert.Equal(t, 4, got.Cursor(doc))
	})
}

func TestVisualRowStarts(t *testing.T) {
	t.Run("wraps whole words at boundaries", func(t *testing.T) {
		vf := &core.VisualMoveFormat{
			ViewportWidth: 20, TabWidth: 4, MaxWrap: 5,
		}
		// "alpha bravo charlie delta": row 0 holds through the trailing space
		// at col 20; "delta" moves to row 1 whole rather than breaking
		got := vf.VisualRowStarts([]rune("alpha bravo charlie delta"))
		assert.Equal(t, []int{20}, got)
	})

	t.Run("breaks a word wider than max-wrap", func(t *testing.T) {
		vf := &core.VisualMoveFormat{
			ViewportWidth: 10, TabWidth: 4, MaxWrap: 2,
		}
		// "supercalifragilistic" cannot move whole, so it fills the row and
		// breaks at the viewport edge
		got := vf.VisualRowStarts([]rune("supercalifragilistic"))
		assert.Equal(t, []int{10}, got)
	})

	t.Run("carries indent onto wrapped rows", func(t *testing.T) {
		vf := &core.VisualMoveFormat{
			ViewportWidth: 14, TabWidth: 4, MaxWrap: 5,
			MaxIndentRetain: 40, WrapIndicatorLen: 0,
		}
		// four-space indent + "alpha bravo gamma"; continuation rows begin at
		// col 4, so fewer content columns are available per wrapped row
		runes := []rune("    alpha bravo gamma")
		got := vf.VisualRowStarts(runes)
		// row 0: "    alpha " (cols 0-9, offsets 0-9)
		// row 1: indent(4) + "bravo " (offsets 10-15)
		// row 2: indent(4) + "gamma" (offsets 16-20)
		assert.Equal(t, []int{10, 16}, got)
	})

	t.Run("single short line does not wrap", func(t *testing.T) {
		vf := &core.VisualMoveFormat{ViewportWidth: 80, TabWidth: 4, MaxWrap: 20}
		assert.Empty(t, vf.VisualRowStarts([]rune("short line")))
	})
}

func TestMoveVerticallyVisualIndented(t *testing.T) {
	vf := &core.VisualMoveFormat{
		ViewportWidth: 14, TabWidth: 4, MaxWrap: 5,
		MaxIndentRetain: 40, WrapIndicatorLen: 0,
	}
	doc := core.NewRope("    alpha bravo gamma")

	t.Run("down keeps visual column", func(t *testing.T) {
		// cursor on "a" of "alpha" at offset 4 (visual col 4). Moving down to
		// row 1 (which begins at col 4 after the carried indent) should land on
		// the first content char of row 1, "bravo" at offset 10 (also col 4)
		r := core.PointRange(4)
		got := vf.MoveVerticallyVisual(doc, r, core.DirectionForward, 1)
		assert.Equal(t, 10, got.Head)
	})

	t.Run("down then up round-trips", func(t *testing.T) {
		r := core.PointRange(12) // "a" in "bravo" on row 1
		up := vf.MoveVerticallyVisual(doc, r, core.DirectionBackward, 1)
		down := vf.MoveVerticallyVisual(doc, up, core.DirectionForward, 1)
		assert.Equal(t, 12, down.Head)
	})
}

func TestVisualRows(t *testing.T) {
	vf := &core.VisualMoveFormat{ViewportWidth: 10, TabWidth: 4, MaxWrap: 3}
	doc := core.NewRope("short\nlonger than ten chars here\nend")

	t.Run("single-row line returns 1", func(t *testing.T) {
		assert.Equal(t, 1, vf.VisualRows(doc, 0))
	})

	t.Run("long line returns more than 1", func(t *testing.T) {
		assert.Greater(t, vf.VisualRows(doc, 1), 1)
	})

	t.Run("nil format always returns 1", func(t *testing.T) {
		var nilvf *core.VisualMoveFormat
		assert.Equal(t, 1, nilvf.VisualRows(doc, 0))
	})

	t.Run("empty line returns 1", func(t *testing.T) {
		docEmpty := core.NewRope("a\n\nb")
		assert.Equal(t, 1, vf.VisualRows(docEmpty, 1))
	})

	t.Run("tab-indented line wraps", func(t *testing.T) {
		docTab := core.NewRope("\t\t\tlong content here")
		assert.GreaterOrEqual(t, vf.VisualRows(docTab, 0), 1)
	})
}

func TestVisualRowOfOffset(t *testing.T) {
	vf := &core.VisualMoveFormat{ViewportWidth: 5, TabWidth: 4, MaxWrap: 3}
	doc := core.NewRope("abcdefghij\nend")

	t.Run("offset 0 is row 0", func(t *testing.T) {
		assert.Equal(t, 0, vf.VisualRowOfOffset(doc, 0, 0))
	})

	t.Run("offset past wrap is row 1", func(t *testing.T) {
		assert.Equal(t, 1, vf.VisualRowOfOffset(doc, 0, 7))
	})

	t.Run("nil format always returns 0", func(t *testing.T) {
		var nilvf *core.VisualMoveFormat
		assert.Equal(t, 0, nilvf.VisualRowOfOffset(doc, 0, 5))
	})
}

func TestVisualScrollUp(t *testing.T) {
	vf := &core.VisualMoveFormat{ViewportWidth: 10, TabWidth: 4, MaxWrap: 3}
	doc := core.NewRope("short\nlonger than ten chars\nend")

	t.Run("scroll up within row stays on same line", func(t *testing.T) {
		line, row := vf.VisualScrollUp(doc, 1, 2, 1)
		assert.Equal(t, 1, line)
		assert.Equal(t, 1, row)
	})

	t.Run("scroll up to previous line", func(t *testing.T) {
		line, row := vf.VisualScrollUp(doc, 1, 0, 1)
		assert.Equal(t, 0, line)
		assert.GreaterOrEqual(t, row, 0)
	})

	t.Run("clamps at document start", func(t *testing.T) {
		line, row := vf.VisualScrollUp(doc, 0, 0, 10)
		assert.Equal(t, 0, line)
		assert.Equal(t, 0, row)
	})
}
