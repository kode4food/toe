package view

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
)

type (
	editorSession struct {
		Version   int               `toml:"version"`
		Options   sessionOptions    `toml:"option,omitempty"`
		Registers sessionRegisters  `toml:"register,omitempty"`
		Documents []sessionDocument `toml:"document"`
		Layout    sessionNode       `toml:"layout"`
	}

	sessionOptions map[string]string

	sessionRegisters map[string][]string

	sessionDocument struct {
		Path      string        `toml:"path,omitempty"`
		Scratch   bool          `toml:"scratch,omitempty"`
		Text      string        `toml:"text,omitempty"`
		Lang      string        `toml:"language,omitempty"`
		Selection sessionSelect `toml:"selection"`
	}

	sessionNode struct {
		Kind             string        `toml:"kind"`
		Layout           string        `toml:"layout,omitempty"`
		Ratios           []float64     `toml:"ratios,omitempty"`
		Document         int           `toml:"document,omitempty"`
		DocumentHistory  []int         `toml:"document-history,omitempty"`
		Mode             string        `toml:"mode,omitempty"`
		Anchor           int           `toml:"anchor,omitempty"`
		HorizontalOffset int           `toml:"horizontal-offset,omitempty"`
		VerticalOffset   int           `toml:"vertical-offset,omitempty"`
		FreeScroll       bool          `toml:"free-scroll,omitempty"`
		Focused          bool          `toml:"focused,omitempty"`
		Selection        sessionSelect `toml:"selection"`
		JumpHead         int           `toml:"jump-head,omitempty"`
		Jumps            []sessionJump `toml:"jump,omitempty"`
		Children         []sessionNode `toml:"child"`
	}

	sessionSelect struct {
		Primary int            `toml:"primary"`
		Ranges  []sessionRange `toml:"range"`
	}

	sessionJump struct {
		Document  int           `toml:"document"`
		Anchor    int           `toml:"anchor"`
		Selection sessionSelect `toml:"selection"`
	}

	sessionRange struct {
		Anchor int `toml:"anchor"`
		Head   int `toml:"head"`
	}
)

const (
	sessionVersion = 1
	SessionFile    = "session.toml"

	sessionKindSplit = "split"
	sessionKindView  = "view"
)

var (
	ErrSessionEmpty         = errors.New("session is empty")
	ErrSessionInvalid       = errors.New("session is invalid")
	ErrSessionUnsupported   = errors.New("session version unsupported")
	ErrSessionUnknownOption = errors.New("session option unknown")
)

// SaveSession stores restorable workspace state in path. Runtime option
// strings are supplied by the command registry that owns the option handlers
func (e *Editor) SaveSession(path string, opts map[string]string) error {
	docIndex := map[DocumentId]int{}
	base := sessionBase(path)
	s := editorSession{
		Version: sessionVersion,
	}
	keys := make([]string, 0, len(opts))
	for key := range opts {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	if len(keys) > 0 {
		s.Options = sessionOptions{}
	}
	for _, key := range keys {
		s.Options[key] = opts[key]
	}
	regKeys := make([]rune, 0, len(e.registers))
	for k := range e.registers {
		regKeys = append(regKeys, k)
	}
	slices.Sort(regKeys)
	if len(regKeys) > 0 {
		s.Registers = sessionRegisters{}
	}
	for _, k := range regKeys {
		if vals := e.registers.Read(k); len(vals) > 0 {
			s.Registers[string(k)] = vals
		}
	}
	for _, v := range e.AllViews() {
		d, ok := e.docs[v.docID]
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
	if len(s.Documents) == 0 {
		return nil
	}
	s.Layout = e.sessionNodeFor(e.tree.root, docIndex)
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
	var s editorSession
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if _, err := toml.DecodeFile(path, &s); err != nil {
		return nil, false, err
	}
	if s.Version != sessionVersion {
		return nil, false, ErrSessionUnsupported
	}
	if len(s.Documents) == 0 {
		return nil, false, ErrSessionEmpty
	}
	base := sessionBase(path)

	docs := map[int]DocumentId{}
	nextDocs := map[DocumentId]*Document{}
	for i, sd := range s.Documents {
		var absPath string
		if !sd.Scratch {
			if sd.Path == "" {
				continue
			}
			var err error
			absPath, err = filepath.Abs(sessionAbsPath(base, sd.Path))
			if err != nil {
				return nil, false, err
			}
			if _, err := os.Stat(absPath); err != nil &&
				!errors.Is(err, os.ErrNotExist) {
				continue
			}
		}
		e.nextDocID++
		id := e.nextDocID
		var doc *Document
		if sd.Scratch {
			doc = newDocument(id, &e.opts)
			doc.buf.text = core.NewRope(sd.Text)
			doc.buf.version++
			if sd.Lang != "" {
				doc.SetLang(sd.Lang)
			}
			doc.buf.lastSel = clampSelection(
				sd.Selection.selection(), doc.buf.text.LenChars(),
			)
		} else {
			doc = newPendingDocument(id, absPath, sd.Lang, &e.opts)
			doc.buf.lastSel = sd.Selection.selection()
		}
		nextDocs[doc.ID()] = doc
		docs[i+1] = doc.ID()
	}
	if len(nextDocs) == 0 {
		return nil, false, ErrSessionEmpty
	}

	t := newTree(e.tree.area.Width, e.tree.area.Height)
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
	rs := sessionRestore{docs: docs, documents: nextDocs}
	if err := e.restoreSessionRoot(t, rootID, s.Layout, &rs); err != nil {
		return nil, false, err
	}
	if t.IsEmpty() {
		return nil, false, ErrSessionEmpty
	}
	if rs.focus != InvalidViewId {
		t.focus = rs.focus
	} else {
		t.focus = t.Traverse()[0].ID()
	}
	t.recalculate()

	e.docs = nextDocs
	e.tree = t
	e.lastModifiedDocIDs = [2]DocumentId{}
	e.markDocAccessed()

	e.registers.ClearAll()
	for name, values := range s.Registers {
		runes := []rune(name)
		if len(runes) == 1 {
			e.registers.Write(runes[0], values)
		}
	}

	for _, doc := range e.VisibleDocuments() {
		doc.ensureLoaded()
		e.documentOpened(doc)
	}

	return s.Options, true, nil
}

func WorkspaceSessionFile(dir string) string {
	root, _ := loader.FindWorkspace(dir)
	return filepath.Join(root, loader.WorkspaceDirName, SessionFile)
}

func sessionPath(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil {
		return rel
	}
	return path
}

func sessionAbsPath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}

func sessionBase(path string) string {
	dir := filepath.Dir(path)
	if filepath.Base(dir) == loader.WorkspaceDirName {
		return filepath.Dir(dir)
	}
	return dir
}
