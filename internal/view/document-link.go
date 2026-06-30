package view

// SetDocumentLinks stores document-wide LSP links
func (d *Document) SetDocumentLinks(links []DocumentLink) {
	if len(links) == 0 {
		d.documentLinks = nil
		return
	}
	d.documentLinks = make([]DocumentLink, len(links))
	copy(d.documentLinks, links)
}

// ClearDocumentLinks removes document-wide LSP links
func (d *Document) ClearDocumentLinks() {
	d.documentLinks = nil
}

// DocumentLinks returns document-wide LSP links
func (d *Document) DocumentLinks() []DocumentLink {
	out := make([]DocumentLink, len(d.documentLinks))
	copy(out, d.documentLinks)
	return out
}
