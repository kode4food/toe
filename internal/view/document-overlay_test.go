package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
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

func TestOverlayAnchoring(t *testing.T) {
	t.Run("diagnostic shifts with insert before it", func(t *testing.T) {
		e := testutil.EditorWithText(t, "func foo() {}\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{Range: view.DiagnosticRange{From: 5, To: 8}},
		})

		insertTextAt(t, e, doc, 0, "// x\n")

		diags := doc.Diagnostics()
		assert.Len(t, diags, 1)
		assert.Equal(t, 5+len("// x\n"), diags[0].Range.From)
		assert.Equal(t, 8+len("// x\n"), diags[0].Range.To)
	})

	t.Run("link shifts with insert before it", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{{From: 6, To: 11}})

		insertTextAt(t, e, doc, 0, "xx")

		links := doc.DocumentLinks()
		assert.Len(t, links, 1)
		assert.Equal(t, 8, links[0].From)
		assert.Equal(t, 13, links[0].To)
	})

	t.Run("color shifts with insert before it", func(t *testing.T) {
		e := testutil.EditorWithText(t, "color: #123456\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentColors([]view.DocumentColor{{From: 7, To: 14}})

		insertTextAt(t, e, doc, 0, "xx")

		colors := doc.DocumentColors()
		assert.Len(t, colors, 1)
		assert.Equal(t, 9, colors[0].From)
		assert.Equal(t, 16, colors[0].To)
	})

	t.Run("highlight shifts with insert before it", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world\n")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentHighlights(v.ID(), []view.DocumentHighlight{
			{From: 6, To: 11},
		})

		insertTextAt(t, e, doc, 0, "xx")

		hl := doc.DocumentHighlights(v.ID())
		assert.Len(t, hl, 1)
		assert.Equal(t, 8, hl[0].From)
		assert.Equal(t, 13, hl[0].To)
	})

	t.Run("inlay hint shifts with insert before it", func(t *testing.T) {
		e := testutil.EditorWithText(t, "fn main() {}\n")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetInlayHints(v.ID(), []view.InlayHint{{Pos: 8, Label: ": ()"}})

		insertTextAt(t, e, doc, 0, "xx")

		hints := doc.InlayHints(v.ID())
		assert.Len(t, hints, 1)
		assert.Equal(t, 10, hints[0].Pos)
	})

	t.Run("diagnostic unaffected by edit after it", func(t *testing.T) {
		e := testutil.EditorWithText(t, "func foo() {}\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{Range: view.DiagnosticRange{From: 5, To: 8}},
		})

		insertTextAt(t, e, doc, 12, "xx")

		diags := doc.Diagnostics()
		assert.Len(t, diags, 1)
		assert.Equal(t, 5, diags[0].Range.From)
		assert.Equal(t, 8, diags[0].Range.To)
	})

	t.Run("diagnostic shift survives undo and redo", func(t *testing.T) {
		e := testutil.EditorWithText(t, "func foo() {}\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{Range: view.DiagnosticRange{From: 5, To: 8}},
		})

		insertTextAt(t, e, doc, 0, "// x\n")
		assert.Equal(t, 5+len("// x\n"), doc.Diagnostics()[0].Range.From)

		assert.True(t, e.Undo())
		assert.Equal(t, 5, doc.Diagnostics()[0].Range.From)

		assert.True(t, e.Redo())
		assert.Equal(t, 5+len("// x\n"), doc.Diagnostics()[0].Range.From)
	})
}

func insertTextAt(
	t *testing.T, e *view.Editor, doc *view.Document, at int, s string,
) {
	t.Helper()
	rope := doc.Text()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(at, at, s),
	})
	assert.NoError(t, err)
	tx := core.NewTransaction(rope).WithChanges(cs)
	assert.NoError(t, e.Apply(tx))
}
