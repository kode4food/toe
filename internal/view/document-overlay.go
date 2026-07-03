package view

import "slices"

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

func setOverlaySlice[T any](ls *lsState, field *[]T, items []T) {
	ls.Lock()
	defer ls.Unlock()
	if len(items) == 0 {
		*field = nil
		return
	}
	*field = slices.Clone(items)
}

func clearOverlaySlice[T any](ls *lsState, field *[]T) {
	ls.Lock()
	defer ls.Unlock()
	*field = nil
}

func getOverlaySlice[T any](ls *lsState, field *[]T) []T {
	ls.RLock()
	defer ls.RUnlock()
	return slices.Clone(*field)
}

func setOverlayMap[T any](ls *lsState, m map[Id][]T, vid Id, items []T) {
	ls.Lock()
	defer ls.Unlock()
	if len(items) == 0 {
		delete(m, vid)
		return
	}
	m[vid] = slices.Clone(items)
}

func clearOverlayMap[T any](ls *lsState, m map[Id][]T, vid Id) {
	ls.Lock()
	defer ls.Unlock()
	delete(m, vid)
}

func clearAllOverlayMap[T any](ls *lsState, m map[Id][]T) {
	ls.Lock()
	defer ls.Unlock()
	clear(m)
}

func getOverlayMap[T any](ls *lsState, m map[Id][]T, vid Id) []T {
	ls.RLock()
	defer ls.RUnlock()
	return slices.Clone(m[vid])
}
