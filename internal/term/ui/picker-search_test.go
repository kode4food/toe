package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
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

	t.Run("empty query has no matches", func(t *testing.T) {
		m, _ := globalSearchModel(t, "")
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "a.txt")
		assert.NotContains(t, out, "b.txt")
	})

	t.Run("invalid regex has no matches", func(t *testing.T) {
		m, _ := globalSearchModel(t, "[")
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "a.txt")
		assert.NotContains(t, out, "b.txt")
	})

	t.Run("lowercase query ignores case", func(t *testing.T) {
		m, _ := globalSearchModelWithFiles(t, "findme", map[string]string{
			"a.txt": "alpha\nFindMe here\n",
			"b.txt": "beta\n",
		})
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "a.txt")
	})

	t.Run("uppercase query is case sensitive", func(t *testing.T) {
		m, _ := globalSearchModelWithFiles(t, "FINDME", map[string]string{
			"a.txt": "alpha\nfindme here\n",
			"b.txt": "beta\n",
		})
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "a.txt")
	})

	t.Run("skips binary files", func(t *testing.T) {
		m, _ := globalSearchModelWithFiles(t, "findme", map[string]string{
			"a.bin": "findme\x00here\n",
			"b.txt": "beta\n",
		})
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "a.bin")
	})

	t.Run("searches open document text", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "a.txt")
		assert.NoError(t, os.WriteFile(path, []byte("disk\n"), 0o644))

		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, rope.LenChars(), "memory needle\n"),
		})
		assert.NoError(t, err)
		tx := core.NewTransaction(rope).
			WithChanges(cs).
			WithSelection(core.PointSelection(0))
		assert.NoError(t, e.Apply(tx))

		m := globalSearchModelForEditor(t, e, "needle")
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "a.txt")
	})
}

// globalSearchModel writes two files, opens the global-search picker, and types
// the query. openPickerAndFeed drains the dynamic source's async feed per key
func globalSearchModel(t *testing.T, query string) (ui.Model, *view.Editor) {
	t.Helper()
	return globalSearchModelWithFiles(t, query, map[string]string{
		"a.txt": "alpha\nfindme here\n",
		"b.txt": "beta\n",
	})
}

func globalSearchModelWithFiles(
	t *testing.T, query string, files map[string]string,
) (ui.Model, *view.Editor) {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		path := filepath.Join(dir, name)
		err := os.WriteFile(path, []byte(text), 0o644)
		assert.NoError(t, err)
	}
	e := view.NewEditor(dir)
	return globalSearchModelForEditor(t, e, query), e
}

func globalSearchModelForEditor(
	t *testing.T, e *view.Editor, query string,
) ui.Model {
	t.Helper()
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "global_search", m.GlobalSearchAction(),
		[]command.KeyEvent{char('s')},
	)
	m = resize(m, 120, 30)
	m = openPickerAndFeed(m, 's')
	for _, ch := range query {
		m = openPickerAndFeed(m, ch)
	}
	return m
}
