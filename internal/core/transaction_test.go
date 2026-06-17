package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestChangeSet(t *testing.T) {
	t.Run("builds operations from ordered changes", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(1, 3, "XX"),
			core.DeleteChange(4, 5),
		})

		assert.NoError(t, err)
		assert.Equal(t, 6, cs.Len())
		assert.Equal(t, 5, cs.LenAfter())
		assert.Equal(t, []core.OperationKind{
			core.OperationRetain,
			core.OperationInsert,
			core.OperationDelete,
			core.OperationRetain,
			core.OperationDelete,
			core.OperationRetain,
		}, opKinds(cs.Operations()))
		assert.Equal(t, 2, cs.Operations()[1].LenChars())
		assert.Equal(t, "XX", cs.Operations()[1].Text())
	})

	t.Run("merges adjacent operation builders", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(1, 2, "X"),
			core.TextChange(2, 3, "Y"),
		})

		assert.NoError(t, err)
		assert.Equal(t, []core.OperationKind{
			core.OperationRetain,
			core.OperationInsert,
			core.OperationDelete,
			core.OperationRetain,
		}, opKinds(cs.Operations()))
		assert.Equal(t, "XY", cs.Operations()[1].Text())
		assert.Equal(t, 2, cs.Operations()[2].LenChars())
	})

	t.Run("rejects unordered and out of range changes", func(t *testing.T) {
		doc := core.NewRope("abc")

		_, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.DeleteChange(2, 1),
		})
		assert.True(t, errors.Is(err, core.ErrChangeOrder))

		_, err = core.NewChangeSetFromChanges(doc, []core.Change{
			core.DeleteChange(1, 4),
		})
		assert.True(t, errors.Is(err, core.ErrChangeOutOfRange))
	})

	t.Run("applies replacements and deletions", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(1, 3, "XX"),
			core.DeleteChange(4, 5),
		})
		assert.NoError(t, err)

		out, err := cs.Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, "aXXdf", out.String())
	})

	t.Run("iterates changes", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(1, 3, "XX"),
			core.DeleteChange(4, 5),
		})
		assert.NoError(t, err)

		changes := cs.Changes()

		assert.Equal(t, 2, len(changes))
		assert.Equal(t, 1, changes[0].From)
		assert.Equal(t, 3, changes[0].To)
		assert.Equal(t, "XX", changes[0].Text())
		assert.Equal(t, core.DeleteChange(4, 5), changes[1])
	})

	t.Run("rejects different snapshot length", func(t *testing.T) {
		doc := core.NewRope("abc")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.DeleteChange(1, 2),
		})
		assert.NoError(t, err)

		_, err = cs.Apply(core.NewRope("abcd"))

		assert.True(t, errors.Is(err, core.ErrChangeSetLengthMismatch))
	})

	t.Run("inverts applied changes", func(t *testing.T) {
		doc := core.NewRope("ab世界cd")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(2, 4, "XY"),
		})
		assert.NoError(t, err)
		out, err := cs.Apply(doc)
		assert.NoError(t, err)

		inv, err := cs.Invert(doc)
		assert.NoError(t, err)
		back, err := inv.Apply(out)

		assert.NoError(t, err)
		assert.Equal(t, doc.String(), back.String())
	})

	t.Run("maps positions through inserts and deletes", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(2, 4, "XYZ"),
		})
		assert.NoError(t, err)

		pos, err := cs.MapPos(2, core.AssocBefore)
		assert.NoError(t, err)
		assert.Equal(t, 2, pos)

		pos, err = cs.MapPos(2, core.AssocAfter)
		assert.NoError(t, err)
		assert.Equal(t, 5, pos)

		pos, err = cs.MapPos(3, core.AssocAfter)
		assert.NoError(t, err)
		assert.Equal(t, 5, pos)
	})

	t.Run("maps word-associated positions", func(t *testing.T) {
		doc := core.NewRope("ab")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(1, 1, "xy-z"),
		})
		assert.NoError(t, err)

		pos, err := cs.MapPos(1, core.AssocAfterWord)
		assert.NoError(t, err)
		assert.Equal(t, 3, pos)

		pos, err = cs.MapPos(1, core.AssocBeforeWord)
		assert.NoError(t, err)
		assert.Equal(t, 4, pos)
	})

	t.Run("reports sticky association variants", func(t *testing.T) {
		assert.True(t, core.AssocBeforeSticky.Sticky())
		assert.True(t, core.AssocAfterSticky.Sticky())
		assert.False(t, core.AssocAfter.Sticky())
	})

	t.Run("maps ranges and selections", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(1, 1, "XX"),
		})
		assert.NoError(t, err)
		s := core.SingleSelection(2, 4)

		mapped, err := s.Map(cs)

		assert.NoError(t, err)
		assert.Equal(t, []core.Range{core.NewRange(4, 6)}, mapped.Ranges())
	})

	t.Run("maps point and backward ranges", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(2, 2, "XX"),
		})
		assert.NoError(t, err)

		r, err := cs.MapRange(core.PointRange(2))
		assert.NoError(t, err)
		assert.Equal(t, core.PointRange(4), r)

		r, err = cs.MapRange(core.NewRange(5, 1))
		assert.NoError(t, err)
		assert.Equal(t, core.NewRange(7, 1), r)
	})

	t.Run("returns selection map errors", func(t *testing.T) {
		doc := core.NewRope("abc")
		cs := core.NewChangeSet(doc)
		s := core.SingleSelection(1, 4)

		_, err := s.Map(cs)

		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})

	t.Run("rejects mapping out of bounds positions", func(t *testing.T) {
		cs := core.NewChangeSet(core.NewRope("abc"))

		_, err := cs.MapPos(4, core.AssocAfter)

		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})
}

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

	t.Run("insert then delete covers inserted region", func(t *testing.T) {
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
		mid, csA := apply(t, doc, []core.Change{core.TextChange(5, 8, " test!")})
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
}

func TestTransaction(t *testing.T) {
	t.Run("applies empty transaction", func(t *testing.T) {
		doc := core.NewRope("abc")
		tx := core.NewTransaction(doc)

		out, err := tx.Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, "abc", out.String())
	})

	t.Run("applies and inverts changes", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		cs, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(2, 4, "YY"),
		})
		assert.NoError(t, err)
		tx := core.NewTransaction(doc).WithChanges(cs)

		out, err := tx.Apply(doc)
		assert.NoError(t, err)
		inv, err := tx.Invert(doc)
		assert.NoError(t, err)
		back, err := inv.Apply(out)

		assert.NoError(t, err)
		assert.Equal(t, "abYYef", out.String())
		assert.Equal(t, doc.String(), back.String())
	})

	t.Run("other selection wins on compose", func(t *testing.T) {
		doc := core.NewRope("abcdef")
		csA, err := core.NewChangeSetFromChanges(doc, []core.Change{
			core.TextChange(2, 4, "YY"),
		})
		assert.NoError(t, err)
		txA := core.NewTransaction(doc).WithChanges(csA)
		mid, err := txA.Apply(doc)
		assert.NoError(t, err)
		csB, err := core.NewChangeSetFromChanges(mid, []core.Change{
			core.DeleteChange(0, 2),
		})
		assert.NoError(t, err)
		sel := core.SingleSelection(1, 3)
		txB := core.NewTransaction(mid).WithChanges(csB).WithSelection(sel)

		composed := txA.Compose(txB)
		result, err := composed.Apply(doc)

		assert.NoError(t, err)
		assert.Equal(t, txB.Selection(), composed.Selection())
		_ = result
	})

	t.Run("stores explicit selection", func(t *testing.T) {
		doc := core.NewRope("abc")
		s := core.SingleSelection(1, 2)
		tx := core.NewTransaction(doc).WithSelection(s)

		assert.Equal(t, &s, tx.Selection())
		assert.True(t, tx.Changes().Empty())
	})

	t.Run("returns invert errors", func(t *testing.T) {
		doc := core.NewRope("abc")
		tx := core.NewTransaction(doc)

		_, err := tx.Invert(core.NewRope("abcd"))

		assert.True(t, errors.Is(err, core.ErrChangeSetLengthMismatch))
	})
}

func opKinds(ops []core.Operation) []core.OperationKind {
	out := make([]core.OperationKind, len(ops))
	for i, op := range ops {
		out[i] = op.Kind()
	}
	return out
}
