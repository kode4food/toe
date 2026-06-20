package view

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kode4food/toe/internal/core"
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

	nextDocID DocumentId

	// count is the pending numeric argument (0 means none)
	count int
	// activeRegister is the pending register selection (0 means default)
	activeRegister rune
	// viewHeight is the current content area height, kept in sync by the UI
	viewHeight int
	// viewContentWidth is the text content area width (viewport minus gutter),
	// kept in sync by the UI layer and used for visual-line movement
	viewContentWidth int
	// prevDocID is the doc switched away from most recently (alternate file)
	prevDocID DocumentId
	// lastModifiedDocIDs tracks the two most recently modified-then-left docs
	lastModifiedDocIDs [2]DocumentId
	// lastMotion is the most recently recorded repeatable motion
	lastMotion func(*Editor)
	// statusMsg is a transient message set by commands (error, warning, info)
	// The UI reads and clears it after each keypress via TakeStatusMsg
	statusMsg string
	// hint is a transient display hint set by actions during a continuation
	// (e.g. "f ...", "r ..."). The UI reads and clears it via TakeHint
	hint string
}

var (
	ErrNoDocument        = errors.New("no document")
	ErrNoView            = errors.New("no view")
	ErrReadonly          = errors.New("document is readonly")
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
	return e
}

// Tree returns the layout tree
func (e *Editor) Tree() *Tree { return e.tree }

// ResizeTree resizes the layout tree to the given content area dimensions
func (e *Editor) ResizeTree(width, height int) {
	e.tree.Resize(width, height)
}

// OpenFile replaces the focused view's document with the given file, reusing an
// existing document if it is already open
func (e *Editor) OpenFile(path string) (*View, error) {
	return e.SwitchFile(path)
}

// SwitchFile replaces the focused view's document with the given file, reusing
// an existing document if it is already open
func (e *Editor) SwitchFile(path string) (*View, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	for _, d := range e.docs {
		if d.Path() == absPath {
			e.recordPrevDoc()
			if v, ok := e.FocusedView(); ok {
				v.docID = d.ID()
				v.offset = Position{}
				e.markDocAccessed()
				return v, nil
			}
			return nil, ErrNoView
		}
	}

	doc, err := e.openFile(absPath)
	if err != nil {
		return nil, err
	}
	e.recordPrevDoc()
	e.docs[doc.ID()] = doc
	if v, ok := e.FocusedView(); ok {
		v.docID = doc.ID()
		v.offset = Position{}
		e.markDocAccessed()
		return v, nil
	}
	return nil, ErrNoView
}

// SwitchBuffer replaces the focused view's document with an already-open
// document by ID. Returns false if the document does not exist or there is no
// focused view
func (e *Editor) SwitchBuffer(did DocumentId) bool {
	if _, ok := e.docs[did]; !ok {
		return false
	}
	e.recordPrevDoc()
	if v, ok := e.FocusedView(); ok {
		v.docID = did
		v.offset = Position{}
		e.markDocAccessed()
		return true
	}
	return false
}

// SwitchOrOpenDoc returns an existing document for path, opening it if needed
func (e *Editor) SwitchOrOpenDoc(path string) (*Document, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	for _, d := range e.docs {
		if d.Path() == absPath {
			return d, nil
		}
	}
	doc, err := e.openFile(absPath)
	if err != nil {
		return nil, err
	}
	e.docs[doc.ID()] = doc
	return doc, nil
}

