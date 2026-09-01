package command

import (
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kode4food/toe/internal/toml"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

// Registry owns installed commands, runtime options, and config sections
type Registry struct {
	keymaps  *Keymaps
	sections []Section
	options  map[string]*Option
	prefixes []*Option
}

const (
	currentOptionMarkerIcon  = "\uf42e" // '' - oct-check
	currentOptionMarkerAscii = "*"
)

// NewRegistry returns an empty command registry bound to keymaps
func NewRegistry(km *Keymaps) *Registry {
	return &Registry{keymaps: km}
}

// RegisterModule adds a module's commands, options, and bindings
func (r *Registry) RegisterModule(m Module) error {
	tr := make(i18n.Translations,
		len(m.Commands)+len(m.Options)+len(m.Translations),
	)
	for _, c := range m.Commands {
		tr[docStringKey(kebabName(c.Name))] = c.DocString
	}
	for _, o := range m.Options {
		if o.DocString != "" {
			tr[optionDocStringKey(o.Key)] = o.DocString
		}
	}
	maps.Copy(tr, m.Translations)
	i18n.Register(tr)
	for _, c := range m.Commands {
		c.DocString = i18n.Text(docStringKey(kebabName(c.Name)))
		if err := r.RegisterCommand(c.Name, c); err != nil {
			return err
		}
	}
	if m.Section != nil {
		r.sections = append(r.sections, *m.Section)
	}
	for _, lbl := range m.Labels {
		for _, mode := range lbl.Modes.Split() {
			r.keymaps.LabelNode(mode, lbl.Seq, lbl.Label)
		}
	}
	for i, o := range m.Options {
		if r.options == nil {
			r.options = make(map[string]*Option)
		}
		localizeOptionDoc(&m.Options[i])
		if o.KeyGet != nil || o.KeySet != nil {
			r.prefixes = append(r.prefixes, &m.Options[i])
			continue
		}
		r.options[normalizeOptionKey(o.Key)] = &m.Options[i]
	}
	return nil
}

// RegisterCommand registers a command with kebab-cased name as the first alias
func (r *Registry) RegisterCommand(name string, c Command) error {
	if c.Name == "" {
		c.Name = name
	}
	localizeCommandDoc(&c)
	c.Aliases = append([]string{kebabName(c.Name)}, c.Aliases...)
	if err := r.keymaps.Register(name, c); err != nil {
		return err
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
	text, err := toml.Encode(raw)
	if err != nil {
		return err
	}
	for _, s := range r.sections {
		if err := toml.Decode(text, s.Config); err != nil {
			return err
		}
		if s.Apply != nil {
			s.Apply(e)
		}
	}
	return nil
}

// LookupOption returns the registered Option for the given key, if any
func (r *Registry) LookupOption(key string) *Option {
	if o, ok := r.options[normalizeOptionKey(key)]; ok {
		return o
	}
	o := r.lookupPrefixOption(key)
	if o == nil || o.KeySet == nil {
		return nil
	}
	return &Option{
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
	}
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

// ChangedOptionValues returns the option values that differ from the editor's
// base options. With no base recorded, every option counts as changed
func (r *Registry) ChangedOptionValues(
	e *view.Editor,
) (map[string]string, error) {
	values, err := r.OptionValues(e)
	if err != nil {
		return nil, err
	}
	base := e.BaseOptions()
	out := map[string]string{}
	for key, value := range values {
		if base[key] != value {
			out[key] = value
		}
	}
	return out, nil
}

// ApplyOptionValues applies a set of runtime option strings through the same
// handlers used by :set
func (r *Registry) ApplyOptionValues(
	e *view.Editor, values map[string]string,
) error {
	for key, value := range values {
		o := r.LookupOption(key)
		if o == nil {
			continue
		}
		if err := o.Set(e, value); err != nil {
			return err
		}
	}
	return nil
}

// BoolOptionKeys returns settable option keys that support toggle
func (r *Registry) BoolOptionKeys() []string {
	var keys []string
	for k, o := range r.options {
		if o.Toggle != nil && !o.Private {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

// OptionCompleter completes every option a user may set
func (r *Registry) OptionCompleter() CompletionFunc {
	return func(_ *view.Editor, _ *Args, input string) []Completion {
		var keys []string
		for key, o := range r.options {
			if !o.Private {
				keys = append(keys, key)
			}
		}
		for _, o := range r.prefixes {
			if !o.Private {
				keys = append(keys, o.Key)
			}
		}
		slices.Sort(keys)
		return r.describeOptions(matchFuzzy(keys, input))
	}
}

// BoolOptionCompleter completes only the boolean option keys
func (r *Registry) BoolOptionCompleter() CompletionFunc {
	return func(_ *view.Editor, _ *Args, input string) []Completion {
		return r.describeOptions(matchFuzzy(r.BoolOptionKeys(), input))
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
		return completeOptionValue(e, args, input, r.LookupOption(key))
	}
}

// OptionValueCompleterFor completes values for a fixed option key
func (r *Registry) OptionValueCompleterFor(key string) CompletionFunc {
	return func(e *view.Editor, args *Args, input string) []Completion {
		return completeOptionValue(e, args, input, r.LookupOption(key))
	}
}

func (r *Registry) describeOptions(items []Completion) []Completion {
	for i, item := range items {
		if o, ok := r.options[normalizeOptionKey(item.Text)]; ok {
			items[i].Detail = o.DocString
			continue
		}
		if o := r.lookupPrefixOption(item.Text); o != nil {
			items[i].Detail = o.DocString
		}
	}
	return items
}

func (r *Registry) lookupPrefixOption(key string) *Option {
	key = normalizeOptionKey(key)
	for _, o := range r.prefixes {
		if strings.HasPrefix(key, normalizeOptionKey(o.Key)) {
			return o
		}
	}
	return nil
}

func normalizeOptionKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func completeOptionValue(
	e *view.Editor, args *Args, input string, o *Option,
) []Completion {
	if o == nil {
		return nil
	}
	var current string
	if o.Get != nil {
		current, _ = o.Get(e)
	}
	nerd := e.Options().NerdFonts
	if o.Complete != nil {
		return markCurrentOption(o.Complete(e, args, input), current, nerd)
	}
	if current == "" {
		return nil
	}
	items := matchFuzzy([]string{current}, input)
	return markCurrentOption(items, current, nerd)
}

func markCurrentOption(
	items []Completion, current string, nerd bool,
) []Completion {
	marker := currentOptionMarkerAscii
	if nerd {
		marker = currentOptionMarkerIcon
	}
	for i := range items {
		if items[i].Text != current {
			continue
		}
		display := items[i].Display
		if display == "" {
			display = items[i].Text
		}
		items[i].Display = display + " " + marker
		items[i].Indices = append(
			items[i].Indices, utf8.RuneCountInString(display)+1,
		)
	}
	return items
}
