package vcs_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/vcs"
	"github.com/kode4food/toe/internal/view"
)

func TestDiff(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		r := core.NewRope("a\nb\nc\n")
		assert.Empty(t, vcs.Diff(r, r))
	})

	t.Run("modification", func(t *testing.T) {
		base := core.NewRope("a\nb\nc\n")
		doc := core.NewRope("a\nx\nc\n")
		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
		}, vcs.Diff(base, doc))
	})

	t.Run("pure insertion", func(t *testing.T) {
		base := core.NewRope("a\nc\n")
		doc := core.NewRope("a\nb\nc\n")
		hunks := vcs.Diff(base, doc)
		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 1, BaseTo: 1, From: 1, To: 2},
		}, hunks)
		assert.True(t, hunks[0].PureInsertion())
		assert.False(t, hunks[0].PureRemoval())
	})

	t.Run("pure removal", func(t *testing.T) {
		base := core.NewRope("a\nb\nc\n")
		doc := core.NewRope("a\nc\n")
		hunks := vcs.Diff(base, doc)
		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 1, BaseTo: 2, From: 1, To: 1},
		}, hunks)
		assert.True(t, hunks[0].PureRemoval())
	})

	t.Run("multiple hunks sorted", func(t *testing.T) {
		base := core.NewRope("a\nb\nc\nd\ne\n")
		doc := core.NewRope("a\nx\nc\nd\ny\ne\n")
		hunks := vcs.Diff(base, doc)
		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
			{BaseFrom: 4, BaseTo: 4, From: 4, To: 5},
		}, hunks)
	})

	t.Run("missing final newline differs", func(t *testing.T) {
		base := core.NewRope("a\nb\n")
		doc := core.NewRope("a\nb")
		assert.NotEmpty(t, vcs.Diff(base, doc))
	})

	t.Run("oversized input yields no hunks", func(t *testing.T) {
		big := core.NewRope(strings.Repeat("a\n", vcs.MaxDiffLines+1))
		small := core.NewRope("b\n")
		assert.Nil(t, vcs.Diff(big, small))
		assert.Nil(t, vcs.Diff(small, big))
	})

	t.Run("empty base is one insertion", func(t *testing.T) {
		base := core.NewRope("")
		doc := core.NewRope("a\nb\n")
		hunks := vcs.Diff(base, doc)
		assert.Len(t, hunks, 1)
		assert.True(t, hunks[0].PureInsertion())
	})
}