// VSplit opens docID in a new vertical split (side by side)
func (e *Editor) VSplit(docID DocumentId) (*View, bool) {
	doc, ok := e.docs[docID]
	if !ok {
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
	e.recordPrevDoc()
	v := &View{docID: doc.ID(), mode: ModeNormal}
	e.tree.Split(v, LayoutHorizontal)
	e.markDocAccessed()
	return v, true
}

// VSplitNew opens a new scratch document in a new vertical split
func (e *Editor) VSplitNew() *View {
	doc := e.newDocument()
	e.docs[doc.ID()] = doc
	v := &View{docID: doc.ID(), mode: ModeNormal}
	e.tree.Split(v, LayoutVertical)
	return v
}

// HSplitNew opens a new scratch document in a new horizontal split
func (e *Editor) HSplitNew() *View {
	doc := e.newDocument()
	e.docs[doc.ID()] = doc
	v := &View{docID: doc.ID(), mode: ModeNormal}
	e.tree.Split(v, LayoutHorizontal)
	return v
}

// CloseView closes a view and, if no other view references the same document,
// also closes the document
func (e *Editor) CloseView(vid Id) {
	v := e.tree.Get(vid)
	if v == nil {
		return
	}
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
			doc.modifiedSinceAccessed = false
		}
	}
}

// SetLastMotion records fn as the last repeatable motion
func (e *Editor) SetLastMotion(fn func(*Editor)) { e.lastMotion = fn }

// LastMotion returns the most recently recorded repeatable motion
func (e *Editor) LastMotion() func(*Editor) { return e.lastMotion }

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
func (e *Editor) Options() *Options { return &e.opts }

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

// Apply applies a transaction to the focused document for the focused view
func (e *Editor) Apply(tx core.Transaction) error {
	v, ok := e.FocusedView()
	if !ok {
		return ErrNoView
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return ErrNoDocument
	}
	if v.Mode() == ModeInsert {
		doc.BeginInsertGroup(v.ID())
	}
	return doc.Apply(tx, v.ID())
}

// CommitInsertHistory flushes any pending insert-mode history accumulation on
// the focused document into a single history revision
func (e *Editor) CommitInsertHistory() {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return
	}
	doc.CommitInsertHistory(v.ID())
}

// Undo reverts one history step in the focused document
func (e *Editor) Undo() bool {
	v, ok := e.FocusedView()
	if !ok {
		return false
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return false
	}
	return doc.Undo(v.ID())
}

