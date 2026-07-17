package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestPickerMatch(t *testing.T) {
	t.Run("file picker page keys", func(t *testing.T) {
		tmp := t.TempDir()
		for i := range 30 {
			name := fmt.Sprintf("file-%02d.go", i)
			err := os.WriteFile(
				filepath.Join(tmp, name), []byte("package p\n"), 0o644,
			)
			assert.NoError(t, err)
		}

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 70, 20)
		m = sendKeyAndFeed(m, 'p')
		_ = m.View()

		m = sendSpecialText(m, tea.KeyPgDown, "pgdown")
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, " > file-12.go")

		m = sendSpecialText(m, tea.KeyPgUp, "pgup")
		out = stripANSI(m.View().Content)
		assert.Contains(t, out, " > file-00.go")
	})

	t.Run("buffer picker filters by field", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "alpha.go")
		err := os.WriteFile(path, []byte("package alpha\n"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(bufferPicker),
			[]command.KeyEvent{char('b')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'b')
		for _, ch := range "%path alpha" {
			m = sendKey(m, ch)
		}
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "alpha.go")
		assert.NotContains(t, out, "[scratch]")
	})
}
