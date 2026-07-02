package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

func TestLanguages(t *testing.T) {
	t.Run("merges trusted workspace languages", func(t *testing.T) {
		root := t.TempDir()
		global := writeLanguages(t, root, `
[[language]]
name = "markdown"
text-width = 80
soft-wrap.enable = true
soft-wrap.wrap-indicator = "↪ "
`)
		work := filepath.Join(root, "work")
		workspaceDir := filepath.Join(work,
			loader.WorkspaceDirName,
		)
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(workspaceDir, 0o755)
		assert.NoError(t, err)
		workspace := filepath.Join(workspaceDir, "languages.toml")
		err = os.WriteFile(workspace, []byte(`
[[language]]
name = "markdown"
text-width = 72
soft-wrap.enable = true
soft-wrap.wrap-indicator = "» "
`), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		err = loader.TrustWorkspace(work)
		assert.NoError(t, err)

		langs, ok := language.LoadLanguagesForWorkspace(global, workspace, work)

		assert.True(t, ok)
		lang, ok := findLanguage(langs, "markdown")
		assert.True(t, ok)
		assert.Equal(t, 72, *lang.TextWidth)
		assert.True(t, *lang.SoftWrap.Enable)
		assert.Equal(t, "» ", *lang.SoftWrap.WrapIndicator)
	})

	t.Run("ignores untrusted workspace languages", func(t *testing.T) {
		root := t.TempDir()
		global := writeLanguages(t, root, `
[[language]]
name = "markdown"
text-width = 80
`)
		work := filepath.Join(root, "work")
		workspaceDir := filepath.Join(work,
			loader.WorkspaceDirName,
		)
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(workspaceDir, 0o755)
		assert.NoError(t, err)
		workspace := filepath.Join(workspaceDir, "languages.toml")
		err = os.WriteFile(workspace, []byte(`
[[language]]
name = "markdown"
text-width = 72
`), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_DATA_HOME", t.TempDir())

		langs, ok := language.LoadLanguagesForWorkspace(global, workspace, work)

		assert.True(t, ok)
		lang, ok := findLanguage(langs, "markdown")
		assert.True(t, ok)
		assert.Equal(t, 80, *lang.TextWidth)
	})

	t.Run("merges built-in languages before user", func(t *testing.T) {
		root := t.TempDir()
		global := writeLanguages(t, root, `
[[language]]
name = "markdown"
text-width = 80
`)
		workspace := filepath.Join(root, "missing.toml")

		langs, ok := language.LoadLanguagesForWorkspace(global, workspace, root)

		assert.True(t, ok)
		lang, ok := findLanguage(langs, "markdown")
		assert.True(t, ok)
		assert.Equal(t, 80, *lang.TextWidth)
		assert.Equal(t, "md", lang.FileTypes[0].Extension)
	})
}

func TestConfig(t *testing.T) {
	t.Run("loads constant theme", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
theme = "dark"
`)

		raw, ok := config.LoadRawConfig(path)

		assert.True(t, ok)
		assert.Equal(t, "dark", raw["theme"])
	})

	t.Run("loads adaptive theme", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[theme]
light = "light"
dark = "dark"
fallback = "base"
`)

		raw, ok := config.LoadRawConfig(path)

		assert.True(t, ok)
		theme, ok := raw["theme"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "light", theme["light"])
		assert.Equal(t, "dark", theme["dark"])
		assert.Equal(t, "base", theme["fallback"])
	})

	t.Run("loads editor soft wrap settings", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
text-width = 72

[editor.soft-wrap]
enable = true
max-wrap = 18
max-indent-retain = 24
wrap-indicator = "» "
wrap-at-text-width = true
`)

		_, ok := config.LoadRawConfig(path)

		assert.True(t, ok)
	})

	t.Run("loads statusline mode names", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.statusline]
separator = "|"
left = ["mode", "file-name"]
center = ["spacer"]
right = ["position", "file-encoding"]
diagnostics = ["hint", "error"]
workspace-diagnostics = ["info", "warning"]

[editor.statusline.mode]
normal = "NORMAL"
insert = "INSERT"
select = "SELECT"
`)

		_, ok := config.LoadRawConfig(path)

		assert.True(t, ok)
	})

	t.Run("uses insecure default", func(t *testing.T) {
		assert.Equal(t, "mocha", view.DefaultTheme)
	})

	t.Run("loads insecure option", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
insecure = true
`)

		raw, ok := config.LoadRawConfig(path)

		assert.True(t, ok)
		editor, ok := raw["editor"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, true, editor["insecure"])
	})

	t.Run("merges trusted workspace config", func(t *testing.T) {
		root := t.TempDir()
		global := writeConfig(t, root, `theme = "mocha"`)
		work := filepath.Join(root, "work")
		workspaceDir := filepath.Join(work, loader.WorkspaceDirName)
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(workspaceDir, 0o755)
		assert.NoError(t, err)
		workspace := filepath.Join(workspaceDir, "config.toml")
		err = os.WriteFile(workspace, []byte(`theme = "dracula"`), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		err = loader.TrustWorkspace(work)
		assert.NoError(t, err)

		raw, ok := config.LoadRawConfigForWorkspace(
			global, workspace, work,
		)

		assert.True(t, ok)
		assert.Equal(t, "dracula", raw["theme"])
	})

	t.Run("loads config for directory workspace", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `
theme = "mocha"

[editor]
insecure = true
`)
		t.Setenv("XDG_CONFIG_HOME", root)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		work := filepath.Join(root, "work")
		workspaceDir := filepath.Join(work, loader.WorkspaceDirName)
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(workspaceDir, 0o755)
		assert.NoError(t, err)
		workspace := filepath.Join(workspaceDir, "config.toml")
		err = os.WriteFile(workspace, []byte(`theme = "dracula"`), 0o644)
		assert.NoError(t, err)

		raw, ok := config.LoadRawConfigForDir(work)

		assert.True(t, ok)
		assert.Equal(t, "dracula", raw["theme"])
	})

	t.Run("ignores untrusted workspace config", func(t *testing.T) {
		root := t.TempDir()
		global := writeConfig(t, root, `theme = "mocha"`)
		work := filepath.Join(root, "work")
		workspaceDir := filepath.Join(work, loader.WorkspaceDirName)
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(workspaceDir, 0o755)
		assert.NoError(t, err)
		workspace := filepath.Join(workspaceDir, "config.toml")
		err = os.WriteFile(workspace, []byte(`theme = "dracula"`), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_DATA_HOME", t.TempDir())

		raw, ok := config.LoadRawConfigForWorkspace(
			global, workspace, work,
		)

		assert.True(t, ok)
		assert.Equal(t, "mocha", raw["theme"])
	})

	t.Run("insecure global enables workspace config", func(t *testing.T) {
		root := t.TempDir()
		global := writeConfig(t, root, `
[editor]
insecure = true
`)
		work := filepath.Join(root, "work")
		workspaceDir := filepath.Join(work, loader.WorkspaceDirName)
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(workspaceDir, 0o755)
		assert.NoError(t, err)
		workspace := filepath.Join(workspaceDir, "config.toml")
		err = os.WriteFile(workspace, []byte(`theme = "dracula"`), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_DATA_HOME", t.TempDir())

		raw, ok := config.LoadRawConfigForWorkspace(
			global, workspace, work,
		)

		assert.True(t, ok)
		assert.Equal(t, "dracula", raw["theme"])
	})
}

func TestLineEndingConfig(t *testing.T) {
	t.Run("native resolves to platform line ending", func(t *testing.T) {
		var le core.LineEnding
		assert.NoError(t, le.UnmarshalText([]byte("native")))
		assert.Equal(t, core.NativeLineEnding(), le)
	})

	t.Run("invalid value returns typed error", func(t *testing.T) {
		var le core.LineEnding
		err := le.UnmarshalText([]byte("bogus"))
		assert.True(t, errors.Is(err, core.ErrInvalidLineEnding))
	})
}

func TestWorkspaceConfigPath(t *testing.T) {
	t.Run("uses nearest workspace marker", func(t *testing.T) {
		root := t.TempDir()
		work := filepath.Join(root, "work")
		nested := filepath.Join(work, "a", "b")
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(nested, 0o755)
		assert.NoError(t, err)

		path := loader.WorkspaceConfigFile(nested)

		assert.Equal(t,
			filepath.Join(work, loader.WorkspaceDirName, "config.toml"),
			path)
	})

	t.Run("falls back to start directory", func(t *testing.T) {
		root := t.TempDir()
		err := os.MkdirAll(filepath.Join(root, loader.WorkspaceDirName), 0o755)
		assert.NoError(t, err)

		path := loader.WorkspaceConfigFile(root)

		assert.Equal(t,
			filepath.Join(root, loader.WorkspaceDirName, "config.toml"),
			path)
	})
}

func TestLogFilePath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)

	path, ok := loader.LogFile()

	assert.True(t, ok)
	assert.Equal(t,
		filepath.Join(root, loader.DirName, loader.LogFileName),
		path)
}

func TestWorkspaceTrust(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	cwd := filepath.Join(work, "src")
	err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(cwd, 0o755)
	assert.NoError(t, err)
	dataRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataRoot)

	err = loader.TrustWorkspace(cwd)
	assert.NoError(t, err)

	trusted := loader.QueryWorkspaceTrust(cwd, false) == loader.TrustTrusted
	assert.True(t, trusted)

	err = loader.UntrustWorkspace(cwd)
	assert.NoError(t, err)

	trusted = loader.QueryWorkspaceTrust(cwd, false) == loader.TrustTrusted
	assert.False(t, trusted)
}

func TestTextFormat(t *testing.T) {
	t.Run("uses soft wrap config", func(t *testing.T) {
		sw := language.SoftWrap{}
		sw.Enable = new(true)
		sw.WrapIndicator = new("» ")
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		format := language.TextFormatForConfig(
			language.LoadLanguage("go"), nil, sw, 80,
		)

		assert.True(t, format.SoftWrap)
		assert.Equal(t, "» ", format.WrapIndicator)
	})

	t.Run("wraps at configured width", func(t *testing.T) {
		sw := language.SoftWrap{}
		sw.Enable = new(true)
		sw.WrapAtTextWidth = new(true)
		sw.MaxWrap = new(20)
		sw.MaxIndentRetain = new(40)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		format := language.TextFormatForConfig(
			language.LoadLanguage("go"), new(40), sw, 80,
		)

		assert.True(t, format.SoftWrapAtTextWidth)
		assert.Equal(t, 40, format.ViewportWidth)
		assert.Equal(t, 10, format.MaxWrap)
		assert.Equal(t, 16, format.MaxIndentRetain)
	})

	t.Run("ignores wide text width", func(t *testing.T) {
		sw := language.SoftWrap{}
		sw.Enable = new(true)
		sw.WrapAtTextWidth = new(true)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		format := language.TextFormatForConfig(
			language.LoadLanguage("go"), new(80), sw, 40,
		)

		assert.False(t, format.SoftWrapAtTextWidth)
		assert.Equal(t, 40, format.ViewportWidth)
	})

}

func TestDetectLanguage(t *testing.T) {
	t.Run("uses language file type before fallback", func(t *testing.T) {
		root := t.TempDir()
		setUserLanguages(t, root, `
[[language]]
name = "custom"
file-types = ["foo"]
`)

		lang, ok := language.DetectLanguage(filepath.Join(root, "main.foo"), "")

		assert.True(t, ok)
		assert.Equal(t, "custom", lang)
	})

	t.Run("uses glob before extension", func(t *testing.T) {
		root := t.TempDir()
		setUserLanguages(t, root, `
[[language]]
name = "extension"
file-types = ["conf"]

[[language]]
name = "glob"
file-types = [{ glob = "special.conf" }]
`)

		lang, ok := language.DetectLanguage(
			filepath.Join(root, "special.conf"), "",
		)

		assert.True(t, ok)
		assert.Equal(t, "glob", lang)
	})

	t.Run("uses longest matching glob", func(t *testing.T) {
		root := t.TempDir()
		setUserLanguages(t, root, `
[[language]]
name = "short"
file-types = [{ glob = "src/*.conf" }]

[[language]]
name = "long"
file-types = [{ glob = "project/src/*.conf" }]
`)
		path := filepath.Join(root, "project", "src", "special.conf")

		lang, ok := language.DetectLanguage(path, "")

		assert.True(t, ok)
		assert.Equal(t, "long", lang)
	})

	t.Run("matches brace glob", func(t *testing.T) {
		root := t.TempDir()
		setUserLanguages(t, root, `
[[language]]
name = "config"
file-types = [
  { glob = "my{t,j}sconfig.json" },
  { glob = "{custom-app,custom-addon}/{parts,views}/*.hbs" },
]
`)

		langName, ok := language.DetectLanguage(
			filepath.Join(root, "custom-addon", "views", "main.hbs"), "",
		)

		assert.True(t, ok)
		assert.Equal(t, "config", langName)
		langName, ok = language.DetectLanguage(
			filepath.Join(root, "myjsconfig.json"), "",
		)
		assert.True(t, ok)
		assert.Equal(t, "config", langName)
	})

	t.Run("matches extension brace glob", func(t *testing.T) {
		root := t.TempDir()
		setUserLanguages(t, root, `
[[language]]
name = "conf"
file-types = [{ glob = "myconf/*/*.{inc,conf}" }]
`)
		path := filepath.Join(root, "myconf", "machine", "default.inc")

		lang, ok := language.DetectLanguage(path, "")

		assert.True(t, ok)
		assert.Equal(t, "conf", lang)
	})

	t.Run("uses shebang after filename miss", func(t *testing.T) {
		root := t.TempDir()
		setUserLanguages(t, root, `
[[language]]
name = "script"
shebangs = ["customsh"]
`)

		lang, ok := language.DetectLanguage(
			"no-language-match", "#!/usr/bin/env customsh\n",
		)

		assert.True(t, ok)
		assert.Equal(t, "script", lang)
	})

	t.Run("uses injection regex after misses", func(t *testing.T) {
		root := t.TempDir()
		setUserLanguages(t, root, `
[[language]]
name = "short"
injection-regex = "foo"

[[language]]
name = "long"
injection-regex = "foo-bar"
`)

		lang, ok := language.DetectLanguage(
			"no-language-match", "embedded foo-bar marker",
		)

		assert.True(t, ok)
		assert.Equal(t, "long", lang)
	})
}

func TestConfigPaths(t *testing.T) {
	t.Run("path under XDG_CONFIG_HOME", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", root)
		path, ok := loader.ConfigFile()
		assert.True(t, ok)
		assert.Contains(t, path, root)
	})

	t.Run("IgnorePath returns non-empty string", func(t *testing.T) {
		assert.NotEmpty(t, loader.ConfigIgnoreFile())
	})
}

func TestLoadRawUserConfig(t *testing.T) {
	t.Run("returns false when no config", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		_, ok := config.LoadRawUserConfig()
		assert.False(t, ok)
	})

	t.Run("returns merged config", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `theme = "nord"`)
		t.Setenv("XDG_CONFIG_HOME", root)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		m, ok := config.LoadRawUserConfig()
		assert.True(t, ok)
		assert.Equal(t, "nord", m["theme"])
	})
}

func TestAutoSaveUnmarshal(t *testing.T) {
	t.Run("bool true sets FocusLost", func(t *testing.T) {
		var a config.AutoSave
		assert.NoError(t, a.UnmarshalTOML(true))
		assert.True(t, *a.FocusLost)
	})

	t.Run("bool false sets FocusLost false", func(t *testing.T) {
		var a config.AutoSave
		assert.NoError(t, a.UnmarshalTOML(false))
		assert.False(t, *a.FocusLost)
	})

	t.Run("map sets focus-lost and after-delay", func(t *testing.T) {
		var a config.AutoSave
		timeout := int64(2000)
		err := a.UnmarshalTOML(map[string]any{
			"focus-lost": true,
			"after-delay": map[string]any{
				"enable":  true,
				"timeout": timeout,
			},
		})
		assert.NoError(t, err)
		assert.True(t, *a.FocusLost)
		assert.True(t, *a.AfterDelay.Enable)
	})

	t.Run("non-map after-delay is empty", func(t *testing.T) {
		var a config.AutoSave
		err := a.UnmarshalTOML(map[string]any{
			"after-delay": "bad",
		})

		assert.NoError(t, err)
		assert.Nil(t, a.AfterDelay.Enable)
		assert.Nil(t, a.AfterDelay.Timeout)
	})

	t.Run("nil input returns error", func(t *testing.T) {
		var a config.AutoSave
		assert.Error(t, a.UnmarshalTOML(nil))
	})

	t.Run("invalid type returns error", func(t *testing.T) {
		var a config.AutoSave
		assert.Error(t, a.UnmarshalTOML("bad"))
	})
}

func writeLanguages(t *testing.T, root, text string) string {
	t.Helper()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	path := filepath.Join(dir, "languages.toml")
	err = os.WriteFile(path, []byte(text), 0o644)
	assert.NoError(t, err)
	return path
}

func findLanguage(
	langs language.Languages, name string,
) (language.Language, bool) {
	var found language.Language
	ok := false
	for _, lang := range langs.Languages {
		if lang.Name == name {
			found = lang
			ok = true
		}
	}
	return found, ok
}

func setUserLanguages(t *testing.T, root, text string) {
	t.Helper()
	writeLanguages(t, root, text)
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeConfig(t *testing.T, root, text string) string {
	t.Helper()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	path := filepath.Join(dir, "config.toml")
	err = os.WriteFile(path, []byte(text), 0o644)
	assert.NoError(t, err)
	return path
}
