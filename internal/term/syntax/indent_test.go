package syntax_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/syntax"
)

func TestIndentForNewline(t *testing.T) {
	t.Run("indents after opener", func(t *testing.T) {
		got, ok := syntax.IndentForNewline(syntax.IndentForNewlineArgs{
			Text:  core.NewRope("if ok {"),
			Lang:  "go",
			Line:  0,
			Pos:   7,
			Style: core.Tabs(),
		})
		assert.True(t, ok)
		assert.Equal(t, "\t", got)
	})

	t.Run("ignores opener in string", func(t *testing.T) {
		got, ok := syntax.IndentForNewline(syntax.IndentForNewlineArgs{
			Text:  core.NewRope("package main\nvar s = \"{\""),
			Lang:  "go",
			Line:  1,
			Pos:   10,
			Style: core.Tabs(),
		})
		assert.True(t, ok)
		assert.Equal(t, "", got)
	})

	t.Run("outdents language keyword", func(t *testing.T) {
		got, ok := syntax.IndentForNewline(syntax.IndentForNewlineArgs{
			Text:  core.NewRope("    else {"),
			Lang:  "javascript",
			Line:  0,
			Pos:   10,
			Style: core.Spaces(4),
		})
		assert.True(t, ok)
		assert.Equal(t, "    ", got)
	})
}
