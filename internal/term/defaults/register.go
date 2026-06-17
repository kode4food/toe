package defaults

import (
	"slices"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type registry struct {
	km *command.Keymaps
}

// RegisterCommand registers a command with its configuration. The action name
// is automatically prepended to Aliases so it is typeable from the command line
func (r *registry) RegisterCommand(name string, c command.Command) {
	if c.Run == nil {
		return
	}
	if !slices.Contains(c.Aliases, name) {
		c.Aliases = append([]string{name}, c.Aliases...)
	}
	r.km.Register(name, c)
}

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
	return []command.KeyEvent{{Code: command.KeyCode{Char: r}}}
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

func sig() command.Signature {
	return command.DefaultSignature()
}

func minArgs(n int) command.Signature {
	return command.Signature{Positionals: command.Positionals{Min: n}}
}

func exactArgs(n int) command.Signature {
	return command.Signature{
		Positionals: command.Positionals{Min: n, Max: n},
	}
}

func optionalArg() command.Signature {
	return command.Signature{
		Positionals: command.Positionals{Min: 0, Max: 1},
	}
}

func rawAfter(after, lo, hi int, flags []command.Flag) command.Signature {
	return command.Signature{
		Positionals: command.Positionals{Min: lo, Max: hi},
		RawAfter:    after,
		Flags:       flags,
	}
}
