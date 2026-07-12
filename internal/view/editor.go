package view

import (
	"errors"
	"sync"

	"github.com/kode4food/toe/internal/view/register"
)

// Editor holds the full state of the editor session: all open documents, the
// view layout tree, and shared editor state
type Editor struct {
	docs         map[DocumentId]*Document
	tree         *Tree
	cwd          string
	dirStack     []string
	opts         Options
	configReload func() error
	registers    register.Registers
	clipboard    Clipboard
	docObservers []DocumentObserver
	langServers  LanguageServerController
	indenter     Indenter

	versionControl VersionControl

	nextDocID          DocumentId
	nextAccess         int64
	lastModifiedDocIDs [2]DocumentId
	pendingTerminals   []Id

	count          int
	activeRegister rune
	lastMotion     func(*Editor)

	viewHeight       int
	viewContentWidth int

	statusMsg   string
	statusMsgMu sync.Mutex
	hint        string
}

var (
	ErrNoDocument        = errors.New("no document")
	ErrNoView            = errors.New("no view")
	ErrReadOnly          = errors.New("document is readonly")
	ErrDocumentNoPath    = errors.New("document has no path")
	ErrEmptyDirStack     = errors.New("directory stack is empty")
	ErrConfigUnavailable = errors.New("config path unavailable")
	ErrUnsavedChanges    = errors.New("unsaved changes")
	ErrFileChangedOnDisk = errors.New(
		"file modified by an external process, use :w! to overwrite",
	)
	ErrFileReadOnly = errors.New("path is read only")
)

// NewEditor creates an empty editor with one scratch document and view
func NewEditor(cwd string) *Editor {
	e := &Editor{
		docs:      map[DocumentId]*Document{},
		tree:      newTree(0, 0),
		cwd:       cwd,
		opts:      defaultOptions(),
		registers: register.New(),
		clipboard: noopClipboard{},
	}
	doc := e.newDocument()
	e.docs[doc.ID()] = doc
	v := &View{docID: doc.ID(), mode: ModeNormal}
	e.tree.Insert(v)
	e.markDocAccessed()
	return e
}

// Tree returns the layout tree
func (e *Editor) Tree() *Tree {
	return e.tree
}

// ResizeTree resizes the layout tree to the given content area dimensions
func (e *Editor) ResizeTree(width, height int) {
	e.tree.Resize(width, height)
}

// VSplit opens docID in a new vertical split (side by side)
func (e *Editor) VSplit(docID DocumentId) (*View, bool) {
	doc, ok := e.docs[docID]
	if !ok {
		return nil, false
	}
	if !e.tree.CanSplit(LayoutVertical) {
		return nil, false
	}
	e.recordLeavingDoc()
	v := &View{docID: doc.ID(), mode: ModeNormal}
	if src, ok := e.FocusedView(); ok {
		v.jumps = src.jumps.Clone()
	}
	e.tree.Split(v, LayoutVertical)
	e.markDocAccessed()
	return v, true
}

// HSplit opens docID in a new horizontal split (stacked)
func (e *Editor) HSplit(docID DocumentId) (*View, bool) {
	doc, ok := e.docs[docID]
	if !ok {
		return nil, false
	}
	if !e.tree.CanSplit(LayoutHorizontal) {
		return nil, false
	}
	e.recordLeavingDoc()
	v := &View{docID: doc.ID(), mode: ModeNormal}
	if src, ok := e.FocusedView(); ok {
		v.jumps = src.jumps.Clone()
	}
	e.tree.Split(v, LayoutHorizontal)
	e.markDocAccessed()
	return v, true
}

// VSplitNew opens a new scratch document in a new vertical split
func (e *Editor) VSplitNew() *View {
	if !e.tree.CanSplit(LayoutVertical) {
		return nil
	}
	doc := e.newDocument()
	e.docs[doc.ID()] = doc
	v := &View{docID: doc.ID(), mode: ModeNormal}
	if src, ok := e.FocusedView(); ok {
		v.jumps = src.jumps.Clone()
	}
	e.tree.Split(v, LayoutVertical)
	e.markDocAccessed()
	return v
}

// HSplitNew opens a new scratch document in a new horizontal split
func (e *Editor) HSplitNew() *View {
	if !e.tree.CanSplit(LayoutHorizontal) {
		return nil
	}
	doc := e.newDocument()
	e.docs[doc.ID()] = doc
	v := &View{docID: doc.ID(), mode: ModeNormal}
	if src, ok := e.FocusedView(); ok {
		v.jumps = src.jumps.Clone()
	}
	e.tree.Split(v, LayoutHorizontal)
	e.markDocAccessed()
	return v
}

