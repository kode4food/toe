package command

import "github.com/kode4food/toe/internal/view"

// LabelNode names the node reached by each alternative in prefix, so a shared
// menu (e.g. the Space and Ctrl-\ leaders) is labelled everywhere it is reached
func (k *Keymaps) LabelNode(mode view.Mode, prefix KeyBinding, name string) {
	for _, seq := range prefix {
		if node := k.lookup(mode, seq); node != nil {
			node.label = name
		}
	}
}

// PendingHints returns the title and (key, label) pairs for the node
// reached by seq in mode, used to populate the pending-key info popup
func (k *Keymaps) PendingHints(
	mode view.Mode, seq []KeyEvent,
) (string, []KeyHint) {
	node := k.lookup(mode, seq)
	if node == nil || len(node.children) == 0 {
		return "", nil
	}
	// Iterate in insertion order so hints remain stable across renders
	hints := make([]KeyHint, 0, len(node.order))
	seen := map[string]int{}
	for _, ev := range node.order {
		child := node.children[ev]
		lbl := child.label
		if lbl == "" {
			continue
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
