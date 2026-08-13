package ui_test

import (
	"strings"
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

		cmdline := stripANSI(lastLine(m.View().Content))

		assert.Contains(t, cmdline, "REC a")
		assert.True(t, strings.HasSuffix(cmdline, " REC a "))
		assert.NotContains(t, lastLine(m.View().Content), "\x1b[5m")
	})

	t.Run("badge survives the command prompt", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), 'Q'), 'a')

		m = sendKey(m, ':')
		prompt := stripANSI(lastLine(m.View().Content))
		m = sendKey(m, 'w')
		typing := stripANSI(lastLine(m.View().Content))

		assert.True(t, strings.HasSuffix(prompt, " REC a "))
		assert.True(t, strings.HasSuffix(typing, " REC a "))
		assert.Contains(t, typing, "w")
	})

	t.Run("badge blinks off and back on", func(t *testing.T) {
		m, _ := macroModel(t)
		m = sendKey(m, 'z')
		next, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
		m = next.(ui.Model)
		lit := stripANSI(lastLine(m.View().Content))

		next, cmd = m.Update(firstMsg(cmd))
		m = next.(ui.Model)
		dark := stripANSI(lastLine(m.View().Content))
		next, _ = m.Update(firstMsg(cmd))
		m = next.(ui.Model)
		relit := stripANSI(lastLine(m.View().Content))

		assert.Contains(t, lit, "REC a")
		assert.NotContains(t, dark, "REC a")
		assert.Equal(t, len(lit), len(dark))
		assert.Contains(t, relit, "REC a")
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

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
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

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
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
	bindNormalTestKeyAction(km, "rec", m.MacroRecordAction,
		[]command.KeyEvent{char('z')})
	bindNormalTestKeyAction(km, "play", m.MacroReplayAction,
		[]command.KeyEvent{char('v')})
	// 'g' returns a continuation; followed by 'x' it inserts "X"
	bindNormalTestKeyAction(km, "g-prefix",
		func(ed *view.Editor) command.Continuation {
			return func(
				ed *view.Editor, k command.KeyEvent,
			) command.Continuation {
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
	bindNormalTestKeyAction(km, "rec", m.MacroRecordAction,
		[]command.KeyEvent{char('z')})
	bindNormalTestKeyAction(km, "play", m.MacroReplayAction,
		[]command.KeyEvent{char('v')})
	bindNormalTestAction(km, "to_insert",
		func(e *view.Editor) {
			action.InsertMode(e)
		}, []command.KeyEvent{char('i')})
	bindTestAction(bindTestActionArgs{
		km: km, mode: view.ModeInsert, name: "to_normal",
		fn: func(e *view.Editor) command.Continuation {
			action.NormalMode(e)
			return nil
		},
		seqs: [][]command.KeyEvent{{special(command.Escape)}},
	})
	m = resize(m, 80, 24)
	return m, e
}