// CloseView closes a view and, if no other view references the same document,
// also closes the document
func (e *Editor) CloseView(vid Id) {
	v, ok := e.tree.Get(vid).(*View)
	if !ok {
		return
	}
	focused := e.tree.Focus() == vid
	docID := v.docID

	if doc, ok := e.docs[docID]; ok {
		doc.RemoveView(vid)
	}

	e.tree.Remove(vid)

	if !e.hasView(func(ov *View) bool { return ov.docID == docID }) {
		if doc, ok := e.docs[docID]; ok {
			e.documentClosed(doc)
		}
		delete(e.docs, docID)
	}
	if focused {
		e.markDocAccessed()
	}
}

// ReplacePane swaps the pane at id for p in place, with no split or reflow, and
// returns the displaced pane so a caller can restore it later
func (e *Editor) ReplacePane(id Id, p Pane) Pane {
	old := e.tree.Get(id)
	e.tree.ReplacePane(id, p)
	return old
}

// DiscardPane closes p's document, if p is a view and this was its last
// reference — for a displaced pane the caller has decided not to keep
func (e *Editor) DiscardPane(p Pane) {
	v, ok := p.(*View)
	if !ok {
		return
	}
	doc, ok := e.docs[v.docID]
	if !ok {
		return
	}
	doc.RemoveView(v.id)
	if e.hasView(func(ov *View) bool { return ov.docID == v.docID }) {
		return
	}
	e.documentClosed(doc)
	delete(e.docs, v.docID)
}

// ClosePane closes the pane at id. If it is the tree's only pane, it is
// replaced with a fresh scratch document instead of leaving the tree empty
func (e *Editor) ClosePane(id Id) {
	if e.tree.Count() <= 1 {
		doc := e.newDocument()
		e.docs[doc.ID()] = doc
		e.tree.ReplacePane(id, &View{docID: doc.ID(), mode: ModeNormal})
		e.markDocAccessed()
		return
	}
	if _, ok := e.tree.Get(id).(*View); ok {
		e.CloseView(id)
		return
	}
	e.tree.Remove(id)
}

// FocusedView returns the currently focused view
func (e *Editor) FocusedView() (*View, bool) {
	v, ok := e.tree.Get(e.tree.Focus()).(*View)
	return v, ok
}

// FocusView moves focus to the given view
func (e *Editor) FocusView(vid Id) {
	if _, ok := e.tree.Get(vid).(*View); ok {
		e.recordLeavingDoc()
		e.tree.SetFocus(vid)
		e.markDocAccessed()
	}
}

// FocusNextView moves focus to the next view in DFS order
func (e *Editor) FocusNextView() {
	e.recordLeavingDoc()
	e.tree.SetFocus(e.tree.Next())
	e.markDocAccessed()
}

// FocusPrevView moves focus to the previous view in DFS order
func (e *Editor) FocusPrevView() {
	e.recordLeavingDoc()
	e.tree.SetFocus(e.tree.Prev())
	e.markDocAccessed()
}

// FocusDirection moves focus to the nearest split in the given direction
func (e *Editor) FocusDirection(dir Direction) {
	if id, ok := e.tree.FindSplitInDirection(e.tree.Focus(), dir); ok {
		e.recordLeavingDoc()
		e.tree.SetFocus(id)
		e.markDocAccessed()
	}
}

// SwapSplitInDirection swaps focus with the nearest split in the direction
func (e *Editor) SwapSplitInDirection(dir Direction) {
	e.tree.SwapSplitInDirection(dir)
}

// Transpose flips the layout of the container holding the focused view
func (e *Editor) Transpose() {
	e.tree.Transpose()
}

// SetLastMotion records fn as the last repeatable motion
func (e *Editor) SetLastMotion(fn func(*Editor)) {
	e.lastMotion = fn
}

// LastMotion returns the most recently recorded repeatable motion
func (e *Editor) LastMotion() func(*Editor) {
	return e.lastMotion
}

// LastModifiedDocIDs returns the two most recently modified-and-left documents,
// with the most recent first. Invalid entries have value InvalidDocumentId
func (e *Editor) LastModifiedDocIDs() [2]DocumentId {
	return e.lastModifiedDocIDs
}

