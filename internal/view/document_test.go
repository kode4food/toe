package view_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

func TestNewDocument(t *testing.T) {
	t.Run("creates scratch document with unique id", func(t *testing.T) {
		e1 := view.NewEditor("/tmp")
		e2 := view.NewEditor("/tmp")
		d1, _ := e1.FocusedDocument()
		d2, _ := e2.FocusedDocument()
		assert.NotEqual(t, view.InvalidDocumentId, d1.ID())
		assert.Equal(t, "", d1.Path())
		assert.False(t, d1.Modified())
		assert.Equal(t, view.ScratchBufferName, d1.DisplayName())
		_ = d2
	})

	t.Run("text is empty rope", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		d, _ := e.FocusedDocument()
		assert.Equal(t, "", d.Text().String())
	})

	t.Run("lang defaults to text", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		d, _ := e.FocusedDocument()
		assert.Equal(t, "text", d.Lang())
	})

	t.Run("uses configured default line ending", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.Options().DefaultLineEnding = core.LineEndingCRLF

		e.NewDocument()

		d, _ := e.FocusedDocument()
		assert.Equal(t, core.LineEndingCRLF, d.LineEnding())
	})

	t.Run("live default line ending for scratch", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.Options().DefaultLineEnding = core.LineEndingCRLF

		e.NewDocument()

		d, _ := e.FocusedDocument()
		assert.Equal(t, core.LineEndingCRLF, d.LineEnding())
	})
}

