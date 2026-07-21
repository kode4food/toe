// Package kit holds the shared vocabulary for declaring default command
// modules: key-sequence builders, signature helpers, option and completion
// constructors. Module packages import it to keep their bindings terse
package kit

import (
	"slices"
	"unicode"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

var (
	LeaderBinding = Or(Char(' '), Ctrl('\\'))

	// LeaderPrefix enters the shared leader menu, reachable by Space or Ctrl-\
	LeaderPrefix = Prefixed(LeaderBinding)
)

// Terse bindings for the plain special keys, named by their keycap form
var (
	Up    = Special(command.Up)
	Down  = Special(command.Down)
	Left  = Special(command.Left)
	Right = Special(command.Right)
	Home  = Special(command.Home)
	End   = Special(command.End)
	PgUp  = Special(command.PageUp)
	PgDn  = Special(command.PageDown)
	Tab   = Special(command.Tab)
	Esc   = Special(command.Escape)
	Ret   = Special(command.Enter)
	Del   = Special(command.Delete)
	Bksp  = Special(command.Backspace)
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

// Char is the single-key binding for a rune, adding Shift for an uppercase
func Char(r rune) command.KeyBinding {
	var mods command.KeyModifiers
	if unicode.IsUpper(r) {
		mods |= command.ModShift
	}
	return single(command.KeyCode{Char: r}, mods)
}

// Special is the binding for a special key
func Special(s command.Special) command.KeyBinding {
	return single(command.KeyCode{Special: s}, 0)
}

// Ctrl is the binding for Ctrl held with a rune
func Ctrl(r rune) command.KeyBinding {
	return single(command.KeyCode{Char: r}, command.ModCtrl)
}

// Alt is the binding for Alt held with a rune
func Alt(r rune) command.KeyBinding {
	return single(command.KeyCode{Char: r}, command.ModAlt)
}

// Shift is the binding for Shift held with a special key
func Shift(s command.Special) command.KeyBinding {
	return single(command.KeyCode{Special: s}, command.ModShift)
}

// AltSpecial is the binding for Alt held with a special key
func AltSpecial(s command.Special) command.KeyBinding {
	return single(command.KeyCode{Special: s}, command.ModAlt)
}

// Or unions its arguments into a single binding of alternative sequences, so
// any one of them triggers the command or enters the shared prefix state
func Or(alts ...command.KeyBinding) command.KeyBinding {
	return slices.Concat(alts...)
}

// Seq concatenates parts into sequences, expanding an Or at any position into
// the cross-product of the alternatives around it
func Seq(parts ...command.KeyBinding) command.KeyBinding {
	out := command.KeyBinding{nil}
	for _, part := range parts {
		var next command.KeyBinding
		for _, prefix := range out {
			for _, suffix := range part {
				next = append(next, slices.Concat(prefix, suffix))
			}
		}
		out = next
	}
	return out
}

// Prefixed returns a builder that joins prefix ahead of any of the sub-bindings
func Prefixed(
	prefix command.KeyBinding,
) func(...command.KeyBinding) command.KeyBinding {
	return func(subs ...command.KeyBinding) command.KeyBinding {
		return Seq(prefix, Or(subs...))
	}
}

// KeyBinding unions alternatives into the one-element binding list a mode maps
// to, for building a per-mode Keys map by hand
func KeyBinding(alts ...command.KeyBinding) []command.KeyBinding {
	return []command.KeyBinding{Or(alts...)}
}

// Keys builds the all-modes ("*") binding map from the given alternatives
func Keys(alts ...command.KeyBinding) map[string][]command.KeyBinding {
	return map[string][]command.KeyBinding{"*": KeyBinding(alts...)}
}

// Leader binds ch under the shared leader (see [LeaderPrefix])
func Leader(ch rune) map[string][]command.KeyBinding {
	return map[string][]command.KeyBinding{"*": {LeaderPrefix(Char(ch))}}
}

// Window binds each sub under the Ctrl-w window chord or the leader's w menu,
// so window management shares the leader wherever the leader is reachable
func Window(subs ...command.KeyBinding) map[string][]command.KeyBinding {
	prefix := Prefixed(Or(Ctrl('w'), LeaderPrefix(Char('w'))))
	return map[string][]command.KeyBinding{"*": {prefix(subs...)}}
}

// Label names a prefix key sequence for the pending-key hint popup
func Label(
	label string, prefix command.KeyBinding, modes ...string,
) command.PrefixLabel {
	return command.PrefixLabel{Modes: modes, Seq: prefix, Label: label}
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

// single wraps one key event into a binding of a one-key sequence
func single(
	code command.KeyCode, mods command.KeyModifiers,
) command.KeyBinding {
	return command.KeyBinding{{{Code: code, Mods: mods}}}
}
