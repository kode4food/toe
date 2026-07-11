package ui

import (
	"fmt"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type jumplistPickerSource struct {
	pickerMeta
}

func (j *jumplistPickerSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	v, ok := e.FocusedView()
	if !ok {
		return nil, nil, func() {}
	}
	jumps := v.Jumps()
	items := make([]PickerItem, 0, len(jumps))
	for i := len(jumps) - 1; i >= 0; i-- {
		entry := jumps[i]
		doc, ok := e.Document(entry.DocID)
		if !ok {
			continue
		}
		name := doc.RelativeName(e.Cwd())
		text := doc.Text()
		line, lines := jumpLineRange(text, entry.Selection)
		display := fmt.Sprintf("%s:%d", name, line+1)
		items = append(items, PickerItem{
			Display: display,
			Columns: []string{display},
			Location: PickerLocation{
				Target: PickerTarget{ID: entry.DocID},
				Lines:  lines,
			},
			Payload: entry,
		})
	}
	return items, nil, func() {}
}

func (j *jumplistPickerSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	entry, ok := item.Payload.(view.JumpEntry)
	if !ok {
		return
	}
	v, ok := acceptDocumentID(e, entry.DocID, action)
	if !ok {
		return
	}
	if doc, ok := e.Document(v.DocID()); ok {
		doc.SetSelectionFor(v.ID(), jumpSelection(entry))
		alignAcceptedView(e, v, doc)
	}
}

// JumplistPicker opens a picker listing the jump history for the focused view
func JumplistPicker(e *view.Editor) *Picker {
	return NewPicker(e, &jumplistPickerSource{
		pickerMeta: pickerMeta{
			title:       "Jumplist",
			columns:     []string{"path"},
			matchColumn: 0,
			proportions: []int{1},
		},
	})
}

func jumpSelection(j view.JumpEntry) core.Selection {
	return j.Selection
}

func jumpLineRange(text core.Rope, sel core.Selection) (int, *PickerLineRange) {
	line, err := sel.Primary().CursorLine(text)
	if err != nil {
		return 0, nil
	}
	return line, &PickerLineRange{From: line, To: line}
}
