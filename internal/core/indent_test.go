package core_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestIndentStyle(t *testing.T) {
	t.Run("Tabs reports as tabs", func(t *testing.T) {
		s := core.Tabs()
		assert.True(t, s.IsTabs())
		assert.Equal(t, uint8(0), s.Width())
	})

	t.Run("Spaces reports width", func(t *testing.T) {
		s := core.Spaces(4)
		assert.False(t, s.IsTabs())
		assert.Equal(t, uint8(4), s.Width())
	})

	t.Run("Spaces(0) clamps to 1", func(t *testing.T) {
		s := core.Spaces(0)
		assert.Equal(t, uint8(1), s.Width())
	})

	t.Run("Spaces over MaxIndent clamps to 1", func(t *testing.T) {
		s := core.Spaces(255)
		assert.Equal(t, uint8(1), s.Width())
	})
}

func TestParseIndentStyle(t *testing.T) {
	t.Run("empty string gives tabs", func(t *testing.T) {
		assert.True(t, core.ParseIndentStyle("").IsTabs())
	})

	t.Run("tab string gives tabs", func(t *testing.T) {
		assert.True(t, core.ParseIndentStyle("\t").IsTabs())
	})

	t.Run("four-space string gives Spaces(4)", func(t *testing.T) {
		s := core.ParseIndentStyle("    ")
		assert.False(t, s.IsTabs())
		assert.Equal(t, uint8(4), s.Width())
	})

	t.Run("two-space string gives Spaces(2)", func(t *testing.T) {
		s := core.ParseIndentStyle("  ")
		assert.Equal(t, uint8(2), s.Width())
	})

	t.Run("clamps to MaxIndent", func(t *testing.T) {
		s := core.ParseIndentStyle(strings.Repeat(" ", 20))
		assert.Equal(t, uint8(core.MaxIndent), s.Width())
	})
}

func TestAsStr(t *testing.T) {
	t.Run("Tabs() returns tab character", func(t *testing.T) {
		assert.Equal(t, "\t", core.Tabs().AsStr())
	})

	t.Run("Spaces(4) returns four spaces", func(t *testing.T) {
		assert.Equal(t, "    ", core.Spaces(4).AsStr())
	})

	t.Run("Spaces(2) returns two spaces", func(t *testing.T) {
		assert.Equal(t, "  ", core.Spaces(2).AsStr())
	})
}

func TestIndentWidth(t *testing.T) {
	t.Run("Tabs() uses tabWidth", func(t *testing.T) {
		assert.Equal(t, 4, core.Tabs().IndentWidth(4))
	})

	t.Run("Spaces(4) ignores tabWidth", func(t *testing.T) {
		assert.Equal(t, 4, core.Spaces(4).IndentWidth(8))
	})

	t.Run("Spaces(2) returns 2", func(t *testing.T) {
		assert.Equal(t, 2, core.Spaces(2).IndentWidth(4))
	})
}

func TestLevelForLine(t *testing.T) {
	t.Run("eight spaces at width 4 is level 2", func(t *testing.T) {
		line := core.NewRope("        fn new")
		assert.Equal(t, 2, core.LevelForLine(line, 4, 4))
	})

	t.Run("three tabs is level 3", func(t *testing.T) {
		line := core.NewRope("\t\t\tfn new")
		assert.Equal(t, 3, core.LevelForLine(line, 4, 4))
	})

	t.Run("tab+four-spaces+tab is level 3", func(t *testing.T) {
		line := core.NewRope("\t    \tfn new")
		assert.Equal(t, 3, core.LevelForLine(line, 4, 4))
	})

	t.Run("sixteen spaces at width 16 is level 1", func(t *testing.T) {
		line := core.NewRope(strings.Repeat(" ", 16) + "x")
		assert.Equal(t, 1, core.LevelForLine(line, 16, 16))
	})

	t.Run("thirty-two spaces at width 16 is level 2", func(t *testing.T) {
		line := core.NewRope(strings.Repeat(" ", 32) + "x")
		assert.Equal(t, 2, core.LevelForLine(line, 16, 16))
	})

	t.Run("zero indentWidth returns 0", func(t *testing.T) {
		line := core.NewRope("    x")
		assert.Equal(t, 0, core.LevelForLine(line, 4, 0))
	})

	t.Run("no indent is level 0", func(t *testing.T) {
		line := core.NewRope("fn new")
		assert.Equal(t, 0, core.LevelForLine(line, 4, 4))
	})
}

func TestAutoDetect(t *testing.T) {
	t.Run("detects tab indentation", func(t *testing.T) {
		doc := core.NewRope(
			"func foo() {\n\tbar()\n\tbaz()\n\t\tqux()\n}",
		)
		style, ok := core.AutoDetect(doc)
		assert.True(t, ok)
		assert.True(t, style.IsTabs())
	})

	t.Run("detects four-space indentation", func(t *testing.T) {
		doc := core.NewRope(
			"func foo() {\n    bar()\n    baz()\n        qux()\n}",
		)
		style, ok := core.AutoDetect(doc)
		assert.True(t, ok)
		assert.False(t, style.IsTabs())
		assert.Equal(t, uint8(4), style.Width())
	})

	t.Run("detects two-space indentation", func(t *testing.T) {
		doc := core.NewRope(
			"func foo() {\n  bar()\n  baz()\n    qux()\n}",
		)
		style, ok := core.AutoDetect(doc)
		assert.True(t, ok)
		assert.False(t, style.IsTabs())
		assert.Equal(t, uint8(2), style.Width())
	})

	t.Run("returns false for unindented doc", func(t *testing.T) {
		doc := core.NewRope("foo\nbar\nbaz\n")
		_, ok := core.AutoDetect(doc)
		assert.False(t, ok)
	})

	t.Run("returns false for single-line doc", func(t *testing.T) {
		doc := core.NewRope("hello")
		_, ok := core.AutoDetect(doc)
		assert.False(t, ok)
	})
}
