package view

// SetDocumentColors stores document-wide LSP colors
func (d *Document) SetDocumentColors(colors []DocumentColor) {
	setOverlaySlice(&d.ls, &d.ls.colors, colors)
}

// ClearDocumentColors removes document-wide LSP colors
func (d *Document) ClearDocumentColors() {
	clearOverlaySlice(&d.ls, &d.ls.colors)
}

// DocumentColors returns document-wide LSP colors
func (d *Document) DocumentColors() []DocumentColor {
	return getOverlaySlice(&d.ls, &d.ls.colors)
}
