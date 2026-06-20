package defaults

import (
	"bytes"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

// Registry owns the installed default commands, their runtime options, and
// their config sections
type Registry struct {
	km       *command.Keymaps
	sections []command.Section
	options  map[string]command.Option
}

// Keymaps returns the registry-owned key mappings
func (r *Registry) Keymaps() *command.Keymaps {
	return r.km
}

// RegisterCommand registers a command. The action name is automatically
// prepended to Aliases so it is typeable from the command line
func (r *Registry) RegisterCommand(name string, c command.Command) error {
	if c.Run == nil {
		return nil
	}
	if !slices.Contains(c.Aliases, name) {
		c.Aliases = append([]string{name}, c.Aliases...)
	}
	return r.km.Register(name, c)
}

// ApplyTOML resets all sections to defaults, decodes the merged TOML map into
// each section, then calls each section's Apply to push typed values into
// editor Options. Pass an empty map when no config file is present.
func (r *Registry) ApplyTOML(e *view.Editor, raw map[string]any) error {
	for _, s := range r.sections {
		s.Reset()
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(raw); err != nil {
		return err
	}
	tomlStr := buf.String()
	for _, s := range r.sections {
		if _, err := toml.Decode(tomlStr, s.Config); err != nil {
			return err
		}
		if s.Apply != nil {
			s.Apply(e)
		}
	}
	return nil
}

// ResetSections restores all section configs to their defaults before a config
// reload
func (r *Registry) ResetSections() {
	for _, s := range r.sections {
		s.Reset()
	}
}

// LookupOption returns the registered Option for the given key, if any
func (r *Registry) LookupOption(key string) (command.Option, bool) {
	o, ok := r.options[normalizeOptionKey(key)]
	return o, ok
}

// OptionKeys returns all registered option keys in sorted order
func (r *Registry) OptionKeys() []string {
	keys := make([]string, 0, len(r.options))
	for k := range r.options {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// BoolOptionKeys returns registered option keys that support toggle
func (r *Registry) BoolOptionKeys() []string {
	var keys []string
	for k, o := range r.options {
		if o.Toggle != nil {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

// optionCompleter returns a CompletionFunc over all registered option keys
func (r *Registry) optionCompleter() command.CompletionFunc {
	return func(_ *view.Editor, input string) []command.Completion {
		return command.StaticCompleter(r.OptionKeys()...)(nil, input)
	}
}

// boolOptionCompleter returns a CompletionFunc over toggleable option keys
func (r *Registry) boolOptionCompleter() command.CompletionFunc {
	return func(_ *view.Editor, input string) []command.Completion {
		return command.StaticCompleter(r.BoolOptionKeys()...)(nil, input)
	}
}

func (r *Registry) registerModule(m command.Module) error {
	for name, cmd := range m.Commands {
		if err := r.RegisterCommand(name, cmd); err != nil {
			return err
		}
	}
	if m.Section != nil {
		r.sections = append(r.sections, *m.Section)
	}
	for _, o := range m.Options {
		if r.options == nil {
			r.options = make(map[string]command.Option)
		}
		r.options[normalizeOptionKey(o.Key)] = o
	}
	return nil
}

func normalizeOptionKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
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
