package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestOptionsStatusLine(t *testing.T) {
	t.Run("separator default", func(t *testing.T) {
		o := view.Options{}
		assert.Equal(t, "│", o.StatusLineSeparator())
	})

	t.Run("separator custom", func(t *testing.T) {
		o := view.Options{StatusLine: view.StatusLine{Separator: "|"}}
		assert.Equal(t, "|", o.StatusLineSeparator())
	})

	t.Run("left default elements returned", func(t *testing.T) {
		o := view.Options{}
		left := o.StatusLineLeft()
		assert.NotEmpty(t, left)
	})

	t.Run("left custom overrides default", func(t *testing.T) {
		o := view.Options{StatusLine: view.StatusLine{
			Left: []view.StatusLineElement{view.StatusLineMode},
		}}
		assert.Equal(t,
			[]view.StatusLineElement{view.StatusLineMode}, o.StatusLineLeft(),
		)
	})

	t.Run("center returns clone of elements", func(t *testing.T) {
		o := view.Options{StatusLine: view.StatusLine{
			Center: []view.StatusLineElement{view.StatusLineFileName},
		}}
		assert.Equal(t,
			[]view.StatusLineElement{view.StatusLineFileName},
			o.StatusLineCenter())
	})

	t.Run("center empty returns empty", func(t *testing.T) {
		o := view.Options{}
		assert.Empty(t, o.StatusLineCenter())
	})

	t.Run("right default elements returned", func(t *testing.T) {
		o := view.Options{}
		right := o.StatusLineRight()
		assert.NotEmpty(t, right)
	})

	t.Run("right custom overrides default", func(t *testing.T) {
		o := view.Options{StatusLine: view.StatusLine{
			Right: []view.StatusLineElement{view.StatusLinePosition},
		}}
		assert.Equal(t,
			[]view.StatusLineElement{view.StatusLinePosition},
			o.StatusLineRight())
	})
}

func TestOptionsModeNames(t *testing.T) {
	t.Run("default normal", func(t *testing.T) {
		o := view.Options{}
		assert.Equal(t, "NOR", o.ModeNameForMode("NOR"))
	})

	t.Run("default insert", func(t *testing.T) {
		o := view.Options{}
		assert.Equal(t, "INS", o.ModeNameForMode("insert"))
	})

	t.Run("default select via sel", func(t *testing.T) {
		o := view.Options{}
		assert.Equal(t, "SEL", o.ModeNameForMode("sel"))
	})

	t.Run("custom normal", func(t *testing.T) {
		o := view.Options{StatusLine: view.StatusLine{
			Mode: view.StatusLineModeNames{Normal: "NRM"},
		}}
		assert.Equal(t, "NRM", o.ModeNameForMode("normal"))
	})

	t.Run("custom insert", func(t *testing.T) {
		o := view.Options{StatusLine: view.StatusLine{
			Mode: view.StatusLineModeNames{Insert: "I"},
		}}
		assert.Equal(t, "I", o.ModeNameForMode("ins"))
	})

	t.Run("custom select", func(t *testing.T) {
		o := view.Options{StatusLine: view.StatusLine{
			Mode: view.StatusLineModeNames{Select: "S"},
		}}
		assert.Equal(t, "S", o.ModeNameForMode("select"))
	})
}

func TestOptionsCursorShape(t *testing.T) {
	t.Run("default returns block", func(t *testing.T) {
		o := view.Options{}
		assert.Equal(t, view.CursorKindBlock, o.CursorShapeForMode("NOR"))
		assert.Equal(t, view.CursorKindBlock, o.CursorShapeForMode("INS"))
		assert.Equal(t, view.CursorKindBlock, o.CursorShapeForMode("SEL"))
	})

	t.Run("custom normal shape", func(t *testing.T) {
		o := view.Options{CursorShape: view.CursorShape{
			Normal: view.CursorKindBar,
		}}
		assert.Equal(t, view.CursorKindBar, o.CursorShapeForMode("NOR"))
	})

	t.Run("custom insert shape", func(t *testing.T) {
		o := view.Options{CursorShape: view.CursorShape{
			Insert: view.CursorKindBar,
		}}
		assert.Equal(t, view.CursorKindBar, o.CursorShapeForMode("INS"))
	})

	t.Run("custom select shape", func(t *testing.T) {
		o := view.Options{CursorShape: view.CursorShape{
			Select: view.CursorKindUnderline,
		}}
		assert.Equal(t, view.CursorKindUnderline, o.CursorShapeForMode("SEL"))
	})
}
