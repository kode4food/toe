package syntax_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/syntax"
)

func TestFindSurroundPair(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
	// cursor inside "alpha" — innermost pair is argument_list ()
	cursor := strings.Index(src, "alpha") + 2
	parenOpen := strings.Index(src, "(alpha)")
	parenClose := parenOpen + len("(alpha)") - 1
	braceOpen := strings.Index(src, "{")
	braceClose := strings.LastIndex(src, "}")

	t.Run("skip 1 finds innermost pair", func(t *testing.T) {
		r, ok := syntax.FindSurroundPair(src, "go", cursor, 1)
		assert.True(t, ok)
		assert.Equal(t, parenOpen, r.From)
		assert.Equal(t, parenClose, r.To)
	})

	t.Run("skip 2 finds next outer pair", func(t *testing.T) {
		r, ok := syntax.FindSurroundPair(src, "go", cursor, 2)
		assert.True(t, ok)
		assert.Equal(t, braceOpen, r.From)
		assert.Equal(t, braceClose, r.To)
	})

	t.Run("skip beyond depth returns false", func(t *testing.T) {
		_, ok := syntax.FindSurroundPair(src, "go", cursor, 99)
		assert.False(t, ok)
	})

	t.Run("unknown language returns false", func(t *testing.T) {
		_, ok := syntax.FindSurroundPair(src, "nope", cursor, 1)
		assert.False(t, ok)
	})

	t.Run("out-of-bounds cursor returns false", func(t *testing.T) {
		_, ok := syntax.FindSurroundPair(src, "go", -1, 1)
		assert.False(t, ok)
		_, ok = syntax.FindSurroundPair(src, "go", len([]rune(src)), 1)
		assert.False(t, ok)
	})

	t.Run("inside string not matched", func(t *testing.T) {
		strSrc := "package main\n\nfunc main() {\n" +
			"\tx := \"(foo)\"\n\t_ = x\n}\n"
		inStr := strings.Index(strSrc, "(foo)") + 2
		r, ok := syntax.FindSurroundPair(strSrc, "go", inStr, 1)
		assert.True(t, ok)
		// must not match the parens inside the string; first real pair is {}
		brOpen := strings.Index(strSrc, "{")
		brClose := strings.LastIndex(strSrc, "}")
		assert.Equal(t, brOpen, r.From)
		assert.Equal(t, brClose, r.To)
	})
}

func TestFindSurroundPairFor(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
	cursor := strings.Index(src, "alpha") + 2
	parenOpen := strings.Index(src, "(alpha)")
	parenClose := parenOpen + len("(alpha)") - 1
	braceOpen := strings.Index(src, "{")
	braceClose := strings.LastIndex(src, "}")

	t.Run("find paren pair", func(t *testing.T) {
		r, ok := syntax.FindSurroundPairFor(src, "go", cursor, '(', 1)
		assert.True(t, ok)
		assert.Equal(t, parenOpen, r.From)
		assert.Equal(t, parenClose, r.To)
	})

	t.Run("closing bracket also matches pair", func(t *testing.T) {
		r, ok := syntax.FindSurroundPairFor(src, "go", cursor, ')', 1)
		assert.True(t, ok)
		assert.Equal(t, parenOpen, r.From)
		assert.Equal(t, parenClose, r.To)
	})

	t.Run("find brace pair skip 1", func(t *testing.T) {
		r, ok := syntax.FindSurroundPairFor(src, "go", cursor, '{', 1)
		assert.True(t, ok)
		assert.Equal(t, braceOpen, r.From)
		assert.Equal(t, braceClose, r.To)
	})

	t.Run("symmetric char returns false", func(t *testing.T) {
		_, ok := syntax.FindSurroundPairFor(src, "go", cursor, '"', 1)
		assert.False(t, ok)
	})

	t.Run("unknown language returns false", func(t *testing.T) {
		_, ok := syntax.FindSurroundPairFor(src, "nope", cursor, '(', 1)
		assert.False(t, ok)
	})

	t.Run("out-of-bounds cursor returns false", func(t *testing.T) {
		_, ok := syntax.FindSurroundPairFor(src, "go", -1, '(', 1)
		assert.False(t, ok)
	})
}
