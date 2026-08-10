package view

import "github.com/kode4food/toe/internal/core"

type sessionRestore struct {
	base     string
	docIDs   map[int]DocumentId
	docIndex map[DocumentId]*Document
}

func (e *Editor) restoreSessionRoot(
	t *Tree, root Id, sn *sessNode, rs *sessionRestore,
) error {
	if sn.Kind == SessionKindView || sn.Kind == SessionKindImage ||
		sn.Kind == SessionKindTerminal || sn.Kind == SessionKindBinary {
		id, err := e.restoreSessionNode(t, root, sn, rs)
		if err != nil {
			return err
		}
		t.nodes[root].container.children = []Id{id}
		return nil
	}
	if sn.Kind != SessionKindSplit {
		return ErrSessionInvalid
	}
	c := t.nodes[root].container
	c.layout = sessionLayout(sn.Layout)
	c.ratios = sn.Ratios
	t.nodes[root].focusSeq = sn.FocusSeq
	for i := range sn.Children {
		id, err := e.restoreSessionNode(t, root, &sn.Children[i], rs)
		if err != nil {
			return err
		}
		c.children = append(c.children, id)
	}
	return nil
}

func (e *Editor) restoreSessionNode(
	t *Tree, parent Id, sn *sessNode, rs *sessionRestore,
) (Id, error) {
	id, err := e.restoreSessionKind(t, parent, sn, rs)
	if err != nil {
		return 0, err
	}
	t.nodes[id].focusSeq = sn.FocusSeq
	return id, nil
}

func (e *Editor) restoreSessionKind(
	t *Tree, parent Id, sn *sessNode, rs *sessionRestore,
) (Id, error) {
	switch sn.Kind {
	case SessionKindSplit:
		id := t.allocID()
		t.nodes[id] = &treeNode{
			parent: parent,
			container: &treeContainer{
				layout: sessionLayout(sn.Layout),
				ratios: sn.Ratios,
			},
		}
		for i := range sn.Children {
			childID, err := e.restoreSessionNode(t, id, &sn.Children[i], rs)
			if err != nil {
				return 0, err
			}
			c := t.nodes[id].container
			c.children = append(c.children, childID)
		}
		return id, nil
	case SessionKindView:
		docID, ok := rs.docIDs[sn.Document]
		if !ok {
			return 0, ErrSessionInvalid
		}
		id := t.allocID()
		return e.restoreSessionView(restoreSessionViewArgs{
			tree:    t,
			parent:  parent,
			viewID:  id,
			docID:   docID,
			session: sn,
			restore: rs,
		}), nil
	default:
		pane, err := e.restorePane(restorePaneArgs{
			kind: sn.Kind,
			session: &PaneSession{
				path: sessionAbsPath(sessRef{
					base: rs.base,
					path: sn.Path,
				}),
				values: sn.Values,
			},
		})
		if err != nil {
			return 0, err
		}
		id := t.allocID()
		t.attach(pane, id)
		node := &treeNode{parent: parent, pane: pane}
		for _, h := range sn.History {
			if hp := e.restoreDisplacedPane(t, id, &h, rs); hp != nil {
				node.history = append(node.history, hp)
			}
		}
		t.nodes[id] = node
		return id, nil
	}
}

type restoreSessionViewArgs struct {
	tree    *Tree
	parent  Id
	viewID  Id
	docID   DocumentId
	session *sessNode
	restore *sessionRestore
}

func (e *Editor) restoreSessionView(args restoreSessionViewArgs) Id {
	v := e.newSessionView(args)
	args.tree.nodes[args.viewID] = &treeNode{parent: args.parent, pane: v}
	return args.viewID
}

// newSessionView rebuilds a view from its session node without attaching it to
// the tree or touching focus, so it can serve as a detached restore pane
func (e *Editor) newSessionView(args restoreSessionViewArgs) *View {
	v := &View{
		id:     args.viewID,
		editor: e,
		docID:  args.docID,
		mode:   ParseMode(args.session.Mode),
		offset: sessionPosition(args.session),
	}
	for _, idx := range args.session.DocHistory {
		if did, ok := args.restore.docIDs[idx]; ok {
			v.docHistory = append(v.docHistory, did)
		}
	}
	for _, so := range args.session.DocOffs {
		did, ok := args.restore.docIDs[so.Document]
		if !ok {
			continue
		}
		if v.docOffsets == nil {
			v.docOffsets = map[DocumentId]Position{}
		}
		v.docOffsets[did] = Position{
			Anchor:           so.Anchor,
			HorizontalOffset: so.HorzOff,
			VerticalOffset:   so.VertOff,
		}
	}
	entries := make([]JumpEntry, 0, len(args.session.Jumps))
	for _, j := range args.session.Jumps {
		jDocID, ok := args.restore.docIDs[j.Document]
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
	if doc, ok := args.restore.docIndex[args.docID]; ok {
		doc.SetSelectionFor(args.viewID, args.session.Selection.selection())
	}
	return v
}

// restoreDisplacedPane rebuilds a stashed pane detached from the tree, or nil
// when its document or kind cannot be resolved
func (e *Editor) restoreDisplacedPane(
	t *Tree, parent Id, sn *sessNode, rs *sessionRestore,
) Pane {
	if sn == nil {
		return nil
	}
	if sn.Kind == SessionKindView {
		docID, ok := rs.docIDs[sn.Document]
		if !ok {
			return nil
		}
		return e.newSessionView(restoreSessionViewArgs{
			tree:    t,
			parent:  parent,
			viewID:  t.allocID(),
			docID:   docID,
			session: sn,
			restore: rs,
		})
	}
	pane, err := e.restorePane(restorePaneArgs{
		kind: sn.Kind,
		session: &PaneSession{
			path:   sessionAbsPath(sessRef{base: rs.base, path: sn.Path}),
			values: sn.Values,
		},
	})
	if err != nil {
		return nil
	}
	t.attach(pane, t.allocID())
	stashPane(pane)
	return pane
}

func (s sessSelect) selection() core.Selection {
	if len(s.Ranges) == 0 {
		return core.PointSelection(0)
	}
	ranges := make([]core.Range, 0, len(s.Ranges))
	for _, r := range s.Ranges {
		ranges = append(ranges, core.Range{Anchor: r.Anchor, Head: r.Head})
	}
	if sel, err := core.NewSelection(ranges, s.Primary); err == nil {
		return sel
	}
	return core.PointSelection(0)
}

func sessionLayout(name string) Layout {
	if name == "horizontal" {
		return LayoutHorizontal
	}
	return LayoutVertical
}

func sessionPosition(sn *sessNode) Position {
	return Position{
		Anchor:           sn.Anchor,
		HorizontalOffset: sn.HorzOff,
		VerticalOffset:   sn.VertOff,
	}
}
