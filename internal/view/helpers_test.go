package view_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
)

func selectionAnchorHead(t *testing.T, e *view.Editor) (anchor, head int) {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	sel := doc.SelectionFor(v.ID())
	r := sel.Primary()
	return r.Anchor, r.Head
}

func writeCommandLanguages(t *testing.T, text string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	)
	assert.NoError(t, err)
	t.Setenv("XDG_CONFIG_HOME", root)
}

func newSelection(
	t *testing.T, ranges []core.Range, primary int,
) core.Selection {
	t.Helper()
	sel, err := core.NewSelection(ranges, primary)
	assert.NoError(t, err)
	return sel
}

func replaceFocusedText(t *testing.T, e *view.Editor, text string) {
	t.Helper()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	rope := doc.Text()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(0, rope.LenChars(), text),
	})
	assert.NoError(t, err)
	tx := core.NewTransaction(rope).WithChanges(cs)
	assert.NoError(t, e.Apply(tx))
}
