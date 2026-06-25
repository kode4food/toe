package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestBufferPicker(t *testing.T) {
	t.Run("accept switches to the selected buffer", func(t *testing.T) {
		dir := t.TempDir()
		a := filepath.Join(dir, "a.txt")
		b := filepath.Join(dir, "b.txt")
		assert.NoError(t, os.WriteFile(a, []byte("AAA"), 0o644))
		assert.NoError(t, os.WriteFile(b, []byte("BBB"), 0o644))

		e := view.NewEditor(dir)
		_, err := e.OpenFile(a)
		assert.NoError(t, err)
		_, err = e.OpenFile(b)
		assert.NoError(t, err)

		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(ui.BufferPicker),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 120, 30)
		m = sendKey(m, 'p')
		// filter to a.txt and accept it
		for _, ch := range "a.txt" {
			m = sendKey(m, ch)
		}
		_ = sendSpecial(m, tea.KeyEnter)

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Contains(t, doc.Text().String(), "AAA")
	})

	t.Run("accept calls SwitchBuffer for orphaned doc", func(t *testing.T) {
		dir := t.TempDir()
		a := filepath.Join(dir, "a.txt")
		b := filepath.Join(dir, "b.txt")
		assert.NoError(t, os.WriteFile(a, []byte("AAA"), 0o644))
		assert.NoError(t, os.WriteFile(b, []byte("BBB"), 0o644))

		e := view.NewEditor(dir)
		vA, err := e.OpenFile(a)
		assert.NoError(t, err)
		_, err = e.OpenFile(b)
		assert.NoError(t, err)
		// make both views show A — leaves B orphaned in AllDocuments
		e.SwitchBuffer(vA.DocID())

		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(ui.BufferPicker),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 120, 30)
		m = sendKey(m, 'p')
		for _, ch := range "b.txt" {
			m = sendKey(m, ch)
		}
		_ = sendSpecial(m, tea.KeyEnter)

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Contains(t, doc.Text().String(), "BBB")
	})
}
