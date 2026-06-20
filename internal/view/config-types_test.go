package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestConfigTypes(t *testing.T) {
	t.Run("parses cursor kind", func(t *testing.T) {
		v, err := view.ParseCursorKind("bar")
		assert.NoError(t, err)
		assert.Equal(t, view.CursorKindBar, v)
	})

	t.Run("rejects cursor kind", func(t *testing.T) {
		_, err := view.ParseCursorKind("bad")
		assert.ErrorIs(t, err, view.ErrInvalidCursorKind)
	})

	t.Run("parses line number", func(t *testing.T) {
		v, err := view.ParseLineNumber("absolute")
		assert.NoError(t, err)
		assert.Equal(t, view.LineNumberAbsolute, v)
	})

	t.Run("rejects line number", func(t *testing.T) {
		_, err := view.ParseLineNumber("bad")
		assert.ErrorIs(t, err, view.ErrInvalidLineNumber)
	})

	t.Run("parses bufferline", func(t *testing.T) {
		v, err := view.ParseBufferLine("never")
		assert.NoError(t, err)
		assert.Equal(t, view.BufferLineNever, v)
	})

	t.Run("rejects bufferline", func(t *testing.T) {
		_, err := view.ParseBufferLine("bad")
		assert.ErrorIs(t, err, view.ErrInvalidBufferLine)
	})

	t.Run("parses whitespace render", func(t *testing.T) {
		v, err := view.ParseWhitespaceRenderValue("all")
		assert.NoError(t, err)
		assert.Equal(t, view.WhitespaceRenderAll, v)
	})

	t.Run("rejects whitespace render", func(t *testing.T) {
		_, err := view.ParseWhitespaceRenderValue("bad")
		assert.ErrorIs(t, err, view.ErrInvalidWhitespaceRender)
	})
}
