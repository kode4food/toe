package view

import "slices"

// SetInlayHints stores language-server inlay hints for a view
func (d *Document) SetInlayHints(vid Id, hints []InlayHint) {
	d.ls.Lock()
	defer d.ls.Unlock()
	if len(hints) == 0 {
		delete(d.ls.hints, vid)
		return
	}
	d.ls.hints[vid] = slices.Clone(hints)
}

// ClearInlayHints removes language-server inlay hints for a view
func (d *Document) ClearInlayHints(vid Id) {
	d.ls.Lock()
	defer d.ls.Unlock()
	delete(d.ls.hints, vid)
}

// ClearAllInlayHints removes language-server inlay hints for every view
func (d *Document) ClearAllInlayHints() {
	d.ls.Lock()
	defer d.ls.Unlock()
	clear(d.ls.hints)
}

// InlayHints returns language-server inlay hints for a view
func (d *Document) InlayHints(vid Id) []InlayHint {
	d.ls.RLock()
	defer d.ls.RUnlock()
	return slices.Clone(d.ls.hints[vid])
}
