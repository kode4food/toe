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
		modes    map[string]*keyTrieNode
		commands []Command
		byName   map[string]int
		byAlias  map[string]int
	}

	keyTrieNode struct {
		children map[KeyEvent]*keyTrieNode
		order    []KeyEvent // insertion order for info popup display
		action   KeyAction
		label    string
	}
)

var (
	ErrDuplicateCommand = errors.New("duplicate command registration")
	ErrUnknownMode      = errors.New("keys references mode not in modes")
)

// NewKeymaps creates an empty Keymaps
func NewKeymaps() *Keymaps {
	return &Keymaps{
		modes:   map[string]*keyTrieNode{},
		byName:  map[string]int{},
		byAlias: map[string]int{},
	}
}

// Register adds a command entry and wires its key bindings. Returns
// ErrDuplicateCommand if name is already registered - eachcommand must be fully
// declared once, in the module that owns it
func (k *Keymaps) Register(name string, cmd Command) error {
	if _, ok := k.byName[name]; ok {
		return fmt.Errorf("%w: %s", ErrDuplicateCommand, name)
	}
	idx := len(k.commands)
	k.commands = append(k.commands, cmd)
	k.byName[name] = idx
	if len(cmd.Modes) > 0 {
		for mode := range cmd.Keys {
			if mode != "*" && !slices.Contains(cmd.Modes, mode) {
				return fmt.Errorf("%w: %s in %s", ErrUnknownMode, mode, name)
			}
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
		modes := cmd.Modes
		if len(modes) == 0 {
			modes = defaultModes()
		}
		for _, mode := range modes {
			bindings, ok := cmd.Keys[mode]
			if !ok {
				bindings = cmd.Keys["*"]
			}
			for _, binding := range bindings {
				k.bindActionWithLabel(mode, action, label, binding...)
			}
		}
	}
	for _, alias := range cmd.Aliases {
		k.byAlias[alias] = idx
	}
	return nil
}

// ResolveCommand looks up a command by typeable alias
func (k *Keymaps) ResolveCommand(name string) (Command, bool) {
	idx, ok := k.byAlias[name]
	if !ok {
		return Command{}, false
	}
	return k.commands[idx], true
}

// Commands returns all registered commands in registration order
func (k *Keymaps) Commands() []Command {
	return slices.Clone(k.commands)
}

// Bind adds extra key sequences to an already-registered command
func (k *Keymaps) Bind(mode string, name string, seqs ...[]KeyEvent) {
	cmd, ok := k.command(name)
	if !ok || cmd.Run == nil {
		return
	}
	action := func(e *view.Editor) Continuation {
		return cmd.Run(e, nil).Continuation
	}
	k.bindAction(mode, action, seqs...)
}

// Lookup traverses the key trie. Returns (action, true, false) on a complete
// match, (nil, false, true) on a valid prefix, (nil, false, false) otherwise
func (k *Keymaps) Lookup(
	mode string, seq []KeyEvent,
) (action KeyAction, found, prefix bool) {
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
		return node.action, true, false
	}
	if len(node.children) > 0 {
		return nil, false, true
	}
	return nil, false, false
}

func (k *Keymaps) command(name string) (Command, bool) {
	idx, ok := k.byName[name]
	if !ok {
		return Command{}, false
	}
	return k.commands[idx], true
}

// LabelNode sets the display name for a prefix node (e.g., "Goto" for `g`)
func (k *Keymaps) LabelNode(mode string, seq []KeyEvent, name string) {
	root, ok := k.modes[mode]
	if !ok {
		return
	}
	node := root
	for _, ev := range seq {
		child, ok := node.children[ev]
		if !ok {
			return
		}
		node = child
	}
	node.label = name
}

// PendingHints returns the title and (key, label) pairs for the node
// reached by seq in mode, used to populate the pending-key info popup
func (k *Keymaps) PendingHints(
	mode string, seq []KeyEvent,
) (string, []KeyHint) {
	root, ok := k.modes[mode]
	if !ok {
		return "", nil
	}
	node := root
	for _, ev := range seq {
		child, ok := node.children[ev]
		if !ok {
			return "", nil
		}
		node = child
	}
	if len(node.children) == 0 {
		return "", nil
	}
	// Iterate in insertion order so hints remain stable across renders
	hints := make([]KeyHint, 0, len(node.order))
	seen := map[string]int{}
	for _, ev := range node.order {
		child := node.children[ev]
		lbl := child.label
		if lbl == "" {
			lbl = ev.String()
		}
		if idx, ok := seen[lbl]; ok {
			hints[idx].Key += ", " + ev.String()
		} else {
			seen[lbl] = len(hints)
			hints = append(hints, KeyHint{Key: ev.String(), Label: lbl})
		}
	}
	return node.label, hints
}

func (k *Keymaps) bindAction(
	mode string, action KeyAction, seqs ...[]KeyEvent,
) {
	k.bindActionWithLabel(mode, action, "", seqs...)
}

func (k *Keymaps) bindActionWithLabel(
	mode string, action KeyAction, label string, seqs ...[]KeyEvent,
) {
	root, ok := k.modes[mode]
	if !ok {
		root = &keyTrieNode{children: map[KeyEvent]*keyTrieNode{}}
		k.modes[mode] = root
	}
	for _, seq := range seqs {
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
		node.action = action
		if label != "" {
			node.label = label
		}
	}
}

func (k *keyTrieNode) set(ev KeyEvent, child *keyTrieNode) {
	if _, exists := k.children[ev]; !exists {
		k.order = append(k.order, ev)
	}
	k.children[ev] = child
}

func defaultModes() []string {
	return []string{"NOR", "SEL", "INS"}
}
