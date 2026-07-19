package view

import "github.com/kode4food/toe/internal/core"

type sessionRestore struct {
	base      string
	docs      map[int]DocumentId
	documents map[DocumentId]*Document
	focus     Id
}

func (e *Editor) restoreSessionRoot(
	t *Tree, root Id, sn sessionNode, rs *sessionRestore,
) error {
	if sn.Kind == SessionKindView || sn.Kind == SessionKindImage ||
		sn.Kind == SessionKindTerminal {
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
	case SessionKindSplit:
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
	case SessionKindView:
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
	default:
		pane, err := e.restorePane(restorePaneArgs{
			kind: sn.Kind,
			session: &PaneSession{
				path:   sessionAbsPath(rs.base, sn.Path),
				values: sn.Values,
			},
		})
		if err != nil {
			return 0, err
		}
		id := t.allocID()
		pane.SetID(id)
		t.nodes[id] = &treeNode{parent: parent, pane: pane}
		if sn.Focused {
			rs.focus = id
		}
		return id, nil
	}
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
		editor:     e,
		docID:      args.docID,
		mode:       ParseMode(args.session.Mode),
		offset:     sessionPosition(args.session),
		freeScroll: args.session.FreeScroll,
	}
	for _, idx := range args.session.DocumentHistory {
		if did, ok := args.restore.docs[idx]; ok {
			v.docHistory = append(v.docHistory, did)
		}
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
	args.tree.nodes[args.id] = &treeNode{parent: args.parent, pane: v}
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

func sessionPosition(sn sessionNode) Position {
	return Position{
		Anchor:           sn.Anchor,
		HorizontalOffset: sn.HorizontalOffset,
		VerticalOffset:   sn.VerticalOffset,
	}
}
