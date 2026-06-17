package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestChars(t *testing.T) {
	t.Run("categorizes line endings", func(t *testing.T) {
		assert.Equal(t, core.CharCategoryEOL, core.CategorizeChar('\n'))
		assert.False(t, core.CharIsLineEnding('\r'))
	})

	t.Run("categorizes whitespace", func(t *testing.T) {
		for _, ch := range "  　   " {
			assert.Equal(t, core.CharCategoryWhitespace,
				core.CategorizeChar(ch),
			)
		}
	})

	t.Run("matches reference whitespace helper", func(t *testing.T) {
		assert.True(t, core.CharIsWhitespace('\t'))
		assert.True(t, core.CharIsWhitespace('\u2009'))
		assert.False(t, core.CharIsWhitespace('\u1680'))
		assert.False(t, core.CharIsWhitespace('a'))
	})

	t.Run("categorizes word characters", func(t *testing.T) {
		text := "_hello_world_あいうえおー1234567890"
		text += "１２３４５６７８９０"
		for _, ch := range text {
			assert.Equal(t, core.CharCategoryWord, core.CategorizeChar(ch))
		}
	})

	t.Run("recognizes word characters", func(t *testing.T) {
		assert.True(t, core.CharIsWord('_'))
		assert.True(t, core.CharIsWord('a'))
		assert.True(t, core.CharIsWord('１'))
		assert.False(t, core.CharIsWord('-'))
	})

	t.Run("categorizes punctuation and symbols", func(t *testing.T) {
		for _, ch := range "!\"#$%&'()*+,-./:;<=>?@[\\]^`{|}~" {
			assert.Equal(t, core.CharCategoryPunctuation,
				core.CategorizeChar(ch),
			)
		}
	})

	t.Run("recognizes punctuation and symbols", func(t *testing.T) {
		assert.True(t, core.CharIsPunctuation('!'))
		assert.True(t, core.CharIsPunctuation('$'))
		assert.False(t, core.CharIsPunctuation('a'))
	})

	t.Run("categorizes unknown characters", func(t *testing.T) {
		assert.Equal(t, core.CharCategoryUnknown,
			core.CategorizeChar('\u0000'),
		)
	})
}
