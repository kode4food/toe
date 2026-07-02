package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestDocumentInlayHints(t *testing.T) {
	e := editorWithText(t, "fn main() {}\n")
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	vid := v.ID()

	hints := []view.InlayHint{
		{
			Pos:          2,
			Label:        ": ()",
			Kind:         "type",
			PaddingLeft:  true,
			PaddingRight: true,
		},
	}
	doc.SetInlayHints(vid, hints)
	assert.Equal(t, hints, doc.InlayHints(vid))

	doc.ClearInlayHints(vid)
	assert.Empty(t, doc.InlayHints(vid))

	doc.SetInlayHints(vid, hints)
	doc.SetInlayHints(vid, nil)
	assert.Empty(t, doc.InlayHints(vid))
}

func TestDocumentInlayHintsAll(t *testing.T) {
	e := editorWithText(t, "fn main() {}\n")
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	other := view.Id(999)

	hints := []view.InlayHint{{Pos: 2, Label: ": ()"}}
	doc.SetInlayHints(v.ID(), hints)
	doc.SetInlayHints(other, hints)

	doc.ClearAllInlayHints()

	assert.Empty(t, doc.InlayHints(v.ID()))
	assert.Empty(t, doc.InlayHints(other))
}
