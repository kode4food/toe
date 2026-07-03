package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

func TestDocumentColors(t *testing.T) {
	e := testutil.EditorWithText(t, "color: #123456\n")
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)

	colors := []view.DocumentColor{
		{From: 7, To: 14, Red: 0x12, Green: 0x34, Blue: 0x56},
	}
	doc.SetDocumentColors(colors)
	assert.Equal(t, colors, doc.DocumentColors())

	doc.ClearDocumentColors()
	assert.Empty(t, doc.DocumentColors())

	doc.SetDocumentColors(colors)
	doc.SetDocumentColors(nil)
	assert.Empty(t, doc.DocumentColors())
}

func TestDocumentLinks(t *testing.T) {
	e := testutil.EditorWithText(t, "hello world\n")
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)

	links := []view.DocumentLink{
		{From: 0, To: 5, Target: "/a"},
		{From: 6, To: 11, Target: "/b"},
	}
	doc.SetDocumentLinks(links)
	assert.Equal(t, links, doc.DocumentLinks())

	doc.ClearDocumentLinks()
	assert.Empty(t, doc.DocumentLinks())

	doc.SetDocumentLinks(links)
	doc.SetDocumentLinks(nil)
	assert.Empty(t, doc.DocumentLinks())
}

func TestDocumentHighlights(t *testing.T) {
	e := testutil.EditorWithText(t, "hello world\n")
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	vid := v.ID()

	highlights := []view.DocumentHighlight{
		{From: 0, To: 5},
		{From: 6, To: 11},
	}
	doc.SetDocumentHighlights(vid, highlights)
	assert.Equal(t, highlights, doc.DocumentHighlights(vid))

	doc.ClearDocumentHighlights(vid)
	assert.Empty(t, doc.DocumentHighlights(vid))

	doc.SetDocumentHighlights(vid, highlights)
	doc.SetDocumentHighlights(vid, nil)
	assert.Empty(t, doc.DocumentHighlights(vid))
}

func TestDocumentHighlightsAll(t *testing.T) {
	e := testutil.EditorWithText(t, "hello world\n")
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	other := view.Id(999)

	highlights := []view.DocumentHighlight{{From: 0, To: 5}}
	doc.SetDocumentHighlights(v.ID(), highlights)
	doc.SetDocumentHighlights(other, highlights)

	doc.ClearAllDocumentHighlights()

	assert.Empty(t, doc.DocumentHighlights(v.ID()))
	assert.Empty(t, doc.DocumentHighlights(other))
}

func TestDocumentInlayHints(t *testing.T) {
	e := testutil.EditorWithText(t, "fn main() {}\n")
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
	e := testutil.EditorWithText(t, "fn main() {}\n")
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
