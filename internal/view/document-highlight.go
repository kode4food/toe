package view

import "slices"

// SetDocumentHighlights stores the same-document highlight ranges for a view
func (d *Document) SetDocumentHighlights(
	vid Id, highlights []DocumentHighlight,
) {
	d.ls.Lock()
	defer d.ls.Unlock()
	if len(highlights) == 0 {
		delete(d.ls.highlights, vid)
		return
	}
	d.ls.highlights[vid] = slices.Clone(highlights)
}

// ClearDocumentHighlights removes highlight ranges for a view
func (d *Document) ClearDocumentHighlights(vid Id) {
	d.ls.Lock()
	defer d.ls.Unlock()
	delete(d.ls.highlights, vid)
}

// ClearAllDocumentHighlights removes highlight ranges for every view
func (d *Document) ClearAllDocumentHighlights() {
	d.ls.Lock()
	defer d.ls.Unlock()
	clear(d.ls.highlights)
}

// DocumentHighlights returns same-document highlight ranges for a view
func (d *Document) DocumentHighlights(vid Id) []DocumentHighlight {
	d.ls.RLock()
	defer d.ls.RUnlock()
	return slices.Clone(d.ls.highlights[vid])
}