// PrevDocID returns the document ID of the last focused document
func (e *Editor) PrevDocID() (DocumentId, bool) {
	return e.PopPrevDocID()
}

// PopPrevDocID returns and removes the most recently accessed document for the
// focused view
func (e *Editor) PopPrevDocID() (DocumentId, bool) {
	v, ok := e.FocusedView()
	if !ok {
		return InvalidDocumentId, false
	}
	for len(v.docHistory) > 0 {
		last := len(v.docHistory) - 1
		did := v.docHistory[last]
		v.docHistory = v.docHistory[:last]
		if did == v.DocID() {
			continue
		}
		if _, ok := e.docs[did]; ok {
			return did, true
		}
	}
	return InvalidDocumentId, false
}

// Options returns the typed runtime config values for the editor session
func (e *Editor) Options() *Options {
	return &e.opts
}

// SetConfigReload registers the function called by ReloadConfig to reset
// module section state and re-apply the merged TOML config
func (e *Editor) SetConfigReload(fn func() error) {
	e.configReload = fn
}

// ReloadConfig reloads the live editor config and resets module section state.
// Falls back to loading user config only when no reload function is registered
func (e *Editor) ReloadConfig() error {
	if e.configReload == nil {
		return ErrConfigUnavailable
	}
	err := e.configReload()
	e.opts.Gen++
	return err
}

// View returns a view by id
func (e *Editor) View(vid Id) (*View, bool) {
	v, ok := e.tree.Get(vid).(*View)
	return v, ok
}

// Document returns a document by id
func (e *Editor) Document(did DocumentId) (*Document, bool) {
	d, ok := e.docs[did]
	return d, ok
}

// DeleteDocument removes a document from the editor without closing its views.
// Views that referenced the document will report no focused document.
func (e *Editor) DeleteDocument(did DocumentId) {
	delete(e.docs, did)
	e.tree.Range(func(p Pane) bool {
		if v, ok := p.(*View); ok {
			v.removeDocHistory(did)
		}
		return true
	})
}

// FocusedDocument returns the document displayed by the focused view
func (e *Editor) FocusedDocument() (*Document, bool) {
	v, ok := e.FocusedView()
	if !ok {
		return nil, false
	}
	return e.Document(v.DocID())
}

// Mode returns the mode of the focused view
func (e *Editor) Mode() Mode {
	if p := e.tree.Get(e.tree.Focus()); p != nil {
		return p.Mode()
	}
	return ModeNormal
}

// SetMode sets the mode of the focused view
func (e *Editor) SetMode(m Mode) {
	if v, ok := e.FocusedView(); ok {
		v.SetMode(m)
	}
}

// Count returns the pending numeric count argument (0 = none)
func (e *Editor) Count() int {
	return e.count
}

// SetCount sets the pending numeric count
func (e *Editor) SetCount(n int) {
	e.count = n
}

// ResetCount clears the pending count
func (e *Editor) ResetCount() {
	e.count = 0
}

// ActiveRegister returns the pending register rune (0 = default)
func (e *Editor) ActiveRegister() rune {
	return e.activeRegister
}

// SetRegister sets the pending register selection
func (e *Editor) SetRegister(r rune) {
	e.activeRegister = r
}

// ResetRegister clears the pending register to the default
func (e *Editor) ResetRegister() {
	e.activeRegister = 0
}

// Registers returns the editor's register store
func (e *Editor) Registers() register.Registers {
	return e.registers
}

func (e *Editor) Clipboard() Clipboard {
	return e.clipboard
}

func (e *Editor) SetClipboard(c Clipboard) {
	e.clipboard = c
}

// ViewHeight returns the last-reported content area height
func (e *Editor) ViewHeight() int {
	return e.viewHeight
}

// SetViewHeight sets the content area height (called by the UI on resize)
func (e *Editor) SetViewHeight(h int) {
	e.viewHeight = h
}

// ViewContentWidth returns the last-reported text content width (viewport minus
// gutter), used for visual-line movement when soft-wrap is active
func (e *Editor) ViewContentWidth() int {
	return e.viewContentWidth
}

// SetViewContentWidth stores the text content width (called by the renderer
// after computing the gutter width for the focused document)
func (e *Editor) SetViewContentWidth(w int) {
	e.viewContentWidth = w
}

// Cwd returns the editor working directory
func (e *Editor) Cwd() string {
	return e.cwd
}

