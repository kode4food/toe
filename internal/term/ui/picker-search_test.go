package ui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestGlobalSearch(t *testing.T) {
	t.Run("finds matching lines across files", func(t *testing.T) {
		m, _ := globalSearchModel(t, "findme")
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "a.txt")
		assert.NotContains(t, out, "b.txt")
	})

	t.Run("accept opens the match and selects its line", func(t *testing.T) {
		m, e := globalSearchModel(t, "findme")
		m = sendSpecial(m, tea.KeyEnter)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.True(t, strings.HasSuffix(doc.Path(), "a.txt"))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		line, err := doc.Text().CharToLine(
			doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text()),
		)
		assert.NoError(t, err)
		assert.Equal(t, 1, line) // 0-indexed line 2 holds "findme here"
	})
}
