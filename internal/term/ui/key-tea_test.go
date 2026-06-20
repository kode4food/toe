package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
)

func TestFromTeaKey(t *testing.T) {
	t.Run("page keys ignore text fallback", func(t *testing.T) {
		up := ui.FromTeaKey(tea.KeyPressMsg{
			Code: tea.KeyPgUp, Text: "pgup",
		})
		down := ui.FromTeaKey(tea.KeyPressMsg{
			Code: tea.KeyPgDown, Text: "pgdown",
		})
		assert.Equal(t, command.Special("pageup"), up)
		assert.Equal(t, command.Special("pagedown"), down)
	})

	t.Run("keypad page keys", func(t *testing.T) {
		up := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyKpPgUp})
		down := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyKpPgDown})
		assert.Equal(t, command.Special("pageup"), up)
		assert.Equal(t, command.Special("pagedown"), down)
	})

	t.Run("enter key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyEnter})
		assert.Equal(t, command.Special("ret"), got)
	})

	t.Run("backspace key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
		assert.Equal(t, command.Special("backspace"), got)
	})

	t.Run("delete key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyDelete})
		assert.Equal(t, command.Special("del"), got)
	})

	t.Run("escape key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyEscape})
		assert.Equal(t, command.Special("esc"), got)
	})

	t.Run("tab key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyTab})
		assert.Equal(t, command.Special("tab"), got)
	})

	t.Run("arrow keys", func(t *testing.T) {
		assert.Equal(t, command.Special("up"),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyUp}))
		assert.Equal(t, command.Special("down"),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyDown}))
		assert.Equal(t, command.Special("left"),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyLeft}))
		assert.Equal(t, command.Special("right"),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyRight}))
	})

	t.Run("home and end keys", func(t *testing.T) {
		assert.Equal(t, command.Special("home"),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyHome}))
		assert.Equal(t, command.Special("end"),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyEnd}))
	})

	t.Run("space key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeySpace})
		assert.Equal(t, command.KeyEvent{Code: command.KeyCode{Char: ' '}}, got)
	})

	t.Run("lowercase text produces char", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: 'a', Text: "a"})
		assert.Equal(t,
			command.KeyEvent{Code: command.KeyCode{Char: 'a'}}, got)
	})

	t.Run("uppercase text adds shift modifier", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: 'A', Text: "A"})
		assert.Equal(t, command.KeyEvent{
			Code: command.KeyCode{Char: 'A'},
			Mods: command.ModShift,
		}, got)
	})

	t.Run("ctrl+letter from code when no text", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		assert.Equal(t, command.KeyEvent{
			Code: command.KeyCode{Char: 'c'},
			Mods: command.ModCtrl,
		}, got)
	})

	t.Run("alt modifier on special key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{
			Code: tea.KeyEnter,
			Mod:  tea.ModAlt,
		})
		assert.Equal(t, command.KeyEvent{
			Code: command.KeyCode{Special: "ret"},
			Mods: command.ModAlt,
		}, got)
	})

	t.Run("shift modifier on special key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{
			Code: tea.KeyTab,
			Mod:  tea.ModShift,
		})
		assert.Equal(t, command.KeyEvent{
			Code: command.KeyCode{Special: "tab"},
			Mods: command.ModShift,
		}, got)
	})

	t.Run("unknown key falls back to key string", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyF1})
		assert.NotEmpty(t, got.Code.Special)
	})
}
