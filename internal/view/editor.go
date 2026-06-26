package view

import (
	"errors"

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

	nextDocID          DocumentId
	nextAccess         int64
	prevDocID          DocumentId
	lastModifiedDocIDs [2]DocumentId

	count          int
	activeRegister rune
	lastMotion     func(*Editor)

	viewHeight       int
	viewContentWidth int

	statusMsg string
	hint      string
}

var (
	ErrNoDocument        = errors.New("no document")
	ErrNoView            = errors.New("no view")
	ErrReadOnly          = errors.New("document is readonly")
	ErrDocumentNoPath    = errors.New("document has no path")
	ErrEmptyDirStack     = errors.New("directory stack is empty")
	ErrConfigUnavailable = errors.New("config path unavailable")
)

// NewEditor creates an empty editor with one scratch document and view
func NewEditor(cwd string) *Editor {
	e := &Editor{
		docs:      map[DocumentId]*Document{},
		tree:      newTree(0, 0),
		cwd:       cwd,
		opts:      defaultOptions(),
		registers: register.New(),
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
	e.recordPrevDoc()
	v := &View{docID: doc.ID(), mode: ModeNormal}
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
	e.recordPrevDoc()
	v := &View{docID: doc.ID(), mode: ModeNormal}
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
	e.tree.Split(v, LayoutHorizontal)
	e.markDocAccessed()
	return v
}

// CloseView closes a view and, if no other view references the same document,
// also closes the document
func (e *Editor) CloseView(vid Id) {
	v := e.tree.Get(vid)
	if v == nil {
		return
	}
	focused := e.tree.Focus() == vid
	docID := v.docID

	if doc, ok := e.docs[docID]; ok {
		doc.RemoveView(vid)
	}

	e.tree.Remove(vid)

	// clean up document if no longer referenced
	referenced := false
	for _, ov := range e.tree.Traverse() {
		if ov.docID == docID {
			referenced = true
			break
		}
	}
	if !referenced {
		delete(e.docs, docID)
	}
	if focused {
		e.markDocAccessed()
	}
}

// FocusedView returns the currently focused view
func (e *Editor) FocusedView() (*View, bool) {
	v := e.tree.Get(e.tree.Focus())
	if v == nil {
		return nil, false
	}
	return v, true
}

// FocusView moves focus to the given view
func (e *Editor) FocusView(vid Id) {
	if e.tree.Get(vid) != nil {
		e.recordPrevDoc()
		e.tree.SetFocus(vid)
		e.markDocAccessed()
	}
}

// FocusNextView moves focus to the next view in DFS order
func (e *Editor) FocusNextView() {
	e.recordPrevDoc()
	e.tree.SetFocus(e.tree.Next())
	e.markDocAccessed()
}

// FocusPrevView moves focus to the previous view in DFS order
func (e *Editor) FocusPrevView() {
	e.recordPrevDoc()
	e.tree.SetFocus(e.tree.Prev())
	e.markDocAccessed()
}

// FocusDirection moves focus to the nearest split in the given direction
func (e *Editor) FocusDirection(dir Direction) {
	if id, ok := e.tree.FindSplitInDirection(e.tree.Focus(), dir); ok {
		e.recordPrevDoc()
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
	if e.prevDocID == InvalidDocumentId {
		return InvalidDocumentId, false
	}
	if _, ok := e.docs[e.prevDocID]; !ok {
		return InvalidDocumentId, false
	}
	return e.prevDocID, true
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
	if e.configReload != nil {
		return e.configReload()
	}
	return ErrConfigUnavailable
}

// View returns a view by id
func (e *Editor) View(vid Id) (*View, bool) {
	v := e.tree.Get(vid)
	if v == nil {
		return nil, false
	}
	return v, true
}

// Document returns a document by id
func (e *Editor) Document(did DocumentId) (*Document, bool) {
	d, ok := e.docs[did]
	return d, ok
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
	if v, ok := e.FocusedView(); ok {
		return v.Mode()
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
	return e.tree.Traverse()
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
	e.statusMsg = msg
}

// TakeStatusMsg returns the pending status message and clears it
func (e *Editor) TakeStatusMsg() string {
	msg := e.statusMsg
	e.statusMsg = ""
	return msg
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

// recordPrevDoc saves the current focused document as the alternate file and
// updates the last-modified list if the document was changed since last access
func (e *Editor) recordPrevDoc() {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	did := v.DocID()
	e.prevDocID = did
	if doc, ok := e.docs[did]; ok && doc.modifiedSinceAccessed {
		if e.lastModifiedDocIDs[0] != did {
			e.lastModifiedDocIDs[1] = e.lastModifiedDocIDs[0]
			e.lastModifiedDocIDs[0] = did
		}
	}
}

// markDocAccessed clears the modifiedSinceAccessed flag on the focused document
func (e *Editor) markDocAccessed() {
	if v, ok := e.FocusedView(); ok {
		if doc, ok := e.docs[v.DocID()]; ok {
			e.nextAccess++
			doc.accessedAt = e.nextAccess
			doc.modifiedSinceAccessed = false
		}
	}
}
