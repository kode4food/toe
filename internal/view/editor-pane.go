package view

import "fmt"

// VSplit opens docID in a new vertical split (side by side)
func (e *Editor) VSplit(docID DocumentId) (*View, bool) {
	doc, ok := e.documents.byID[docID]
	if !ok {
		return nil, false
	}
	if !e.panes.tree.CanSplit(LayoutVertical) {
		return nil, false
	}
	e.recordLeavingDoc()
	v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
	if src, ok := e.FocusedView(); ok {
		v.jumps = src.jumps.Clone()
	}
	e.panes.tree.Split(v, LayoutVertical)
	e.markDocAccessed()
	return v, true
}

// HSplit opens docID in a new horizontal split (stacked)
func (e *Editor) HSplit(docID DocumentId) (*View, bool) {
	doc, ok := e.documents.byID[docID]
	if !ok {
		return nil, false
	}
	if !e.panes.tree.CanSplit(LayoutHorizontal) {
		return nil, false
	}
	e.recordLeavingDoc()
	v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
	if src, ok := e.FocusedView(); ok {
		v.jumps = src.jumps.Clone()
	}
	e.panes.tree.Split(v, LayoutHorizontal)
	e.markDocAccessed()
	return v, true
}

// SplitFocused opens the focused pane in a new split
func (e *Editor) SplitFocused(layout Layout) error {
	if !e.panes.tree.CanSplit(layout) {
		return ErrCannotSplit
	}
	p := e.panes.tree.Get(e.panes.tree.Focus())
	if p == nil {
		return ErrNoView
	}
	next, err := p.Split()
	if err != nil {
		return err
	}
	if !e.SplitPane(next, layout) {
		return ErrCannotSplit
	}
	return nil
}

// SplitPane adds p in a new split
func (e *Editor) SplitPane(p Pane, layout Layout) bool {
	if !e.panes.tree.CanSplit(layout) {
		return false
	}
	e.recordLeavingDoc()
	e.panes.tree.Split(p, layout)
	e.markDocAccessed()
	return true
}

// VSplitNew opens a new scratch document in a new vertical split
func (e *Editor) VSplitNew() *View {
	if !e.panes.tree.CanSplit(LayoutVertical) {
		return nil
	}
	doc := e.newDocument()
	e.documents.byID[doc.ID()] = doc
	v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
	if src, ok := e.FocusedView(); ok {
		v.jumps = src.jumps.Clone()
	}
	e.panes.tree.Split(v, LayoutVertical)
	e.markDocAccessed()
	return v
}

// HSplitNew opens a new scratch document in a new horizontal split
func (e *Editor) HSplitNew() *View {
	if !e.panes.tree.CanSplit(LayoutHorizontal) {
		return nil
	}
	doc := e.newDocument()
	e.documents.byID[doc.ID()] = doc
	v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
	if src, ok := e.FocusedView(); ok {
		v.jumps = src.jumps.Clone()
	}
	e.panes.tree.Split(v, LayoutHorizontal)
	e.markDocAccessed()
	return v
}

// CloseView closes a view and, if no other view references the same document,
// also closes the document
func (e *Editor) CloseView(vid Id) {
	v, ok := e.panes.tree.Get(vid).(*View)
	if !ok {
		return
	}
	focused := e.panes.tree.Focus() == vid
	docID := v.docID

	if doc, ok := e.documents.byID[docID]; ok {
		doc.RemoveView(vid)
	}

	e.panes.tree.Remove(vid)

	if !e.hasView(func(ov *View) bool { return ov.docID == docID }) {
		if doc, ok := e.documents.byID[docID]; ok {
			e.documentClosed(doc)
		}
		delete(e.documents.byID, docID)
	}
	if focused {
		e.markDocAccessed()
	}
}

// ReplacePane swaps the pane at id for p in place, with no split or reflow, and
// returns the displaced pane so a caller can restore it later
func (e *Editor) ReplacePane(id Id, p Pane) Pane {
	old := e.panes.tree.Get(id)
	e.panes.tree.ReplacePane(id, p)
	return old
}

// DiscardPane closes p's document, if p is a view and this was its last
// reference — for a displaced pane the caller has decided not to keep
func (e *Editor) DiscardPane(p Pane) {
	p.Discard()
}

// ClosePane closes the pane at id. If it is the tree's only pane, it is
// replaced with a fresh scratch document instead of leaving the tree empty
func (e *Editor) ClosePane(id Id) {
	p := e.panes.tree.Get(id)
	if p == nil {
		return
	}
	if e.panes.tree.Count() <= 1 {
		doc := e.newDocument()
		e.documents.byID[doc.ID()] = doc
		v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
		e.panes.tree.ReplacePane(id, v)
		p.Discard()
		e.markDocAccessed()
		return
	}
	p.Close()
}

// RemovePane removes a non-document pane from the layout. If it is the only
// pane, it is replaced with a fresh scratch document
func (e *Editor) RemovePane(id Id) {
	if e.panes.tree.Count() <= 1 {
		doc := e.newDocument()
		e.documents.byID[doc.ID()] = doc
		v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
		e.panes.tree.ReplacePane(id, v)
		e.markDocAccessed()
		return
	}
	e.panes.tree.Remove(id)
}

// RegisterPaneRestorer registers how to rebuild a leaf pane of the given
// session kind
func (e *Editor) RegisterPaneRestorer(kind SessionKind, fn PaneRestorer) {
	if e.panes.restorers == nil {
		e.panes.restorers = map[SessionKind]PaneRestorer{}
	}
	e.panes.restorers[kind] = fn
}

func (e *Editor) discardView(v *View) {
	doc, ok := e.documents.byID[v.docID]
	if !ok {
		return
	}
	doc.RemoveView(v.id)
	if e.hasView(func(ov *View) bool { return ov.docID == v.docID }) {
		return
	}
	e.documentClosed(doc)
	delete(e.documents.byID, v.docID)
}

// restorePane rebuilds a leaf pane of the given kind via its registered
// restorer
type restorePaneArgs struct {
	kind    SessionKind
	session *PaneSession
}

func (e *Editor) restorePane(args restorePaneArgs) (Pane, error) {
	fn, ok := e.panes.restorers[args.kind]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSessionInvalid, args.kind)
	}
	return fn(e, args.session)
}
