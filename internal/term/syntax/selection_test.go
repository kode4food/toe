package syntax_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/syntax"
)

func TestSelection(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
	idFrom := strings.Index(src, "alpha")
	idTo := idFrom + len("alpha")
	cursor := idFrom + 1

	t.Run("expand point selects leaf", func(t *testing.T) {
		res, ok := syntax.ExpandSelection(syntax.SelectionArgs{
			Text:   src,
			Lang:   "go",
			Cursor: cursor,
			Range:  syntax.Range{From: cursor, To: cursor},
		})
		assert.True(t, ok)
		assert.Equal(t, idFrom, res.From)
		assert.Equal(t, idTo, res.To)
	})

	t.Run("expand range selects parent", func(t *testing.T) {
		res, ok := syntax.ExpandSelection(syntax.SelectionArgs{
			Text:   src,
			Lang:   "go",
			Cursor: cursor,
			Range:  syntax.Range{From: idFrom, To: idTo},
		})
		assert.True(t, ok)
		assert.Less(t, res.From, idFrom)
		assert.Greater(t, res.To, idTo)
	})

	t.Run("shrink range selects child", func(t *testing.T) {
		callFrom := strings.Index(src, "println")
		callTo := strings.Index(src, ")\n") + 1
		res, ok := syntax.ShrinkSelection(syntax.SelectionArgs{
			Text:   src,
			Lang:   "go",
			Cursor: cursor,
			Range:  syntax.Range{From: callFrom, To: callTo},
		})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, res.From, callFrom)
		assert.LessOrEqual(t, res.To, callTo)
		assert.Less(t, res.To-res.From, callTo-callFrom)
	})

	t.Run("unknown language noops", func(t *testing.T) {
		_, ok := syntax.ExpandSelection(syntax.SelectionArgs{
			Text:   src,
			Lang:   "nope",
			Cursor: cursor,
			Range:  syntax.Range{From: cursor, To: cursor},
		})
		assert.False(t, ok)
	})
}
