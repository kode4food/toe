package action_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
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

func primaryRange(t *testing.T, e *view.Editor) (*view.Document, core.Range) {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	return doc, doc.SelectionFor(v.ID()).Primary()
}
