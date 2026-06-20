package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestGlobalSearch(t *testing.T) {
	t.Run("finds matching lines across files", func(t *testing.T) {
		m, _ := globalSearchModel(t, "findme")
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "a.txt")
		assert.NotContains(t, out, "b.txt")
	})

	t.Run("accept opens match", func(t *testing.T) {
		m, e := globalSearchModel(t, "findme")
		_ = sendSpecial(m, tea.KeyEnter)
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

// globalSearchModel writes two files, opens the global-search picker, and types
// the query. openPickerAndFeed drains the dynamic source's async feed per key
func globalSearchModel(t *testing.T, query string) (ui.Model, *view.Editor) {
	t.Helper()
	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "a.txt"), []byte("alpha\nfindme here\n"), 0o644,
	))
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "b.txt"), []byte("beta\n"), 0o644,
	))

	e := view.NewEditor(dir)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "global_search", m.GlobalSearchAction(),
		[]command.KeyEvent{command.Char('s')},
	)
	m = resize(m, 120, 30)
	m = openPickerAndFeed(m, 's')
	for _, ch := range query {
		m = openPickerAndFeed(m, ch)
	}
	return m, e
}
