package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestRopeSliceString(t *testing.T) {
	// build a rope large enough to span multiple leaves, including multi-byte
	// runes, then check SliceString matches Slice().String() across ranges
	src := strings.Repeat("abcdé\tfghij\n", 500)
	r := core.NewRope(src)
	n := r.LenChars()

	ranges := [][2]int{{0, 0}, {0, n}, {0, 7}, {3, 9}, {n - 10, n}, {n / 2, n/2 + 5}}
	for _, rg := range ranges {
		from, to := rg[0], rg[1]
		want, err := r.Slice(from, to)
		assert.NoError(t, err)
		got, err := r.SliceString(from, to)
		assert.NoError(t, err)
		assert.Equal(t, want.String(), got, "range [%d, %d)", from, to)
	}

	_, err := r.SliceString(-1, 3)
	assert.Error(t, err)
	_, err = r.SliceString(5, 3)
	assert.Error(t, err)
	_, err = r.SliceString(0, n+1)
	assert.Error(t, err)
}

func TestRope(t *testing.T) {
	t.Run("handles empty text", func(t *testing.T) {
		r := core.NewRope("")

		assert.Equal(t, "", r.String())
		assert.Equal(t, 0, r.LenChars())
		assert.Equal(t, 1, r.LenLines())

		line, err := r.Line(0)
		assert.NoError(t, err)
		assert.Equal(t, "", line.String())
	})

	t.Run("stores text in balanced leaves", func(t *testing.T) {
		text := strings.Repeat("ab世\n", 700)
		r := core.NewRope(text)

		assert.Equal(t, text, r.String())
		assert.Equal(t, 2800, r.LenChars())
		assert.Equal(t, 701, r.LenLines())
	})

	t.Run("edits both sides of large tree", func(t *testing.T) {
		text := strings.Repeat("a", 1200) + strings.Repeat("b", 1200)
		r := core.NewRope(text)

		next, err := r.Insert(1, "L")
		assert.NoError(t, err)
		next, err = next.Insert(next.LenChars()-1, "R")
		assert.NoError(t, err)
		next, err = next.Delete(1190, 1210)

		assert.NoError(t, err)
		assert.Equal(t, 2382, next.LenChars())
		assert.Equal(t, "aL", next.String()[:2])
		assert.Equal(t, "Rb", next.String()[len(next.String())-2:])
	})

	t.Run("slices by character offsets", func(t *testing.T) {
		r := core.NewRope("a世b界c")

		s, err := r.Slice(1, 4)

		assert.NoError(t, err)
		assert.Equal(t, "世b界", s.String())
		assert.Equal(t, 3, s.LenChars())
	})

	t.Run("slices large text across leaves", func(t *testing.T) {
		text := strings.Repeat("a", 1100) + "世界" + strings.Repeat("b", 1100)
		r := core.NewRope(text)

		s, err := r.Slice(1099, 1103)

		assert.NoError(t, err)
		assert.Equal(t, "a世界b", s.String())
	})

	t.Run("rejects invalid slices", func(t *testing.T) {
		r := core.NewRope("abc")

		_, err := r.Slice(2, 1)
		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))

		_, err = r.Slice(0, 4)
		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})

	t.Run("inserts text without changing original", func(t *testing.T) {
		r := core.NewRope("ab界")

		next, err := r.Insert(2, "世")

		assert.NoError(t, err)
		assert.Equal(t, "ab界", r.String())
		assert.Equal(t, "ab世界", next.String())
	})

	t.Run("deletes text without changing original", func(t *testing.T) {
		r := core.NewRope("ab世界cd")

		next, err := r.Delete(2, 4)

		assert.NoError(t, err)
		assert.Equal(t, "ab世界cd", r.String())
		assert.Equal(t, "abcd", next.String())
	})

	t.Run("rejects invalid edits", func(t *testing.T) {
		r := core.NewRope("abc")

		_, err := r.Insert(4, "x")
		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))

		_, err = r.Delete(-1, 1)
		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})

	t.Run("reads characters by character index", func(t *testing.T) {
		r := core.NewRope("a世b")

		ch, err := r.CharAt(1)

		assert.NoError(t, err)
		assert.Equal(t, '世', ch)

		_, err = r.CharAt(3)
		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})

	t.Run("reads characters across leaves", func(t *testing.T) {
		text := strings.Repeat("a", 1300) + "世" + strings.Repeat("b", 1300)
		r := core.NewRope(text)

		ch, err := r.CharAt(1300)

		assert.NoError(t, err)
		assert.Equal(t, '世', ch)
	})

	t.Run("maps between lines and character offsets", func(t *testing.T) {
		r := core.NewRope("one\ntwo\r\nthree")

		pos, err := r.LineToChar(1)
		assert.NoError(t, err)
		assert.Equal(t, 4, pos)

		pos, err = r.LineToChar(2)
		assert.NoError(t, err)
		assert.Equal(t, 9, pos)

		line, err := r.CharToLine(10)
		assert.NoError(t, err)
		assert.Equal(t, 2, line)
	})

	t.Run("maps nonstandard line endings", func(t *testing.T) {
		r := core.NewRope("one\rtwo\u2028three")

		pos, err := r.LineToChar(1)
		assert.NoError(t, err)
		assert.Equal(t, 4, pos)

		pos, err = r.LineToChar(2)
		assert.NoError(t, err)
		assert.Equal(t, 8, pos)

		line, err := r.CharToLine(9)
		assert.NoError(t, err)
		assert.Equal(t, 2, line)
	})

	t.Run("maps positions across large line sets", func(t *testing.T) {
		r := core.NewRope(strings.Repeat("x\n", 1500) + "tail")

		pos, err := r.LineToChar(1499)
		assert.NoError(t, err)
		assert.Equal(t, 2998, pos)

		line, err := r.CharToLine(3001)
		assert.NoError(t, err)
		assert.Equal(t, 1500, line)
	})

	t.Run("returns lines with line endings", func(t *testing.T) {
		r := core.NewRope("one\ntwo\r\nthree")

		l, err := r.Line(1)

		assert.NoError(t, err)
		assert.Equal(t, "two\r\n", l.String())
	})

	t.Run("returns final line without line ending", func(t *testing.T) {
		r := core.NewRope("one\ntwo")

		l, err := r.Line(1)

		assert.NoError(t, err)
		assert.Equal(t, "two", l.String())
	})

	t.Run("computes line end before line ending", func(t *testing.T) {
		r := core.NewRope("one\ntwo\r\nthree")

		pos, err := r.LineEndCharIndex(1)

		assert.NoError(t, err)
		assert.Equal(t, 7, pos)
	})

	t.Run("computes line end before unicode ending", func(t *testing.T) {
		r := core.NewRope("one\u2028two")

		pos, err := r.LineEndCharIndex(0)

		assert.NoError(t, err)
		assert.Equal(t, 3, pos)
	})

	t.Run("computes final line end", func(t *testing.T) {
		r := core.NewRope("one\ntwo")

		pos, err := r.LineEndCharIndex(1)

		assert.NoError(t, err)
		assert.Equal(t, 7, pos)
	})

	t.Run("rejects invalid line indexes", func(t *testing.T) {
		r := core.NewRope("one")

		_, err := r.Line(1)
		assert.True(t, errors.Is(err, core.ErrRopeLineOutOfRange))

		_, err = r.LineToChar(-1)
		assert.True(t, errors.Is(err, core.ErrRopeLineOutOfRange))

		_, err = r.CharToLine(4)
		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})

	t.Run("ForEachSegment clamps negative from", func(t *testing.T) {
		r := core.NewRope("abcde")
		var got string
		r.ForEachSegment(-5, 3, func(s string) { got += s })
		assert.Equal(t, "abc", got)
	})

	t.Run("ForEachSegment clamps to past end", func(t *testing.T) {
		r := core.NewRope("abcde")
		var got string
		r.ForEachSegment(2, 999, func(s string) { got += s })
		assert.Equal(t, "cde", got)
	})

	t.Run("ForEachSegment empty range calls nothing", func(t *testing.T) {
		r := core.NewRope("abcde")
		var called bool
		r.ForEachSegment(3, 3, func(s string) { called = true })
		assert.False(t, called)
	})

	t.Run("left-rotate rebalances right-heavy tree", func(t *testing.T) {
		// 4097 chars builds a depth-4 rope; inserting at 0 creates a
		// depth-2 vs depth-4 imbalance (diff=2 > maxRopeDepthSkew=1)
		// which triggers rotateRopeLeft
		base := strings.Repeat("a", 4097)
		r := core.NewRope(base)
		next, err := r.Insert(0, "x")
		assert.NoError(t, err)
		assert.Equal(t, 4098, next.LenChars())
		assert.Equal(t, "x", string([]rune(next.String())[:1]))
	})
}
