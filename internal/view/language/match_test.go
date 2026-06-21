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

func TestExpandGlobBraces(t *testing.T) {
	t.Run("brace expansion detects c files", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "c"
file-types = [{ glob = "*.{c,h}" }]
`)
		name, ok := language.DetectLanguage("main.c", "")
		assert.True(t, ok)
		assert.Equal(t, "c", name)
	})

	t.Run("brace expansion matches header", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "c"
file-types = [{ glob = "*.{c,h}" }]
`)
		name, ok := language.DetectLanguage("types.h", "")
		assert.True(t, ok)
		assert.Equal(t, "c", name)
	})
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
