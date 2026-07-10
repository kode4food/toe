package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestReflowHardWrap(t *testing.T) {
	t.Run("wraps long line at word boundary", func(t *testing.T) {
		got := core.ReflowHardWrap("one two three four five", 10)
		assert.Equal(t, "one two\nthree four\nfive", got)
	})

	t.Run("breaks long words", func(t *testing.T) {
		got := core.ReflowHardWrap("superlongword", 5)
		assert.Equal(t, "super\nlongw\nord", got)
	})

	t.Run("collapses line breaks and re-wraps", func(t *testing.T) {
		got := core.ReflowHardWrap("one\ntwo three\nfour", 10)
		assert.Equal(t, "one two\nthree four", got)
	})

	t.Run("avoids short final line", func(t *testing.T) {
		text := "This is a demo of the short last line penalty."
		got := core.ReflowHardWrap(text, 37)
		assert.Equal(t, "This is a demo of the short last\nline penalty.", got)
	})

	t.Run("empty string returns empty string", func(t *testing.T) {
		assert.Equal(t, "", core.ReflowHardWrap("", 80))
	})

	t.Run("zero width returns input unchanged", func(t *testing.T) {
		assert.Equal(t, "hello", core.ReflowHardWrap("hello", 0))
	})

	t.Run("single word fits within width", func(t *testing.T) {
		assert.Equal(t, "hello", core.ReflowHardWrap("hello", 80))
	})

	t.Run("all-whitespace collapses to empty", func(t *testing.T) {
		assert.Equal(t, "", core.ReflowHardWrap("   \n  \n  ", 80))
	})

	t.Run("preserves list prefixes", func(t *testing.T) {
		got := core.ReflowHardWrap("* This is my\n  list item.", 20)
		assert.Equal(t, "* This is my list\n  item.", got)
	})

	t.Run("preserves comment prefixes", func(t *testing.T) {
		got := core.ReflowHardWrap("    // foo bar\n    // baz quux", 16)
		assert.Equal(t, "    // foo bar\n    // baz quux", got)
	})

	t.Run("preserves quote prefixes", func(t *testing.T) {
		got := core.ReflowHardWrap("> Memory\n> safety without garbage", 20)
		assert.Equal(t, "> Memory safety\n> without garbage", got)
	})

	t.Run("preserves trailing crlf", func(t *testing.T) {
		got := core.ReflowHardWrap("> foo\r\n> bar\r\n", 20)
		assert.Equal(t, "> foo bar\r\n", got)
	})
}
