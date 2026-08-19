package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestGotoPath(t *testing.T) {
	t.Run("keeps the open document's buffer", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "a.txt")
		assert.NoError(t, os.WriteFile(path, []byte("AAA\n"), 0o644))

		e := view.NewEditor(dir)
		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		testutil.SetEditorText(t, e, "edited ")

		landed, ok := ui.GotoPath(e, path, nil, ui.PickerAcceptReplace)

		assert.True(t, ok)
		assert.Equal(t, v.DocID(), landed.DocID())
		assert.Equal(t,
			"edited AAA\n", e.FocusedDocument().Text().String(),
		)
	})

	t.Run("records a jump back to the departed file", func(t *testing.T) {
		dir := t.TempDir()
		pathA := filepath.Join(dir, "a.txt")
		pathB := filepath.Join(dir, "b.txt")
		assert.NoError(t, os.WriteFile(pathA, []byte("AAA\n"), 0o644))
		assert.NoError(t, os.WriteFile(pathB, []byte("BBB\n"), 0o644))

		e := view.NewEditor(dir)
		_, err := e.OpenFile(pathA)
		assert.NoError(t, err)
		docA := e.FocusedDocument().ID()

		_, ok := ui.GotoPath(e, pathB, nil, ui.PickerAcceptReplace)
		assert.True(t, ok)
		assert.NotEqual(t, docA, e.FocusedDocument().ID())

		action.JumpBackward(e)

		assert.Equal(t, docA, e.FocusedDocument().ID())
	})

	t.Run("places the selector's selection", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "a.txt")
		assert.NoError(t, os.WriteFile(path, []byte("one\ntwo\n"), 0o644))

		e := view.NewEditor(dir)
		lines := &core.Span{From: 1, To: 1}
		v, ok := ui.GotoPath(
			e, path, ui.GotoLines(lines), ui.PickerAcceptReplace,
		)

		assert.True(t, ok)
		doc := e.Document(v.DocID())
		sel := doc.SelectionFor(v.ID())
		line, err := sel.Primary().CursorLine(doc.Text())
		assert.NoError(t, err)
		assert.Equal(t, 1, line)
	})

	t.Run("empty path reports failure", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		_, ok := ui.GotoPath(e, "", nil, ui.PickerAcceptReplace)
		assert.False(t, ok)
	})
}

func TestGotoJump(t *testing.T) {
	t.Run("keeps history either side of the entry", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef\nghi\njkl")
		v := e.FocusedView()
		doc := e.FocusedDocument()
		for _, at := range []int{0, 4, 8} {
			v.PushJump(doc.ID(), at, core.PointSelection(at))
		}

		_, ok := ui.GotoJump(e, 0, ui.PickerAcceptReplace)

		assert.True(t, ok)
		assert.Len(t, v.Jumps(), 3)
		assert.Equal(t, 0, testutil.CursorPos(t, e))

		// forward still walks the entries the pick did not discard
		action.JumpForward(e)
		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("out of range index reports failure", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		_, ok := ui.GotoJump(e, 3, ui.PickerAcceptReplace)
		assert.False(t, ok)
	})
}
