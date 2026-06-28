package ui

import (
	"fmt"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type jumplistPickerSource struct {
	pickerMeta
}

const (
	runeTruncEllipsis       = "\u2026" // '…' - horizontal ellipsis
	jumplistContentsMaxRune = 80
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
			Columns: []string{
				fmt.Sprintf("%d", entry.DocID), display, "",
				jumplistContents(text, entry.Selection),
			},
			Location: PickerLocation{
				Target: PickerTarget{ID: entry.DocID},
				Lines:  lines,
			},
			Payload: entry,
		})
	}
	return items, nil, func() {}
}

func (j *jumplistPickerSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, j.Columns(), j.Primary())
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

func jumplistContents(text core.Rope, sel core.Selection) string {
	var parts []string
	for _, r := range sel.Ranges() {
		frag, err := r.Fragment(text)
		if err != nil {
			continue
		}
		parts = append(parts, frag)
	}
	s := strings.Join(parts, " ")
	s = strings.TrimRight(s, "\r\n")
	runes := []rune(s)
	if len(runes) > jumplistContentsMaxRune {
		return string(runes[:jumplistContentsMaxRune]) + runeTruncEllipsis
	}
	return s
}
