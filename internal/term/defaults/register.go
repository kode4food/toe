package defaults

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

// method wraps an editor method into a command Run
func method(fn func(*view.Editor)) command.Run {
	return Runner(fn)
}

// Key sequence helpers

func char(r rune) []command.KeyEvent {
	ev := command.KeyEvent{Code: command.KeyCode{Char: r}}
	if unicode.IsUpper(r) {
		ev.Mods |= command.ModShift
	}
	return []command.KeyEvent{ev}
}

func special(name string) []command.KeyEvent {
	return []command.KeyEvent{{Code: command.KeyCode{Special: name}}}
}

func ctrl(r rune) []command.KeyEvent {
	return []command.KeyEvent{
		{Code: command.KeyCode{Char: r}, Mods: command.ModCtrl},
	}
}

func alt(r rune) []command.KeyEvent {
	return []command.KeyEvent{
		{Code: command.KeyCode{Char: r}, Mods: command.ModAlt},
	}
}

func shift(name string) []command.KeyEvent {
	return []command.KeyEvent{
		{Code: command.KeyCode{Special: name}, Mods: command.ModShift},
	}
}

func altSpecial(name string) []command.KeyEvent {
	return []command.KeyEvent{
		{Code: command.KeyCode{Special: name}, Mods: command.ModAlt},
	}
}

func prefixed(
	prefix []command.KeyEvent,
) func(...[]command.KeyEvent) []command.KeyEvent {
	return func(seqs ...[]command.KeyEvent) []command.KeyEvent {
		result := make([]command.KeyEvent, len(prefix)+len(seqs[0]))
		copy(result, prefix)
		copy(result[len(prefix):], seqs[0])
		return result
	}
}

func keyBinding(seqs ...[]command.KeyEvent) []command.KeyBinding {
	return []command.KeyBinding{seqs}
}

func keys(seqs ...[]command.KeyEvent) map[string][]command.KeyBinding {
	return map[string][]command.KeyBinding{"*": keyBinding(seqs...)}
}

func sig() command.Signature {
	return command.DefaultSignature()
}

func minArgs(n int) command.Signature {
	return command.Signature{Positionals: command.Positionals{Min: n}}
}

func optionalArg() command.Signature {
	return command.Signature{
		Positionals: command.Positionals{Min: 0, Max: 1},
	}
}
