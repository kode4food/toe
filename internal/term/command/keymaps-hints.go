package command

// LabelNode names the node reached by each alternative in prefix, so a shared
// menu (e.g. the Space and Ctrl-\ leaders) is labelled everywhere it is reached
func (k *Keymaps) LabelNode(mode string, prefix KeyBinding, name string) {
	root, ok := k.modes[mode]
	if !ok {
		return
	}
	for _, seq := range prefix {
		node := root
		for _, ev := range seq {
			if node = node.children[ev]; node == nil {
				break
			}
		}
		if node != nil {
			node.label = name
		}
	}
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
