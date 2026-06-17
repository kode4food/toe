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
			[]command.KeyEvent{command.Char('p')},
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

	t.Run("buffer picker columns", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(ui.BufferPicker),
			[]command.KeyEvent{command.Char('b')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'b')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "┬")
		assert.Contains(t, out, "id")
		assert.Contains(t, out, "flags")
		assert.Contains(t, out, "path")
	})
}
