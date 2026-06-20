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

const jumplistAnchor = 6

func TestJumplistPicker(t *testing.T) {
	t.Run("lists jumps with line contents", func(t *testing.T) {
		m, _, _ := jumplistModel(t)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "file.txt")
		assert.Contains(t, out, "TARGET")
	})

	t.Run("accept moves cursor to the jump", func(t *testing.T) {
		m, e, anchor := jumplistModel(t)
		_ = sendSpecial(m, tea.KeyEnter)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		pos := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
		assert.Equal(t, anchor, pos)
	})
}

// jumplistModel opens a file, records a jump to the TARGET line, and returns a
// model with the jumplist picker open plus the editor and recorded anchor
func jumplistModel(t *testing.T) (ui.Model, *view.Editor, int) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	assert.NoError(t, os.WriteFile(path, []byte("l0\nl1\nTARGET\n"), 0o644))

	e := view.NewEditor(dir)
	v, err := e.OpenFile(path)
	assert.NoError(t, err)
	v.PushJump(v.DocID(), jumplistAnchor)

	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "jumplist", m.PickerAction(ui.JumplistPicker),
		[]command.KeyEvent{command.Char('j')},
	)
	m = resize(m, 120, 30)
	return sendKey(m, 'j'), e, jumplistAnchor
}
