package command

import "github.com/kode4food/toe/internal/view"

// LabelNode names the node reached by each alternative in prefix, so a shared
// menu (e.g. the Space and Ctrl-\ leaders) is labelled everywhere it is reached
func (k *Keymaps) LabelNode(mode view.Mode, prefix KeyBinding, name string) {
	for _, seq := range prefix {
		k.declare(mode, seq).label = name
	}
}

// PendingHints returns the title and (key, label) pairs offered after seq in
// mode, from the node's children or its hint provider. A binding rejected by
// :when, or unusable with the count typed, is omitted
func (k *Keymaps) PendingHints(
	e *view.Editor, mode view.Mode, seq []KeyEvent, counting bool,
) (string, []KeyHint) {
	node := k.lookup(mode, seq)
	if node == nil {
		return "", nil
	}
	if node.hints != nil {
		return node.label, node.hints(e)
	}
	if len(node.children) == 0 {
		if node.action == nil {
			return "", nil
		}
		return node.label, nil
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
		if counting && !child.countable() {
			continue
		}
		if child.available != nil && !child.available(e) {
			continue
		}
		if idx, ok := seen[lbl]; ok {
			hints[idx].Key += ", " + ev.String()
		} else {
			seen[lbl] = len(hints)
			hints = append(hints, KeyHint{
				Key:    ev.String(),
				Label:  lbl,
				Prefix: child.isPrefix(),
			})
		}
	}
	return node.label, hints
}
