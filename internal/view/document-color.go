package view

// SetDocumentColors stores document-wide LSP colors
func (d *Document) SetDocumentColors(colors []DocumentColor) {
	if len(colors) == 0 {
		d.documentColors = nil
		return
	}
	d.documentColors = append(d.documentColors[:0], colors...)
}

// ClearDocumentColors removes document-wide LSP colors
func (d *Document) ClearDocumentColors() {
	d.documentColors = nil
}

// DocumentColors returns document-wide LSP colors
func (d *Document) DocumentColors() []DocumentColor {
	out := make([]DocumentColor, len(d.documentColors))
	copy(out, d.documentColors)
	return out
}
