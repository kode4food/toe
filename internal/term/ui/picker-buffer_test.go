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
			km, "buffer_picker", m.PickerAction(bufferPicker),
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
			km, "buffer_picker", m.PickerAction(bufferPicker),
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

	t.Run("defaults to most recent buffer", func(t *testing.T) {
		m, e := bufferPickerMRUModel(t)
		_ = sendSpecial(m, tea.KeyEnter)

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Contains(t, doc.Text().String(), "CCC")
	})

	t.Run("start previous selects previous buffer", func(t *testing.T) {
		_, e := bufferPickerMRUModel(t)
		m := openBufferPicker(t, e, ui.BufferPickerOptions{
			StartPosition: ui.PickerStartPrevious,
		})
		_ = sendSpecial(m, tea.KeyEnter)

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Contains(t, doc.Text().String(), "BBB")
	})

	t.Run("ctrl-s opens horizontal split", func(t *testing.T) {
		m, e := bufferPickerMRUModel(t)
		before := len(e.AllViews())
		_ = sendModified(m, 's', tea.ModCtrl)

		assert.Equal(t, before+1, len(e.AllViews()))
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Contains(t, doc.Text().String(), "CCC")
	})

	t.Run("ctrl-v opens vertical split", func(t *testing.T) {
		m, e := bufferPickerMRUModel(t)
		before := len(e.AllViews())
		_ = sendModified(m, 'v', tea.ModCtrl)

		assert.Equal(t, before+1, len(e.AllViews()))
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Contains(t, doc.Text().String(), "CCC")
	})
}

func bufferPickerMRUModel(t *testing.T) (ui.Model, *view.Editor) {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{
		"a.txt": "AAA",
		"b.txt": "BBB",
		"c.txt": "CCC",
	} {
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, name), []byte(text), 0o644,
		))
	}

	e := view.NewEditor(dir)
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		_, err := e.OpenFile(filepath.Join(dir, name))
		assert.NoError(t, err)
	}
	return openBufferPicker(t, e), e
}

func openBufferPicker(
	t *testing.T, e *view.Editor, opts ...ui.BufferPickerOptions,
) ui.Model {
	t.Helper()
	cfg := ui.BufferPickerOptions{
		StartPosition: ui.PickerStartTop,
	}
	if len(opts) > 0 {
		cfg = opts[0]
	}
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "buffer_picker",
		m.PickerAction(func(e *view.Editor) *ui.Picker {
			return ui.NewBufferPicker(e, cfg)
		}),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 120, 30)
	return sendKey(m, 'p')
}

func TestPickerStartPositionUnmarshal(t *testing.T) {
	t.Run("valid values unmarshal", func(t *testing.T) {
		var p ui.PickerStartPosition
		assert.NoError(t, p.UnmarshalText([]byte("top")))
		assert.Equal(t, ui.PickerStartTop, p)
		assert.NoError(t, p.UnmarshalText([]byte("previous")))
		assert.Equal(t, ui.PickerStartPrevious, p)
	})
	t.Run("invalid value returns error", func(t *testing.T) {
		var p ui.PickerStartPosition
		err := p.UnmarshalText([]byte("invalid"))
		assert.ErrorIs(t, err, ui.ErrInvalidPickerStart)
	})
}
