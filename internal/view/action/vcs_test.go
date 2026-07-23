package action_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

// stubVC serves canned diff state for one document
type stubVC struct {
	hunks []view.DiffHunk
	base  string
}

var _ view.VersionControl = (*stubVC)(nil)

func (s *stubVC) DiffHunks(*view.Document) []view.DiffHunk {
	return s.hunks
}

func (s *stubVC) DiffBase(*view.Document) (string, bool) {
	return s.base, s.base != ""
}

func (s *stubVC) DiffHunksForPath(string) []view.DiffHunk {
	return s.hunks
}

func (s *stubVC) DiffBaseForPath(string) (string, bool) {
	return s.base, s.base != ""
}

func (s *stubVC) HeadName(*view.Document) (string, bool) {
	return "", false
}

func (s *stubVC) ChangedFiles() ([]view.FileChange, error) {
	return nil, nil
}

func (s *stubVC) Refresh() {}

func (s *stubVC) Updates() <-chan struct{} {
	return nil
}

func TestGotoChange(t *testing.T) {
	// doc lines: 0 "one", 1 "CHANGED", 2 "three", 3 "ADDED", 4 ""
	setup := func(t *testing.T) *view.Editor {
		e := testutil.EditorWithText(t, "one\nCHANGED\nthree\nADDED\n")
		e.SetVersionControl(&stubVC{
			base: "one\ntwo\nthree\n",
			hunks: []view.DiffHunk{
				{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
				{BaseFrom: 3, BaseTo: 3, From: 3, To: 4},
			},
		})
		return e
	}

	t.Run("next change selects first hunk", func(t *testing.T) {
		e := setup(t)
		testutil.SetCursor(t, e, 0)
		action.GotoNextChange(e)
		// hunk 1 covers chars [4,12); forward cursor sits before head
		assert.Equal(t, 11, testutil.CursorPos(t, e))
	})

	t.Run("next change skips current hunk", func(t *testing.T) {
		e := setup(t)
		testutil.SetCursor(t, e, 5) // inside line 1
		action.GotoNextChange(e)
		// hunk 2 covers chars [18,24)
		assert.Equal(t, 23, testutil.CursorPos(t, e))
	})

	t.Run("prev change moves back", func(t *testing.T) {
		e := setup(t)
		testutil.SetCursor(t, e, 19) // line 3
		action.GotoPrevChange(e)
		// backward range puts the cursor at the hunk start
		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("first and last change", func(t *testing.T) {
		e := setup(t)
		testutil.SetCursor(t, e, 19)
		action.GotoFirstChange(e)
		assert.Equal(t, 11, testutil.CursorPos(t, e))
		action.GotoLastChange(e)
		assert.Equal(t, 23, testutil.CursorPos(t, e))
	})

	t.Run("no version control sets status", func(t *testing.T) {
		e := testutil.EditorWithText(t, "one\n")
		testutil.SetCursor(t, e, 0)
		action.GotoNextChange(e)
		assert.Equal(t, 0, testutil.CursorPos(t, e))
		assert.Equal(t,
			i18n.Text(i18n.StatusDiffUnavailable), e.TakeStatusMsg())
	})

	t.Run("count jumps over hunks", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nB\nc\nD\ne\nF\n")
		e.SetVersionControl(&stubVC{
			base: "a\nb\nc\nd\ne\nf\n",
			hunks: []view.DiffHunk{
				{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
				{BaseFrom: 3, BaseTo: 4, From: 3, To: 4},
				{BaseFrom: 5, BaseTo: 6, From: 5, To: 6},
			},
		})
		testutil.SetCursor(t, e, 0)
		e.SetCount(2)

		action.GotoNextChange(e)

		assert.Equal(t, 7, testutil.CursorPos(t, e))
	})

	t.Run("select mode extends to hunk", func(t *testing.T) {
		e := setup(t)
		testutil.SetCursor(t, e, 0)
		e.SetMode(view.ModeSelect)

		action.GotoNextChange(e)

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		r := doc.SelectionFor(v.ID()).Primary()
		assert.Equal(t, 0, r.Anchor)
		assert.Equal(t, 12, r.Head)
	})

	t.Run("prev finds pure removal", func(t *testing.T) {
		e := testutil.EditorWithText(t, "one\nthree\nfour\n")
		e.SetVersionControl(&stubVC{
			base: "one\ntwo\nthree\nfour\n",
			hunks: []view.DiffHunk{
				{BaseFrom: 1, BaseTo: 2, From: 1, To: 1},
			},
		})
		testutil.SetCursor(t, e, 11)

		action.GotoPrevChange(e)

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("empty hunks keep cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "one\n")
		e.SetVersionControl(&stubVC{})
		testutil.SetCursor(t, e, 2)

		action.GotoNextChange(e)
		action.GotoFirstChange(e)
		action.GotoLastChange(e)

		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})
}

func TestResetDiffChange(t *testing.T) {
	t.Run("reverts hunk under cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "one\nCHANGED\nthree\n")
		e.SetVersionControl(&stubVC{
			base: "one\ntwo\nthree\n",
			hunks: []view.DiffHunk{
				{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
			},
		})
		testutil.SetCursor(t, e, 5) // line 1
		n, err := action.ResetDiffChange(e)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "one\ntwo\nthree\n", doc.Text().String())
	})

	t.Run("reverts pure removal", func(t *testing.T) {
		e := testutil.EditorWithText(t, "one\nthree\n")
		e.SetVersionControl(&stubVC{
			base: "one\ntwo\nthree\n",
			hunks: []view.DiffHunk{
				{BaseFrom: 1, BaseTo: 2, From: 1, To: 1},
			},
		})
		testutil.SetCursor(t, e, 4) // line 1
		n, err := action.ResetDiffChange(e)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "one\ntwo\nthree\n", doc.Text().String())
	})

	t.Run("reverts multiple selected hunks", func(t *testing.T) {
		e := testutil.EditorWithText(t, "one\nCHANGED\nthree\nADDED\n")
		e.SetVersionControl(&stubVC{
			base: "one\ntwo\nthree\n",
			hunks: []view.DiffHunk{
				{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
				{BaseFrom: 3, BaseTo: 3, From: 3, To: 4},
			},
		})
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(4, 24),
		}, 0)

		n, err := action.ResetDiffChange(e)

		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "one\ntwo\nthree\n", doc.Text().String())
	})

	t.Run("errors when selection has no changes", func(t *testing.T) {
		e := testutil.EditorWithText(t, "one\nCHANGED\nthree\n")
		e.SetVersionControl(&stubVC{
			base: "one\ntwo\nthree\n",
			hunks: []view.DiffHunk{
				{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
			},
		})
		testutil.SetCursor(t, e, 0) // line 0, outside the hunk
		_, err := action.ResetDiffChange(e)
		assert.True(t, errors.Is(err, action.ErrNoChangesInSelection))
	})

	t.Run("errors without version control", func(t *testing.T) {
		e := testutil.EditorWithText(t, "one\n")
		_, err := action.ResetDiffChange(e)
		assert.True(t, errors.Is(err, action.ErrDiffUnavailable))
	})
}
