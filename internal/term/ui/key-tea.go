package ui

import (
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
)

var teaSpecialKeys = map[rune]command.Special{
	tea.KeyEnter:     command.Enter,
	tea.KeyBackspace: command.Backspace,
	tea.KeyDelete:    command.Delete,
	tea.KeyEscape:    command.Escape,
	tea.KeyTab:       command.Tab,
	tea.KeyUp:        command.Up,
	tea.KeyDown:      command.Down,
	tea.KeyLeft:      command.Left,
	tea.KeyRight:     command.Right,
	tea.KeyHome:      command.Home,
	tea.KeyEnd:       command.End,
	tea.KeyPgUp:      command.PageUp,
	tea.KeyKpPgUp:    command.PageUp,
	tea.KeyPgDown:    command.PageDown,
	tea.KeyKpPgDown:  command.PageDown,
}

// FromTeaKey converts a Bubbletea v2 KeyPressMsg to a KeyEvent
func FromTeaKey(k tea.KeyPressMsg) command.KeyEvent {
	var mods command.KeyModifiers
	if k.Mod&tea.ModAlt != 0 {
		mods |= command.ModAlt
	}
	if k.Mod&tea.ModCtrl != 0 {
		mods |= command.ModCtrl
	}
	if k.Mod&tea.ModShift != 0 {
		mods |= command.ModShift
	}
	if k.Code == tea.KeySpace {
		return command.KeyEvent{Code: command.KeyCode{Char: ' '}, Mods: mods}
	}
	if name, ok := teaSpecialKeys[k.Code]; ok {
		return command.KeyEvent{
			Code: command.KeyCode{Special: name}, Mods: mods,
		}
	}
	if k.Text != "" {
		ch := []rune(k.Text)[0]
		if unicode.IsUpper(ch) {
			mods |= command.ModShift
		}
		return command.KeyEvent{Code: command.KeyCode{Char: ch}, Mods: mods}
	}
	if mods.Has(command.ModCtrl) && isASCIIGraphic(k.Code) {
		return command.KeyEvent{Code: command.KeyCode{Char: k.Code}, Mods: mods}
	}
	return command.KeyEvent{
		Code: command.KeyCode{Special: command.SpecialUnknown}, Mods: mods,
	}
}

func isASCIIGraphic(r rune) bool {
	return r > ' ' && r < 0x7f
}

func signalToCmd(s command.Signal) tea.Cmd {
	switch s {
	case command.SignalQuit:
		return tea.Quit
	case command.SignalClearScreen:
		return tea.ClearScreen
	default:
		return nil
	}
}
