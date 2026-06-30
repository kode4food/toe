package view

// SetDocumentHighlights stores the same-document highlight ranges for a view
func (d *Document) SetDocumentHighlights(
	vid Id, highlights []DocumentHighlight,
) {
	if len(highlights) == 0 {
		delete(d.documentHighlights, vid)
		return
	}
	if d.documentHighlights == nil {
		d.documentHighlights = map[Id][]DocumentHighlight{}
	}
	out := make([]DocumentHighlight, len(highlights))
	copy(out, highlights)
	d.documentHighlights[vid] = out
}

// ClearDocumentHighlights removes highlight ranges for a view
func (d *Document) ClearDocumentHighlights(vid Id) {
	delete(d.documentHighlights, vid)
}

// ClearAllDocumentHighlights removes highlight ranges for every view
func (d *Document) ClearAllDocumentHighlights() {
	clear(d.documentHighlights)
}

// DocumentHighlights returns same-document highlight ranges for a view
func (d *Document) DocumentHighlights(vid Id) []DocumentHighlight {
	highlights := d.documentHighlights[vid]
	out := make([]DocumentHighlight, len(highlights))
	copy(out, highlights)
	return out
}
