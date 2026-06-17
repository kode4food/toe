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

	t.Run("no split of long words", func(t *testing.T) {
		got := core.ReflowHardWrap("superlongword", 5)
		assert.Equal(t, "superlongword", got)
	})

	t.Run("collapses line breaks and re-wraps", func(t *testing.T) {
		got := core.ReflowHardWrap("one\ntwo three\nfour", 10)
		assert.Equal(t, "one two\nthree four", got)
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
}
