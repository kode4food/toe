package view

import (
	"slices"

	"github.com/kode4food/toe/internal/core"
)

// SetDocumentColors stores document-wide LSP colors
func (d *Document) SetDocumentColors(colors []DocumentColor) {
	if d.overlays.setSlice(&d.overlays.colors, colors) {
		d.MarkDirty()
	}
}

// ClearDocumentColors removes document-wide LSP colors
func (d *Document) ClearDocumentColors() {
	if d.overlays.clearSlice(&d.overlays.colors) {
		d.MarkDirty()
	}
}

// DocumentColors returns document-wide LSP colors
func (d *Document) DocumentColors() []DocumentColor {
	return d.overlays.getSlice(&d.overlays.colors)
}

// SetDocumentLinks stores document-wide LSP links
func (d *Document) SetDocumentLinks(links []DocumentLink) {
	if d.overlays.setSlice(&d.overlays.links, links) {
		d.MarkDirty()
	}
}

// ClearDocumentLinks removes document-wide LSP links
func (d *Document) ClearDocumentLinks() {
	if d.overlays.clearSlice(&d.overlays.links) {
		d.MarkDirty()
	}
}

// DocumentLinks returns document-wide LSP links
func (d *Document) DocumentLinks() []DocumentLink {
	return d.overlays.getSlice(&d.overlays.links)
}

// SetDocumentHighlights stores the same-document highlight ranges for a view
func (d *Document) SetDocumentHighlights(
	vid Id, highlights []DocumentHighlight,
) {
	if d.overlays.setMap(d.overlays.highlights, vid, highlights) {
		d.markViewDirty(vid)
	}
}

// ClearDocumentHighlights removes highlight ranges for a view
func (d *Document) ClearDocumentHighlights(vid Id) {
	if d.overlays.clearMap(d.overlays.highlights, vid) {
		d.markViewDirty(vid)
	}
}

// ClearAllDocumentHighlights removes highlight ranges for every view
func (d *Document) ClearAllDocumentHighlights() {
	if d.overlays.clearAllMap(d.overlays.highlights) {
		d.MarkDirty()
	}
}

// DocumentHighlights returns same-document highlight ranges for a view
func (d *Document) DocumentHighlights(vid Id) []DocumentHighlight {
	return d.overlays.getMap(d.overlays.highlights, vid)
}

// SetInlayHints stores language-server inlay hints for a view
func (d *Document) SetInlayHints(vid Id, hints []InlayHint) {
	if d.overlays.setMap(d.overlays.hints, vid, hints) {
		d.markViewDirty(vid)
	}
}

// ClearInlayHints removes language-server inlay hints for a view
func (d *Document) ClearInlayHints(vid Id) {
	if d.overlays.clearMap(d.overlays.hints, vid) {
		d.markViewDirty(vid)
	}
}

// ClearAllInlayHints removes language-server inlay hints for every view
func (d *Document) ClearAllInlayHints() {
	if d.overlays.clearAllMap(d.overlays.hints) {
		d.MarkDirty()
	}
}

// InlayHints returns language-server inlay hints for a view
func (d *Document) InlayHints(vid Id) []InlayHint {
	return d.overlays.getMap(d.overlays.hints, vid)
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

func (s *overlayState) setSlice[T comparable](field *[]T, items []T) bool {
	s.Lock()
	defer s.Unlock()
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

func (s *overlayState) clearSlice[T any](field *[]T) bool {
	s.Lock()
	defer s.Unlock()
	changed := len(*field) != 0
	*field = nil
	return changed
}

func (s *overlayState) getSlice[T any](field *[]T) []T {
	s.RLock()
	defer s.RUnlock()
	return slices.Clone(*field)
}

func (s *overlayState) setMap[T comparable](
	m map[Id][]T, vid Id, items []T,
) bool {
	s.Lock()
	defer s.Unlock()
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

func (s *overlayState) clearMap[T any](m map[Id][]T, vid Id) bool {
	s.Lock()
	defer s.Unlock()
	_, ok := m[vid]
	delete(m, vid)
	return ok
}

func (s *overlayState) clearAllMap[T any](m map[Id][]T) bool {
	s.Lock()
	defer s.Unlock()
	changed := len(m) != 0
	clear(m)
	return changed
}

func (s *overlayState) getMap[T any](m map[Id][]T, vid Id) []T {
	s.RLock()
	defer s.RUnlock()
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
