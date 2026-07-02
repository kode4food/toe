package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
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
		defer func() { _ = session.Close() }()
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
		defer func() { _ = session.Close() }()
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

	t.Run("requests declaration target", func(t *testing.T) {
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
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(0))

		locations, err := session.GotoDeclaration(doc, v.ID())

		assert.NoError(t, err)
		assert.Equal(t, []view.Location{
			{Path: target, From: 3, To: 6},
		}, locations)
	})

	t.Run("requests type definition target", func(t *testing.T) {
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
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(0))

		locations, err := session.GotoTypeDefinition(doc, v.ID())

		assert.NoError(t, err)
		assert.Equal(t, []view.Location{
			{Path: target, From: 3, To: 6},
		}, locations)
	})

	t.Run("requests implementation target", func(t *testing.T) {
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
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(0))

		locations, err := session.GotoImplementation(doc, v.ID())

		assert.NoError(t, err)
		assert.Equal(t, []view.Location{
			{Path: target, From: 3, To: 6},
		}, locations)
	})
}

func TestNavigationLinkSlices(t *testing.T) {
	t.Run("handles declaration link slice", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		source := filepath.Join(dir, "main.session")
		target := filepath.Join(dir, "target.session")
		writeNavigationLinksLanguages(t, exe, target)
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(source)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(0))

		locations, err := session.GotoDeclaration(doc, v.ID())

		assert.NoError(t, err)
		assert.Equal(t, []view.Location{
			{Path: target, From: 3, To: 6},
		}, locations)
	})

	t.Run("handles definition link slice", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		source := filepath.Join(dir, "main.session")
		target := filepath.Join(dir, "target.session")
		writeNavigationLinksLanguages(t, exe, target)
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(source)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
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
}

func TestNavigationLocationSlice(t *testing.T) {
	t.Run("handles location slice result", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		source := filepath.Join(dir, "main.session")
		target := filepath.Join(dir, "target.session")
		writeNavigationLocationSliceLanguages(t, exe, target)
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(source)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(0))

		locations, err := session.GotoDeclaration(doc, v.ID())
		assert.NoError(t, err)
		assert.Equal(t, []view.Location{{Path: target, From: 3, To: 6}}, locations)

		locations, err = session.GotoDefinition(doc, v.ID())
		assert.NoError(t, err)
		assert.Equal(t, []view.Location{{Path: target, From: 3, To: 6}}, locations)
	})
}

func TestNavigationOutOfRangePosition(t *testing.T) {
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
	defer func() { _ = session.Close() }()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc.SetSelectionFor(v.ID(), core.PointSelection(9999))

	_, err = session.GotoDefinition(doc, v.ID())
	assert.Error(t, err)
}

func writeNavigationLocationSliceLanguages(t *testing.T, exe, target string) {
	t.Helper()
	writeNavigationConfig(t,
		exe, target, testServerNavLocationSliceEnv+` = "1", `)
}

func writeNavigationLanguages(t *testing.T, exe, target string) {
	t.Helper()
	writeNavigationConfig(t, exe, target, "")
}

func writeNavigationLinksLanguages(t *testing.T, exe, target string) {
	t.Helper()
	writeNavigationConfig(t, exe, target, testServerNavLinksEnv+` = "1", `)
}

func writeNavigationConfig(t *testing.T, exe, target, extra string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` + extra +
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
