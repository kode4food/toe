package files

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type (
	BufferPickerOptions struct {
		StartPosition PickerStartPosition `toml:"start-position"`
	}

	PickerStartPosition string

	bufferPickerSource struct {
		ui.PickerBase
	}
)

const (
	PickerStartTop      PickerStartPosition = "top"
	PickerStartPrevious PickerStartPosition = "previous"
)

const (
	bufferPickerModifiedIcon      = "\uf448" //  - pencil icon
	bufferPickerModifiedIconAscii = "*"
)

var ErrInvalidPickerStart = errors.New("invalid picker start position")

func NewBufferPicker(e *view.Editor, opts BufferPickerOptions) *ui.Picker {
	p := ui.NewPicker(e, &bufferPickerSource{
		PickerBase: ui.NewPickerBase(
			"open-buffer", []string{"", ""}, 1, []int{0, 1},
		),
	})
	if opts.StartPosition == PickerStartPrevious && p.MatchCount() > 1 {
		p.SelectIndex(1)
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
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	docs := e.AllDocuments()
	id := view.InvalidDocumentId
	if doc, _ := e.FocusedDocument(); doc != nil {
		id = doc.ID()
	}
	slices.SortStableFunc(docs, func(a, b *view.Document) int {
		ra := bufferPickerRank(a, id)
		rb := bufferPickerRank(b, id)
		if c := cmp.Compare(ra, rb); c != 0 {
			return c
		}
		if c := cmp.Compare(b.AccessedAt(), a.AccessedAt()); c != 0 {
			return c
		}
		return cmp.Compare(a.ID(), b.ID())
	})

	modifiedIcon := bufferPickerModifiedIcon
	if !e.Options().NerdFonts {
		modifiedIcon = bufferPickerModifiedIconAscii
	}
	items := make([]ui.PickerItem, 0, len(docs))
	for _, doc := range docs {
		flags := ""
		if doc.Modified() {
			flags = modifiedIcon
		}
		name := doc.RelativeName(e.Cwd())
		items = append(items, ui.PickerItem{
			Display: name,
			Columns: []string{flags, name},
			SortKey: name,
			Location: ui.PickerLocation{
				Target: ui.PickerTarget{ID: doc.ID()},
			},
		})
	}
	return items, nil, func() {}
}

func (b *bufferPickerSource) Accept(
	e *view.Editor, item ui.PickerItem, action ui.PickerAcceptAction,
) {
	id := item.Location.Target.ID
	if id == view.InvalidDocumentId {
		return
	}
	v, ok := ui.AcceptDocumentID(e, id, action)
	if !ok {
		return
	}
	if doc, ok := e.Document(v.DocID()); ok {
		ui.AlignAcceptedView(e, v, doc)
	}
}

func bufferPickerRank(doc *view.Document, focusedID view.DocumentId) int {
	switch {
	case doc.ID() == focusedID:
		return 0
	case doc.Modified():
		return 1
	default:
		return 2
	}
}
