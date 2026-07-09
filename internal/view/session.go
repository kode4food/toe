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

	sessionRestore struct {
		docs      map[int]DocumentId
		documents map[DocumentId]*Document
		focus     Id
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
			if sd.Path == "" {
				e.nextDocID--
				continue
			}
			absPath, err := filepath.Abs(sessionAbsPath(base, sd.Path))
			if err != nil {
				return nil, false, err
			}
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
	e.prevDocID = InvalidDocumentId
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

func (e *Editor) sessionDocument(d *Document, base string) sessionDocument {
	if d.Path() == "" {
		return sessionDocument{
			Scratch:   true,
			Text:      d.Text().String(),
			Lang:      d.Lang(),
			Selection: sessionSelection(d.Selection()),
		}
	}
	return sessionDocument{
		Path:      sessionPath(base, d.Path()),
		Lang:      d.Lang(),
		Selection: sessionSelection(d.Selection()),
	}
}

func (e *Editor) sessionNodeFor(
	id Id, docIndex map[DocumentId]int,
) sessionNode {
	n := e.tree.nodes[id]
	if n.view != nil {
		return e.sessionViewNode(n.view, docIndex)
	}
	c := n.container
	out := sessionNode{
		Kind:     sessionKindSplit,
		Layout:   sessionLayoutName(c.layout),
		Ratios:   c.ratios,
		Children: make([]sessionNode, 0, len(c.children)),
	}
	for _, child := range c.children {
		out.Children = append(out.Children, e.sessionNodeFor(child, docIndex))
	}
	return out
}

func (e *Editor) sessionViewNode(
	v *View, docIndex map[DocumentId]int,
) sessionNode {
	doc, ok := e.docs[v.docID]
	if !ok {
		return sessionNode{Kind: sessionKindView}
	}
	entries := v.jumps.Entries()
	savedHead := v.jumps.Head()
	jumps := make([]sessionJump, 0, len(entries))
	newHead := 0
	for i, j := range entries {
		idx, ok := docIndex[j.DocID]
		if !ok {
			continue
		}
		if i < savedHead {
			newHead++
		}
		jumps = append(jumps, sessionJump{
			Document:  idx,
			Anchor:    j.Anchor,
			Selection: sessionSelection(j.Selection),
		})
	}
	return sessionNode{
		Kind:             sessionKindView,
		Document:         docIndex[doc.ID()],
		Mode:             v.Mode().String(),
		Anchor:           v.offset.Anchor,
		HorizontalOffset: v.offset.HorizontalOffset,
		VerticalOffset:   v.offset.VerticalOffset,
		FreeScroll:       v.freeScroll,
		Focused:          e.tree.focus == v.id,
		Selection:        sessionSelection(doc.SelectionFor(v.id)),
		JumpHead:         newHead,
		Jumps:            jumps,
	}
}

func (e *Editor) restoreSessionRoot(
	t *Tree, root Id, sn sessionNode, rs *sessionRestore,
) error {
	if sn.Kind == sessionKindView {
		id, err := e.restoreSessionNode(t, root, sn, rs)
		if err != nil {
			return err
		}
		t.nodes[root].container.children = []Id{id}
		return nil
	}
	if sn.Kind != sessionKindSplit {
		return ErrSessionInvalid
	}
	c := t.nodes[root].container
	c.layout = sessionLayout(sn.Layout)
	c.ratios = sn.Ratios
	for _, child := range sn.Children {
		id, err := e.restoreSessionNode(t, root, child, rs)
		if err != nil {
			return err
		}
		c.children = append(c.children, id)
	}
	return nil
}

func (e *Editor) restoreSessionNode(
	t *Tree, parent Id, sn sessionNode, rs *sessionRestore,
) (Id, error) {
	switch sn.Kind {
	case sessionKindSplit:
		id := t.allocID()
		t.nodes[id] = &treeNode{
			parent: parent,
			container: &treeContainer{
				layout: sessionLayout(sn.Layout),
				ratios: sn.Ratios,
			},
		}
		for _, child := range sn.Children {
			childID, err := e.restoreSessionNode(t, id, child, rs)
			if err != nil {
				return 0, err
			}
			c := t.nodes[id].container
			c.children = append(c.children, childID)
		}
		return id, nil
	case sessionKindView:
		docID, ok := rs.docs[sn.Document]
		if !ok {
			return 0, ErrSessionInvalid
		}
		id := t.allocID()
		return e.restoreSessionView(restoreSessionViewArgs{
			tree:    t,
			parent:  parent,
			id:      id,
			docID:   docID,
			session: sn,
			restore: rs,
		}), nil
	}
	return 0, ErrSessionInvalid
}

type restoreSessionViewArgs struct {
	tree    *Tree
	parent  Id
	id      Id
	docID   DocumentId
	session sessionNode
	restore *sessionRestore
}

func (e *Editor) restoreSessionView(args restoreSessionViewArgs) Id {
	v := &View{
		id:         args.id,
		docID:      args.docID,
		mode:       sessionMode(args.session.Mode),
		offset:     sessionPosition(args.session),
		freeScroll: args.session.FreeScroll,
	}
	entries := make([]JumpEntry, 0, len(args.session.Jumps))
	for _, j := range args.session.Jumps {
		jDocID, ok := args.restore.docs[j.Document]
		if !ok {
			continue
		}
		entries = append(entries, JumpEntry{
			DocID:     jDocID,
			Anchor:    j.Anchor,
			Selection: j.Selection.selection(),
		})
	}
	head := args.session.JumpHead
	if head == 0 || head > len(entries) {
		head = len(entries)
	}
	v.jumps.Restore(entries, head)
	args.tree.nodes[args.id] = &treeNode{parent: args.parent, view: v}
	if doc, ok := args.restore.documents[args.docID]; ok {
		sel := args.session.Selection.selection()
		doc.SetSelectionFor(args.id, sel)
		if v.freeScroll {
			v.BeginFreeScroll(doc.Revision(), sel)
		}
	}
	if args.session.Focused {
		args.restore.focus = args.id
	}
	return args.id
}

func sessionSelection(sel core.Selection) sessionSelect {
	ranges := sel.Ranges()
	out := sessionSelect{
		Primary: sel.PrimaryIndex(),
		Ranges:  make([]sessionRange, 0, len(ranges)),
	}
	for _, r := range ranges {
		out.Ranges = append(out.Ranges, sessionRange{
			Anchor: r.Anchor,
			Head:   r.Head,
		})
	}
	return out
}

func (s sessionSelect) selection() core.Selection {
	if len(s.Ranges) == 0 {
		return core.PointSelection(0)
	}
	ranges := make([]core.Range, 0, len(s.Ranges))
	for _, r := range s.Ranges {
		ranges = append(ranges, core.NewRange(r.Anchor, r.Head))
	}
	sel, err := core.NewSelection(ranges, s.Primary)
	if err != nil {
		return core.PointSelection(0)
	}
	return sel
}

func sessionLayout(name string) Layout {
	if name == "horizontal" {
		return LayoutHorizontal
	}
	return LayoutVertical
}

func sessionLayoutName(l Layout) string {
	if l == LayoutHorizontal {
		return "horizontal"
	}
	return "vertical"
}

func sessionMode(name string) Mode {
	switch name {
	case "INS":
		return ModeInsert
	case "SEL":
		return ModeSelect
	default:
		return ModeNormal
	}
}

func sessionPosition(sn sessionNode) Position {
	return Position{
		Anchor:           sn.Anchor,
		HorizontalOffset: sn.HorizontalOffset,
		VerticalOffset:   sn.VerticalOffset,
	}
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