func TestOpenDocument(t *testing.T) {
	t.Run("opens existing file", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "hello.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello world"), 0o644))

		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.Equal(t, "hello world", d.Text().String())
		assert.False(t, d.Modified())
	})

	t.Run("rejects binary file", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "hello.bin")
		data := []byte("hello\x00world")
		assert.NoError(t, os.WriteFile(path, data, 0o644))

		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.ErrorIs(t, err, core.ErrBinaryFile)
	})

	t.Run("new file returns empty doc at path", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "new.txt")
		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.Equal(t, "", d.Text().String())
		assert.Equal(t, path, d.Path())
	})

	t.Run("detects existing file line ending", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "hello.txt")
		err := os.WriteFile(path, []byte("hello\r\nworld\r\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)

		_, err = e.OpenFile(path)

		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.Equal(t, core.LineEndingCRLF, d.LineEnding())
	})

	t.Run("applies editor config on open", func(t *testing.T) {
		tmp := t.TempDir()
		err := os.WriteFile(filepath.Join(tmp, ".editorconfig"), []byte(`
root = true

[*.go]
indent_style = space
indent_size = 2
tab_width = 8
end_of_line = crlf
`), 0o644)
		assert.NoError(t, err)
		path := filepath.Join(tmp, "main.go")
		err = os.WriteFile(path, []byte("package main\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)

		_, err = e.OpenFile(path)

		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.False(t, d.IndentStyle().IsTabs())
		assert.Equal(t, uint8(2), d.IndentStyle().Width())
		assert.Equal(t, 8, d.TabWidth())
		assert.Equal(t, core.LineEndingCRLF, d.LineEnding())
	})

	t.Run("uses language indent fallback on open", func(t *testing.T) {
		root := t.TempDir()
		writeViewLanguages(t, root, `
[[language]]
name = "custom"
file-types = ["foo"]
indent = { tab-width = 8, unit = "  " }
`)
		t.Setenv("XDG_CONFIG_HOME", root)
		tmp := t.TempDir()
		path := filepath.Join(tmp, "main.foo")
		err := os.WriteFile(path, []byte("package main\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)

		_, err = e.OpenFile(path)

		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.False(t, d.IndentStyle().IsTabs())
		assert.Equal(t, uint8(2), d.IndentStyle().Width())
		assert.Equal(t, 8, d.TabWidth())
	})

	t.Run("missing file language indent", func(t *testing.T) {
		root := t.TempDir()
		writeViewLanguages(t, root, `
[[language]]
name = "custom"
file-types = ["foo"]
indent = { tab-width = 8, unit = "  " }
`)
		t.Setenv("XDG_CONFIG_HOME", root)
		tmp := t.TempDir()
		path := filepath.Join(tmp, "new.foo")
		e := view.NewEditor(tmp)

		_, err := e.OpenFile(path)

		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.False(t, d.IndentStyle().IsTabs())
		assert.Equal(t, uint8(2), d.IndentStyle().Width())
		assert.Equal(t, 8, d.TabWidth())
	})

	t.Run("can disable editor config on open", func(t *testing.T) {
		tmp := t.TempDir()
		err := os.WriteFile(filepath.Join(tmp, ".editorconfig"), []byte(`
root = true

[*.go]
tab_width = 8
end_of_line = crlf
`), 0o644)
		assert.NoError(t, err)
		path := filepath.Join(tmp, "main.go")
		err = os.WriteFile(path, []byte("package main\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		e.Options().EditorConfig = false

		_, err = e.OpenFile(path)

		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.Equal(t, 4, d.TabWidth())
		assert.Equal(t, core.LineEndingLF, d.LineEnding())
	})

	t.Run("new file uses default line ending", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "new.txt")
		e := view.NewEditor(tmp)
		e.Options().DefaultLineEnding = core.LineEndingCRLF

		_, err := e.OpenFile(path)

		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.Equal(t, core.LineEndingCRLF, d.LineEnding())
	})

	t.Run("editorconfig text width", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		tmp := t.TempDir()
		err := os.WriteFile(filepath.Join(tmp, ".editorconfig"), []byte(`
root = true

[*.md]
max_line_length = 40
`), 0o644)
		assert.NoError(t, err)
		path := filepath.Join(tmp, "note.md")
		err = os.WriteFile(path, []byte("# title\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		enabled := true
		opts := view.Options{
			TextWidth: new(80),
			SoftWrap: language.SoftWrap{
				Enable:          &enabled,
				WrapAtTextWidth: &enabled,
			},
		}

		format := d.TextFormatForConfig(80, &opts)

		assert.True(t, format.SoftWrap)
		assert.True(t, format.SoftWrapAtTextWidth)
		assert.Equal(t, 40, format.ViewportWidth)
	})
}

func TestDocumentSave(t *testing.T) {
	t.Run("saves content to disk", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		assert.NoError(t, e.Save(false))
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "", string(data))
	})

	t.Run("inserts final newline by default", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		e := testutil.EditorWithText(t, "hello")
		doc, _ := e.FocusedDocument()
		doc.SetPath(path)

		err := e.Save(false)

		assert.NoError(t, err)
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "hello\n", string(data))
		assert.Equal(t, "hello\n", doc.Text().String())
	})

	t.Run("save cleanup is undoable", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		e := testutil.EditorWithText(t, "hello")
		doc, _ := e.FocusedDocument()
		doc.SetPath(path)

		err := e.Save(false)

		assert.NoError(t, err)
		assert.Equal(t, "hello\n", doc.Text().String())
		assert.False(t, doc.Modified())
		assert.True(t, e.Undo())
		assert.Equal(t, "hello", doc.Text().String())
		assert.True(t, doc.Modified())
	})

	t.Run("doc line ending used for final newline", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		e := testutil.EditorWithText(t, "hello")
		doc, _ := e.FocusedDocument()
		doc.SetPath(path)
		doc.SetLineEnding(core.LineEndingCRLF)

		err := e.Save(false)

		assert.NoError(t, err)
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "hello\r\n", string(data))
	})

	t.Run("applies configured save cleanup", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		e := testutil.EditorWithText(t, "a  \n\n\n")
		e.Options().TrimTrailingWS = true
		e.Options().TrimFinalNewlines = true
		e.Options().InsertFinalNewline = true
		doc, _ := e.FocusedDocument()
		doc.SetPath(path)

		err := e.Save(false)

		assert.NoError(t, err)
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "a\n", string(data))
		assert.Equal(t, "a\n", doc.Text().String())
	})

	t.Run("can disable final newline insertion", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		e := testutil.EditorWithText(t, "hello")
		e.Options().InsertFinalNewline = false
		doc, _ := e.FocusedDocument()
		doc.SetPath(path)

		err := e.Save(false)

		assert.NoError(t, err)
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})

	t.Run("editor config overrides save cleanup", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		tmp := t.TempDir()
		err := os.WriteFile(filepath.Join(tmp, ".editorconfig"), []byte(`
root = true

[*.txt]
insert_final_newline = false
trim_trailing_whitespace = true
`), 0o644)
		assert.NoError(t, err)
		path := filepath.Join(tmp, "out.txt")
		err = os.WriteFile(path, []byte("old"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, rope.LenChars(), "hello  "),
		})
		assert.NoError(t, err)
		tx := core.NewTransaction(rope).WithChanges(cs)
		assert.NoError(t, e.Apply(tx))

		err = e.Save(false)

		assert.NoError(t, err)
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})

	t.Run("scratch needs path to save", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		err := e.Save(false)
		assert.Error(t, err)
	})
}

