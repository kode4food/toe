package command

import (
	"bytes"
	"maps"
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
	prefixes []Option
}

func NewRegistry(km *Keymaps) *Registry {
	return &Registry{km: km}
}

// RegisterCommand registers a command with its kebab-cased name as the first
// alias
func (r *Registry) RegisterCommand(name string, c Command) error {
	if c.Run == nil {
		return nil
	}
	alias := strings.ReplaceAll(name, "_", "-")
	c.Aliases = append([]string{alias}, c.Aliases...)
	if err := r.km.Register(name, c); err != nil {
		return err
	}
	r.km.byAlias[name] = r.km.byName[name]
	return nil
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
	for _, lbl := range m.Labels {
		for _, mode := range lbl.Modes {
			r.km.LabelNode(mode, lbl.Seq, lbl.Label)
		}
	}
	for _, o := range m.Options {
		if r.options == nil {
			r.options = make(map[string]Option)
		}
		if o.KeyGet != nil || o.KeySet != nil {
			r.prefixes = append(r.prefixes, o)
			continue
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
	if o, ok := r.options[normalizeOptionKey(key)]; ok {
		return o, true
	}
	o, ok := r.lookupPrefixOption(key)
	if !ok || o.KeySet == nil {
		return Option{}, false
	}
	return Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			values, err := o.KeyGet(e)
			if err != nil {
				return "", err
			}
			return values[key], nil
		},
		Set: func(e *view.Editor, value string) error {
			return o.KeySet(e, key, value)
		},
		Complete: o.Complete,
	}, true
}

func (r *Registry) lookupPrefixOption(key string) (Option, bool) {
	key = normalizeOptionKey(key)
	for _, o := range r.prefixes {
		if strings.HasPrefix(key, normalizeOptionKey(o.Key)) {
			return o, true
		}
	}
	return Option{}, false
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
	for _, o := range r.prefixes {
		if o.KeyGet == nil {
			continue
		}
		values, err := o.KeyGet(e)
		if err != nil {
			return nil, err
		}
		maps.Copy(out, values)
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
			continue
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
	return func(_ *view.Editor, _ *Args, input string) []Completion {
		keys := r.OptionKeys()
		for _, o := range r.prefixes {
			keys = append(keys, o.Key)
		}
		slices.Sort(keys)
		return matchPrefix(keys, input)
	}
}

func (r *Registry) BoolOptionCompleter() CompletionFunc {
	return func(_ *view.Editor, _ *Args, input string) []Completion {
		return matchPrefix(r.BoolOptionKeys(), input)
	}
}

// OptionValueCompleter completes an option's value, dispatching to the
// completer registered against the option named by the already-parsed first
// positional argument (e.g. the key in "set <key> <value>")
func (r *Registry) OptionValueCompleter() CompletionFunc {
	return func(e *view.Editor, args *Args, input string) []Completion {
		key, ok := args.Get(0)
		if !ok {
			return nil
		}
		o, ok := r.LookupOption(key)
		if !ok {
			return nil
		}
		if o.Complete != nil {
			return o.Complete(e, args, input)
		}
		if o.Get == nil {
			return nil
		}
		value, err := o.Get(e)
		if err != nil {
			return nil
		}
		return matchPrefix([]string{value}, input)
	}
}

func normalizeOptionKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
