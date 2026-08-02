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
			Left: []view.StatusLineItem{
				{Element: view.StatusLineMode},
			},
		}}
		assert.Equal(t,
			[]view.StatusLineItem{
				{Element: view.StatusLineMode},
			}, o.StatusLineLeft(),
		)
	})

	t.Run("right default elements returned", func(t *testing.T) {
		o := view.Options{}
		right := o.StatusLineRight()
		assert.NotEmpty(t, right)
	})

	t.Run("left default includes spinner", func(t *testing.T) {
		o := view.Options{}
		assert.Contains(t, o.StatusLineLeft(),
			view.StatusLineItem{Element: view.StatusLineSpinner})
	})

	t.Run("right default includes file encoding", func(t *testing.T) {
		o := view.Options{}
		assert.Contains(t, o.StatusLineRight(),
			view.StatusLineItem{Element: view.StatusLineFileEncoding})
	})

	t.Run("right custom overrides default", func(t *testing.T) {
		o := view.Options{StatusLine: view.StatusLine{
			Right: []view.StatusLineItem{{Element: view.StatusLinePosition}},
		}}
		assert.Equal(t,
			[]view.StatusLineItem{{Element: view.StatusLinePosition}},
			o.StatusLineRight())
	})
}

func TestOptionsRulers(t *testing.T) {
	o := view.Options{}
	o.SetRulers([]int{120, 80, 120, 80})
	assert.Equal(t, []int{80, 120}, o.Rulers)
}

func TestOptionsCursorShape(t *testing.T) {
	t.Run("default returns block", func(t *testing.T) {
		o := view.Options{}
		assert.Equal(t,
			view.CursorKindBlock, o.CursorShapeForMode(view.ModeNormal))
		assert.Equal(t,
			view.CursorKindBlock, o.CursorShapeForMode(view.ModeInsert))
		assert.Equal(t,
			view.CursorKindBlock, o.CursorShapeForMode(view.ModeSelect))
	})

	t.Run("custom normal shape", func(t *testing.T) {
		o := view.Options{CursorShape: view.CursorShape{
			Normal: view.CursorKindBar,
		}}
		assert.Equal(t,
			view.CursorKindBar, o.CursorShapeForMode(view.ModeNormal))
	})

	t.Run("custom insert shape", func(t *testing.T) {
		o := view.Options{CursorShape: view.CursorShape{
			Insert: view.CursorKindBar,
		}}
		assert.Equal(t,
			view.CursorKindBar, o.CursorShapeForMode(view.ModeInsert))
	})

	t.Run("custom select shape", func(t *testing.T) {
		o := view.Options{CursorShape: view.CursorShape{
			Select: view.CursorKindUnderline,
		}}
		assert.Equal(t,
			view.CursorKindUnderline, o.CursorShapeForMode(view.ModeSelect),
		)
	})
}
