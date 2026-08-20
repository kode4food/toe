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
		r := &d.overlays.diagnostics[i].Range
		s := remapSpan(cs, core.Span{From: r.From, To: r.To})
		r.From = s.From
		r.To = s.To
	}
	for i := range d.overlays.links {
		l := &d.overlays.links[i]
		s := remapSpan(cs, core.Span{From: l.From, To: l.To})
		l.From = s.From
		l.To = s.To
	}
	for i := range d.overlays.colors {
		c := &d.overlays.colors[i]
		s := remapSpan(cs, core.Span{From: c.From, To: c.To})
		c.From = s.From
		c.To = s.To
	}
	for _, hl := range d.overlays.highlights {
		for i := range hl {
			s := remapSpan(cs, core.Span{From: hl[i].From, To: hl[i].To})
			hl[i].From = s.From
			hl[i].To = s.To
		}
	}
	for _, hints := range d.overlays.hints {
		for i := range hints {
			hints[i].Pos = remapPos(cs, hints[i].Pos)
		}
	}
}

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

func clearOverlayMap[T any](state *overlayState, m map[Id][]T, vid Id) bool {
	state.Lock()
	defer state.Unlock()
	_, ok := m[vid]
	delete(m, vid)
	return ok
}

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

func remapSpan(cs core.ChangeSet, s core.Span) core.Span {
	if r, err := cs.MapRange(core.Range{
		Anchor: s.From,
		Head:   s.To,
	}); err == nil {
		return core.Span{From: r.Anchor, To: r.Head}
	}
	return s
}

func remapPos(cs core.ChangeSet, pos int) int {
	if p, err := cs.MapPos(pos, core.AssocAfterSticky); err == nil {
		return p
	}
	return pos
}
