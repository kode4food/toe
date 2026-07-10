package action_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view/action"
)

func TestShellPipeTo(t *testing.T) {
	t.Run("pipes selection without changing doc", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		err := action.ShellPipeTo(e, "cat > /dev/null")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})

	t.Run("failing command returns error", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		err := action.ShellPipeTo(e, "false")

		assert.Error(t, err)
	})
}

func TestShellPipeErrors(t *testing.T) {
	t.Run("failing command returns error", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		err := action.ShellPipe(e, "false")

		assert.Error(t, err)
	})

	t.Run("failing command returns stderr", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		err := action.ShellPipe(e, "printf nope >&2; exit 1")

		assert.True(t, errors.Is(err, action.ErrShellCommand))
		assert.True(t, strings.Contains(err.Error(), "nope"))
	})
}

func TestShellOutputErrors(t *testing.T) {
	t.Run("insert failing command returns error", func(t *testing.T) {
		e := testutil.EditorWithText(t, "x")
		testutil.SetCursor(t, e, 0)

		err := action.ShellInsertOutput(e, "false")

		assert.Error(t, err)
	})
}

func TestReadFileErrors(t *testing.T) {
	t.Run("missing file returns error", func(t *testing.T) {
		e := testutil.EditorWithText(t, "x")
		testutil.SetCursor(t, e, 0)

		err := action.ReadFile(e, "/no/such/file/xyz.txt")

		assert.Error(t, err)
	})
}

func TestShellNoView(t *testing.T) {
	t.Run("pipe is noop", func(t *testing.T) {
		e := editorWithNoView(t)

		err := action.ShellPipe(e, "cat")

		assert.NoError(t, err)
	})

	t.Run("pipe to is noop", func(t *testing.T) {
		e := editorWithNoView(t)

		err := action.ShellPipeTo(e, "cat")

		assert.NoError(t, err)
	})

	t.Run("insert output is noop", func(t *testing.T) {
		e := editorWithNoView(t)

		err := action.ShellInsertOutput(e, "printf hello")

		assert.NoError(t, err)
	})

	t.Run("append output is noop", func(t *testing.T) {
		e := editorWithNoView(t)

		err := action.ShellAppendOutput(e, "printf hello")

		assert.NoError(t, err)
	})

	t.Run("keep pipe is noop", func(t *testing.T) {
		e := editorWithNoView(t)

		err := action.ShellKeepPipe(e, "cat")

		assert.NoError(t, err)
	})

	t.Run("read file is noop", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "input.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))
		e := editorWithNoView(t)

		err := action.ReadFile(e, path)

		assert.NoError(t, err)
	})
}

func TestShellEdgeSelections(t *testing.T) {
	t.Run("pipe skips invalid range", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(10, 11)}, 0)

		err := action.ShellPipe(e, "cat")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("pipe to skips invalid range", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(10, 11)}, 0)

		err := action.ShellPipeTo(e, "cat")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("keep pipe skips invalid range", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(10, 11)}, 0)

		err := action.ShellKeepPipe(e, "cat")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("keep pipe keeps none", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		err := action.ShellKeepPipe(e, "false")

		assert.NoError(t, err)
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, []core.Range{core.NewRange(0, 3)}, sel.Ranges())
	})

	t.Run("append output skips duplicate position", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(
			t, e,
			[]core.Range{core.NewRange(0, 1), core.PointRange(1)},
			0,
		)

		err := action.ShellAppendOutput(e, "printf x")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "axbc", doc.Text().String())
	})
}

func TestShell(t *testing.T) {
	t.Run("pipe replaces selection", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		err := action.ShellPipe(e, "tr a-z A-Z")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "ABC", doc.Text().String())
	})

	t.Run("pipe trims synthetic newline", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		err := action.ShellPipe(e, "cat; printf '\\n'")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("pipe keeps input newline", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\n")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 4)}, 0)

		err := action.ShellPipe(e, "cat")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc\n", doc.Text().String())
	})

	t.Run("insert/append at boundaries", func(t *testing.T) {
		e := testutil.EditorWithText(t, "x")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)

		err := action.ShellInsertOutput(e, "printf hello")

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hellox", doc.Text().String())

		e = testutil.EditorWithText(t, "x")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)

		err = action.ShellAppendOutput(e, "printf hello")

		assert.NoError(t, err)
		doc, _ = e.FocusedDocument()
		assert.Equal(t, "xhello", doc.Text().String())
	})

	t.Run("keep pipe by exit status", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetSelection(
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

		e := testutil.EditorWithText(t, "x")
		testutil.SetCursor(t, e, 1)

		err := action.ReadFile(e, path)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xhello", doc.Text().String())
	})

	t.Run("read file normalizes crlf", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "input.txt")
		assert.NoError(t, os.WriteFile(path, []byte("a\r\nb\r\n"), 0o644))

		e := testutil.EditorWithText(t, "")
		testutil.SetCursor(t, e, 0)

		err := action.ReadFile(e, path)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\nb\n", doc.Text().String())
	})

	t.Run("read file skips duplicate position", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "input.txt")
		assert.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

		e := testutil.EditorWithText(t, "ab")
		testutil.SetSelection(
			t, e,
			[]core.Range{core.NewRange(0, 1), core.PointRange(0)},
			0,
		)

		err := action.ReadFile(e, path)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xab", doc.Text().String())
	})
}
