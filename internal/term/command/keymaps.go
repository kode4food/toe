package command

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Keymaps is the combined command registry and key-event dispatch trie
	Keymaps struct {
		modes    map[view.Mode]*keyTrieNode
		commands []Command
		byName   map[string]int
	}

	// KeyMatch is a binding matched by traversing the key trie
	KeyMatch struct {
		Action KeyResultAction
		When   func(*view.Editor) bool
		Name   string
		Prefix bool
	}

	keyTrieNode struct {
		children  map[KeyEvent]*keyTrieNode
		order     []KeyEvent
		action    KeyResultAction
		available func(*view.Editor) bool
		name      string
		label     string
		counted   bool
		hints     HintProvider
	}
)

var (
	ErrDuplicateCommand = errors.New("duplicate command registration")
	ErrNoModes          = errors.New("command has no modes")
	ErrUnknownMode      = errors.New("keys references mode not in modes")

	ErrBindingExists = i18n.NewError(i18n.ErrorBindingExists)
)

// NewKeymaps creates an empty Keymaps
func NewKeymaps() *Keymaps {
	return &Keymaps{
		modes:  map[view.Mode]*keyTrieNode{},
		byName: map[string]int{},
	}
}

// Register adds a command entry and wires its key bindings. Returns
// ErrDuplicateCommand if name is already registered - each command must be
// fully declared once, in the module that owns it
func (k *Keymaps) Register(name string, cmd Command) error {
	if _, ok := k.byName[name]; ok {
		return fmt.Errorf("%w: %s", ErrDuplicateCommand, name)
	}
	if cmd.Name == "" {
		cmd.Name = name
	}
	if cmd.Modes == 0 {
		return fmt.Errorf("%w: %s", ErrNoModes, name)
	}
	idx := len(k.commands)
	k.commands = append(k.commands, cmd)
	k.byName[name] = idx
	for mode := range cmd.Keys {
		if mode != view.ModeAny && cmd.Modes&mode == 0 {
			return fmt.Errorf("%w: %s in %s", ErrUnknownMode, mode, name)
		}
	}
	label := cmd.DocString
	action := func(e *view.Editor) Result {
		return k.commands[idx].run(e)
	}
	for _, mode := range cmd.Modes.Split() {
		bindings, ok := cmd.Keys[mode]
		if !ok {
			bindings = cmd.Keys[view.ModeAny]
		}
		k.bindCommand(bindCommandArgs{
			mode:    mode,
			name:    name,
			action:  action,
			label:   label,
			counted: cmd.Counted,
			hints:   cmd.Hints,
			seqs:    bindings,
		})
	}
	for _, alias := range cmd.Aliases {
		k.byName[alias] = idx
	}
	return nil
}

// ResolveCommand looks up a command by typeable alias
func (k *Keymaps) ResolveCommand(name string) *Command {
	if idx, ok := k.byName[name]; ok {
		return &k.commands[idx]
	}
	return nil
}

// ResolveCommandIn looks up a command by alias and filters it by mode
func (k *Keymaps) ResolveCommandIn(mode view.Mode, name string) *Command {
	if cmd := k.ResolveCommand(name); cmd != nil && cmd.availableIn(mode) {
		return cmd
	}
	return nil
}

// CommandsIn returns registered commands available in the named mode
func (k *Keymaps) CommandsIn(mode view.Mode) []*Command {
	out := make([]*Command, 0, len(k.commands))
	for i := range k.commands {
		if k.commands[i].availableIn(mode) {
			out = append(out, &k.commands[i])
		}
	}
	return out
}

// Bindings returns key sequences bound to a command in a mode
func (k *Keymaps) Bindings(mode view.Mode, name string) KeyBinding {
	root, ok := k.modes[mode]
	if !ok {
		return nil
	}
	var bindings KeyBinding
	root.collectBindings(name, nil, &bindings)
	return bindings
}

// Bind adds extra key sequences to an already-registered command
func (k *Keymaps) Bind(mode view.Mode, name string, seqs ...[]KeyEvent) {
	cmd := k.ResolveCommand(name)
	if cmd == nil {
		return
	}
	action := func(e *view.Editor) Result {
		return cmd.run(e)
	}
	k.bindCommand(bindCommandArgs{
		mode:   mode,
		name:   cmd.Name,
		action: action,
		seqs:   seqs,
	})
}

