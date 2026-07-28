package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestHistory(t *testing.T) {
	t.Run("undoes and redoes edits", func(t *testing.T) {
		h := core.NewHistory()
		st := core.State{
			Doc:       core.NewRope("hello"),
			Selection: core.PointSelection(0),
		}

		commit(commitArgs{
			t: t, h: &h, st: &st,
			c: core.TextChange(5, 5, " world!"),
		})
		assert.Equal(t, "hello world!", st.Doc.String())

		commit(commitArgs{
			t: t, h: &h, st: &st,
			c: core.TextChange(6, 11, "世界"),
		})
		assert.Equal(t, "hello 世界!", st.Doc.String())

		applyUndo(t, &h, &st)
		assert.Equal(t, "hello world!", st.Doc.String())

		applyRedo(t, &h, &st)
		assert.Equal(t, "hello 世界!", st.Doc.String())

		applyUndo(t, &h, &st)
		applyUndo(t, &h, &st)
		assert.Equal(t, "hello", st.Doc.String())

		_, ok := h.Undo()
		assert.False(t, ok)
	})

	t.Run("navigates by steps", func(t *testing.T) {
		h, st := historyFixture(t)

		applyAll(t, h.Earlier(core.UndoSteps(3)), &st)
		assert.Equal(t, "a b c d\n", st.Doc.String())

		applyAll(t, h.Later(core.UndoSteps(50)), &st)
		assert.Equal(t, "a f\n", st.Doc.String())
	})

	t.Run("CurrentRevision tracks commits", func(t *testing.T) {
		h := core.NewHistory()
		st := core.State{
			Doc:       core.NewRope("a"),
			Selection: core.PointSelection(0),
		}
		assert.Equal(t, 0, h.CurrentRevision())
		commit(commitArgs{
			t: t, h: &h, st: &st,
			c: core.TextChange(1, 1, "b"),
		})
		assert.Equal(t, 1, h.CurrentRevision())
	})

	t.Run("CommitRevision advances the tip", func(t *testing.T) {
		h := core.NewHistory()
		st := core.State{
			Doc:       core.NewRope("hello"),
			Selection: core.PointSelection(0),
		}
		tx, err := transaction(st.Doc, core.TextChange(5, 5, "!"))
		assert.NoError(t, err)
		assert.NoError(t, h.CommitRevision(tx, st))
		assert.Equal(t, 1, h.CurrentRevision())
	})

	t.Run("LastEditPos returns change position", func(t *testing.T) {
		h := core.NewHistory()
		st := core.State{
			Doc:       core.NewRope("hello"),
			Selection: core.PointSelection(0),
		}
		commit(commitArgs{
			t: t, h: &h, st: &st,
			c: core.TextChange(3, 3, "X"),
		})
		assert.Equal(t, 3, h.LastEditPos())
	})

	t.Run("Earlier negative steps clamps to zero", func(t *testing.T) {
		h, st := historyFixture(t)
		txns := h.Earlier(core.UndoSteps(-5))
		applyAll(t, txns, &st)
		assert.NotEmpty(t, st.Doc.String())
	})

	t.Run("Later negative steps clamps to zero", func(t *testing.T) {
		h, st := historyFixture(t)
		txns := h.Later(core.UndoSteps(-5))
		applyAll(t, txns, &st)
		assert.NotEmpty(t, st.Doc.String())
	})

	t.Run("Redo at tip returns false", func(t *testing.T) {
		h := core.NewHistory()
		_, ok := h.Redo()
		assert.False(t, ok)
	})

	t.Run("branches redo from current revision", func(t *testing.T) {
		h := core.NewHistory()
		st := core.State{
			Doc:       core.NewRope("a"),
			Selection: core.PointSelection(0),
		}
		commit(commitArgs{
			t: t, h: &h, st: &st, c: core.TextChange(1, 1, "b"),
		})
		applyUndo(t, &h, &st)
		commit(commitArgs{
			t: t, h: &h, st: &st, c: core.TextChange(1, 1, "c"),
		})

		applyUndo(t, &h, &st)
		applyRedo(t, &h, &st)

		assert.Equal(t, "ac", st.Doc.String())
	})
}

func historyFixture(t *testing.T) (*core.History, core.State) {
	t.Helper()

	h := core.NewHistory()
	st := core.State{
		Doc:       core.NewRope("a\n"),
		Selection: core.PointSelection(0),
	}
	commit(commitArgs{
		t: t, h: &h, st: &st, c: core.TextChange(1, 1, " b"),
	})
	commit(commitArgs{
		t: t, h: &h, st: &st, c: core.TextChange(3, 3, " c"),
	})
	commit(commitArgs{
		t: t, h: &h, st: &st, c: core.TextChange(5, 5, " d"),
	})
	applyUndo(t, &h, &st)
	commit(commitArgs{
		t: t, h: &h, st: &st, c: core.TextChange(5, 5, " e"),
	})
	applyUndo(t, &h, &st)
	applyUndo(t, &h, &st)
	commit(commitArgs{
		t: t, h: &h, st: &st, c: core.DeleteChange(1, 3),
	})
	commit(commitArgs{
		t: t, h: &h, st: &st, c: core.TextChange(1, 1, " f"),
	})

	return &h, st
}

type commitArgs struct {
	t  *testing.T
	h  *core.History
	st *core.State
	c  core.Change
}

func commit(args commitArgs) {
	args.t.Helper()

	tx, err := transaction(args.st.Doc, args.c)
	assert.NoError(args.t, err)
	err = args.h.CommitRevision(tx, *args.st)
	assert.NoError(args.t, err)
	args.st.Doc, err = tx.Apply(args.st.Doc)
	assert.NoError(args.t, err)
}

func transaction(doc core.Rope, c core.Change) (core.Transaction, error) {
	cs, err := core.NewChangeSetFromChanges(doc, []core.Change{c})
	if err != nil {
		return core.Transaction{}, err
	}
	return core.NewTransaction(doc).WithChanges(cs), nil
}

func applyUndo(t *testing.T, h *core.History, st *core.State) {
	t.Helper()

	tx, ok := h.Undo()
	assert.True(t, ok)
	var err error
	st.Doc, err = tx.Apply(st.Doc)
	assert.NoError(t, err)
}

func applyRedo(t *testing.T, h *core.History, st *core.State) {
	t.Helper()

	tx, ok := h.Redo()
	assert.True(t, ok)
	var err error
	st.Doc, err = tx.Apply(st.Doc)
	assert.NoError(t, err)
}

func applyAll(t *testing.T, txns []core.Transaction, st *core.State) {
	t.Helper()

	var err error
	for _, tx := range txns {
		st.Doc, err = tx.Apply(st.Doc)
		assert.NoError(t, err)
	}
}
