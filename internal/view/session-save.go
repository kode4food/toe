package view

import (
	"slices"

	"github.com/kode4food/toe/internal/core"
)

// SessionWriter is the opaque target a pane writes session state into
type SessionWriter struct {
	node     sessNode
	docIndex map[DocumentId]int
	base     string
}

// SaveSession stores this view's document state in w. A view onto a buffer
// the editor generates records only that it was showing it
func (v *View) SaveSession(w *SessionWriter) {
	doc, ok := v.editor.documents.byID[v.docID]
	if !ok {
		return
	}
	if doc.Type() == DocTypeLog {
		w.node = sessNode{Kind: SessionKindMessages}
		return
	}
	if _, ok := w.docIndex[doc.ID()]; !ok {
		return
	}
	entries := v.jumps.Entries()
	savedHead := v.jumps.Head()
	jumps := make([]sessJump, 0, len(entries))
	newHead := 0
	for i, j := range entries {
		idx, ok := w.docIndex[j.DocID]
		if !ok {
			continue
		}
		if i < savedHead {
			newHead++
		}
		jumps = append(jumps, sessJump{
			Document:  idx,
			Anchor:    j.Anchor,
			Selection: sessionSelection(j.Selection),
		})
	}
	w.node = sessNode{
		Kind:      SessionKindView,
		Document:  w.docIndex[doc.ID()],
		Mode:      v.Mode().String(),
		Anchor:    v.offset.Anchor,
		HorzOff:   v.offset.HorizontalOffset,
		VertOff:   v.offset.VerticalOffset,
		Selection: sessionSelection(doc.SelectionFor(v.id)),
		JumpHead:  newHead,
		Jumps:     jumps,
	}
	for _, did := range v.docHistory {
		if idx, ok := w.docIndex[did]; ok {
			w.node.DocHistory = append(w.node.DocHistory, idx)
		}
	}
	w.node.DocOffs = v.sessionDocOffsets(w.docIndex)
}

// SaveSlot stores a reopenable pane slot in the session
func (w *SessionWriter) SaveSlot(kind SessionKind, path string) {
	w.node = sessNode{Kind: kind}
	if path != "" {
		w.node.Path = sessionPath(sessRef{base: w.base, path: path})
	}
}

// SaveValue stores module-owned pane state
func (w *SessionWriter) SaveValue(key string, value any) {
	if w.node.Values == nil {
		w.node.Values = map[string]any{}
	}
	w.node.Values[key] = value
}

func (e *Editor) sessionDocument(d *Document, base string) sessDocument {
	if d.Path() == "" {
		return sessDocument{
			Scratch:   true,
			Text:      d.Text().String(),
			Lang:      d.Lang(),
			Selection: sessionSelection(d.Selection()),
		}
	}
	return sessDocument{
		Path:      sessionPath(sessRef{base: base, path: d.Path()}),
		Lang:      d.Lang(),
		Selection: sessionSelection(d.Selection()),
	}
}

func (e *Editor) sessionNodeFor(
	id Id, docIndex map[DocumentId]int, base string,
) sessNode {
	n := e.panes.tree.nodes[id]
	if n.pane != nil {
		w := &SessionWriter{docIndex: docIndex, base: base}
		n.pane.SaveSession(w)
		node := w.node
		node.FocusSeq = n.focusSeq
		for _, displaced := range n.history {
			hw := &SessionWriter{docIndex: docIndex, base: base}
			displaced.SaveSession(hw)
			node.History = append(node.History, hw.node)
		}
		return node
	}
	c := n.container
	out := sessNode{
		Kind:     SessionKindSplit,
		Layout:   sessionLayoutName(c.layout),
		Ratios:   c.ratios,
		FocusSeq: n.focusSeq,
		Children: make([]sessNode, 0, len(c.children)),
	}
	for _, child := range c.children {
		node := e.sessionNodeFor(child, docIndex, base)
		if node.Kind == "" {
			continue
		}
		out.Children = append(out.Children, node)
	}
	return out
}

func (v *View) sessionDocOffsets(docIndex map[DocumentId]int) []sessDocOffset {
	out := make([]sessDocOffset, 0, len(v.docOffsets))
	for did, p := range v.docOffsets {
		idx, ok := docIndex[did]
		if !ok || did == v.docID || p == (Position{}) {
			continue
		}
		out = append(out, sessDocOffset{
			Document: idx,
			Anchor:   p.Anchor,
			HorzOff:  p.HorizontalOffset,
			VertOff:  p.VerticalOffset,
		})
	}
	slices.SortFunc(out, func(a, b sessDocOffset) int {
		return a.Document - b.Document
	})
	return out
}

func sessionSelection(sel core.Selection) sessSelect {
	ranges := sel.Ranges()
	out := sessSelect{
		Primary: sel.PrimaryIndex(),
		Ranges:  make([]sessRange, 0, len(ranges)),
	}
	for _, r := range ranges {
		out.Ranges = append(out.Ranges, sessRange{
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
