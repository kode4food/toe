package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestEditorMacro(t *testing.T) {
	t.Run("records and replays an insertion", func(t *testing.T) {
		m, e := macroModel(t)
		// record into register 'a': enter insert, type x, escape
		m = sendKey(m, 'z')
		m = sendKey(m, 'a')
		m = sendKey(m, 'i')
		m = sendKey(m, 'x')
		m = sendSpecial(m, tea.KeyEscape)
		m = sendKey(m, 'z')
		// replay register 'a' once
		m = sendKey(m, 'v')
		m = sendKey(m, 'a')

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "xx", doc.Text().String())
	})
}

// macroModel wires lowercase keys for record ('z'), replay ('v'), insert ('i'),
// and esc->normal, avoiding the capital Q/q default bindings. Recording and
// replaying an insert then exercises the macro machinery end to end
func macroModel(t *testing.T) (ui.Model, *view.Editor) {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(km, "rec", m.MacroRecordAction,
		[]command.KeyEvent{command.Char('z')})
	bindNormalTestAction(km, "play", m.MacroReplayAction,
		[]command.KeyEvent{command.Char('v')})
	bindNormalTestAction(km, "to_insert",
		func(e *view.Editor) command.Continuation {
			action.InsertMode(e)
			return nil
		}, []command.KeyEvent{command.Char('i')})
	bindTestAction(bindTestActionArgs{
		km: km, mode: "INS", name: "to_normal",
		fn: func(e *view.Editor) command.Continuation {
			action.NormalMode(e)
			return nil
		},
		seqs: [][]command.KeyEvent{{command.Special("esc")}},
	})
	m = resize(m, 80, 24)
	return m, e
}
