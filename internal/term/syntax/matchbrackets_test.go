package syntax_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/syntax"
)

func TestFindMatchingBracket(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"

	parenOpen := strings.Index(src, "(alpha)")
	parenClose := parenOpen + len("(alpha)") - 1
	braceOpen := strings.Index(src, "{")
	braceClose := strings.LastIndex(src, "}")

	t.Run("open paren finds close paren", func(t *testing.T) {
		pos, ok := syntax.FindMatchingBracket(src, "go", parenOpen)
		assert.True(t, ok)
		assert.Equal(t, parenClose, pos)
	})

	t.Run("close paren finds open paren", func(t *testing.T) {
		pos, ok := syntax.FindMatchingBracket(src, "go", parenClose)
		assert.True(t, ok)
		assert.Equal(t, parenOpen, pos)
	})

	t.Run("open brace finds close brace", func(t *testing.T) {
		pos, ok := syntax.FindMatchingBracket(src, "go", braceOpen)
		assert.True(t, ok)
		assert.Equal(t, braceClose, pos)
	})

	t.Run("close brace finds open brace", func(t *testing.T) {
		pos, ok := syntax.FindMatchingBracket(src, "go", braceClose)
		assert.True(t, ok)
		assert.Equal(t, braceOpen, pos)
	})

	t.Run("non-bracket position returns false", func(t *testing.T) {
		_, ok := syntax.FindMatchingBracket(src, "go", 0)
		assert.False(t, ok)
	})

	t.Run("bracket inside string not matched", func(t *testing.T) {
		strSrc := `package main

var x = "(not a bracket)"
`
		openInStr := strings.Index(strSrc, "(not")
		_, ok := syntax.FindMatchingBracket(strSrc, "go", openInStr)
		assert.False(t, ok)
	})

	t.Run("unknown language returns false", func(t *testing.T) {
		_, ok := syntax.FindMatchingBracket(src, "nope", parenOpen)
		assert.False(t, ok)
	})

	t.Run("out of range cursor returns false", func(t *testing.T) {
		_, ok := syntax.FindMatchingBracket(src, "go", len([]rune(src)))
		assert.False(t, ok)
	})
}
