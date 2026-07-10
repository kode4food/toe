package view

import "slices"

// SetDocumentColors stores document-wide LSP colors
func (d *Document) SetDocumentColors(colors []DocumentColor) {
	if setOverlaySlice(&d.ls, &d.ls.colors, colors) {
		d.markAllDirty()
	}
}

// ClearDocumentColors removes document-wide LSP colors
func (d *Document) ClearDocumentColors() {
	if clearOverlaySlice(&d.ls, &d.ls.colors) {
		d.markAllDirty()
	}
}

// DocumentColors returns document-wide LSP colors
func (d *Document) DocumentColors() []DocumentColor {
	return getOverlaySlice(&d.ls, &d.ls.colors)
}

// SetDocumentLinks stores document-wide LSP links
func (d *Document) SetDocumentLinks(links []DocumentLink) {
	if setOverlaySlice(&d.ls, &d.ls.links, links) {
		d.markAllDirty()
	}
}

// ClearDocumentLinks removes document-wide LSP links
func (d *Document) ClearDocumentLinks() {
	if clearOverlaySlice(&d.ls, &d.ls.links) {
		d.markAllDirty()
	}
}

// DocumentLinks returns document-wide LSP links
func (d *Document) DocumentLinks() []DocumentLink {
	return getOverlaySlice(&d.ls, &d.ls.links)
}

// SetDocumentHighlights stores the same-document highlight ranges for a view
func (d *Document) SetDocumentHighlights(
	vid Id, highlights []DocumentHighlight,
) {
	if setOverlayMap(&d.ls, d.ls.highlights, vid, highlights) {
		d.markDirty(vid)
	}
}

// ClearDocumentHighlights removes highlight ranges for a view
func (d *Document) ClearDocumentHighlights(vid Id) {
	if clearOverlayMap(&d.ls, d.ls.highlights, vid) {
		d.markDirty(vid)
	}
}

// ClearAllDocumentHighlights removes highlight ranges for every view
func (d *Document) ClearAllDocumentHighlights() {
	if clearAllOverlayMap(&d.ls, d.ls.highlights) {
		d.markAllDirty()
	}
}

// DocumentHighlights returns same-document highlight ranges for a view
func (d *Document) DocumentHighlights(vid Id) []DocumentHighlight {
	return getOverlayMap(&d.ls, d.ls.highlights, vid)
}

// SetInlayHints stores language-server inlay hints for a view
func (d *Document) SetInlayHints(vid Id, hints []InlayHint) {
	if setOverlayMap(&d.ls, d.ls.hints, vid, hints) {
		d.markDirty(vid)
	}
}

// ClearInlayHints removes language-server inlay hints for a view
func (d *Document) ClearInlayHints(vid Id) {
	if clearOverlayMap(&d.ls, d.ls.hints, vid) {
		d.markDirty(vid)
	}
}

// ClearAllInlayHints removes language-server inlay hints for every view
func (d *Document) ClearAllInlayHints() {
	if clearAllOverlayMap(&d.ls, d.ls.hints) {
		d.markAllDirty()
	}
}

// InlayHints returns language-server inlay hints for a view
func (d *Document) InlayHints(vid Id) []InlayHint {
	return getOverlayMap(&d.ls, d.ls.hints, vid)
}

// setOverlaySlice reports whether field's content actually changed
func setOverlaySlice[T comparable](ls *lsState, field *[]T, items []T) bool {
	ls.Lock()
	defer ls.Unlock()
	if len(items) == 0 {
		changed := len(*field) != 0
		*field = nil
		return changed
	}
	if slices.Equal(*field, items) {
		return false
	}
	*field = slices.Clone(items)
	return true
}

// clearOverlaySlice reports whether field held anything to clear
func clearOverlaySlice[T any](ls *lsState, field *[]T) bool {
	ls.Lock()
	defer ls.Unlock()
	changed := len(*field) != 0
	*field = nil
	return changed
}

func getOverlaySlice[T any](ls *lsState, field *[]T) []T {
	ls.RLock()
	defer ls.RUnlock()
	return slices.Clone(*field)
}

// setOverlayMap reports whether vid's entry actually changed
func setOverlayMap[T comparable](
	ls *lsState, m map[Id][]T, vid Id, items []T,
) bool {
	ls.Lock()
	defer ls.Unlock()
	if len(items) == 0 {
		_, ok := m[vid]
		delete(m, vid)
		return ok
	}
	if slices.Equal(m[vid], items) {
		return false
	}
	m[vid] = slices.Clone(items)
	return true
}

// clearOverlayMap reports whether vid had an entry to clear
func clearOverlayMap[T any](ls *lsState, m map[Id][]T, vid Id) bool {
	ls.Lock()
	defer ls.Unlock()
	_, ok := m[vid]
	delete(m, vid)
	return ok
}

// clearAllOverlayMap reports whether m held any entries to clear
func clearAllOverlayMap[T any](ls *lsState, m map[Id][]T) bool {
	ls.Lock()
	defer ls.Unlock()
	changed := len(m) != 0
	clear(m)
	return changed
}

func getOverlayMap[T any](ls *lsState, m map[Id][]T, vid Id) []T {
	ls.RLock()
	defer ls.RUnlock()
	return slices.Clone(m[vid])
}
