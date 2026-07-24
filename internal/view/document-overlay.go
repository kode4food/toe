package view

import (
	"slices"

	"github.com/kode4food/toe/internal/core"
)

// SetDocumentColors stores document-wide LSP colors
func (d *Document) SetDocumentColors(colors []DocumentColor) {
	if setOverlaySlice(&d.overlays, &d.overlays.colors, colors) {
		d.MarkDirty()
	}
}

// ClearDocumentColors removes document-wide LSP colors
func (d *Document) ClearDocumentColors() {
	if clearOverlaySlice(&d.overlays, &d.overlays.colors) {
		d.MarkDirty()
	}
}

// DocumentColors returns document-wide LSP colors
func (d *Document) DocumentColors() []DocumentColor {
	return getOverlaySlice(&d.overlays, &d.overlays.colors)
}

// SetDocumentLinks stores document-wide LSP links
func (d *Document) SetDocumentLinks(links []DocumentLink) {
	if setOverlaySlice(&d.overlays, &d.overlays.links, links) {
		d.MarkDirty()
	}
}

// ClearDocumentLinks removes document-wide LSP links
func (d *Document) ClearDocumentLinks() {
	if clearOverlaySlice(&d.overlays, &d.overlays.links) {
		d.MarkDirty()
	}
}

// DocumentLinks returns document-wide LSP links
func (d *Document) DocumentLinks() []DocumentLink {
	return getOverlaySlice(&d.overlays, &d.overlays.links)
}

// SetDocumentHighlights stores the same-document highlight ranges for a view
func (d *Document) SetDocumentHighlights(
	vid Id, highlights []DocumentHighlight,
) {
	if setOverlayMap(
		&d.overlays, d.overlays.highlights, vid, highlights,
	) {
		d.markViewDirty(vid)
	}
}

// ClearDocumentHighlights removes highlight ranges for a view
func (d *Document) ClearDocumentHighlights(vid Id) {
	if clearOverlayMap(&d.overlays, d.overlays.highlights, vid) {
		d.markViewDirty(vid)
	}
}

// ClearAllDocumentHighlights removes highlight ranges for every view
func (d *Document) ClearAllDocumentHighlights() {
	if clearAllOverlayMap(&d.overlays, d.overlays.highlights) {
		d.MarkDirty()
	}
}

// DocumentHighlights returns same-document highlight ranges for a view
func (d *Document) DocumentHighlights(vid Id) []DocumentHighlight {
	return getOverlayMap(&d.overlays, d.overlays.highlights, vid)
}

// SetInlayHints stores language-server inlay hints for a view
func (d *Document) SetInlayHints(vid Id, hints []InlayHint) {
	if setOverlayMap(&d.overlays, d.overlays.hints, vid, hints) {
		d.markViewDirty(vid)
	}
}

// ClearInlayHints removes language-server inlay hints for a view
func (d *Document) ClearInlayHints(vid Id) {
	if clearOverlayMap(&d.overlays, d.overlays.hints, vid) {
		d.markViewDirty(vid)
	}
}

// ClearAllInlayHints removes language-server inlay hints for every view
func (d *Document) ClearAllInlayHints() {
	if clearAllOverlayMap(&d.overlays, d.overlays.hints) {
		d.MarkDirty()
	}
}

// InlayHints returns language-server inlay hints for a view
func (d *Document) InlayHints(vid Id) []InlayHint {
	return getOverlayMap(&d.overlays, d.overlays.hints, vid)
}

func (d *Document) remapOverlays(cs core.ChangeSet) {
	d.overlays.Lock()
	defer d.overlays.Unlock()
	for i := range d.overlays.diagnostics {
		from, to := remapRange(
			cs, d.overlays.diagnostics[i].Range.From,
			d.overlays.diagnostics[i].Range.To,
		)
		d.overlays.diagnostics[i].Range.From = from
		d.overlays.diagnostics[i].Range.To = to
	}
	for i := range d.overlays.links {
		d.overlays.links[i].From, d.overlays.links[i].To = remapRange(
			cs, d.overlays.links[i].From, d.overlays.links[i].To,
		)
	}
	for i := range d.overlays.colors {
		d.overlays.colors[i].From, d.overlays.colors[i].To = remapRange(
			cs, d.overlays.colors[i].From, d.overlays.colors[i].To,
		)
	}
	for _, hl := range d.overlays.highlights {
		for i := range hl {
			hl[i].From, hl[i].To = remapRange(cs, hl[i].From, hl[i].To)
		}
	}
	for _, hints := range d.overlays.hints {
		for i := range hints {
			hints[i].Pos = remapPos(cs, hints[i].Pos)
		}
	}
}

// setOverlaySlice reports whether field's content actually changed
func setOverlaySlice[T comparable](
	state *overlayState, field *[]T, items []T,
) bool {
	state.Lock()
	defer state.Unlock()
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
func clearOverlaySlice[T any](state *overlayState, field *[]T) bool {
	state.Lock()
	defer state.Unlock()
	changed := len(*field) != 0
	*field = nil
	return changed
}

func getOverlaySlice[T any](state *overlayState, field *[]T) []T {
	state.RLock()
	defer state.RUnlock()
	return slices.Clone(*field)
}

// setOverlayMap reports whether vid's entry actually changed
func setOverlayMap[T comparable](
	state *overlayState, m map[Id][]T, vid Id, items []T,
) bool {
	state.Lock()
	defer state.Unlock()
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
func clearOverlayMap[T any](state *overlayState, m map[Id][]T, vid Id) bool {
	state.Lock()
	defer state.Unlock()
	_, ok := m[vid]
	delete(m, vid)
	return ok
}

// clearAllOverlayMap reports whether m held any entries to clear
func clearAllOverlayMap[T any](state *overlayState, m map[Id][]T) bool {
	state.Lock()
	defer state.Unlock()
	changed := len(m) != 0
	clear(m)
	return changed
}

func getOverlayMap[T any](state *overlayState, m map[Id][]T, vid Id) []T {
	state.RLock()
	defer state.RUnlock()
	return slices.Clone(m[vid])
}

func remapRange(cs core.ChangeSet, from, to int) (int, int) {
	r, err := cs.MapRange(core.NewRange(from, to))
	if err != nil {
		return from, to
	}
	return r.Anchor, r.Head
}

func remapPos(cs core.ChangeSet, pos int) int {
	p, err := cs.MapPos(pos, core.AssocAfterSticky)
	if err != nil {
		return pos
	}
	return p
}
