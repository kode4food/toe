package testutil

import "github.com/kode4food/toe/internal/term/command"

// Char builds a KeyEvent for a printable character
func Char(ch rune) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Char: ch}}
}

// Special builds a KeyEvent for a special key
func Special(s command.Special) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Special: s}}
}
