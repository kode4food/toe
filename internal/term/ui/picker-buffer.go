package ui

import (
	"fmt"

	"github.com/kode4food/toe/internal/view"
)

type bufferPickerSource struct {
	pickerMeta
}

// BufferPicker opens a picker listing all open documents
func BufferPicker(e *view.Editor) *Picker {
	return NewPicker(e, &bufferPickerSource{
		pickerMeta: pickerMeta{
			title:   "Open buffer",
			columns: []string{"id", "flags", "path"},
			primary: 2,
		},
	})
}

func (b *bufferPickerSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	docs := e.AllDocuments()
	focusedDoc, _ := e.FocusedDocument()
	views := e.AllViews()
	focusedView, _ := e.FocusedView()

	items := make([]PickerItem, 0, len(docs))
	for _, doc := range docs {
		flags := ""
		if focusedDoc != nil && doc.ID() == focusedDoc.ID() {
			flags += "*"
		}
		if doc.Modified() {
			flags += "+"
		}
		name := doc.RelativeName(e.Cwd())
		id := doc.ID()
		lines := bufferPickerLines(doc, views, focusedView)
		items = append(items, PickerItem{
			Display: name,
			Columns: []string{fmt.Sprintf("%d", id), flags, name},
			SortKey: name,
			Location: PickerLocation{
				Target: PickerTarget{ID: id},
				Lines:  lines,
			},
			Payload: id,
		})
	}
	return items, nil, func() {}
}

func (b *bufferPickerSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, b.Columns(), b.Primary())
}

func (b *bufferPickerSource) Accept(e *view.Editor, item PickerItem) {
	id, ok := item.Payload.(view.DocumentId)
	if !ok {
		return
	}
	for _, v := range e.AllViews() {
		if v.DocID() == id {
			e.FocusView(v.ID())
			return
		}
	}
	e.SwitchBuffer(id)
}

func bufferPickerLines(
	doc *view.Document, views []*view.View, focused *view.View,
) *PickerLineRange {
	for _, v := range views {
		if v.DocID() == doc.ID() {
			return selectionLineRange(doc, v.ID())
		}
	}
	if focused == nil {
		return nil
	}
	return selectionLineRange(doc, focused.ID())
}

func selectionLineRange(doc *view.Document, vid view.Id) *PickerLineRange {
	sel := doc.SelectionFor(vid)
	cursor := sel.Primary().Cursor(doc.Text())
	if l, err := doc.Text().CharToLine(cursor); err == nil {
		return &PickerLineRange{From: l, To: l}
	}
	return nil
}
