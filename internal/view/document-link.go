package view

import "slices"

// SetDocumentLinks stores document-wide LSP links
func (d *Document) SetDocumentLinks(links []DocumentLink) {
	d.ls.Lock()
	defer d.ls.Unlock()
	if len(links) == 0 {
		d.ls.links = nil
		return
	}
	d.ls.links = slices.Clone(links)
}

// ClearDocumentLinks removes document-wide LSP links
func (d *Document) ClearDocumentLinks() {
	d.ls.Lock()
	defer d.ls.Unlock()
	d.ls.links = nil
}

// DocumentLinks returns document-wide LSP links
func (d *Document) DocumentLinks() []DocumentLink {
	d.ls.RLock()
	defer d.ls.RUnlock()
	return slices.Clone(d.ls.links)
}
