package ui

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/toe/internal/view"
)

type (
	BufferPickerOptions struct {
		StartPosition PickerStartPosition `toml:"start-position"`
	}

	PickerStartPosition string

	bufferPickerSource struct {
		pickerMeta
	}
)

const (
	PickerStartTop      PickerStartPosition = "top"
	PickerStartPrevious PickerStartPosition = "previous"
)

var (
	ErrInvalidPickerStart = errors.New("invalid picker start position")
)

func NewBufferPicker(e *view.Editor, opts BufferPickerOptions) *Picker {
	p := NewPicker(e, &bufferPickerSource{
		pickerMeta: pickerMeta{
			title:   "Open buffer",
			columns: []string{"id", "flags", "path"},
			primary: 2,
		},
	})
	if opts.StartPosition == PickerStartPrevious && len(p.matched) > 1 {
		p.cursor = 1
	}
	return p
}

func (p *PickerStartPosition) UnmarshalText(text []byte) error {
	switch PickerStartPosition(text) {
	case PickerStartTop, PickerStartPrevious:
		*p = PickerStartPosition(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidPickerStart, text)
	}
	return nil
}

func (b *bufferPickerSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	docs := e.AllDocuments()
	slices.SortStableFunc(docs, func(a, b *view.Document) int {
		if c := cmp.Compare(b.AccessedAt(), a.AccessedAt()); c != 0 {
			return c
		}
		return cmp.Compare(a.ID(), b.ID())
	})
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
		})
	}
	return items, nil, func() {}
}

func (b *bufferPickerSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, b.Columns(), b.Primary())
}

func (b *bufferPickerSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	id := item.Location.Target.ID
	if id == view.InvalidDocumentId {
		return
	}
	acceptDocumentID(e, id, action)
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
