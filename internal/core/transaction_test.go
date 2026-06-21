package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

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