// AllDocuments returns all open documents
func (e *Editor) AllDocuments() []*Document {
	out := make([]*Document, 0, len(e.docs))
	for _, d := range e.docs {
		out = append(out, d)
	}
	return out
}

// AllViews returns all open views in DFS order
func (e *Editor) AllViews() []*View {
	out := make([]*View, 0, e.tree.Count())
	e.tree.Range(func(p Pane) bool {
		if v, ok := p.(*View); ok {
			out = append(out, v)
		}
		return true
	})
	return out
}

// Views returns all open views in DFS order with a focused flag
func (e *Editor) Views() []struct {
	View    *View
	Focused bool
} {
	focus := e.tree.Focus()
	var out []struct {
		View    *View
		Focused bool
	}
	e.tree.Range(func(p Pane) bool {
		if v, ok := p.(*View); ok {
			out = append(out, struct {
				View    *View
				Focused bool
			}{View: v, Focused: v.id == focus})
		}
		return true
	})
	return out
}

// VisibleDocuments returns the deduplicated documents currently shown in a pane
func (e *Editor) VisibleDocuments() []*Document {
	seen := map[DocumentId]bool{}
	var out []*Document
	e.tree.Range(func(p Pane) bool {
		v, ok := p.(*View)
		if !ok || seen[v.docID] {
			return true
		}
		if doc, ok := e.docs[v.docID]; ok {
			seen[v.docID] = true
			out = append(out, doc)
		}
		return true
	})
	return out
}

// CloseCurrentView closes the focused view (and its document if unreferenced)
func (e *Editor) CloseCurrentView() {
	if v, ok := e.FocusedView(); ok {
		e.CloseView(v.ID())
	}
}

// CloseAllOtherViews closes all views except the focused one
func (e *Editor) CloseAllOtherViews() {
	focused := e.tree.Focus()
	for _, v := range e.tree.Traverse() {
		if v.ID() != focused {
			e.CloseView(v.ID())
		}
	}
}

// SetStatusMsg stores a transient status message to be displayed after the
// current keypress. The UI clears it via TakeStatusMsg
func (e *Editor) SetStatusMsg(msg string) {
	e.statusMsgMu.Lock()
	e.statusMsg = msg
	e.statusMsgMu.Unlock()
}

// TakeStatusMsg returns the pending status message and clears it
func (e *Editor) TakeStatusMsg() string {
	e.statusMsgMu.Lock()
	msg := e.statusMsg
	e.statusMsg = ""
	e.statusMsgMu.Unlock()
	return msg
}

// TakePendingTerminals returns and clears the panes that held a terminal in
// the most recently restored session, since a live shell can't be saved
func (e *Editor) TakePendingTerminals() []Id {
	ids := e.pendingTerminals
	e.pendingTerminals = nil
	return ids
}

// SetHint stores a transient hint shown during an active continuation
func (e *Editor) SetHint(h string) {
	e.hint = h
}

// TakeHint returns the pending hint and clears it
func (e *Editor) TakeHint() string {
	h := e.hint
	e.hint = ""
	return h
}

// hasView reports whether any view in the tree satisfies pred
func (e *Editor) hasView(pred func(*View) bool) bool {
	return e.tree.Any(func(p Pane) bool {
		v, ok := p.(*View)
		return ok && pred(v)
	})
}

// recordPrevDoc adds the current document to the focused view's access history
// before replacing it
func (e *Editor) recordPrevDoc() {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	v.addDocHistory(v.DocID())
	e.recordLeavingDocFor(v)
}

func (e *Editor) recordLeavingDoc() {
	if v, ok := e.FocusedView(); ok {
		e.recordLeavingDocFor(v)
	}
}

func (e *Editor) recordLeavingDocFor(v *View) {
	doc, ok := e.docs[v.DocID()]
	if !ok {
		return
	}
	doc.rememberSelection(v.ID())
	if doc.buf.modified {
		did := doc.ID()
		if e.lastModifiedDocIDs[0] != did {
			e.lastModifiedDocIDs[1] = e.lastModifiedDocIDs[0]
			e.lastModifiedDocIDs[0] = did
		}
	}
}

func (e *Editor) markDocAccessed() {
	if v, ok := e.FocusedView(); ok {
		if doc, ok := e.docs[v.DocID()]; ok {
			e.nextAccess++
			doc.accessedAt = e.nextAccess
			doc.buf.modified = false
		}
	}
}
