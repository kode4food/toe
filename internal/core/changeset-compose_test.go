package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestChangeSetCompose(t *testing.T) {
	// Verify composition by applying each step individually and via compose,
	// then checking both paths produce the same result
	apply := func(
		t *testing.T, doc core.Rope, changes []core.Change,
	) (core.Rope, core.ChangeSet) {
		t.Helper()
		cs, err := core.NewChangeSetFromChanges(doc, changes)
		assert.NoError(t, err)
		out, err := cs.Apply(doc)
		assert.NoError(t, err)
		return out, cs
	}

	t.Run("delete covers previously inserted region", func(t *testing.T) {
		doc := core.NewRope("abc")
		mid, csA := apply(t, doc, []core.Change{core.TextChange(1, 1, "XY")})
		// mid = "aXYbc"
		final, csB := apply(t, mid, []core.Change{core.DeleteChange(1, 3)})
		// final = "abc"

		result, err := csA.Compose(csB).Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, final.String(), result.String())
	})

	t.Run("two sequential replacements", func(t *testing.T) {
		doc := core.NewRope("hello world")
		mid, csA := apply(t, doc, []core.Change{core.TextChange(6, 11, "Go")})
		// mid = "hello Go"
		final, csB := apply(t, mid, []core.Change{core.TextChange(0, 5, "hi")})
		// final = "hi Go"

		result, err := csA.Compose(csB).Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, final.String(), result.String())
	})

	t.Run("delete then insert at same position", func(t *testing.T) {
		doc := core.NewRope("abcde")
		mid, csA := apply(t, doc, []core.Change{core.DeleteChange(1, 3)})
		// mid = "ade"
		final, csB := apply(t, mid, []core.Change{core.TextChange(1, 1, "XY")})
		// final = "aXYde"

		result, err := csA.Compose(csB).Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, final.String(), result.String())
	})

	t.Run("unicode: replace multibyte chars", func(t *testing.T) {
		doc := core.NewRope("hello xz")
		// A: replace " xz" with " test!" then retain nothing
		mid, csA := apply(t, doc, []core.Change{
			core.TextChange(5, 8, " test!"),
		})
		// mid = "hello test!"
		final, csB := apply(t, mid, []core.Change{
			core.DeleteChange(0, 5),
			core.TextChange(6, 11, "世orld"),
		})
		// final = " 世orld"

		result, err := csA.Compose(csB).Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, final.String(), result.String())
	})
}

func TestChangeSetComposeCoverage(t *testing.T) {
	t.Run("partial insert consumed by delete", func(t *testing.T) {
		// A inserts "XYZ" at pos 1; B deletes only 1 of those 3 chars
		doc := core.NewRope("ab")
		csA, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(1, 1, "XYZ"),
		})
		assert.NoError(t, err)
		mid, err := csA.Apply(doc) // "aXYZb"
		assert.NoError(t, err)
		csB, err := core.NewChangeSetFromChanges(mid, []core.Change{
			core.DeleteChange(1, 2),
		})
		assert.NoError(t, err)
		final, err := csB.Apply(mid) // "aYZb"
		assert.NoError(t, err)

		result, err := csA.Compose(csB).Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, final.String(), result.String())
	})

	t.Run("small insert consumed by larger delete", func(t *testing.T) {
		// A inserts "X"; B deletes "Xb" (2 chars, more than A inserted)
		doc := core.NewRope("ab")
		csA, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(1, 1, "X"),
		})
		assert.NoError(t, err)
		mid, err := csA.Apply(doc) // "aXb"
		assert.NoError(t, err)
		csB, err := core.NewChangeSetFromChanges(mid, []core.Change{
			core.DeleteChange(1, 3),
		})
		assert.NoError(t, err)
		final, err := csB.Apply(mid) // "a"
		assert.NoError(t, err)

		result, err := csA.Compose(csB).Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, final.String(), result.String())
	})
}

func TestChangeSetComposeEmpty(t *testing.T) {
	t.Run("empty lhs returns rhs", func(t *testing.T) {
		doc := core.NewRope("hello")
		empty := core.NewChangeSet(doc)
		csB, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(0, 5, "world"),
		})
		assert.NoError(t, err)

		result, err := empty.Compose(csB).Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, "world", result.String())
	})

	t.Run("empty rhs returns lhs", func(t *testing.T) {
		doc := core.NewRope("hello")
		csA, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(0, 5, "world"),
		})
		assert.NoError(t, err)
		mid, err := csA.Apply(doc)
		assert.NoError(t, err)
		empty := core.NewChangeSet(mid)

		result, err := csA.Compose(empty).Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, "world", result.String())
	})
}
