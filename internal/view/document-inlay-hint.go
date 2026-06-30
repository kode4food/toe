package view

// SetInlayHints stores language-server inlay hints for a view
func (d *Document) SetInlayHints(vid Id, hints []InlayHint) {
	if len(hints) == 0 {
		delete(d.inlayHints, vid)
		return
	}
	if d.inlayHints == nil {
		d.inlayHints = map[Id][]InlayHint{}
	}
	out := make([]InlayHint, len(hints))
	copy(out, hints)
	d.inlayHints[vid] = out
}

// ClearInlayHints removes language-server inlay hints for a view
func (d *Document) ClearInlayHints(vid Id) {
	delete(d.inlayHints, vid)
}

// ClearAllInlayHints removes language-server inlay hints for every view
func (d *Document) ClearAllInlayHints() {
	clear(d.inlayHints)
}

// InlayHints returns language-server inlay hints for a view
func (d *Document) InlayHints(vid Id) []InlayHint {
	hints := d.inlayHints[vid]
	out := make([]InlayHint, len(hints))
	copy(out, hints)
	return out
}
