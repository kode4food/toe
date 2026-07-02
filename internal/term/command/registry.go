package command

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/kode4food/toe/internal/view"
)

// Registry owns installed commands, runtime options, and config sections
type Registry struct {
	km       *Keymaps
	sections []Section
	options  map[string]Option
}

func NewRegistry(km *Keymaps) *Registry {
	return &Registry{km: km}
}

// RegisterCommand registers a command. The action name is automatically
// prepended to Aliases so it is typeable from the command line
func (r *Registry) RegisterCommand(name string, c Command) error {
	if c.Run == nil {
		return nil
	}
	if !slices.Contains(c.Aliases, name) {
		c.Aliases = append([]string{name}, c.Aliases...)
	}
	return r.km.Register(name, c)
}

func (r *Registry) RegisterModule(m Module) error {
	for _, c := range m.Commands {
		if err := r.RegisterCommand(c.Name, c); err != nil {
			return err
		}
	}
	if m.Section != nil {
		r.sections = append(r.sections, *m.Section)
	}
	for _, o := range m.Options {
		if r.options == nil {
			r.options = make(map[string]Option)
		}
		r.options[normalizeOptionKey(o.Key)] = o
	}
	return nil
}

// ApplyTOML resets all sections to defaults, decodes the merged TOML map into
// each section, then calls each section's Apply to push typed values into
// editor Options. Pass an empty map when no config file is present
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

// LookupOption returns the registered Option for the given key, if any
func (r *Registry) LookupOption(key string) (Option, bool) {
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

// OptionValues returns the current string value for every registered runtime
// option
func (r *Registry) OptionValues(e *view.Editor) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range r.OptionKeys() {
		o := r.options[key]
		value, err := o.Get(e)
		if err != nil {
			return nil, err
		}
		out[key] = value
	}
	return out, nil
}

// ApplyOptionValues applies a set of runtime option strings through the same
// handlers used by :set
func (r *Registry) ApplyOptionValues(
	e *view.Editor, values map[string]string,
) error {
	for key, value := range values {
		o, ok := r.LookupOption(key)
		if !ok {
			return fmt.Errorf("%w: %s", view.ErrSessionUnknownOption, key)
		}
		if err := o.Set(e, value); err != nil {
			return err
		}
	}
	return nil
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

func (r *Registry) OptionCompleter() CompletionFunc {
	return func(_ *view.Editor, input string) []Completion {
		return StaticCompleter(r.OptionKeys()...)(nil, input)
	}
}

func (r *Registry) BoolOptionCompleter() CompletionFunc {
	return func(_ *view.Editor, input string) []Completion {
		return StaticCompleter(r.BoolOptionKeys()...)(nil, input)
	}
}

func normalizeOptionKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
