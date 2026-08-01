package command

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/toe/internal/view"
)

type (
	// Keymaps is the combined command registry and key-event dispatch trie
	Keymaps struct {
		modes    map[view.Mode]*keyTrieNode
		commands []Command
		byName   map[string]int
		byAlias  map[string]int
	}

	keyTrieNode struct {
		children map[KeyEvent]*keyTrieNode
		order    []KeyEvent // insertion order for info popup display
		action   KeyAction
		name     string
		label    string
	}
)

var (
	ErrDuplicateCommand = errors.New("duplicate command registration")
	ErrNoModes          = errors.New("command has no modes")
	ErrUnknownMode      = errors.New("keys references mode not in modes")
)

// NewKeymaps creates an empty Keymaps
func NewKeymaps() *Keymaps {
	return &Keymaps{
		modes:   map[view.Mode]*keyTrieNode{},
		byName:  map[string]int{},
		byAlias: map[string]int{},
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
	if cmd.Run != nil {
		label := cmd.DocString
		if label == "" && len(cmd.Aliases) > 0 {
			label = cmd.Aliases[0]
		}
		action := func(e *view.Editor) Continuation {
			return k.commands[idx].Run(e, nil).Continuation
		}
		for _, mode := range cmd.Modes.Split() {
			bindings, ok := cmd.Keys[mode]
			if !ok {
				bindings = cmd.Keys[view.ModeAny]
			}
			for _, binding := range bindings {
				k.bindCommand(bindCommandArgs{
					mode:   mode,
					name:   name,
					action: action,
					label:  label,
					seqs:   binding,
				})
			}
		}
	}
	for _, alias := range cmd.Aliases {
		k.byAlias[alias] = idx
	}
	return nil
}

// ResolveCommand looks up a command by typeable alias
func (k *Keymaps) ResolveCommand(name string) *Command {
	if idx, ok := k.byAlias[name]; ok {
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

// Commands returns all registered commands in registration order
func (k *Keymaps) Commands() []*Command {
	out := make([]*Command, 0, len(k.commands))
	for i := range k.commands {
		out = append(out, &k.commands[i])
	}
	return out
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
func (k *Keymaps) Bindings(mode view.Mode, name string) []KeyBinding {
	root, ok := k.modes[mode]
	if !ok {
		return nil
	}
	var bindings []KeyBinding
	root.collectBindings(name, nil, &bindings)
	return bindings
}

// Bind adds extra key sequences to an already-registered command
func (k *Keymaps) Bind(mode view.Mode, name string, seqs ...[]KeyEvent) {
	cmd, ok := k.command(name)
	if !ok || cmd.Run == nil {
		return
	}
	action := func(e *view.Editor) Continuation {
		return cmd.Run(e, nil).Continuation
	}
	k.bindCommand(bindCommandArgs{
		mode:   mode,
		name:   name,
		action: action,
		seqs:   seqs,
	})
}

// Lookup traverses the key trie. Returns (action, true, false) on a complete
// match, (nil, false, true) on a valid prefix, (nil, false, false) otherwise
func (k *Keymaps) Lookup(
	mode view.Mode, seq []KeyEvent,
) (action KeyAction, found, prefix bool) {
	node, found, prefix := k.lookup(mode, seq)
	if !found {
		return nil, false, prefix
	}
	return node.action, true, false
}

// LookupCommand traverses the key trie and returns the registered command name
func (k *Keymaps) LookupCommand(
	mode view.Mode, seq []KeyEvent,
) (name string, found, prefix bool) {
	node, found, prefix := k.lookup(mode, seq)
	if !found {
		return "", false, prefix
	}
	return node.name, true, false
}

func (k *Keymaps) lookup(
	mode view.Mode, seq []KeyEvent,
) (*keyTrieNode, bool, bool) {
	root, ok := k.modes[mode]
	if !ok {
		return nil, false, false
	}
	node := root
	for _, ev := range seq {
		child, ok := node.children[ev]
		if !ok {
			return nil, false, false
		}
		node = child
	}
	if node.action != nil {
		return node, true, false
	}
	if len(node.children) > 0 {
		return nil, false, true
	}
	return nil, false, false
}

func (k *Keymaps) command(name string) (Command, bool) {
	if idx, ok := k.byName[name]; ok {
		return k.commands[idx], true
	}
	return Command{}, false
}

func (c Command) availableIn(mode view.Mode) bool {
	return c.Modes&mode != 0
}

type bindCommandArgs struct {
	mode   view.Mode
	name   string
	action KeyAction
	label  string
	seqs   [][]KeyEvent
}

func (k *Keymaps) bindCommand(args bindCommandArgs) {
	root, ok := k.modes[args.mode]
	if !ok {
		root = &keyTrieNode{children: map[KeyEvent]*keyTrieNode{}}
		k.modes[args.mode] = root
	}
	for _, seq := range args.seqs {
		node := root
		for _, ev := range seq {
			child, ok := node.children[ev]
			if !ok {
				child = &keyTrieNode{
					children: map[KeyEvent]*keyTrieNode{},
				}
				node.set(ev, child)
			}
			node = child
		}
		node.action = args.action
		node.name = args.name
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

func (k *keyTrieNode) collectBindings(
	name string, seq []KeyEvent, bindings *[]KeyBinding,
) {
	if k.action != nil && k.name == name {
		*bindings = append(*bindings, KeyBinding{slices.Clone(seq)})
	}
	for _, ev := range k.order {
		child := k.children[ev]
		child.collectBindings(name, append(seq, ev), bindings)
	}
}
