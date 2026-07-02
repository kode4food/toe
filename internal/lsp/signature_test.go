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

func TestSignatureHelp(t *testing.T) {
	t.Run("requests trigger character help", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeSignatureLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("Println(\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(8))

		help, err := session.TriggerSignatureHelp(doc, v.ID())

		assert.NoError(t, err)
		assert.Len(t, help.Signatures, 1)
		if len(help.Signatures) != 1 {
			return
		}
		sig := help.Signatures[0]
		assert.Equal(t, "Println(a ...any)", sig.Label)
		assert.Equal(t, "signature docs", sig.Docs)
		assert.Equal(t, "parameter docs", sig.ParamDocs)
		assert.Equal(t, 8, sig.ActiveStart)
		assert.Equal(t, 16, sig.ActiveEnd)
	})
}

func TestSignatureHelpNonTrigger(t *testing.T) {
	t.Run("fetches help without trigger", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeSignatureLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("Println(\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(8))

		help, err := session.SignatureHelp(doc, v.ID())

		assert.NoError(t, err)
		assert.Len(t, help.Signatures, 1)
	})
}

func TestSignatureHelpOffsetLabel(t *testing.T) {
	t.Run("handles tuple offset label", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeSignatureOffsetLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("Println(a)\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(9))

		help, err := session.SignatureHelp(doc, v.ID())
		assert.NoError(t, err)
		assert.NotEmpty(t, help.Signatures)
		// offset label [8,16] maps to "a ...any" in "Println(a ...any)"
		sig := help.Signatures[0]
		assert.Equal(t, 8, sig.ActiveStart)
		assert.Equal(t, 16, sig.ActiveEnd)
	})
}

func TestSignatureHelpVariants(t *testing.T) {
	t.Run("uses help active parameter", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeSignatureVariantLanguages(
			t, exe, testServerSignatureHelpActiveEnv,
		)
		assert.NoError(t, os.WriteFile(path, []byte("Println(a,\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(10))

		help, err := session.SignatureHelp(doc, v.ID())

		assert.NoError(t, err)
		assert.Len(t, help.Signatures, 1)
		if len(help.Signatures) == 0 {
			return
		}
		sig := help.Signatures[0]
		assert.Equal(t, "second docs", sig.ParamDocs)
		assert.Equal(t, 11, sig.ActiveStart)
		assert.Equal(t, 12, sig.ActiveEnd)
	})

	t.Run("accepts signature without params", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeSignatureVariantLanguages(t, exe, testServerSignatureNoParamsEnv)
		assert.NoError(t, os.WriteFile(path, []byte("Now(\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(4))

		help, err := session.SignatureHelp(doc, v.ID())

		assert.NoError(t, err)
		assert.Len(t, help.Signatures, 1)
		if len(help.Signatures) == 0 {
			return
		}
		sig := help.Signatures[0]
		assert.Equal(t, "Now()", sig.Label)
		assert.Equal(t, 0, sig.ActiveStart)
		assert.Equal(t, 0, sig.ActiveEnd)
	})

	t.Run("missing label has zero range", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeSignatureVariantLanguages(
			t, exe, testServerSignatureMissingLabelEnv,
		)
		assert.NoError(t, os.WriteFile(path, []byte("Println(\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(8))

		help, err := session.SignatureHelp(doc, v.ID())

		assert.NoError(t, err)
		assert.Len(t, help.Signatures, 1)
		if len(help.Signatures) == 0 {
			return
		}
		sig := help.Signatures[0]
		assert.Equal(t, 0, sig.ActiveStart)
		assert.Equal(t, 0, sig.ActiveEnd)
	})

	t.Run("clips active signature", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeSignatureVariantLanguages(t, exe, testServerSignatureActiveOutEnv)
		assert.NoError(t, os.WriteFile(path, []byte("Now(\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(4))

		help, err := session.SignatureHelp(doc, v.ID())

		assert.NoError(t, err)
		assert.Equal(t, 0, help.Active)
		assert.Len(t, help.Signatures, 1)
	})
}

func writeSignatureOffsetLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerSignatureEnv + ` = "1", ` +
		testServerSignatureOffsetEnv + ` = "1" }

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

func writeSignatureVariantLanguages(t *testing.T, exe, variant string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerSignatureEnv + ` = "1", ` +
		variant + ` = "1" }

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

func writeSignatureLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerSignatureEnv + ` = "1" }

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
