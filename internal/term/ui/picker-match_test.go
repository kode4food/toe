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

	t.Run("narrowed equals rebuilt", func(t *testing.T) {
		narrowed := stripANSI(typeQuery(narrowPicker(t), "ale").View().Content)
		m := typeQuery(narrowPicker(t), "alex")
		rebuilt := stripANSI(sendSpecial(m, tea.KeyBackspace).View().Content)
		assert.Equal(t, rebuilt, narrowed)
		assert.Contains(t, narrowed, "ale.go")
		assert.NotContains(t, narrowed, "main.go")
	})

	t.Run("column query is not narrowed", func(t *testing.T) {
		m := typeQuery(narrowPicker(t), "%path ale")
		direct := stripANSI(m.View().Content)
		m = typeQuery(narrowPicker(t), "%path alex")
		back := stripANSI(sendSpecial(m, tea.KeyBackspace).View().Content)
		assert.Equal(t, back, direct)
		assert.Contains(t, direct, "ale.go")
	})

	t.Run("clearing restores every row", func(t *testing.T) {
		full := stripANSI(narrowPicker(t).View().Content)
		m := typeQuery(narrowPicker(t), "ale")
		for range 3 {
			m = sendSpecial(m, tea.KeyBackspace)
		}
		assert.Equal(t, full, stripANSI(m.View().Content))
	})
}

func narrowPicker(t *testing.T) ui.Model {
	t.Helper()
	m := feedPickerModel(t, []string{
		"internal/term/ale.go",
		"internal/lsp/capabilities.go",
		"internal/view/action/selection-lines.go",
		"alembic/config.toml",
		"docs/scale-guide.md",
		"cmd/toe/main.go",
	})
	return sendKeyAndFeed(m, 'p')
}

func typeQuery(m ui.Model, query string) ui.Model {
	for _, ch := range query {
		m = sendKey(m, ch)
	}
	return m
}
