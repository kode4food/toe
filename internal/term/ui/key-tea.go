package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
)

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

	switch k.Code {
	case tea.KeyEnter:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "ret"}, Mods: mods,
		}
	case tea.KeyBackspace:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "backspace"}, Mods: mods,
		}
	case tea.KeyDelete:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "del"}, Mods: mods,
		}
	case tea.KeyEscape:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "esc"}, Mods: mods,
		}
	case tea.KeyTab:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "tab"}, Mods: mods,
		}
	case tea.KeyUp:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "up"}, Mods: mods,
		}
	case tea.KeyDown:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "down"}, Mods: mods,
		}
	case tea.KeyLeft:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "left"}, Mods: mods,
		}
	case tea.KeyRight:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "right"}, Mods: mods,
		}
	case tea.KeyHome:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "home"}, Mods: mods,
		}
	case tea.KeyEnd:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "end"}, Mods: mods,
		}
	case tea.KeyPgUp, tea.KeyKpPgUp:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "pageup"}, Mods: mods,
		}
	case tea.KeyPgDown, tea.KeyKpPgDown:
		return command.KeyEvent{
			Code: command.KeyCode{Special: "pagedown"}, Mods: mods,
		}
	case tea.KeySpace:
		return command.KeyEvent{
			Code: command.KeyCode{Char: ' '}, Mods: mods,
		}
	default:
		if k.Text != "" {
			ch := []rune(k.Text)[0]
			if ch >= 'A' && ch <= 'Z' {
				mods |= command.ModShift
			}
			return command.KeyEvent{
				Code: command.KeyCode{Char: ch}, Mods: mods,
			}
		}
		if k.Code >= 'a' && k.Code <= 'z' && mods.Has(command.ModCtrl) {
			return command.KeyEvent{
				Code: command.KeyCode{Char: k.Code}, Mods: mods,
			}
		}
		return command.KeyEvent{
			Code: command.KeyCode{Special: k.String()}, Mods: mods,
		}
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
