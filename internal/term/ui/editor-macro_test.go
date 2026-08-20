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

		status := stripANSI(lastLine(m.View().Content))

		assert.Contains(t, status, "REC a")
		assert.True(t, strings.HasSuffix(status, " REC a "))
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
		assert.Contains(t, promptText(m), "w")
	})

	t.Run("badge blinks off and back on", func(t *testing.T) {
		if testing.Short() {
			t.Skip("slow: waits on a real blink timer")
		}
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

	t.Run("replays continuation backtracking", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestKeyAction(km, "rec", m.MacroRecordAction,
			[]command.KeyEvent{char('z')})
		bindNormalTestKeyAction(km, "play", m.MacroReplayAction,
			[]command.KeyEvent{char('v')})
		bindNormalTestKeyAction(km, "pop",
			func(*view.Editor) command.Continuation {
				return func(
					*view.Editor, command.KeyEvent,
				) (command.Continuation, command.Transition) {
					return nil, command.ContinuationPop
				}
			}, []command.KeyEvent{char('g'), char('a')})
		bindNormalTestAction(km, "insert",
			func(e *view.Editor) {
				action.InsertMode(e)
				action.InsertChar(e, 'B')
				action.NormalMode(e)
			}, []command.KeyEvent{char('g'), char('b')})
		m = resize(m, 80, 24)

		m = sendKey(m, 'z')
		m = sendKey(m, 'a')
		m = sendKey(m, 'g')
		m = sendKey(m, 'a')
		m = sendSpecial(m, tea.KeyBackspace)
		m = sendKey(m, 'b')
		m = sendKey(m, 'z')
		m = sendKey(m, 'v')
		_ = sendKey(m, 'a')

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, "BB", doc.Text().String())
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

	t.Run("replays a two-key normal-mode binding", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestKeyAction(km, "rec", m.MacroRecordAction,
			[]command.KeyEvent{char('z')})
		bindNormalTestKeyAction(km, "play", m.MacroReplayAction,
			[]command.KeyEvent{char('v')})
		bindNormalTestAction(km, "dd",
			func(e *view.Editor) {
				action.InsertMode(e)
				action.InsertChar(e, 'D')
				action.NormalMode(e)
			}, []command.KeyEvent{char('d'), char('d')})
		m = resize(m, 80, 24)

		m = sendKey(m, 'z')
		m = sendKey(m, 'a')
		m = sendKey(m, 'd')
		m = sendKey(m, 'd')
		m = sendKey(m, 'z')
		m = sendKey(m, 'v')
		_ = sendKey(m, 'a')

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, "DD", doc.Text().String())
	})

	t.Run("replays a recorded count prefix", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestKeyAction(km, "rec", m.MacroRecordAction,
			[]command.KeyEvent{char('z')})
		bindNormalTestKeyAction(km, "play", m.MacroReplayAction,
			[]command.KeyEvent{char('v')})
		bindTestAction(bindTestActionArgs{
			km:      km,
			mode:    view.ModeNormal,
			name:    "insert_x",
			counted: true,
			fn: func(e *view.Editor) command.Continuation {
				n := e.Count()
				if n == 0 {
					n = 1
				}
				action.InsertMode(e)
				for range n {
					action.InsertChar(e, 'x')
				}
				action.NormalMode(e)
				return nil
			},
			seqs: [][]command.KeyEvent{{char('x')}},
		})
		m = resize(m, 80, 24)

		// record into 'a': count 3, then the single-key insert binding
		m = sendKey(m, 'z')
		m = sendKey(m, 'a')
		m = sendKey(m, '3')
		m = sendKey(m, 'x')
		m = sendKey(m, 'z')
		m = sendKey(m, 'v')
		_ = sendKey(m, 'a')

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		// the count-prefixed insert runs once live while recording, then
		// again on replay: 3 x's each time
		assert.Equal(t, "xxxxxx", doc.Text().String())
	})

	t.Run("recording writes to its register", func(t *testing.T) {
		m, e := macroModel(t)
		m = sendKey(m, 'z')
		m = sendKey(m, 'a')
		m = sendKey(m, 'i')
		m = sendKey(m, 'x')
		m = sendSpecial(m, tea.KeyEscape)
		_ = sendKey(m, 'z')

		text, ok := e.Registers().First('a')
		assert.True(t, ok)
		assert.Equal(t, "i x esc", text)
	})

	t.Run("replays a register-written macro", func(t *testing.T) {
		// registers are what the session persists, so a macro restored from
		// one replays without any separate macro state being installed
		m, e := macroModel(t)
		e.Registers().Set('a', "i x esc")

		m = sendKey(m, 'v')
		_ = sendKey(m, 'a')

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, "x", doc.Text().String())
	})

	t.Run("dangling count does not leak past replay", func(t *testing.T) {
		m, e := macroModel(t)
		e.Registers().Set('a', "3")

		m = sendKey(m, 'v')
		_ = sendKey(m, 'a')

		assert.Equal(t, 0, e.Count())
	})
}

func macroModelWithContinuation(t *testing.T) (ui.Model, *view.Editor) {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestKeyAction(km, "rec", m.MacroRecordAction,
		[]command.KeyEvent{char('z')})
	bindNormalTestKeyAction(km, "play", m.MacroReplayAction,
		[]command.KeyEvent{char('v')})
	// 'g' returns a continuation, followed by 'x' it inserts "X"
	bindNormalTestKeyAction(km, "g-prefix",
		func(ed *view.Editor) command.Continuation {
			return func(
				ed *view.Editor, k command.KeyEvent,
			) (command.Continuation, command.Transition) {
				if k.Code.Char == 'x' {
					action.InsertMode(ed)
					action.InsertChar(ed, 'X')
					action.NormalMode(ed)
				}
				return nil, command.ContinuationDone
			}
		}, []command.KeyEvent{char('g')})
	m = resize(m, 80, 24)
	return m, e
}

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