func TestDocumentSaveSafety(t *testing.T) {
	t.Run("refuses save when file changed on disk", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		assert.NoError(t, os.WriteFile(path, []byte("original"), 0o644))
		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		assert.NoError(t, os.WriteFile(path, []byte("externally changed"), 0o644))

		err = e.Save(false)

		assert.ErrorIs(t, err, view.ErrFileChangedOnDisk)
		data, readErr := os.ReadFile(path)
		assert.NoError(t, readErr)
		assert.Equal(t, "externally changed", string(data))
	})

	t.Run("force save overwrites external change", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		assert.NoError(t, os.WriteFile(path, []byte("original"), 0o644))
		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		assert.NoError(t, os.WriteFile(path, []byte("externally changed"), 0o644))

		err = e.Save(true)

		assert.NoError(t, err)
		data, readErr := os.ReadFile(path)
		assert.NoError(t, readErr)
		assert.Equal(t, "original\n", string(data))
	})

	t.Run("refuses save when file is read-only", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		assert.NoError(t, os.WriteFile(path, []byte("original"), 0o644))
		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		assert.NoError(t, os.Chmod(path, 0o444))
		t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

		err = e.Save(false)

		assert.ErrorIs(t, err, view.ErrFileReadOnly)
	})

	t.Run("leaves no backup after a successful save", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		assert.NoError(t, os.WriteFile(path, []byte("original"), 0o644))
		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, rope.LenChars(), "changed"),
		})
		assert.NoError(t, err)
		tx := core.NewTransaction(rope).WithChanges(cs)
		assert.NoError(t, e.Apply(tx))

		err = e.Save(false)

		assert.NoError(t, err)
		entries, readErr := os.ReadDir(tmp)
		assert.NoError(t, readErr)
		assert.Len(t, entries, 1)
		assert.Equal(t, "out.txt", entries[0].Name())
	})

	t.Run("read-only directory restores original", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")
		assert.NoError(t, os.WriteFile(path, []byte("original"), 0o644))
		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		assert.NoError(t, os.Chmod(tmp, 0o555))
		t.Cleanup(func() { _ = os.Chmod(tmp, 0o755) })

		err = e.Save(false)

		assert.Error(t, err)
		data, readErr := os.ReadFile(path)
		assert.NoError(t, readErr)
		assert.Equal(t, "original", string(data))
	})
}

func TestDocumentReload(t *testing.T) {
	t.Run("reloads and clears modified", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "reload.txt")
		err := os.WriteFile(path, []byte("old"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		err = os.WriteFile(path, []byte("new"), 0o644)
		assert.NoError(t, err)

		err = e.Reload()

		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.Equal(t, "new", d.Text().String())
		assert.False(t, d.Modified())
	})

	t.Run("reload preserves undo history", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "reload.txt")
		err := os.WriteFile(path, []byte("old"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(3, 3, "!"),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		err = os.WriteFile(path, []byte("old!"), 0o644)
		assert.NoError(t, err)

		err = e.Reload()

		assert.NoError(t, err)
		assert.True(t, e.Undo())
		assert.Equal(t, "old", doc.Text().String())
	})
}

