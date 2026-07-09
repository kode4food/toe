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
			columns: []string{"flags", "path"},
			primary: 1,
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
		items = append(items, PickerItem{
			Display: name,
			Columns: []string{flags, name},
			SortKey: name,
			Location: PickerLocation{
				Target: PickerTarget{ID: doc.ID()},
			},
		})
	}
	return items, nil, func() {}
}

func (b *bufferPickerSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	id := item.Location.Target.ID
	if id == view.InvalidDocumentId {
		return
	}
	v, ok := acceptDocumentID(e, id, action)
	if !ok {
		return
	}
	if doc, ok := e.Document(v.DocID()); ok {
		alignAcceptedView(e, v, doc)
	}
}
