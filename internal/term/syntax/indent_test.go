package syntax_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/syntax"
)

func TestIndentForNewline(t *testing.T) {
	t.Run("indents after opener", func(t *testing.T) {
		got, ok := syntax.IndentForNewline(
			core.NewRope("if ok {"), "go", 0, 7, core.Tabs(),
		)
		assert.True(t, ok)
		assert.Equal(t, "\t", got)
	})

	t.Run("ignores opener in string", func(t *testing.T) {
		got, ok := syntax.IndentForNewline(
			core.NewRope("package main\nvar s = \"{\""),
			"go", 1, 10, core.Tabs(),
		)
		assert.True(t, ok)
		assert.Equal(t, "", got)
	})

	t.Run("outdents language keyword", func(t *testing.T) {
		got, ok := syntax.IndentForNewline(
			core.NewRope("    else {"), "javascript", 0, 10,
			core.Spaces(4),
		)
		assert.True(t, ok)
		assert.Equal(t, "    ", got)
	})
}
