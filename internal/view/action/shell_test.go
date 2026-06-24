package action_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/action"
)

func TestShellPipeTo(t *testing.T) {
	t.Run("pipes selection without changing doc", func(t *testing.T) {
		e := editorWithText(t, "hello")
		setSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		err := action.ShellPipeTo(e, "cat > /dev/null")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})

	t.Run("failing command returns error", func(t *testing.T) {
		e := editorWithText(t, "hello")
		setSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		err := action.ShellPipeTo(e, "false")

		assert.Error(t, err)
	})
}

func TestShellPipeErrors(t *testing.T) {
	t.Run("failing command returns error", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		err := action.ShellPipe(e, "false")

		assert.Error(t, err)
	})
}

func TestShellOutputErrors(t *testing.T) {
	t.Run("insert failing command returns error", func(t *testing.T) {
		e := editorWithText(t, "x")
		setCursor(t, e, 0)

		err := action.ShellInsertOutput(e, "false")

		assert.Error(t, err)
	})
}

func TestReadFileErrors(t *testing.T) {
	t.Run("missing file returns error", func(t *testing.T) {
		e := editorWithText(t, "x")
		setCursor(t, e, 0)

		err := action.ReadFile(e, "/no/such/file/xyz.txt")

		assert.Error(t, err)
	})
}

func TestShell(t *testing.T) {
	t.Run("pipe replaces selection", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		err := action.ShellPipe(e, "tr a-z A-Z")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "ABC", doc.Text().String())
	})

	t.Run("insert/append at boundaries", func(t *testing.T) {
		e := editorWithText(t, "x")
		setSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)

		err := action.ShellInsertOutput(e, "printf hello")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hellox", doc.Text().String())

		e = editorWithText(t, "x")
		setSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)

		err = action.ShellAppendOutput(e, "printf hello")

		assert.NoError(t, err)
		doc, _ = e.FocusedDocument()
		assert.Equal(t, "xhello", doc.Text().String())
	})

	t.Run("keep pipe by exit status", func(t *testing.T) {
		e := editorWithText(t, "a\nb")
		setSelection(
			t, e,
			[]core.Range{
				core.NewRange(0, 1),
				core.NewRange(2, 3),
			},
			0,
		)

		err := action.ShellKeepPipe(e, "grep a")

		assert.NoError(t, err)
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 1, len(sel.Ranges()))
		assert.Equal(t, core.NewRange(0, 1), sel.Ranges()[0])
	})

	t.Run("read file inserts contents", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "input.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))

		e := editorWithText(t, "x")
		setCursor(t, e, 1)

		err := action.ReadFile(e, path)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xhello", doc.Text().String())
	})
}
