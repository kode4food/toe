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
		documents documentState
		panes     paneState
		workspace workspaceState
		registers registerState
		command   commandState
		messages  messageState

		opts           Options
		configReload   func() error
		docObservers   []DocumentObserver
		langServers    LanguageServerController
		indenter       Indenter
		versionControl VersionControl
	}

	documentState struct {
		byID            map[DocumentId]*Document
		nextID          DocumentId
		nextAccess      int64
		lastModifiedIDs [2]DocumentId
	}

	paneState struct {
		tree        *Tree
		restorers   map[SessionKind]PaneRestorer
		contentSize geom.Size
	}

	workspaceState struct {
		cwd      string
		dirStack []string
	}

	registerState struct {
		values    register.Registers
		clipboard Clipboard
	}

	commandState struct {
		count          int
		activeRegister rune
		lastMotion     func(*Editor)
	}

	messageState struct {
		status   string
		statusMu sync.Mutex
		hint     string
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
		documents: documentState{
			byID: map[DocumentId]*Document{},
		},
		panes: paneState{
			tree: newTree(geom.Size{}),
		},
		workspace: workspaceState{
			cwd: cwd,
		},
		registers: registerState{
			values:    register.New(),
			clipboard: noopClipboard{},
		},
		opts: defaultOptions(),
	}
	doc := e.newDocument()
	e.documents.byID[doc.ID()] = doc
	v := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
	e.panes.tree.Insert(v)
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
	return e.panes.tree
}

// ResizeTree resizes the layout tree to the given content area dimensions
func (e *Editor) ResizeTree(size geom.Size) {
	e.panes.tree.Resize(size)
}

// SetLastMotion records fn as the last repeatable motion
func (e *Editor) SetLastMotion(fn func(*Editor)) {
	e.command.lastMotion = fn
}

// LastMotion returns the most recently recorded repeatable motion
func (e *Editor) LastMotion() func(*Editor) {
	return e.command.lastMotion
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
	v, ok := e.panes.tree.Get(vid).(*View)
	return v, ok
}

// Document returns a document by id
func (e *Editor) Document(did DocumentId) (*Document, bool) {
	d, ok := e.documents.byID[did]
	return d, ok
}

// DeleteDocument removes a document without closing its views; affected views
// will report no focused document
func (e *Editor) DeleteDocument(did DocumentId) {
	delete(e.documents.byID, did)
	e.panes.tree.Range(func(p Pane) bool {
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
	if p := e.panes.tree.Get(e.panes.tree.Focus()); p != nil {
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
	return e.command.count
}

// SetCount sets the pending numeric count
func (e *Editor) SetCount(n int) {
	e.command.count = n
}

// ResetCount clears the pending count
func (e *Editor) ResetCount() {
	e.command.count = 0
}

// ActiveRegister returns the pending register rune (0 = default)
func (e *Editor) ActiveRegister() rune {
	return e.command.activeRegister
}

// SetRegister sets the pending register selection
func (e *Editor) SetRegister(r rune) {
	e.command.activeRegister = r
}

// ResetRegister clears the pending register to the default
func (e *Editor) ResetRegister() {
	e.command.activeRegister = 0
}

// Registers returns the editor's register store
func (e *Editor) Registers() register.Registers {
	return e.registers.values
}

func (e *Editor) Clipboard() Clipboard {
	return e.registers.clipboard
}

func (e *Editor) SetClipboard(c Clipboard) {
	e.registers.clipboard = c
}

// ViewHeight returns the last-reported content area height
func (e *Editor) ViewHeight() int {
	return e.panes.contentSize.Height
}

// SetViewHeight sets the content area height (called by the UI on resize)
func (e *Editor) SetViewHeight(h int) {
	e.panes.contentSize.Height = h
}

// ViewContentWidth returns the last-reported text content width (viewport minus
// gutter), used for visual-line movement when soft-wrap is active
func (e *Editor) ViewContentWidth() int {
	return e.panes.contentSize.Width
}

// SetViewContentWidth stores the text content width (called by the renderer
// after computing the gutter width for the focused document)
func (e *Editor) SetViewContentWidth(w int) {
	e.panes.contentSize.Width = w
}

// Cwd returns the editor working directory
func (e *Editor) Cwd() string {
	return e.workspace.cwd
}

// AllDocuments returns all open documents
func (e *Editor) AllDocuments() []*Document {
	out := make([]*Document, 0, len(e.documents.byID))
	for _, d := range e.documents.byID {
		out = append(out, d)
	}
	return out
}

// AllViews returns all open views in DFS order
func (e *Editor) AllViews() []*View {
	out := make([]*View, 0, e.panes.tree.Count())
	e.panes.tree.Range(func(p Pane) bool {
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
	focus := e.panes.tree.Focus()
	var out []struct {
		View    *View
		Focused bool
	}
	e.panes.tree.Range(func(p Pane) bool {
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
	e.panes.tree.Range(func(p Pane) bool {
		v, ok := p.(*View)
		if !ok || seen[v.docID] {
			return true
		}
		if doc, ok := e.documents.byID[v.docID]; ok {
			seen[v.docID] = true
			out = append(out, doc)
		}
		return true
	})
	return out
}

// CloseCurrentView closes the focused pane
func (e *Editor) CloseCurrentView() {
	p := e.panes.tree.Get(e.panes.tree.Focus())
	if p == nil {
		return
	}
	p.Close()
}

// CloseAllOtherViews closes all views except the focused one
func (e *Editor) CloseAllOtherViews() {
	focused := e.panes.tree.Focus()
	for _, p := range e.panes.tree.Traverse() {
		if p.ID() != focused {
			e.ClosePane(p.ID())
		}
	}
}

// SetStatusMsg stores a transient status message to be displayed after the
// current keypress. The UI clears it via TakeStatusMsg
func (e *Editor) SetStatusMsg(msg string) {
	e.messages.statusMu.Lock()
	e.messages.status = msg
	e.messages.statusMu.Unlock()
}

// TakeStatusMsg returns the pending status message and clears it
func (e *Editor) TakeStatusMsg() string {
	e.messages.statusMu.Lock()
	msg := e.messages.status
	e.messages.status = ""
	e.messages.statusMu.Unlock()
	return msg
}

// SetHint stores a transient hint shown during an active continuation
func (e *Editor) SetHint(h string) {
	e.messages.hint = h
}

// TakeHint returns the pending hint and clears it
func (e *Editor) TakeHint() string {
	h := e.messages.hint
	e.messages.hint = ""
	return h
}