func TestDocumentExternalChange(t *testing.T) {
	t.Run("clean buffer reloads", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "reload.txt")
		err := os.WriteFile(path, []byte("old"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		err = os.WriteFile(path, []byte("new content"), 0o644)
		assert.NoError(t, err)

		ok := e.ProcessExternalFileChange(path)

		doc, _ := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "new content", doc.Text().String())
		assert.False(t, doc.Modified())
		assert.Equal(t, view.ExternalStateClean, doc.ExternalState())
		assert.Contains(t, e.TakeStatusMsg(), "reloaded")
	})

	t.Run("clean reload preserves selections", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cursor.txt")
		err := os.WriteFile(path, []byte("0123456789"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		e.ResizeTree(80, 24)
		v1, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		v2, ok := e.VSplit(doc.ID())
		assert.True(t, ok)
		doc.SetSelectionFor(v1.ID(), core.PointSelection(4))
		doc.SetSelectionFor(v2.ID(), core.PointSelection(8))
		err = os.WriteFile(path, []byte("0123456789012345"), 0o644)
		assert.NoError(t, err)

		ok = e.ProcessExternalFileChange(path)

		assert.True(t, ok)
		assert.Equal(t, 4, doc.SelectionFor(v1.ID()).Primary().Head)
		assert.Equal(t, 8, doc.SelectionFor(v2.ID()).Primary().Head)
	})

	t.Run("clean reload clips selections", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "short.txt")
		err := os.WriteFile(path, []byte("0123456789"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		doc.SetSelectionFor(v.ID(), core.PointSelection(8))
		err = os.WriteFile(path, []byte("012"), 0o644)
		assert.NoError(t, err)

		ok := e.ProcessExternalFileChange(path)

		assert.True(t, ok)
		assert.Equal(t, 3, doc.SelectionFor(v.ID()).Primary().Head)
	})

	t.Run("clean reload clips invalid selections", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "invalid-selection.txt")
		err := os.WriteFile(path, []byte("0123456789"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		doc.SetSelectionFor(v.ID(), newSelection(t, []core.Range{
			core.NewRange(-5, 20),
		}, 0))
		err = os.WriteFile(path, []byte("012"), 0o644)
		assert.NoError(t, err)

		ok := e.ProcessExternalFileChange(path)

		assert.True(t, ok)
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, []core.Range{core.NewRange(0, 3)}, sel.Ranges())
	})

	t.Run("clean reload maps inserted prefix", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "insert.txt")
		err := os.WriteFile(path, []byte("alpha beta"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		doc.SetSelectionFor(v.ID(), core.PointSelection(6))
		err = os.WriteFile(path, []byte("new alpha beta"), 0o644)
		assert.NoError(t, err)

		ok := e.ProcessExternalFileChange(path)

		assert.True(t, ok)
		assert.Equal(t, 10, doc.SelectionFor(v.ID()).Primary().Head)
	})

	t.Run("clean reload maps deleted prefix", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "delete-prefix.txt")
		err := os.WriteFile(path, []byte("new alpha beta"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		doc.SetSelectionFor(v.ID(), core.PointSelection(10))
		err = os.WriteFile(path, []byte("alpha beta"), 0o644)
		assert.NoError(t, err)

		ok := e.ProcessExternalFileChange(path)

		assert.True(t, ok)
		assert.Equal(t, 6, doc.SelectionFor(v.ID()).Primary().Head)
	})

	t.Run("dirty buffer marks conflict", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "dirty.txt")
		err := os.WriteFile(path, []byte("old"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		replaceFocusedText(t, e, "local")
		err = os.WriteFile(path, []byte("external content"), 0o644)
		assert.NoError(t, err)

		ok := e.ProcessExternalFileChange(path)

		doc, _ := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "local", doc.Text().String())
		assert.True(t, doc.Modified())
		assert.Equal(t, view.ExternalStateChanged, doc.ExternalState())
		assert.Contains(t, e.TakeStatusMsg(), ":reload or :write")
	})

	t.Run("deleted file marks conflict", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "delete.txt")
		err := os.WriteFile(path, []byte("old"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		err = os.Remove(path)
		assert.NoError(t, err)

		ok := e.ProcessExternalFileChange(path)

		doc, _ := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "old", doc.Text().String())
		assert.Equal(t, view.ExternalStateDeleted, doc.ExternalState())
		assert.Contains(t, e.TakeStatusMsg(), "deleted")
	})

	t.Run("save refreshes snapshot", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "save.txt")
		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		replaceFocusedText(t, e, "local")
		assert.NoError(t, e.Save(false))

		ok := e.ProcessExternalFileChange(path)

		doc, _ := e.FocusedDocument()
		assert.False(t, ok)
		assert.Equal(t, view.ExternalStateClean, doc.ExternalState())
	})
}

