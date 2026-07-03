package view

// SetInlayHints stores language-server inlay hints for a view
func (d *Document) SetInlayHints(vid Id, hints []InlayHint) {
	setOverlayMap(&d.ls, d.ls.hints, vid, hints)
}

// ClearInlayHints removes language-server inlay hints for a view
func (d *Document) ClearInlayHints(vid Id) {
	clearOverlayMap(&d.ls, d.ls.hints, vid)
}

// ClearAllInlayHints removes language-server inlay hints for every view
func (d *Document) ClearAllInlayHints() {
	clearAllOverlayMap(&d.ls, d.ls.hints)
}

// InlayHints returns language-server inlay hints for a view
func (d *Document) InlayHints(vid Id) []InlayHint {
	return getOverlayMap(&d.ls, d.ls.hints, vid)
}
