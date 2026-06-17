package command

import (
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

var allModes = []string{"NOR", "SEL", "INS"}

// NewKeymaps creates an empty Keymaps
func NewKeymaps() *Keymaps {
	return &Keymaps{
		modes:   map[string]*keyTrieNode{},
		byName:  map[string]int{},
		byAlias: map[string]int{},
	}
}

// Register adds or updates a command entry and wires its key bindings
func (k *Keymaps) Register(name string, cmd Command) {
	idx, ok := k.byName[name]
	if !ok {
		idx = len(k.commands)
		k.commands = append(k.commands, cmd)
		k.byName[name] = idx
	} else {
		k.commands[idx] = mergeCommand(k.commands[idx], cmd)
	}
	if k.commands[idx].Run != nil {
		stored := k.commands[idx]
		label := stored.DocString
		if label == "" && len(stored.Aliases) > 0 {
			label = stored.Aliases[0]
		}
		action := func(e *view.Editor) Continuation {
			return k.commands[idx].Run(e, nil).Continuation
		}
		modes := cmd.Modes
		if len(modes) == 0 {
			modes = allModes
		}
		for _, mode := range modes {
			for _, binding := range cmd.Keys {
				k.bindActionWithLabel(
					mode, action, label, binding...,
				)
			}
		}
	}
	for _, alias := range k.commands[idx].Aliases {
		k.byAlias[alias] = idx
	}
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

func mergeCommand(a, b Command) Command {
	if b.Run != nil {
		a.Run = b.Run
	}
	if len(b.Aliases) > 0 {
		a.Aliases = b.Aliases
	}
	if commandSignatureSet(b.Signature) {
		a.Signature = b.Signature
	}
	return a
}

func commandSignatureSet(sig Signature) bool {
	return sig.Positionals.Min != 0 ||
		sig.Positionals.Max != 0 ||
		sig.RawAfter != 0 ||
		len(sig.Flags) > 0 ||
		len(sig.Completer.Positionals) > 0 ||
		sig.Completer.Raw != nil
}