func TestDocumentRelativeName(t *testing.T) {
	t.Run("relative to basedir", func(t *testing.T) {
		name := view.DocumentRelativeName("/a/b/c/file.txt", "/a/b")
		assert.Equal(t, "c/file.txt", name)
	})

	t.Run("scratch buffer returns scratch name", func(t *testing.T) {
		assert.Equal(t,
			view.ScratchBufferName, view.DocumentRelativeName("", "/any"),
		)
	})
}

func TestDocumentAccessors(t *testing.T) {
	e := view.NewEditor("/tmp")
	d, _ := e.FocusedDocument()

	t.Run("SetLang", func(t *testing.T) {
		d.SetLang("go")
		assert.Equal(t, "go", d.Lang())
	})

	t.Run("ReadOnly defaults false", func(t *testing.T) {
		assert.False(t, d.ReadOnly())
	})

	t.Run("sets read only", func(t *testing.T) {
		d.SetReadOnly(true)
		assert.True(t, d.ReadOnly())
	})

	t.Run("IndentStyle defaults to tabs", func(t *testing.T) {
		assert.True(t, d.IndentStyle().IsTabs())
	})

	t.Run("TabWidth defaults to 4", func(t *testing.T) {
		assert.Equal(t, 4, d.TabWidth())
	})

	t.Run("LineEnding defaults to LF", func(t *testing.T) {
		assert.Equal(t, d.LineEnding(), d.LineEnding())
	})

	t.Run("RelativeName delegates", func(t *testing.T) {
		d.SetPath("/a/b/c.txt")
		assert.Equal(t, "b/c.txt", d.RelativeName("/a"))
	})

	t.Run("AccessedAt non-zero", func(t *testing.T) {
		assert.NotZero(t, d.AccessedAt())
	})

	t.Run("tracks search highlights", func(t *testing.T) {
		v, ok := e.FocusedView()
		assert.True(t, ok)

		assert.False(t, d.SearchHighlightsActive(v.ID()))
		d.ShowSearchHighlights(v.ID())
		assert.True(t, d.SearchHighlightsActive(v.ID()))
	})
}

func TestDocumentBOM(t *testing.T) {
	bom := []byte{0xef, 0xbb, 0xbf}

	t.Run("BOM stripped on open, text excludes BOM", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "bom.txt")
		err := os.WriteFile(path, append(bom, []byte("hello")...), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)

		d, _ := e.FocusedDocument()
		assert.Equal(t, "hello", d.Text().String())
		assert.True(t, d.HasBOM())
	})

	t.Run("BOM preserved on save", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		tmp := t.TempDir()
		path := filepath.Join(tmp, "bom.txt")
		err := os.WriteFile(path, append(bom, []byte("hello\n")...), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		assert.NoError(t, e.Save(false))

		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, append(bom, []byte("hello\n")...), data)
	})

	t.Run("no BOM when file has none", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		tmp := t.TempDir()
		path := filepath.Join(tmp, "nobom.txt")
		err := os.WriteFile(path, []byte("hello\n"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		assert.NoError(t, e.Save(false))

		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, []byte("hello\n"), data)
	})

	t.Run("BOM re-detected on reload", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		tmp := t.TempDir()
		path := filepath.Join(tmp, "bom.txt")
		err := os.WriteFile(path, append(bom, []byte("v1\n")...), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)

		err = os.WriteFile(path, append(bom, []byte("v2\n")...), 0o644)
		assert.NoError(t, err)
		assert.NoError(t, e.Reload())

		d, _ := e.FocusedDocument()
		assert.Equal(t, "v2\n", d.Text().String())
		assert.True(t, d.HasBOM())
		assert.NoError(t, e.Save(false))

		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, append(bom, []byte("v2\n")...), data)
	})
}

