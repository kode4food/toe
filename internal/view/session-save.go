package view

import "github.com/kode4food/toe/internal/core"

// SessionWriter is the opaque target a pane writes session state into
type SessionWriter struct {
	node     sessionNode
	docIndex map[DocumentId]int
	base     string
	focused  bool
}

// SaveSession stores this view's document state in w
func (v *View) SaveSession(w *SessionWriter) {
	doc, ok := v.editor.docs[v.docID]
	if !ok {
		w.node = sessionNode{Kind: SessionKindView}
		return
	}
	entries := v.jumps.Entries()
	savedHead := v.jumps.Head()
	jumps := make([]sessionJump, 0, len(entries))
	newHead := 0
	for i, j := range entries {
		idx, ok := w.docIndex[j.DocID]
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
	w.node = sessionNode{
		Kind:             SessionKindView,
		Document:         w.docIndex[doc.ID()],
		Mode:             v.Mode().String(),
		Anchor:           v.offset.Anchor,
		HorizontalOffset: v.offset.HorizontalOffset,
		VerticalOffset:   v.offset.VerticalOffset,
		FreeScroll:       v.freeScroll,
		Focused:          w.focused,
		Selection:        sessionSelection(doc.SelectionFor(v.id)),
		JumpHead:         newHead,
		Jumps:            jumps,
	}
	for _, did := range v.docHistory {
		if idx, ok := w.docIndex[did]; ok {
			w.node.DocumentHistory = append(w.node.DocumentHistory, idx)
		}
	}
}

// SaveSlot stores a reopenable pane slot in the session
func (w *SessionWriter) SaveSlot(kind, path string) {
	w.node = sessionNode{Kind: kind, Focused: w.focused}
	if path != "" {
		w.node.Path = sessionPath(w.base, path)
	}
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
	id Id, docIndex map[DocumentId]int, base string,
) sessionNode {
	n := e.tree.nodes[id]
	if n.pane != nil {
		w := &SessionWriter{
			docIndex: docIndex,
			base:     base,
			focused:  e.tree.focus == id,
		}
		n.pane.SaveSession(w)
		return w.node
	}
	c := n.container
	out := sessionNode{
		Kind:     SessionKindSplit,
		Layout:   sessionLayoutName(c.layout),
		Ratios:   c.ratios,
		Children: make([]sessionNode, 0, len(c.children)),
	}
	for _, child := range c.children {
		out.Children = append(
			out.Children, e.sessionNodeFor(child, docIndex, base),
		)
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
