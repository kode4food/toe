package view

import (
	"errors"
	"sync"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view/register"
)

type (
	// Editor holds the full state of the editor session: all open documents,
	// the view layout tree, and shared editor state
	Editor struct {
		docs          map[DocumentId]*Document
		tree          *Tree
		cwd           string
		dirStack      []string
		opts          Options
		configReload  func() error
		registers     register.Registers
		clipboard     Clipboard
		docObservers  []DocumentObserver
		langServers   LanguageServerController
		indenter      Indenter
		paneRestorers map[string]PaneRestorer

		versionControl VersionControl

		nextDocID          DocumentId
		nextAccess         int64
		lastModifiedDocIDs [2]DocumentId

		count          int
		activeRegister rune
		lastMotion     func(*Editor)

		viewSize geom.Size

		statusMsg   string
		statusMsgMu sync.Mutex
		hint        string
	}

	// PaneRestorer rebuilds a leaf pane of a given session kind from its
	// persisted state
	PaneRestorer func(*Editor, *PaneSession) (Pane, error)

	// PaneSession exposes module-owned state for a restored pane
	PaneSession struct {
		path   string
		values map[string]any
	}
)

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
	ErrCannotSplit  = errors.New("pane is too small to split")
)

// NewEditor creates an empty editor with one scratch document and view
func NewEditor(cwd string) *Editor {
	e := &Editor{
		docs:      map[DocumentId]*Document{},
		tree:      newTree(geom.Size{}),
		cwd:       cwd,
		opts:      defaultOptions(),
		registers: register.New(),
		clipboard: noopClipboard{},
	}
	doc := e.newDocument()
	e.docs[doc.ID()] = doc
	v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
	e.tree.Insert(v)
	e.markDocAccessed()
	return e
}

// Path returns the pane's persisted path
func (s *PaneSession) Path() string {
	return s.path
}

// Value returns module-owned pane state by key
func (s *PaneSession) Value(key string) (any, bool) {
	value, ok := s.values[key]
	return value, ok
}

// Tree returns the layout tree
func (e *Editor) Tree() *Tree {
	return e.tree
}

// ResizeTree resizes the layout tree to the given content area dimensions
func (e *Editor) ResizeTree(size geom.Size) {
	e.tree.Resize(size)
}

// SetLastMotion records fn as the last repeatable motion
func (e *Editor) SetLastMotion(fn func(*Editor)) {
	e.lastMotion = fn
}

// LastMotion returns the most recently recorded repeatable motion
func (e *Editor) LastMotion() func(*Editor) {
	return e.lastMotion
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

// DeleteDocument removes a document without closing its views; affected views
// will report no focused document
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
	return e.viewSize.Height
}

// SetViewHeight sets the content area height (called by the UI on resize)
func (e *Editor) SetViewHeight(h int) {
	e.viewSize.Height = h
}

// ViewContentWidth returns the last-reported text content width (viewport minus
// gutter), used for visual-line movement when soft-wrap is active
func (e *Editor) ViewContentWidth() int {
	return e.viewSize.Width
}

// SetViewContentWidth stores the text content width (called by the renderer
// after computing the gutter width for the focused document)
func (e *Editor) SetViewContentWidth(w int) {
	e.viewSize.Width = w
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

// CloseCurrentView closes the focused pane
func (e *Editor) CloseCurrentView() {
	p := e.tree.Get(e.tree.Focus())
	if p == nil {
		return
	}
	p.Close()
}

// CloseAllOtherViews closes all views except the focused one
func (e *Editor) CloseAllOtherViews() {
	focused := e.tree.Focus()
	for _, p := range e.tree.Traverse() {
		if p.ID() != focused {
			e.ClosePane(p.ID())
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
