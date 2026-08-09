package view

import "fmt"

// VSplit opens docID in a new vertical split (side by side)
func (e *Editor) VSplit(docID DocumentId) *View {
	doc, ok := e.documents.byID[docID]
	if !ok {
		return nil
	}
	if !e.panes.tree.CanSplit(LayoutVertical) {
		return nil
	}
	e.recordLeavingDoc()
	v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
	if src := e.FocusedView(); src != nil {
		v.jumps = src.jumps.Clone()
	}
	e.panes.tree.Split(v, LayoutVertical)
	e.markDocAccessed()
	return v
}

// HSplit opens docID in a new horizontal split (stacked)
func (e *Editor) HSplit(docID DocumentId) *View {
	doc, ok := e.documents.byID[docID]
	if !ok {
		return nil
	}
	if !e.panes.tree.CanSplit(LayoutHorizontal) {
		return nil
	}
	e.recordLeavingDoc()
	v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
	if src := e.FocusedView(); src != nil {
		v.jumps = src.jumps.Clone()
	}
	e.panes.tree.Split(v, LayoutHorizontal)
	e.markDocAccessed()
	return v
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
	if src := e.FocusedView(); src != nil {
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
	if src := e.FocusedView(); src != nil {
		v.jumps = src.jumps.Clone()
	}
	e.panes.tree.Split(v, LayoutHorizontal)
	e.markDocAccessed()
	return v
}

// ReplacePane swaps the pane at id for p in place, discarding any panes stashed
// behind id, and returns the evicted pane for the caller to dispose of
func (e *Editor) ReplacePane(id Id, p Pane) Pane {
	old := e.panes.tree.Get(id)
	e.panes.tree.DiscardHistory(id)
	e.panes.tree.ReplacePane(id, p)
	return old
}

// DisplacePane swaps the pane at id for p, stashing the displaced pane on the
// node so RevertPane can restore it when p closes
func (e *Editor) DisplacePane(id Id, p Pane) {
	e.panes.tree.DisplacePane(id, p)
}

// RevertPane restores the pane most recently displaced at id, reporting whether
// one was available
func (e *Editor) RevertPane(id Id) bool {
	return e.panes.tree.RevertPane(id)
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
		e.panes.tree.DiscardHistory(id)
		e.panes.tree.ReplacePane(id, v)
		p.Discard()
		e.markDocAccessed()
		return
	}
	focused := e.panes.tree.Focus() == id
	e.panes.tree.Remove(id)
	p.Discard()
	if _, ok := p.(*View); focused && ok {
		e.markDocAccessed()
	}
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
	if fn, ok := e.panes.restorers[args.kind]; ok {
		return fn(e, args.session)
	}
	return nil, fmt.Errorf("%w: %s", ErrSessionInvalid, args.kind)
}
