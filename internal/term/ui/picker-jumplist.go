package ui

import (
	"fmt"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type (
	jumplistPickerSource struct {
		pickerMeta
	}

	jumplistPayload struct {
		docID  view.DocumentId
		anchor int
	}
)

// JumplistPicker opens a picker listing the jump history for the focused view
func JumplistPicker(e *view.Editor) *Picker {
	return NewPicker(e, &jumplistPickerSource{
		pickerMeta: pickerMeta{
			title:   "Jumplist",
			columns: []string{"id", "path", "flags", "contents"},
			primary: 1,
		},
	})
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
		j := jumps[i]
		doc, ok := e.Document(j.DocID)
		if !ok {
			continue
		}
		name := doc.RelativeName(e.Cwd())
		text := doc.Text()
		line := 0
		if l, err := text.CharToLine(j.Anchor); err == nil {
			line = l + 1
		}
		display := fmt.Sprintf("%s:%d", name, line)
		did := j.DocID
		anchor := j.Anchor
		items = append(items, PickerItem{
			Display: display,
			Columns: []string{
				fmt.Sprintf("%d", did), display, "",
				jumplistContents(text, anchor),
			},
			Location: PickerLocation{
				Target: PickerTarget{ID: did},
				Lines:  &PickerLineRange{From: line - 1, To: line - 1},
			},
			Payload: jumplistPayload{docID: did, anchor: anchor},
		})
	}
	return items, nil, func() {}
}

func (j *jumplistPickerSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, j.Columns(), j.Primary())
}

func (j *jumplistPickerSource) Accept(e *view.Editor, item PickerItem) {
	payload, ok := item.Payload.(jumplistPayload)
	if !ok {
		return
	}
	for _, v := range e.AllViews() {
		if v.DocID() == payload.docID {
			e.FocusView(v.ID())
			if doc, ok := e.FocusedDocument(); ok {
				doc.SetSelectionFor(v.ID(), core.PointSelection(payload.anchor))
			}
			return
		}
	}
}

func jumplistContents(text core.Rope, anchor int) string {
	lineIdx := 0
	if l, err := text.CharToLine(anchor); err == nil {
		lineIdx = l
	}
	line, err := text.Line(lineIdx)
	if err != nil {
		return ""
	}
	s := strings.TrimRight(line.String(), "\r\n")
	runes := []rune(s)
	if len(runes) > 80 {
		return string(runes[:80]) + "…"
	}
	return s
}
