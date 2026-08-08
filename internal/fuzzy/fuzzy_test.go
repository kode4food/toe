package fuzzy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/fuzzy"
)

func TestMatch(t *testing.T) {
	t.Run("prefix matches", func(t *testing.T) {
		res, ok := fuzzy.Match(fuzzy.MatchArgs{Pattern: "ba", Text: "baz"})
		assert.True(t, ok)
		assert.Equal(t, []int{0, 1}, res.Indices)
	})

	t.Run("subsequence matches out of order in text", func(t *testing.T) {
		res, ok := fuzzy.Match(fuzzy.MatchArgs{
			Pattern: "dim", Text: "inactive-dim",
		})
		assert.True(t, ok)
		assert.Equal(t, []int{9, 10, 11}, res.Indices)
	})

	t.Run("case insensitive", func(t *testing.T) {
		_, ok := fuzzy.Match(fuzzy.MatchArgs{
			Pattern: "DIM", Text: "inactive-dim",
		})
		assert.True(t, ok)
	})

	t.Run("no subsequence does not match", func(t *testing.T) {
		_, ok := fuzzy.Match(fuzzy.MatchArgs{
			Pattern: "xyz", Text: "inactive-dim",
		})
		assert.False(t, ok)
	})

	t.Run("empty pattern matches everything", func(t *testing.T) {
		res, ok := fuzzy.Match(fuzzy.MatchArgs{Text: "anything"})
		assert.True(t, ok)
		assert.Empty(t, res.Indices)
	})

	t.Run("exact match scores highest", func(t *testing.T) {
		exact, _ := fuzzy.Match(fuzzy.MatchArgs{Pattern: "dim", Text: "dim"})
		loose, _ := fuzzy.Match(fuzzy.MatchArgs{
			Pattern: "dim", Text: "inactive-dim",
		})
		assert.Greater(t, exact.Score, loose.Score)
	})

	t.Run("Matcher reuses buffers across calls", func(t *testing.T) {
		m := fuzzy.NewMatcher("dim")
		_, ok1 := m.Match("inactive-dim")
		_, ok2 := m.Match("dim")
		_, ok3 := m.Match("nope")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.False(t, ok3)
	})
}
