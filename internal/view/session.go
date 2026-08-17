package view

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"

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
		Version   int            `json:"version"`
		Options   sessOptions    `json:"options,omitempty"`
		Registers sessRegisters  `json:"registers,omitempty"`
		Maximized bool           `json:"maximized,omitempty"`
		Documents []sessDocument `json:"documents"`
		Layout    sessNode       `json:"layout"`
	}

	sessOptions map[string]string

	sessRegisters map[string][]string

	sessDocument struct {
		Path      string     `json:"path,omitempty"`
		Scratch   bool       `json:"scratch,omitempty"`
		Text      string     `json:"text,omitempty"`
		Lang      string     `json:"language,omitempty"`
		Selection sessSelect `json:"selection"`
	}

	sessNode struct {
		Kind   SessionKind    `json:"kind"`
		Path   string         `json:"path,omitempty"`
		Values map[string]any `json:"values,omitempty"`

		Layout string    `json:"layout,omitempty"`
		Ratios []float64 `json:"ratios,omitempty"`

		Document   int             `json:"document,omitempty"`
		DocHistory []int           `json:"document-history,omitempty"`
		DocOffs    []sessDocOffset `json:"document-offsets,omitempty"`
		Mode       string          `json:"mode,omitempty"`

		Anchor   int `json:"anchor,omitempty"`
		HorzOff  int `json:"horizontal-offset,omitempty"`
		VertOff  int `json:"vertical-offset,omitempty"`
		FocusSeq int `json:"focus-seq,omitempty"`

		Selection sessSelect `json:"selection"`
		JumpHead  int        `json:"jump-head,omitempty"`
		Jumps     []sessJump `json:"jumps,omitempty"`

		Children []sessNode `json:"children"`
		History  []sessNode `json:"history,omitempty"`
	}

	sessDocOffset struct {
		Document int `json:"document"`
		Anchor   int `json:"anchor,omitempty"`
		HorzOff  int `json:"horizontal-offset,omitempty"`
		VertOff  int `json:"vertical-offset,omitempty"`
	}

	sessSelect struct {
		Primary int         `json:"primary"`
		Ranges  []sessRange `json:"ranges"`
	}

	sessJump struct {
		Document  int        `json:"document"`
		Anchor    int        `json:"anchor"`
		Selection sessSelect `json:"selection"`
	}

	sessRange struct {
		Anchor int `json:"anchor"`
		Head   int `json:"head"`
	}
)

const (
	SessionFile = "session.json"

	SessionKindSplit    SessionKind = "split"
	SessionKindView     SessionKind = "view"
	SessionKindImage    SessionKind = "image"
	SessionKindTerminal SessionKind = "terminal"
	SessionKindBinary   SessionKind = "binary"
	SessionKindMessages SessionKind = "messages"

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
		if d.Type() == DocTypeLog {
			continue
		}
		docIndex[d.ID()] = len(s.Documents) + 1
		s.Documents = append(s.Documents, e.sessionDocument(d, base))
	}
	for _, d := range e.AllDocuments() {
		if _, ok := docIndex[d.ID()]; ok {
			continue
		}
		if d.Type() == DocTypeLog {
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
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
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
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, false, nil
	}
	if err != nil {
		return s, false, err
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(f)
	dec.UseNumber()
	if err := dec.Decode(&s); err != nil {
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
	case SessionKindImage, SessionKindTerminal, SessionKindBinary,
		SessionKindMessages:
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
