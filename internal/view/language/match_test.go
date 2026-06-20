package language_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/language"
)

func TestDetectLanguage(t *testing.T) {
	t.Run("detects go by extension", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		name, ok := language.DetectLanguage("main.go", "")
		assert.True(t, ok)
		assert.Equal(t, "go", name)
	})

	t.Run("detects language by shebang", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "python"
shebangs = ["python3"]
`)
		name, ok := language.DetectLanguage(
			"script", "#!/usr/bin/env python3\nprint('hi')\n",
		)
		assert.True(t, ok)
		assert.Equal(t, "python", name)
	})

	t.Run("returns false for unknown", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		_, ok := language.DetectLanguage("unknown.xyz99qwerty", "")
		assert.False(t, ok)
	})

	t.Run("detects bash by shebang", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		name, ok := language.DetectLanguage(
			"run", "#!/bin/bash\necho hi\n",
		)
		assert.True(t, ok)
		assert.Equal(t, "bash", name)
	})
}

func TestLoadLanguages(t *testing.T) {
	t.Run("loads glob file types", func(t *testing.T) {
		path := writeLangToml(t, `
[[language]]
name = "owners"
file-types = [{ glob = "OWNERS" }]
`)
		langs, ok := language.LoadLanguages(path)
		assert.True(t, ok)
		assert.Equal(t, "owners", langs.Languages[0].Name)
		assert.Equal(t, "*/OWNERS", langs.Languages[0].FileTypes[0].Glob)
	})

	t.Run("loads brace-expansion globs", func(t *testing.T) {
		path := writeLangToml(t, `
[[language]]
name = "c"
file-types = [{ glob = "*.{c,h}" }]
`)
		langs, ok := language.LoadLanguages(path)
		assert.True(t, ok)
		assert.Equal(t, "*/*.{c,h}", langs.Languages[0].FileTypes[0].Glob)
	})

	t.Run("missing file returns false", func(t *testing.T) {
		_, ok := language.LoadLanguages(
			filepath.Join(t.TempDir(), "missing.toml"),
		)
		assert.False(t, ok)
	})
}

func TestFindLanguageRoot(t *testing.T) {
	t.Run("finds root with marker file", func(t *testing.T) {
		dir := t.TempDir()
		_, err := os.Create(filepath.Join(dir, "go.mod"))
		assert.NoError(t, err)
		lang := &language.Language{Roots: []string{"go.mod"}}
		root, ok := language.FindLanguageRoot(dir, lang)
		assert.True(t, ok)
		assert.Equal(t, dir, root)
	})

	t.Run("no roots returns false", func(t *testing.T) {
		_, ok := language.FindLanguageRoot(t.TempDir(), &language.Language{})
		assert.False(t, ok)
	})

	t.Run("marker absent returns false", func(t *testing.T) {
		lang := &language.Language{Roots: []string{"go.mod"}}
		_, ok := language.FindLanguageRoot(t.TempDir(), lang)
		assert.False(t, ok)
	})

	t.Run("finds root from file path", func(t *testing.T) {
		dir := t.TempDir()
		_, err := os.Create(filepath.Join(dir, "go.mod"))
		assert.NoError(t, err)
		sub := filepath.Join(dir, "pkg")
		assert.NoError(t, os.Mkdir(sub, 0o755))
		f, err := os.Create(filepath.Join(sub, "main.go"))
		assert.NoError(t, err)
		_ = f.Close()
		lang := &language.Language{Roots: []string{"go.mod"}}
		root, ok := language.FindLanguageRoot(
			filepath.Join(sub, "main.go"), lang)
		assert.True(t, ok)
		assert.Equal(t, dir, root)
	})
}

func TestFindLSPWorkspace(t *testing.T) {
	t.Run("file outside workspace returns false", func(t *testing.T) {
		_, ok := language.FindLSPWorkspace(language.LSPWorkspaceArgs{
			File:      "/some/other/path/file.go",
			Workspace: "/workspace",
		})
		assert.False(t, ok)
	})

	t.Run("fallback returns workspace root", func(t *testing.T) {
		dir := t.TempDir()
		f, err := os.Create(filepath.Join(dir, "file.go"))
		assert.NoError(t, err)
		_ = f.Close()
		root, ok := language.FindLSPWorkspace(language.LSPWorkspaceArgs{
			File:              filepath.Join(dir, "file.go"),
			Workspace:         dir,
			WorkspaceFallback: true,
		})
		assert.True(t, ok)
		assert.Equal(t, dir, root)
	})

	t.Run("no fallback or root marker returns false", func(t *testing.T) {
		dir := t.TempDir()
		f, err := os.Create(filepath.Join(dir, "file.go"))
		assert.NoError(t, err)
		_ = f.Close()
		_, ok := language.FindLSPWorkspace(language.LSPWorkspaceArgs{
			File:      filepath.Join(dir, "file.go"),
			Workspace: dir,
		})
		assert.False(t, ok)
	})

	t.Run("matching rootDir returns workspace", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		assert.NoError(t, os.Mkdir(src, 0o755))
		f, err := os.Create(filepath.Join(src, "file.go"))
		assert.NoError(t, err)
		_ = f.Close()
		root, ok := language.FindLSPWorkspace(language.LSPWorkspaceArgs{
			File:      filepath.Join(src, "file.go"),
			RootDirs:  []string{"src"},
			Workspace: dir,
		})
		assert.True(t, ok)
		assert.Equal(t, dir, root)
	})
}

func TestSelectedGrammars(t *testing.T) {
	langs := language.Languages{
		Grammars: []language.Grammar{
			{Name: "go"},
			{Name: "rust"},
			{Name: "skip"},
		},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		langs.GrammarSelection = language.GrammarSelection{}
		assert.Len(t, langs.SelectedGrammars(), 3)
	})

	t.Run("only filter selects subset", func(t *testing.T) {
		langs.GrammarSelection = language.GrammarSelection{
			Only: []string{"go"},
		}
		grammars := langs.SelectedGrammars()
		assert.Len(t, grammars, 1)
		assert.Equal(t, "go", grammars[0].Name)
	})

	t.Run("except filter excludes", func(t *testing.T) {
		langs.GrammarSelection = language.GrammarSelection{
			Except: []string{"skip"},
		}
		grammars := langs.SelectedGrammars()
		assert.Len(t, grammars, 2)
	})
}

func TestAutoPairConfig(t *testing.T) {
	t.Run("absent returns defaults", func(t *testing.T) {
		a := language.AutoPairConfig{}
		pairs, ok := a.OrDefault()
		assert.True(t, ok)
		_, got := pairs.Get('(')
		assert.True(t, got)
	})

	t.Run("enable false disables", func(t *testing.T) {
		a := language.AutoPairConfig{Present: true, Enable: new(false)}
		_, ok := a.AutoPairs()
		assert.False(t, ok)
	})

	t.Run("enable true returns defaults", func(t *testing.T) {
		a := language.AutoPairConfig{Present: true, Enable: new(true)}
		pairs, ok := a.AutoPairs()
		assert.True(t, ok)
		_, got := pairs.Get('(')
		assert.True(t, got)
	})

	t.Run("custom pairs returned", func(t *testing.T) {
		a := language.AutoPairConfig{
			Present: true,
			Pairs:   [][2]rune{{'<', '>'}},
		}
		pairs, ok := a.AutoPairs()
		assert.True(t, ok)
		p, got := pairs.Get('<')
		assert.True(t, got)
		assert.Equal(t, '>', p.Close)
	})

	t.Run("unmarshal bool false disables", func(t *testing.T) {
		var a language.AutoPairConfig
		assert.NoError(t, a.UnmarshalTOML(false))
		_, ok := a.AutoPairs()
		assert.False(t, ok)
	})

	t.Run("unmarshal invalid errors", func(t *testing.T) {
		var a language.AutoPairConfig
		assert.Error(t, a.UnmarshalTOML(42))
	})
}

func TestLoadBundledLanguages(t *testing.T) {
	t.Run("loads without error", func(t *testing.T) {
		langs, ok := language.LoadBundledLanguages()
		assert.True(t, ok)
		assert.NotEmpty(t, langs.Languages)
	})

	t.Run("contains go language", func(t *testing.T) {
		langs, ok := language.LoadBundledLanguages()
		assert.True(t, ok)
		var found bool
		for _, l := range langs.Languages {
			if l.Name == "go" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

func writeLangToml(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "languages.toml")
	assert.NoError(t, os.WriteFile(path, []byte(text), 0o644))
	return path
}

func setUserLangs(t *testing.T, text string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}
