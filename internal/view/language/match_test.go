package language_test

import (
	"fmt"
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

	t.Run("empty shebang fields returns false", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		_, ok := language.DetectLanguage("", "#!")
		assert.False(t, ok)
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

	t.Run("unmarshal nil errors", func(t *testing.T) {
		var a language.AutoPairConfig
		assert.Error(t, a.UnmarshalTOML(nil))
	})

	t.Run("unmarshal string map enables pairs", func(t *testing.T) {
		var a language.AutoPairConfig
		assert.NoError(t, a.UnmarshalTOML(map[string]string{"(": ")"}))
		pairs, ok := a.AutoPairs()
		assert.True(t, ok)
		p, got := pairs.Get('(')
		assert.True(t, got)
		assert.Equal(t, ')', p.Close)
	})

	t.Run("OrDefault with present config", func(t *testing.T) {
		a := language.AutoPairConfig{
			Present: true,
			Pairs:   [][2]rune{{'[', ']'}},
		}
		pairs, ok := a.OrDefault()
		assert.True(t, ok)
		_, got := pairs.Get('[')
		assert.True(t, got)
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

func TestLanguageForMatch(t *testing.T) {
	t.Run("injection regex matches content", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "injlang"
injection-regex = "injlang|il"
`)
		name, ok := language.DetectLanguage("", "il")
		assert.True(t, ok)
		assert.Equal(t, "injlang", name)
	})

	t.Run("injection regex no match returns false", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "injlang2"
injection-regex = "uniqueinjlang2_abc"
`)
		_, ok := language.DetectLanguage("", "ZZZNOMATCH_ABCDEF_123")
		assert.False(t, ok)
	})

	t.Run("invalid injection regex skipped", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "badregexlang"
injection-regex = "["
`)
		// content doesn't match name → 2nd loop tries invalid regex, skips it
		_, ok := language.DetectLanguage("", "ZZZNOMATCH_ABCDEF_123")
		assert.False(t, ok)
	})
}

func TestGlobMatch(t *testing.T) {
	t.Run("absolute glob matches exact path", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "exact.abs")
		setUserLangs(t, fmt.Sprintf(`
[[language]]
name = "abslang"
file-types = [{glob = %q}]
`, target))
		name, ok := language.DetectLanguage(target, "")
		assert.True(t, ok)
		assert.Equal(t, "abslang", name)
	})

	t.Run("starslash prefix glob matches", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "starlang"
file-types = [{ glob = "*/foo.star" }]
`)
		name, ok := language.DetectLanguage("/any/dir/foo.star", "")
		assert.True(t, ok)
		assert.Equal(t, "starlang", name)
	})

	t.Run("double-star glob matches any path", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "dblstar"
file-types = [{ glob = "**" }]
`)
		name, ok := language.DetectLanguage("anything/foo.xyz", "")
		assert.True(t, ok)
		assert.Equal(t, "dblstar", name)
	})

	t.Run("glob with double-star non-match is noop", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "nomatch"
file-types = [{ glob = "**/nope.nomatch" }]
`)
		_, ok := language.DetectLanguage("some/path/other.txt", "")
		assert.False(t, ok)
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