// BindActionArgs bundles the inputs for BindResultAction
type BindActionArgs struct {
	Modes  []view.Mode
	Action KeyResultAction
	When   func(*view.Editor) bool
	Label  string
	Seqs   [][]KeyEvent
}

// BindResultAction adds key sequences for a result-returning action
func (k *Keymaps) BindResultAction(args BindActionArgs) error {
	for _, mode := range args.Modes {
		for _, seq := range args.Seqs {
			if k.hasBindingConflict(mode, seq) {
				return fmt.Errorf("%w in mode %s", ErrBindingExists, mode)
			}
		}
	}
	for _, mode := range args.Modes {
		k.bindCommand(bindCommandArgs{
			mode:      mode,
			action:    args.Action,
			available: args.When,
			label:     args.Label,
			seqs:      args.Seqs,
		})
	}
	return nil
}

// Enabled reports whether a matched binding's :when predicate allows it
func (k KeyMatch) Enabled(e *view.Editor) bool {
	return k.When == nil || k.When(e)
}

// Lookup traverses the key trie. The bool reports a complete match
func (k *Keymaps) Lookup(mode view.Mode, seq []KeyEvent) (KeyMatch, bool) {
	node := k.lookup(mode, seq)
	if node == nil {
		return KeyMatch{}, false
	}
	if node.isPrefix() {
		return KeyMatch{Prefix: true}, false
	}
	return KeyMatch{
		Action: node.action,
		When:   node.available,
		Name:   node.name,
	}, true
}

// AcceptsCount reports whether seq reaches a node with a counted command
func (k *Keymaps) AcceptsCount(mode view.Mode, seq []KeyEvent) bool {
	node := k.lookup(mode, seq)
	return node != nil && node.countable()
}

func (k *Keymaps) hasBindingConflict(mode view.Mode, seq []KeyEvent) bool {
	node, ok := k.modes[mode]
	if !ok {
		return false
	}
	for _, ev := range seq {
		if node.action != nil {
			return true
		}
		node, ok = node.children[ev]
		if !ok {
			return false
		}
	}
	return node.action != nil || len(node.children) > 0
}

func (k *Keymaps) declare(mode view.Mode, seq []KeyEvent) *keyTrieNode {
	root, ok := k.modes[mode]
	if !ok {
		root = &keyTrieNode{children: map[KeyEvent]*keyTrieNode{}}
		k.modes[mode] = root
	}
	node := root
	for _, ev := range seq {
		child, ok := node.children[ev]
		if !ok {
			child = &keyTrieNode{children: map[KeyEvent]*keyTrieNode{}}
			node.set(ev, child)
		}
		node = child
	}
	return node
}

func (k *Keymaps) lookup(mode view.Mode, seq []KeyEvent) *keyTrieNode {
	root, ok := k.modes[mode]
	if !ok {
		return nil
	}
	node := root
	for _, ev := range seq {
		child, ok := node.children[ev]
		if !ok {
			return nil
		}
		node = child
	}
	return node
}

type bindCommandArgs struct {
	mode      view.Mode
	name      string
	action    KeyResultAction
	available func(*view.Editor) bool
	label     string
	counted   bool
	hints     HintProvider
	seqs      [][]KeyEvent
}

func (k *Keymaps) bindCommand(args bindCommandArgs) {
	for _, seq := range args.seqs {
		node := k.declare(args.mode, seq)
		node.action = args.action
		node.available = args.available
		node.name = args.name
		node.counted = args.counted
		node.hints = args.hints
		if args.label != "" {
			node.label = args.label
		}
	}
}

func (k *keyTrieNode) set(ev KeyEvent, child *keyTrieNode) {
	if _, exists := k.children[ev]; !exists {
		k.order = append(k.order, ev)
	}
	k.children[ev] = child
}

func (k *keyTrieNode) isPrefix() bool {
	return k.action == nil && len(k.children) > 0
}

func (k *keyTrieNode) countable() bool {
	if k.counted {
		return true
	}
	for _, child := range k.children {
		if child.countable() {
			return true
		}
	}
	return false
}

func (k *keyTrieNode) collectBindings(
	name string, seq []KeyEvent, bindings *KeyBinding,
) {
	if k.action != nil && k.name == name {
		*bindings = append(*bindings, slices.Clone(seq))
	}
	for _, ev := range k.order {
		child := k.children[ev]
		child.collectBindings(name, append(seq, ev), bindings)
	}
}
