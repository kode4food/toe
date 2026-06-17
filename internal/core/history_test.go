package core_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

type commitArgs struct {
	t    *testing.T
	h    *core.History
	st   *core.State
	c    core.Change
	when time.Time
}

func TestHistory(t *testing.T) {
	t.Run("undoes and redoes edits", func(t *testing.T) {
		h := core.NewHistory()
		st := core.State{
			Doc:       core.NewRope("hello"),
			Selection: core.PointSelection(0),
		}

		commit(commitArgs{
			t: t, h: &h, st: &st,
			c: core.TextChange(5, 5, " world!"), when: time.Now(),
		})
		assert.Equal(t, "hello world!", st.Doc.String())

		commit(commitArgs{
			t: t, h: &h, st: &st,
			c: core.TextChange(6, 11, "世界"), when: time.Now(),
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

	t.Run("navigates by time", func(t *testing.T) {
		h, st := historyFixture(t)

		applyAll(t, h.Earlier(core.UndoSteps(4)), &st)
		assert.Equal(t, "a b c\n", st.Doc.String())

		applyAll(t, h.Later(core.UndoTimePeriod(5*time.Second)), &st)
		assert.Equal(t, "a b c d\n", st.Doc.String())

		applyAll(t, h.Later(core.UndoTimePeriod(6*time.Second)), &st)
		assert.Equal(t, "a b c e\n", st.Doc.String())
	})

	t.Run("branches redo from current revision", func(t *testing.T) {
		h := core.NewHistory()
		st := core.State{
			Doc:       core.NewRope("a"),
			Selection: core.PointSelection(0),
		}
		now := time.Now()
		commit(commitArgs{
			t: t, h: &h, st: &st, c: core.TextChange(1, 1, "b"), when: now,
		})
		applyUndo(t, &h, &st)
		commit(commitArgs{
			t: t, h: &h, st: &st, c: core.TextChange(1, 1, "c"), when: now,
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
	t0 := time.Now()

	commit(commitArgs{
		t: t, h: &h, st: &st, c: core.TextChange(1, 1, " b"), when: t0,
	})
	commit(commitArgs{
		t: t, h: &h, st: &st,
		c: core.TextChange(3, 3, " c"), when: t0.Add(10 * time.Second),
	})
	commit(commitArgs{
		t: t, h: &h, st: &st,
		c: core.TextChange(5, 5, " d"), when: t0.Add(20 * time.Second),
	})
	applyUndo(t, &h, &st)
	commit(commitArgs{
		t: t, h: &h, st: &st,
		c: core.TextChange(5, 5, " e"), when: t0.Add(30 * time.Second),
	})
	applyUndo(t, &h, &st)
	applyUndo(t, &h, &st)
	commit(commitArgs{
		t: t, h: &h, st: &st,
		c: core.DeleteChange(1, 3), when: t0.Add(40 * time.Second),
	})
	commit(commitArgs{
		t: t, h: &h, st: &st,
		c: core.TextChange(1, 1, " f"), when: t0.Add(50 * time.Second),
	})

	return &h, st
}

func commit(args commitArgs) {
	args.t.Helper()

	tx, err := transaction(args.st.Doc, args.c)
	assert.NoError(args.t, err)
	err = args.h.CommitRevisionAt(tx, *args.st, args.when)
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
