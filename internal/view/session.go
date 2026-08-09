package view

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
)

type (
	// SessionKind identifies a leaf pane's restorer in a saved session
	SessionKind string

	// sessRef is a path recorded in a session, alongside the session's base
	// directory that relative paths resolve against
	sessRef struct {
		base string
		path string
	}

	sessEditor struct {
		Version   int            `toml:"version"`
		Options   sessOptions    `toml:"option,omitempty"`
		Registers sessRegisters  `toml:"register,omitempty"`
		Maximized bool           `toml:"maximized,omitempty"`
		Documents []sessDocument `toml:"document"`
		Layout    sessNode       `toml:"layout"`
	}

	sessOptions map[string]string

	sessRegisters map[string][]string

	sessDocument struct {
		Path      string     `toml:"path,omitempty"`
		Scratch   bool       `toml:"scratch,omitempty"`
		Text      string     `toml:"text,omitempty"`
		Lang      string     `toml:"language,omitempty"`
		Selection sessSelect `toml:"selection"`
	}

	sessNode struct {
		Kind   SessionKind    `toml:"kind"`
		Path   string         `toml:"path,omitempty"`
		Values map[string]any `toml:"value,omitempty"`

		Layout string    `toml:"layout,omitempty"`
		Ratios []float64 `toml:"ratios,omitempty"`

		Document   int             `toml:"document,omitempty"`
		DocHistory []int           `toml:"document-history,omitempty"`
		DocOffs    []sessDocOffset `toml:"document-offset,omitempty"`
		Mode       string          `toml:"mode,omitempty"`

		Anchor   int `toml:"anchor,omitempty"`
		HorzOff  int `toml:"horizontal-offset,omitempty"`
		VertOff  int `toml:"vertical-offset,omitempty"`
		FocusSeq int `toml:"focus-seq,omitempty"`

		Selection sessSelect `toml:"selection"`
		JumpHead  int        `toml:"jump-head,omitempty"`
		Jumps     []sessJump `toml:"jump,omitempty"`

		Children []sessNode `toml:"child"`
		History  []sessNode `toml:"history,omitempty"`
	}

	// sessDocOffset is a pane's remembered scroll position for a document
	// it is not currently displaying
	sessDocOffset struct {
		Document         int `toml:"document"`
		Anchor           int `toml:"anchor,omitempty"`
		HorizontalOffset int `toml:"horizontal-offset,omitempty"`
		VerticalOffset   int `toml:"vertical-offset,omitempty"`
	}

	sessSelect struct {
		Primary int         `toml:"primary"`
		Ranges  []sessRange `toml:"range"`
	}

	sessJump struct {
		Document  int        `toml:"document"`
		Anchor    int        `toml:"anchor"`
		Selection sessSelect `toml:"selection"`
	}

	sessRange struct {
		Anchor int `toml:"anchor"`
		Head   int `toml:"head"`
	}
)

const (
	SessionFile = "session.toml"

	SessionKindSplit    SessionKind = "split"
	SessionKindView     SessionKind = "view"
	SessionKindImage    SessionKind = "image"
	SessionKindTerminal SessionKind = "terminal"
	SessionKindBinary   SessionKind = "binary"

	sessionVersion = 1
)

var (
	ErrSessionEmpty       = errors.New("session is empty")
	ErrSessionInvalid     = errors.New("session is invalid")
	ErrSessionUnsupported = errors.New("session version unsupported")
)

// SaveSession stores restorable workspace state in path. Runtime option
// strings are supplied by the command registry that owns the option handlers
func (e *Editor) SaveSession(path string, opts map[string]string) error {
	docIndex := map[DocumentId]int{}
	base := sessionBase(path)
	e.panes.tree.compactFocusSeq()
	s := sessEditor{
		Version:   sessionVersion,
		Maximized: e.panes.tree.Maximized(),
	}
	keys := slices.Sorted(maps.Keys(opts))
	if len(keys) > 0 {
		s.Options = sessOptions{}
	}
	for _, key := range keys {
		s.Options[key] = opts[key]
	}
	regKeys := slices.Sorted(maps.Keys(e.registers.values))
	if len(regKeys) > 0 {
		s.Registers = sessRegisters{}
	}
	for _, k := range regKeys {
		if vals := e.registers.values.Read(k); len(vals) > 0 {
			s.Registers[string(k)] = vals
		}
	}
	for _, v := range e.AllViews() {
		d, ok := e.documents.byID[v.docID]
		if !ok {
			continue
		}
		d.rememberSelection(v.id)
		if _, ok := docIndex[d.ID()]; ok {
			continue
		}
		docIndex[d.ID()] = len(s.Documents) + 1
		s.Documents = append(s.Documents, e.sessionDocument(d, base))
	}
	for _, d := range e.AllDocuments() {
		if _, ok := docIndex[d.ID()]; ok {
			continue
		}
		docIndex[d.ID()] = len(s.Documents) + 1
		s.Documents = append(s.Documents, e.sessionDocument(d, base))
	}
	s.Layout = e.sessionNodeFor(e.panes.tree.root, docIndex, base)
	if len(s.Documents) == 0 && !layoutHasReopenablePane(&s.Layout) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return toml.NewEncoder(f).Encode(s)
}

