package view

// SetDocumentHighlights stores the same-document highlight ranges for a view
func (d *Document) SetDocumentHighlights(
	vid Id, highlights []DocumentHighlight,
) {
	setOverlayMap(&d.ls, d.ls.highlights, vid, highlights)
}

// ClearDocumentHighlights removes highlight ranges for a view
func (d *Document) ClearDocumentHighlights(vid Id) {
	clearOverlayMap(&d.ls, d.ls.highlights, vid)
}

// ClearAllDocumentHighlights removes highlight ranges for every view
func (d *Document) ClearAllDocumentHighlights() {
	clearAllOverlayMap(&d.ls, d.ls.highlights)
}

// DocumentHighlights returns same-document highlight ranges for a view
func (d *Document) DocumentHighlights(vid Id) []DocumentHighlight {
	return getOverlayMap(&d.ls, d.ls.highlights, vid)
}