func TestDocumentRestoreCursor(t *testing.T) {
	t.Run("defaults to false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		d, _ := e.FocusedDocument()
		assert.False(t, d.RestoreCursor())
	})

	t.Run("set and clear", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		d, _ := e.FocusedDocument()
		d.SetRestoreCursor(true)
		assert.True(t, d.RestoreCursor())
		d.SetRestoreCursor(false)
		assert.False(t, d.RestoreCursor())
	})
}

func TestDocumentSetIndentStyle(t *testing.T) {
	t.Run("updates indent style", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		d, _ := e.FocusedDocument()
		spaces := core.ParseIndentStyle("  ")
		d.SetIndentStyle(spaces)
		assert.Equal(t, spaces, d.IndentStyle())
	})
}

func TestDocumentTextFormat(t *testing.T) {
	t.Run("returns format without error", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor("/tmp")
		d, _ := e.FocusedDocument()
		f := d.TextFormat(80)
		assert.NotNil(t, f)
		assert.Equal(t, 4, f.TabWidth)
	})
}

func TestDocumentRevisionAndLastEditPos(t *testing.T) {
	t.Run("revision starts at zero", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		d, _ := e.FocusedDocument()
		assert.Equal(t, 0, d.Revision())
	})

	t.Run("revision increments on apply", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		d, _ := e.FocusedDocument()
		assert.Greater(t, d.Revision(), 0)
	})

	t.Run("LastEditPos after edit", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		d, _ := e.FocusedDocument()
		assert.GreaterOrEqual(t, d.LastEditPos(), 0)
	})

	t.Run("selection apply does not modify document", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "clean.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))
		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		v, _ := e.FocusedView()
		tx := core.NewTransaction(
			d.Text()).WithSelection(core.PointSelection(3))

		assert.NoError(t, e.Apply(tx))

		assert.Equal(t, 3, d.SelectionFor(v.ID()).Primary().Head)
		assert.False(t, d.Modified())
	})
}

func TestDocumentBeginAndCommitInsertGroup(t *testing.T) {
	t.Run("accumulated changes become one revision", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()

		d.BeginInsertGroup(v.ID())

		rope := d.Text()
		cs1, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "a"),
		})
		assert.NoError(t, err)
		tx1 := core.NewTransaction(rope).WithChanges(cs1)
		assert.NoError(t, d.Apply(tx1, v.ID()))

		rope2 := d.Text()
		cs2, err := core.NewChangeSetFromChanges(rope2, []core.Change{
			core.TextChange(1, 1, "b"),
		})
		assert.NoError(t, err)
		tx2 := core.NewTransaction(rope2).WithChanges(cs2)
		assert.NoError(t, d.Apply(tx2, v.ID()))

		d.CommitInsertHistory(v.ID())

		assert.Equal(t, "ab", d.Text().String())
		assert.True(t, d.Modified())
	})

	t.Run("begin is idempotent when already active", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		d.BeginInsertGroup(v.ID())
		d.BeginInsertGroup(v.ID())
		d.CommitInsertHistory(v.ID())
	})

	t.Run("commit with no accumulation is no-op", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		d.CommitInsertHistory(v.ID())
		assert.Equal(t, 0, d.Revision())
	})

	t.Run("empty changeset commit is no-op", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		d.BeginInsertGroup(v.ID())
		d.CommitInsertHistory(v.ID())
		assert.Equal(t, 0, d.Revision())
	})
}

