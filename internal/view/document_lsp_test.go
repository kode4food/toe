package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestDocumentHighlights(t *testing.T) {
	e := editorWithText(t, "hello world\n")
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

	// setting empty slice removes entry
	doc.SetDocumentHighlights(vid, highlights)
	doc.SetDocumentHighlights(vid, nil)
	assert.Empty(t, doc.DocumentHighlights(vid))
}

func TestDocumentHighlightsAll(t *testing.T) {
	e := editorWithText(t, "hello world\n")
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

func TestDocumentAccessedAt(t *testing.T) {
	e := editorWithText(t, "hello\n")
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	assert.NotZero(t, doc.AccessedAt())
}

func TestDocumentLinks(t *testing.T) {
	e := editorWithText(t, "hello world\n")
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

	// setting empty slice removes links
	doc.SetDocumentLinks(links)
	doc.SetDocumentLinks(nil)
	assert.Empty(t, doc.DocumentLinks())
}
