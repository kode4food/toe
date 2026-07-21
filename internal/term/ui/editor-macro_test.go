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

func TestMacroRecordingStatus(t *testing.T) {
	t.Run("status shows register while recording", func(t *testing.T) {
		m, _ := macroModel(t)
		m = sendKey(m, 'z')
		m = sendKey(m, 'a')

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "[a]")
	})
}

func TestEditorMacro(t *testing.T) {
	t.Run("replays two-key continuation command", func(t *testing.T) {
		m, e := macroModelWithContinuation(t)
		// record into 'a': g then x (two-key command)
		m = sendKey(m, 'z')
		m = sendKey(m, 'a')
		m = sendKey(m, 'g')
		m = sendKey(m, 'x')
		m = sendKey(m, 'z')
		// replay
		m = sendKey(m, 'v')
		_ = sendKey(m, 'a')

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "XX", doc.Text().String())
	})

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
		_ = sendKey(m, 'a')

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "xx", doc.Text().String())
	})
}

// macroModelWithContinuation extends macroModel with a two-key command
// ('g' then any key) so that macro replay exercises the continuation loop
func macroModelWithContinuation(t *testing.T) (ui.Model, *view.Editor) {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(km, "rec", m.MacroRecordAction,
		[]command.KeyEvent{char('z')})
	bindNormalTestAction(km, "play", m.MacroReplayAction,
		[]command.KeyEvent{char('v')})
	// 'g' returns a continuation; followed by 'x' it inserts "X"
	bindNormalTestAction(km, "g-prefix",
		func(ed *view.Editor) command.Continuation {
			return func(ed *view.Editor, k command.KeyEvent) command.Continuation {
				if k.Code.Char == 'x' {
					action.InsertMode(ed)
					action.InsertChar(ed, 'X')
					action.NormalMode(ed)
				}
				return nil
			}
		}, []command.KeyEvent{char('g')})
	m = resize(m, 80, 24)
	return m, e
}

// macroModel wires lowercase keys that avoid default bindings while exercising
// insert recording and replay end to end
func macroModel(t *testing.T) (ui.Model, *view.Editor) {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(km, "rec", m.MacroRecordAction,
		[]command.KeyEvent{char('z')})
	bindNormalTestAction(km, "play", m.MacroReplayAction,
		[]command.KeyEvent{char('v')})
	bindNormalTestAction(km, "to_insert",
		func(e *view.Editor) command.Continuation {
			action.InsertMode(e)
			return nil
		}, []command.KeyEvent{char('i')})
	bindTestAction(bindTestActionArgs{
		km: km, mode: "INS", name: "to_normal",
		fn: func(e *view.Editor) command.Continuation {
			action.NormalMode(e)
			return nil
		},
		seqs: [][]command.KeyEvent{{special(command.Escape)}},
	})
	m = resize(m, 80, 24)
	return m, e
}