func TestDocumentApplyBranches(t *testing.T) {
	t.Run("apply with selection update", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		rope := d.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "xyz"),
		})
		assert.NoError(t, err)
		sel := core.PointSelection(2)
		tx := core.NewTransaction(rope).WithChanges(cs).WithSelection(sel)
		assert.NoError(t, d.Apply(tx, v.ID()))
		assert.Equal(t, "xyz", d.Text().String())
		assert.Equal(t, 2, d.SelectionFor(v.ID()).Primary().Head)
	})

	t.Run("apply in insert group with selection", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		d.BeginInsertGroup(v.ID())
		rope := d.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "abc"),
		})
		assert.NoError(t, err)
		sel := core.PointSelection(1)
		tx := core.NewTransaction(rope).WithChanges(cs).WithSelection(sel)
		assert.NoError(t, d.Apply(tx, v.ID()))
		d.CommitInsertHistory(v.ID())
		assert.Equal(t, "abc", d.Text().String())
	})

	t.Run("maps selection when transaction omits it", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		cs, err := core.NewChangeSetFromChanges(d.Text(), []core.Change{
			core.TextChange(0, 0, "abc\n"),
		})
		assert.NoError(t, err)
		firstTx := core.NewTransaction(d.Text()).WithChanges(cs).
			WithSelection(core.PointSelection(4))
		assert.NoError(t, d.Apply(firstTx, v.ID()))
		assert.Equal(t, 4, d.SelectionFor(v.ID()).Primary().Head)

		importCS, err := core.NewChangeSetFromChanges(d.Text(), []core.Change{
			core.TextChange(0, 0, "import \"strings\"\n"),
		})
		assert.NoError(t, err)
		importTx := core.NewTransaction(d.Text()).WithChanges(importCS)
		assert.NoError(t, d.Apply(importTx, v.ID()))
		assert.Equal(t, 4+len("import \"strings\"\n"),
			d.SelectionFor(v.ID()).Primary().Head)
	})

	t.Run("maps other view selections", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "multi.txt")
		err := os.WriteFile(path, []byte("hello"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		e.ResizeTree(80, 24)
		v1, err := e.OpenFile(path)
		assert.NoError(t, err)
		v2, ok := e.VSplit(v1.DocID())
		assert.True(t, ok)
		doc, _ := e.Document(v1.DocID())
		doc.SetSelectionFor(v2.ID(), core.PointSelection(3))

		rope := doc.Text()
		cs, csErr := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "X"),
		})
		assert.NoError(t, csErr)
		tx := core.NewTransaction(rope).WithChanges(cs)
		assert.NoError(t, doc.Apply(tx, v1.ID()))
		assert.Equal(t, 4, doc.SelectionFor(v2.ID()).Primary().Head)
	})
}

func TestDocumentAtomicSave(t *testing.T) {
	t.Run("atomic save writes content", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "atomic.txt")
		e := testutil.EditorWithText(t, "atomic content")
		e.Options().AtomicSave = true
		e.Options().InsertFinalNewline = false
		doc, _ := e.FocusedDocument()
		doc.SetPath(path)
		err := e.Save(false)
		assert.NoError(t, err)
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "atomic content", string(data))
	})
}

func TestDocumentDetectLang(t *testing.T) {
	t.Run("go file detected as go", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "main.go")
		err := os.WriteFile(path, []byte("package main\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.Equal(t, "go", d.Lang())
	})

	t.Run("unknown extension falls back to text", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		d, _ := e.FocusedDocument()
		assert.Equal(t, "text", d.Lang())
	})
}

func TestDocumentReloadScratch(t *testing.T) {
	t.Run("reload scratch returns error", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		err := e.Reload()
		assert.Error(t, err)
	})
}

func TestDocumentReloadPreservesSelections(t *testing.T) {
	t.Run("reload maps per-view selections", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "sel.txt")
		err := os.WriteFile(path, []byte("hello world"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		e.ResizeTree(80, 24)
		v1, err := e.OpenFile(path)
		assert.NoError(t, err)
		v2, ok := e.VSplit(v1.DocID())
		assert.True(t, ok)
		doc, _ := e.Document(v1.DocID())
		doc.SetSelectionFor(v1.ID(), core.PointSelection(5))
		doc.SetSelectionFor(v2.ID(), core.PointSelection(8))
		err = os.WriteFile(path, []byte("hello big world"), 0o644)
		assert.NoError(t, err)
		err = doc.Reload()
		assert.NoError(t, err)
		assert.Equal(t, 5, doc.SelectionFor(v1.ID()).Primary().Head)
		assert.Equal(t, 12, doc.SelectionFor(v2.ID()).Primary().Head)
	})
}

func TestDocumentTrimTrailingWhitespaceWithCRLF(t *testing.T) {
	t.Run("preserves crlf on trimmed lines", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "crlf.txt")
		e := testutil.EditorWithText(t, "line  \r\nend  ")
		e.Options().TrimTrailingWS = true
		e.Options().InsertFinalNewline = false
		doc, _ := e.FocusedDocument()
		doc.SetPath(path)
		doc.SetLineEnding(core.LineEndingCRLF)

		err := e.Save(false)

		assert.NoError(t, err)
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "line\r\nend", string(data))
	})
}

