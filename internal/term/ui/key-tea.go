package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
)

var teaSpecialKeys = map[rune]string{
	tea.KeyEnter:     "ret",
	tea.KeyBackspace: "backspace",
	tea.KeyDelete:    "del",
	tea.KeyEscape:    "esc",
	tea.KeyTab:       "tab",
	tea.KeyUp:        "up",
	tea.KeyDown:      "down",
	tea.KeyLeft:      "left",
	tea.KeyRight:     "right",
	tea.KeyHome:      "home",
	tea.KeyEnd:       "end",
	tea.KeyPgUp:      "pageup",
	tea.KeyKpPgUp:    "pageup",
	tea.KeyPgDown:    "pagedown",
	tea.KeyKpPgDown:  "pagedown",
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
		if ch >= 'A' && ch <= 'Z' {
			mods |= command.ModShift
		}
		return command.KeyEvent{Code: command.KeyCode{Char: ch}, Mods: mods}
	}
	if k.Code >= 'a' && k.Code <= 'z' && mods.Has(command.ModCtrl) {
		return command.KeyEvent{Code: command.KeyCode{Char: k.Code}, Mods: mods}
	}
	return command.KeyEvent{
		Code: command.KeyCode{Special: k.String()}, Mods: mods,
	}
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