// RestoreSession restores file-backed documents and view state from path. It
// returns any runtime option strings stored in the session for the caller to
// apply through the command registry
func (e *Editor) RestoreSession(path string) (map[string]string, bool, error) {
	s, ok, err := readSession(path)
	if !ok || err != nil {
		return nil, ok, err
	}
	reopenable := layoutHasReopenablePane(&s.Layout)
	base := sessionBase(path)

	docIDs := map[int]DocumentId{}
	docIndex := map[DocumentId]*Document{}
	for i, sd := range s.Documents {
		var absPath string
		if !sd.Scratch {
			if sd.Path == "" {
				continue
			}
			var err error
			absPath, err = filepath.Abs(sessionAbsPath(sessRef{
				base: base,
				path: sd.Path,
			}))
			if err != nil {
				return nil, false, err
			}
			if _, err := os.Stat(absPath); err != nil &&
				!errors.Is(err, os.ErrNotExist) {
				continue
			}
		}
		e.documents.nextID++
		id := e.documents.nextID
		var doc *Document
		if sd.Scratch {
			doc = newDocument(id, &e.opts)
			doc.content.text = core.NewRope(sd.Text)
			doc.content.version++
			if sd.Lang != "" {
				doc.SetLang(sd.Lang)
			}
			doc.views.lastSelection = clampSelection(
				sd.Selection.selection(), doc.content.text.LenChars(),
			)
		} else {
			doc = newPendingDocument(newPendingDocumentArgs{
				id:      id,
				absPath: absPath,
				lang:    sd.Lang,
				opts:    &e.opts,
			})
			doc.views.lastSelection = sd.Selection.selection()
		}
		docIndex[doc.ID()] = doc
		docIDs[i+1] = doc.ID()
	}
	if len(docIndex) == 0 && !reopenable {
		return nil, false, ErrSessionEmpty
	}

	t := newTree(e.panes.tree.area.Size)
	t.redraw = e.panes.tree.redraw
	t.nodes = map[Id]*treeNode{}
	t.nextID = 0
	rootID := t.allocID()
	t.root = rootID
	t.focus = rootID
	t.nodes[rootID] = &treeNode{
		parent: rootID,
		container: &treeContainer{
			layout: LayoutVertical,
		},
	}
	rs := sessionRestore{
		base:     base,
		docIDs:   docIDs,
		docIndex: docIndex,
	}
	if err := e.restoreSessionRoot(t, rootID, &s.Layout, &rs); err != nil {
		return nil, false, err
	}
	if t.IsEmpty() {
		return nil, false, ErrSessionEmpty
	}
	// read before compacting, while a zero sequence still means never focused
	if id, ok := t.leafWithLatestFocus(); ok {
		t.focus = id
	} else {
		t.focus = t.Traverse()[0].ID()
	}
	t.compactFocusSeq()
	t.advanceFocusSeq(t.focus)
	if s.Maximized && t.Count() > 1 {
		t.maximized = t.focus
	}
	t.recalculate()

	e.documents.byID = docIndex
	e.panes.tree = t
	e.documents.lastModifiedIDs = [2]DocumentId{}
	e.markDocAccessed()

	e.registers.values.ClearAll()
	for name, values := range s.Registers {
		runes := []rune(name)
		if len(runes) == 1 {
			e.registers.values.Write(runes[0], values)
		}
	}

	for _, doc := range e.VisibleDocuments() {
		doc.ensureLoaded()
		e.documentOpened(doc)
	}

	return s.Options, true, nil
}

// WorkspaceSessionFile returns the session file path for dir's workspace
func WorkspaceSessionFile(dir string) string {
	root, _ := loader.FindWorkspace(dir)
	return filepath.Join(root, loader.WorkspaceDirName, SessionFile)
}

func readSession(path string) (sessEditor, bool, error) {
	var s sessEditor
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return s, false, nil
	}
	if _, err := toml.DecodeFile(path, &s); err != nil {
		return s, false, err
	}
	if s.Version != sessionVersion {
		return s, false, ErrSessionUnsupported
	}
	if len(s.Documents) == 0 && !layoutHasReopenablePane(&s.Layout) {
		return s, false, ErrSessionEmpty
	}
	return s, true, nil
}

func sessionPath(at sessRef) string {
	if rel, err := filepath.Rel(at.base, at.path); err == nil {
		return rel
	}
	return at.path
}

func sessionAbsPath(at sessRef) string {
	if filepath.IsAbs(at.path) {
		return at.path
	}
	return filepath.Join(at.base, at.path)
}

func sessionBase(path string) string {
	dir := filepath.Dir(path)
	if filepath.Base(dir) == loader.WorkspaceDirName {
		return filepath.Dir(dir)
	}
	return dir
}

func layoutHasReopenablePane(n *sessNode) bool {
	switch n.Kind {
	case SessionKindImage, SessionKindTerminal, SessionKindBinary:
		return true
	case SessionKindSplit:
		for i := range n.Children {
			if layoutHasReopenablePane(&n.Children[i]) {
				return true
			}
		}
	}
	return false
}