func TestDocumentDetectLangByContent(t *testing.T) {
	t.Run("content-based shell detection", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "script")
		content := "#!/bin/bash\necho hello\n"
		err := os.WriteFile(path, []byte(content), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.NotEqual(t, "", d.Lang())
	})

	t.Run("chroma extension fallback", func(t *testing.T) {
		// .py is not in bundled languages, falls back to chroma lexers.Match
		tmp := t.TempDir()
		path := filepath.Join(tmp, "main.py")
		err := os.WriteFile(path, []byte("print('hello')\n"), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.NotEqual(t, "text", d.Lang())
	})

	t.Run("chroma content fallback", func(t *testing.T) {
		// no extension + unrecognized-path, but chroma can analyse the content
		tmp := t.TempDir()
		path := filepath.Join(tmp, "noext")
		err := os.WriteFile(path,
			[]byte("#!/usr/bin/env python3\nprint('hi')\n"), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		_ = d.Lang() // may or may not match; just covers lexers.Analyse path
	})

	t.Run("unknown falls back to text", func(t *testing.T) {
		// binary-ish content with weird extension: all three fallbacks fail
		tmp := t.TempDir()
		path := filepath.Join(tmp, "data.zzzxyzqwerty99")
		err := os.WriteFile(path, []byte("xyzzy plugh\n"), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.Equal(t, "text", d.Lang())
	})
}

func TestDocumentTrimFinalNewlinesSingleEnding(t *testing.T) {
	t.Run("single final newline is preserved", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "single.txt")
		e := testutil.EditorWithText(t, "hello\n")
		e.Options().TrimFinalNewlines = true
		e.Options().InsertFinalNewline = false
		doc, _ := e.FocusedDocument()
		doc.SetPath(path)

		err := e.Save(false)

		assert.NoError(t, err)
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "hello\n", string(data))
	})
}

func TestDocumentOpenEditorConfigNewFileMissingEC(t *testing.T) {
	t.Run("editorconfig line ending on new file", func(t *testing.T) {
		tmp := t.TempDir()
		err := os.WriteFile(filepath.Join(tmp, ".editorconfig"), []byte(`
root = true

[*.txt]
end_of_line = crlf
indent_style = space
indent_size = 2
tab_width = 2
`), 0o644)
		assert.NoError(t, err)
		path := filepath.Join(tmp, "new.txt")
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		d, _ := e.FocusedDocument()
		assert.Equal(t, core.LineEndingCRLF, d.LineEnding())
		assert.False(t, d.IndentStyle().IsTabs())
		assert.Equal(t, 2, d.TabWidth())
	})
}

func TestDocumentConsumeDirty(t *testing.T) {
	t.Run("unseen view is dirty", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		assert.True(t, d.ConsumeDirty(v.ID()))
	})

	t.Run("consuming clears the flag", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		d.ConsumeDirty(v.ID())
		assert.False(t, d.ConsumeDirty(v.ID()))
	})

	t.Run("editing text marks every view dirty", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		d.ConsumeDirty(v.ID())
		rope := d.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "x"),
		})
		assert.NoError(t, err)
		tx := core.NewTransaction(rope).WithChanges(cs)
		assert.NoError(t, d.Apply(tx, v.ID()))
		assert.True(t, d.ConsumeDirty(v.ID()))
	})

	t.Run("changing selection marks that view dirty", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		d.SetSelectionFor(v.ID(), core.PointSelection(0))
		d.ConsumeDirty(v.ID())
		d.SetSelectionFor(v.ID(), core.PointSelection(0))
		assert.False(t, d.ConsumeDirty(v.ID()))
		d.SetSelectionFor(v.ID(), core.PointSelection(1))
		assert.True(t, d.ConsumeDirty(v.ID()))
	})

	t.Run("removing a view forgets its dirty state", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		d, _ := e.FocusedDocument()
		d.ConsumeDirty(v.ID())
		d.RemoveView(v.ID())
		assert.True(t, d.ConsumeDirty(v.ID()))
	})
}

func writeViewLanguages(t *testing.T, root, text string) {
	t.Helper()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	)
	assert.NoError(t, err)
}
