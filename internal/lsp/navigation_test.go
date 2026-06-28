package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
)

func TestNavigation(t *testing.T) {
	t.Run("requests definition target", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		source := filepath.Join(dir, "main.session")
		target := filepath.Join(dir, "target.session")
		writeNavigationLanguages(t, exe, target)
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(source)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(0))

		locations, err := session.GotoDefinition(doc, v.ID())

		assert.NoError(t, err)
		assert.Equal(t, []view.Location{
			{Path: target, From: 3, To: 6},
		}, locations)
	})

	t.Run("requests reference target", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		source := filepath.Join(dir, "main.session")
		target := filepath.Join(dir, "target.session")
		writeNavigationLanguages(t, exe, target)
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(source)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(0))

		locations, err := session.GotoReference(doc, v.ID())

		assert.NoError(t, err)
		assert.Equal(t, []view.Location{
			{Path: target, From: 3, To: 6},
		}, locations)
	})
}

func writeNavigationLanguages(t *testing.T, exe, target string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerNavigationEnv + ` = "1", ` +
		testServerNavigationTargetEnv + ` = "` + target + `" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}