// Redo reapplies one reverted step in the focused document
func (e *Editor) Redo() bool {
	v, ok := e.FocusedView()
	if !ok {
		return false
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return false
	}
	return doc.Redo(v.ID())
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
func (e *Editor) Count() int { return e.count }

// SetCount sets the pending numeric count
func (e *Editor) SetCount(n int) { e.count = n }

// ResetCount clears the pending count
func (e *Editor) ResetCount() { e.count = 0 }

// ActiveRegister returns the pending register rune (0 = default)
func (e *Editor) ActiveRegister() rune { return e.activeRegister }

// SetRegister sets the pending register selection
func (e *Editor) SetRegister(r rune) { e.activeRegister = r }

// ResetRegister clears the pending register to the default
func (e *Editor) ResetRegister() { e.activeRegister = 0 }

// Registers returns the editor's register store
func (e *Editor) Registers() register.Registers { return e.registers }

// ViewHeight returns the last-reported content area height
func (e *Editor) ViewHeight() int { return e.viewHeight }

// SetViewHeight sets the content area height (called by the UI on resize)
func (e *Editor) SetViewHeight(h int) { e.viewHeight = h }

// ViewContentWidth returns the last-reported text content width (viewport minus
// gutter), used for visual-line movement when soft-wrap is active
func (e *Editor) ViewContentWidth() int { return e.viewContentWidth }

// SetViewContentWidth stores the text content width (called by the renderer
// after computing the gutter width for the focused document)
func (e *Editor) SetViewContentWidth(w int) { e.viewContentWidth = w }

// Cwd returns the editor working directory
func (e *Editor) Cwd() string { return e.cwd }

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

// Save saves the focused document
func (e *Editor) Save() error {
	doc, ok := e.FocusedDocument()
	if !ok {
		return ErrNoDocument
	}
	return doc.Save(&e.opts)
}

// NewDocument creates a new empty scratch document and makes it the focused
// view
func (e *Editor) NewDocument() *View {
	doc := e.newDocument()
	e.docs[doc.ID()] = doc
	v, ok := e.FocusedView()
	if ok {
		v.docID = doc.ID()
		v.offset = Position{}
		return v
	}
	nv := &View{docID: doc.ID(), mode: ModeNormal}
	e.tree.Insert(nv)
	return nv
}

// SaveAll saves all modified documents
func (e *Editor) SaveAll() []error {
	var errs []error
	for _, doc := range e.docs {
		if doc.Modified() {
			if err := doc.Save(&e.opts); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
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

// Reload reloads the focused document from disk
func (e *Editor) Reload() error {
	doc, ok := e.FocusedDocument()
	if !ok {
		return ErrNoDocument
	}
	return doc.Reload()
}

// ReloadAll reloads all documents that have a file path
func (e *Editor) ReloadAll() []error {
	var errs []error
	for _, doc := range e.docs {
		if doc.Path() != "" {
			if err := doc.Reload(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// Chdir changes the editor working directory
func (e *Editor) Chdir(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.Chdir(abs); err != nil {
		return err
	}
	e.cwd = abs
	return nil
}

// PushDirectory pushes the current directory onto the stack then chdirs
func (e *Editor) PushDirectory(path string) error {
	e.dirStack = append(e.dirStack, e.cwd)
	return e.Chdir(path)
}

// PopDirectory changes to the top of the directory stack, if any
func (e *Editor) PopDirectory() error {
	if len(e.dirStack) == 0 {
		return ErrEmptyDirStack
	}
	top := e.dirStack[len(e.dirStack)-1]
	e.dirStack = e.dirStack[:len(e.dirStack)-1]
	return e.Chdir(top)
}

// DirStack returns a copy of the directory stack (bottom to top)
func (e *Editor) DirStack() []string {
	cp := make([]string, len(e.dirStack))
	copy(cp, e.dirStack)
	return cp
}

// SetStatusMsg stores a transient status message to be displayed after the
// current keypress. The UI clears it via TakeStatusMsg
func (e *Editor) SetStatusMsg(msg string) { e.statusMsg = msg }

// TakeStatusMsg returns the pending status message and clears it
func (e *Editor) TakeStatusMsg() string {
	msg := e.statusMsg
	e.statusMsg = ""
	return msg
}

// SetHint stores a transient hint shown during an active continuation
func (e *Editor) SetHint(h string) { e.hint = h }

// TakeHint returns the pending hint and clears it
func (e *Editor) TakeHint() string {
	h := e.hint
	e.hint = ""
	return h
}

// Earlier navigates history backward by the given UndoKind
func (e *Editor) Earlier(kind core.UndoKind) bool {
	v, ok := e.FocusedView()
	if !ok {
		return false
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return false
	}
	txns := doc.history.Earlier(kind)
	for _, tx := range txns {
		inv, err := tx.Invert(doc.text)
		if err != nil {
			return false
		}
		newText, err := inv.Apply(doc.text)
		if err != nil {
			return false
		}
		doc.text = newText
		if txSel := tx.Selection(); txSel != nil {
			doc.SetSelectionFor(v.ID(), *txSel)
		}
	}
	doc.modified = len(txns) > 0 || doc.modified
	return len(txns) > 0
}

// Later navigates history forward by the given UndoKind
func (e *Editor) Later(kind core.UndoKind) bool {
	v, ok := e.FocusedView()
	if !ok {
		return false
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return false
	}
	txns := doc.history.Later(kind)
	for _, tx := range txns {
		newText, err := tx.Apply(doc.text)
		if err != nil {
			return false
		}
		doc.text = newText
		if txSel := tx.Selection(); txSel != nil {
			doc.SetSelectionFor(v.ID(), *txSel)
		}
	}
	doc.modified = len(txns) > 0 || doc.modified
	return len(txns) > 0
}

func (e *Editor) newDocument() *Document {
	e.nextDocID++
	return newDocument(e.nextDocID, &e.opts)
}

func (e *Editor) openFile(path string) (*Document, error) {
	e.nextDocID++
	return openDocument(e.nextDocID, path, &e.opts)
}
