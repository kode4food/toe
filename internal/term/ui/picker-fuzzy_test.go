package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/ui"
)

func TestPickerFuzzy(t *testing.T) {
	base := ui.NewPickerBase("test", []string{"path"}, 0, nil)
	match := func(query, path string) (int, []int, bool) {
		return base.Match(query, ui.PickerItem{Display: path})
	}
	best := func(query string, paths ...string) string {
		out, score := "", 0
		for i, p := range paths {
			s, _, ok := match(query, p)
			if ok && (i == 0 || s > score) {
				out, score = p, s
			}
		}
		return out
	}

	t.Run("prefers name over path", func(t *testing.T) {
		as := assert.New(t)
		paths := []string{
			"internal/view/action/selection-lines.go",
			"internal/core/rope-balance.go",
			"internal/lsp/capabilities.go",
			"internal/term/ale.go",
		}
		for _, q := range []string{"ale", "ale.go", "alego", "term/ale"} {
			as.Equal("internal/term/ale.go", best(q, paths...), "query %q", q)
		}
	})

	t.Run("indices land on the name", func(t *testing.T) {
		as := assert.New(t)
		_, idx, ok := match("ale", "internal/term/ale.go")
		as.True(ok)
		as.Equal([]int{14, 15, 16}, idx)
	})

	t.Run("indices count runes", func(t *testing.T) {
		as := assert.New(t)
		_, idx, ok := match("go", "é/go.mod")
		as.True(ok)
		as.Equal([]int{2, 3}, idx)
	})

	t.Run("case insensitive", func(t *testing.T) {
		as := assert.New(t)
		_, _, ok := match("ALE", "internal/term/ale.go")
		as.True(ok)
		as.Equal("fooBarBaz.go", best("fbb", "fizzbuzz.go", "fooBarBaz.go"))
	})

	t.Run("rejects non-matches", func(t *testing.T) {
		as := assert.New(t)
		_, _, ok := match("zzz", "internal/term/ale.go")
		as.False(ok)
		_, _, ok = match("elа", "ale.go")
		as.False(ok)
	})
}
