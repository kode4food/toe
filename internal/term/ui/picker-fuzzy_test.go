package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/ui"
)

func TestPickerFuzzy(t *testing.T) {
	base := ui.NewPickerBase("test", []string{"path"}, 0, nil)
	match := func(query, path string) (ui.MatchResult, bool) {
		return base.PrepareMatcher(query)(&ui.PickerItem{Display: path})
	}
	best := func(query string, paths ...string) string {
		out, score := "", 0
		for i, p := range paths {
			res, ok := match(query, p)
			if ok && (i == 0 || res.Score > score) {
				out, score = p, res.Score
			}
		}
		return out
	}

	t.Run("prefers name over path", func(t *testing.T) {
		paths := []string{
			"internal/view/action/selection-lines.go",
			"internal/core/rope-balance.go",
			"internal/lsp/capabilities.go",
			"internal/term/ale.go",
		}
		for _, q := range []string{"ale", "ale.go", "alego", "term/ale"} {
			assert.Equal(t, "internal/term/ale.go", best(q, paths...))
		}
	})

	t.Run("indices land on the name", func(t *testing.T) {
		res, ok := match("ale", "internal/term/ale.go")
		assert.True(t, ok)
		assert.Equal(t, []int{14, 15, 16}, res.Indices)
	})

	t.Run("indices count runes", func(t *testing.T) {
		res, ok := match("go", "é/go.mod")
		assert.True(t, ok)
		assert.Equal(t, []int{2, 3}, res.Indices)
	})

	t.Run("case insensitive", func(t *testing.T) {
		_, ok := match("ALE", "internal/term/ale.go")
		assert.True(t, ok)
		camel := best("fbb", "fizzbuzz.go", "fooBarBaz.go")
		assert.Equal(t, "fooBarBaz.go", camel)
	})

	t.Run("rejects non-matches", func(t *testing.T) {
		_, ok := match("zzz", "internal/term/ale.go")
		assert.False(t, ok)
		_, ok = match("elа", "ale.go")
		assert.False(t, ok)
	})
}
