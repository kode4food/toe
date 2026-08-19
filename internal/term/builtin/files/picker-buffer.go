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
	bufferPickerModifiedIcon  = "\uf448" // '' - pencil icon
	bufferPickerModifiedAscii = "*"
)

var ErrInvalidPickerStart = errors.New("invalid picker start position")

// NewBufferPicker returns a picker over the open buffers
func NewBufferPicker(e *view.Editor, opts BufferPickerOptions) *ui.Picker {
	p := ui.NewPicker(e, &bufferPickerSource{
		PickerBase: ui.PickerBase{
			Ident:       "open-buffer",
			Label:       "Buffers",
			Cols:        []string{"", "", ""},
			MatchCol:    2,
			Proportions: []int{0, 0, 1},
		},
	})
	if opts.StartPosition == PickerStartPrevious && p.MatchCount() > 1 {
		p.SelectIndex(1)
	}
	return p
}

// UnmarshalText parses which entry the picker starts on
func (p *PickerStartPosition) UnmarshalText(text []byte) error {
	switch PickerStartPosition(text) {
	case PickerStartTop, PickerStartPrevious:
		*p = PickerStartPosition(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidPickerStart, text)
	}
	return nil
}

// Load lists the open buffers, most recently used first
func (b *bufferPickerSource) Load(e *view.Editor) ui.PickerLoad {
	docs := e.AllDocuments()
	id := view.InvalidDocumentId
	if doc := e.FocusedDocument(); doc != nil {
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
		modifiedIcon = bufferPickerModifiedAscii
	}
	items := make([]*ui.PickerItem, 0, len(docs))
	var slab ui.PickerItemSlab
	for _, doc := range docs {
		flags := ""
		if doc.Modified() {
			flags = modifiedIcon
		}
		name := doc.RelativeName(e.Cwd())
		lbl, sec := ui.PickerNamePath(name)
		items = append(items, slab.Add(ui.PickerItem{
			Columns: []string{flags, "", lbl},
			SortKey: name,
			SecFrom: sec,
			Location: ui.PickerLocation{
				Target: ui.PickerTarget{ID: doc.ID()},
			},
		}))
	}
	return ui.PickerLoad{Items: items, Stop: func() {}}
}

// Accept switches the focused pane to the chosen buffer
func (b *bufferPickerSource) Accept(
	e *view.Editor, item *ui.PickerItem, action ui.PickerAcceptAction,
) {
	id := item.Location.Target.ID
	if id == view.InvalidDocumentId {
		return
	}
	ui.GotoDocument(e, id, nil, action)
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
