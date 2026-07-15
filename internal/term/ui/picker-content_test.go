package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

type (
	noPreviewPickerSource struct{}

	columnPickerSource struct{}
)

func TestPickerRender(t *testing.T) {
	t.Run("file picker preview pane", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "main.go")
		err := os.WriteFile(path, []byte("package main\n"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "┬")
		assert.Contains(t, out, "┤")
		assert.NotContains(t, out, "┼")
		assert.Contains(t, out, " > main.go")
		assert.NotContains(t, out, "►")
		assert.Contains(t, out, "main.go")
		assertPromptCountRightPadding(t, out)
	})

	t.Run("file picker empty preview pane", func(t *testing.T) {
		tmp := t.TempDir()

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "┬")
		assert.Contains(t, out, "┤")
		assert.Contains(t, out, "0/0")
	})

	t.Run("buffer picker columns", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(bufferPicker),
			[]command.KeyEvent{char('b')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'b')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "┬")
		assert.Contains(t, out, "[scratch]")
	})

	t.Run("preview follows live cursor", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "a.txt")
		text := "one\ntwo\nthree\nfour\nfive\n"
		assert.NoError(t, os.WriteFile(path, []byte(text), 0o644))

		render := func(cursor int) string {
			e := view.NewEditor(dir)
			_, err := e.OpenFile(path)
			assert.NoError(t, err)
			testutil.SetCursor(t, e, cursor)

			km := command.NewKeymaps()
			m := ui.New(e, km)
			bindNormalTestAction(
				km, "buffer_picker", m.PickerAction(bufferPicker),
				[]command.KeyEvent{char('p')},
			)
			m = resize(m, 100, 30)
			m = sendKey(m, 'p')
			return m.View().Content
		}

		atStart := render(0)
		atFour := render(strings.Index(text, "four"))

		oneStart := previewPaneLine(t, atStart, "one")
		oneFour := previewPaneLine(t, atFour, "one")
		fourStart := previewPaneLine(t, atStart, "four")
		fourFour := previewPaneLine(t, atFour, "four")

		assert.NotEqual(t, oneStart, oneFour,
			"preview highlight on doc's first line should move"+
				" once the cursor leaves it")
		assert.NotEqual(t, fourStart, fourFour,
			"preview highlight should follow the cursor to its"+
				" current line")
	})

	t.Run("file explorer shows root title", func(t *testing.T) {
		tmp := t.TempDir()
		dir := filepath.Join(tmp, "project")
		assert.NoError(t, os.MkdirAll(dir, 0o755))
		assert.NoError(t,
			os.WriteFile(filepath.Join(dir, "main.go"), []byte(""), 0o644),
		)
		e := view.NewEditor(dir)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_explorer",
			m.PickerAction(func(e *view.Editor) *ui.Picker {
				return files.NewFileExplorer(
					e, files.DefaultFileExplorerOptions(),
				)
			}),
			[]command.KeyEvent{char('e')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'e')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "project")
		assert.Contains(t, out, "main.go")
	})

	t.Run("plain source preview pane", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "plain_picker",
			m.PickerAction(func(e *view.Editor) *ui.Picker {
				return ui.NewPicker(e, noPreviewPickerSource{})
			}),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "┬")
		assert.Contains(t, out, "┤")
		assert.Contains(t, out, " > plain")
	})

	t.Run("plain source small single pane", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "plain_picker",
			m.PickerAction(func(e *view.Editor) *ui.Picker {
				return ui.NewPicker(e, noPreviewPickerSource{})
			}),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 60, 12)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " > plain")
		assert.NotContains(t, out, "┬")
	})

	t.Run("proportional columns and long query", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "column_picker",
			m.PickerAction(func(e *view.Editor) *ui.Picker {
				return ui.NewPicker(e, columnPickerSource{})
			}),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 30, 12)
		m = sendKey(m, 'p')
		for _, ch := range "abcdefghijklmnopqrstuvwxyz" {
			m = sendKey(m, ch)
		}
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "uvwxyz")
		assert.Contains(t, out, " > go")
		assert.Contains(t, out, "internal/ter")
		assert.Contains(t, out, "non")
	})
}

func (noPreviewPickerSource) ID() string {
	return "Plain"
}

func (noPreviewPickerSource) Columns() []string {
	return []string{"name"}
}

func (noPreviewPickerSource) MatchColumn() int {
	return 0
}

func (noPreviewPickerSource) ColumnProportions() []int {
	return []int{1}
}

func (noPreviewPickerSource) Load(
	*view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	return []ui.PickerItem{{
		Display: "plain",
		Columns: []string{"plain"},
	}}, nil, func() {}
}

func (noPreviewPickerSource) Accept(
	*view.Editor, ui.PickerItem, ui.PickerAcceptAction,
) {
}

func (columnPickerSource) ID() string {
	return "Columns"
}

func (columnPickerSource) Columns() []string {
	return []string{"kind", "path", "description"}
}

func (columnPickerSource) MatchColumn() int {
	return 1
}

func (columnPickerSource) ColumnProportions() []int {
	return []int{0, 4, 1}
}

func (columnPickerSource) Load(
	*view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	return []ui.PickerItem{{
		Display: "first",
		Columns: []string{
			"go",
			"internal/term/ui/picker-render-with-a-very-long-name.go",
			"non-primary columns clip before the primary one",
		},
	}, {
		Display: "second",
		Columns: []string{"txt", "README.md", "short"},
	}}, nil, func() {}
}

func (columnPickerSource) Match(string, ui.PickerItem) (int, []int, bool) {
	return 1, nil, true
}

func (columnPickerSource) Accept(
	*view.Editor, ui.PickerItem, ui.PickerAcceptAction,
) {
}
