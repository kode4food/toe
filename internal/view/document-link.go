package view

// SetDocumentLinks stores document-wide LSP links
func (d *Document) SetDocumentLinks(links []DocumentLink) {
	setOverlaySlice(&d.ls, &d.ls.links, links)
}

// ClearDocumentLinks removes document-wide LSP links
func (d *Document) ClearDocumentLinks() {
	clearOverlaySlice(&d.ls, &d.ls.links)
}

// DocumentLinks returns document-wide LSP links
func (d *Document) DocumentLinks() []DocumentLink {
	return getOverlaySlice(&d.ls, &d.ls.links)
}
