// Package kit holds the shared vocabulary for declaring default command
// modules: key-sequence builders, signature helpers, option and completion
// constructors. Module packages import it to keep their bindings terse
package kit

import (
	"unicode"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

// Runner wraps a view Action into a command Run
func Runner(fn command.Action) command.Run {
	return func(e *view.Editor, _ *command.Args) command.Result {
		fn(e)
		return command.Result{}
	}
}

// Continuation wraps a KeyAction into a command Run
func Continuation(fn command.KeyAction) command.Run {
	return func(e *view.Editor, _ *command.Args) command.Result {
		return command.Result{Continuation: fn(e)}
	}
}

// Char is the key sequence for a single rune, adding Shift for an uppercase
func Char(r rune) []command.KeyEvent {
	ev := command.KeyEvent{Code: command.KeyCode{Char: r}}
	if unicode.IsUpper(r) {
		ev.Mods |= command.ModShift
	}
	return []command.KeyEvent{ev}
}

// Special is the key sequence for a named special key (e.g. "left", "enter")
func Special(name string) []command.KeyEvent {
	return []command.KeyEvent{{Code: command.KeyCode{Special: name}}}
}

// Ctrl is the key sequence for Ctrl held with a rune
func Ctrl(r rune) []command.KeyEvent {
	return []command.KeyEvent{
		{Code: command.KeyCode{Char: r}, Mods: command.ModCtrl},
	}
}

// Alt is the key sequence for Alt held with a rune
func Alt(r rune) []command.KeyEvent {
	return []command.KeyEvent{
		{Code: command.KeyCode{Char: r}, Mods: command.ModAlt},
	}
}

// Shift is the key sequence for Shift held with a named special key
func Shift(name string) []command.KeyEvent {
	return []command.KeyEvent{
		{Code: command.KeyCode{Special: name}, Mods: command.ModShift},
	}
}

// AltSpecial is the key sequence for Alt held with a named special key
func AltSpecial(name string) []command.KeyEvent {
	return []command.KeyEvent{
		{Code: command.KeyCode{Special: name}, Mods: command.ModAlt},
	}
}

// Prefixed returns a builder that joins prefix ahead of each given sequence
func Prefixed(
	prefix []command.KeyEvent,
) func(...[]command.KeyEvent) []command.KeyEvent {
	return func(seqs ...[]command.KeyEvent) []command.KeyEvent {
		result := make([]command.KeyEvent, len(prefix)+len(seqs[0]))
		copy(result, prefix)
		copy(result[len(prefix):], seqs[0])
		return result
	}
}

// KeyBinding groups key sequences into a single binding
func KeyBinding(seqs ...[]command.KeyEvent) []command.KeyBinding {
	return []command.KeyBinding{seqs}
}

// Keys builds the all-modes ("*") binding map from the given sequences
func Keys(seqs ...[]command.KeyEvent) map[string][]command.KeyBinding {
	return map[string][]command.KeyBinding{"*": KeyBinding(seqs...)}
}

// Label names a prefix key sequence for the pending-key hint popup
func Label(label string, seq []command.KeyEvent, modes ...string) command.PrefixLabel {
	return command.PrefixLabel{Modes: modes, Seq: seq, Label: label}
}

// Sig is the default command signature
func Sig() command.Signature {
	return command.DefaultSignature()
}

// MinArgs is a signature requiring at least n positional arguments
func MinArgs(n int) command.Signature {
	return command.Signature{Positionals: command.Positionals{Min: n}}
}

// OptionalArg is a signature accepting zero or one positional argument
func OptionalArg() command.Signature {
	return command.Signature{
		Positionals: command.Positionals{Min: 0, Max: 1},
	}
}
