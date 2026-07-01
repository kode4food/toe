package view

import "slices"

// SetDocumentColors stores document-wide LSP colors
func (d *Document) SetDocumentColors(colors []DocumentColor) {
	d.ls.Lock()
	defer d.ls.Unlock()
	if len(colors) == 0 {
		d.ls.colors = nil
		return
	}
	d.ls.colors = slices.Clone(colors)
}

// ClearDocumentColors removes document-wide LSP colors
func (d *Document) ClearDocumentColors() {
	d.ls.Lock()
	defer d.ls.Unlock()
	d.ls.colors = nil
}

// DocumentColors returns document-wide LSP colors
func (d *Document) DocumentColors() []DocumentColor {
	d.ls.RLock()
	defer d.ls.RUnlock()
	return slices.Clone(d.ls.colors)
}
