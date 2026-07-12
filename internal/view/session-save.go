package view

import "github.com/kode4food/toe/internal/core"

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
	id Id, docIndex map[DocumentId]int, s *editorSession,
) sessionNode {
	n := e.tree.nodes[id]
	if v, ok := n.pane.(*View); ok {
		return e.sessionViewNode(v, docIndex)
	}
	if n.pane != nil {
		// a non-View leaf's live state (e.g. a terminal's shell) isn't
		// serializable, but its slot is: mark it so restore can reopen one
		return sessionNode{
			Kind:    sessionKindTerminal,
			Focused: e.tree.focus == id,
		}
	}
	c := n.container
	out := sessionNode{
		Kind:     sessionKindSplit,
		Layout:   sessionLayoutName(c.layout),
		Ratios:   c.ratios,
		Children: make([]sessionNode, 0, len(c.children)),
	}
	for _, child := range c.children {
		out.Children = append(
			out.Children, e.sessionNodeFor(child, docIndex, s),
		)
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
	out := sessionNode{
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
	for _, did := range v.docHistory {
		if idx, ok := docIndex[did]; ok {
			out.DocumentHistory = append(out.DocumentHistory, idx)
		}
	}
	return out
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

func sessionLayoutName(l Layout) string {
	if l == LayoutHorizontal {
		return "horizontal"
	}
	return "vertical"
}
