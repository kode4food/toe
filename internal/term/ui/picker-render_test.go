package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type noPreviewPickerSource struct{}

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
			m.PickerAction(ui.FilePickerInDir(tmp)),
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
			m.PickerAction(ui.FilePickerInDir(tmp)),
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
			km, "buffer_picker", m.PickerAction(ui.BufferPicker),
			[]command.KeyEvent{char('b')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'b')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "┬")
		assert.Contains(t, out, "id")
		assert.Contains(t, out, "flags")
		assert.Contains(t, out, "path")
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
}

func (noPreviewPickerSource) Title() string {
	return "Plain"
}

func (noPreviewPickerSource) Columns() []string {
	return []string{"name"}
}

func (noPreviewPickerSource) Primary() int {
	return 0
}

func (noPreviewPickerSource) Load(
	*view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	return []ui.PickerItem{{
		Display: "plain",
		Columns: []string{"plain"},
	}}, nil, func() {}
}

func (noPreviewPickerSource) Accept(*view.Editor, ui.PickerItem) {}
