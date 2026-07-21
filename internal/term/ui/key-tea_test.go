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
		assert.Equal(t, special(command.PageUp), up)
		assert.Equal(t, special(command.PageDown), down)
	})

	t.Run("keypad page keys", func(t *testing.T) {
		up := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyKpPgUp})
		down := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyKpPgDown})
		assert.Equal(t, special(command.PageUp), up)
		assert.Equal(t, special(command.PageDown), down)
	})

	t.Run("enter key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyEnter})
		assert.Equal(t, special(command.Enter), got)
	})

	t.Run("backspace key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
		assert.Equal(t, special(command.Backspace), got)
	})

	t.Run("delete key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyDelete})
		assert.Equal(t, special(command.Delete), got)
	})

	t.Run("escape key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyEscape})
		assert.Equal(t, special(command.Escape), got)
	})

	t.Run("tab key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyTab})
		assert.Equal(t, special(command.Tab), got)
	})

	t.Run("arrow keys", func(t *testing.T) {
		assert.Equal(t, special(command.Up),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyUp}))
		assert.Equal(t, special(command.Down),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyDown}))
		assert.Equal(t, special(command.Left),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyLeft}))
		assert.Equal(t, special(command.Right),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyRight}))
	})

	t.Run("home and end keys", func(t *testing.T) {
		assert.Equal(t, special(command.Home),
			ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyHome}))
		assert.Equal(t, special(command.End),
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

	t.Run("unicode uppercase text adds shift", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: 'å', Text: "Å"})
		assert.Equal(t, command.KeyEvent{
			Code: command.KeyCode{Char: 'Å'},
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

	t.Run("ctrl+punctuation from code when no text", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: '\\', Mod: tea.ModCtrl})
		assert.Equal(t, command.KeyEvent{
			Code: command.KeyCode{Char: '\\'},
			Mods: command.ModCtrl,
		}, got)
	})

	t.Run("alt modifier on special key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{
			Code: tea.KeyEnter,
			Mod:  tea.ModAlt,
		})
		assert.Equal(t, command.KeyEvent{
			Code: command.KeyCode{Special: command.Enter},
			Mods: command.ModAlt,
		}, got)
	})

	t.Run("shift modifier on special key", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{
			Code: tea.KeyTab,
			Mod:  tea.ModShift,
		})
		assert.Equal(t, command.KeyEvent{
			Code: command.KeyCode{Special: command.Tab},
			Mods: command.ModShift,
		}, got)
	})

	t.Run("unknown key falls back to key string", func(t *testing.T) {
		got := ui.FromTeaKey(tea.KeyPressMsg{Code: tea.KeyF1})
		assert.NotEmpty(t, got.Code.Special)
	})
}
