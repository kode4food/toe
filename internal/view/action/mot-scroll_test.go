package action_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestPageSelect(t *testing.T) {
	text := strings.Repeat("x\n", 80)

	t.Run("page down extends", func(t *testing.T) {
		e := testutil.EditorWithText(t, text)
		e.SetMode(view.ModeSelect)

		action.PageDown(e)

		doc, r := primaryRange(t, e)
		line, err := doc.Text().CharToLine(r.Cursor(doc.Text()))
		assert.NoError(t, err)
		assert.Equal(t, 0, r.From())
		assert.True(t, r.To() > r.From())
		assert.True(t, line > 0)
	})

	t.Run("page up extends", func(t *testing.T) {
		e := testutil.EditorWithText(t, text)
		testutil.SetSelection(t, e, []core.Range{core.PointRange(100)}, 0)
		e.SetMode(view.ModeSelect)

		action.PageUp(e)

		doc, r := primaryRange(t, e)
		line, err := doc.Text().CharToLine(r.Cursor(doc.Text()))
		assert.NoError(t, err)
		assert.True(t, r.To() > r.From())
		assert.True(t, line < 50)
	})
}

func TestPageHeightPerPane(t *testing.T) {
	text := strings.Repeat("x\n", 400)

	t.Run("page down uses the pane, not the editor", func(t *testing.T) {
		e := splitPane(t, text)
		full := e.ViewHeight()
		pane := e.FocusedView().ContentHeight()
		assert.Less(t, pane, full)
		before := cursorLine(t, e)

		action.PageDown(e)

		assert.Equal(t, before+pane, cursorLine(t, e))
	})

	t.Run("align center uses the pane", func(t *testing.T) {
		e := splitPane(t, text)
		v := e.FocusedView()
		pane := v.ContentHeight()
		testutil.SetCursor(t, e, 400)

		action.AlignViewCenter(e)

		doc := e.FocusedDocument()
		anchorLine, err := doc.Text().CharToLine(v.Offset().Anchor)
		assert.NoError(t, err)
		assert.Equal(t, 200-(pane-1)/2, anchorLine)
	})
}

func splitPane(t *testing.T, text string) *view.Editor {
	t.Helper()
	e := testutil.EditorWithText(t, text)
	// scrolloff would pull the cursor further down after the page
	e.Options().ScrollOff = 0
	e.SetViewHeight(100)
	e.ResizeTree(geom.Size{Width: 80, Height: 100})
	docID := e.FocusedDocument().ID()
	assert.NotNil(t, e.HSplitNew())
	e.ShowDocument(docID)
	return e
}

func primaryRange(t *testing.T, e *view.Editor) (*view.Document, core.Range) {
	t.Helper()
	v := e.FocusedView()
	assert.NotNil(t, v)
	doc := e.FocusedDocument()
	assert.NotNil(t, doc)
	return doc, doc.SelectionFor(v.ID()).Primary()
}
