package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

func TestLanguages(t *testing.T) {
	t.Run("loads soft wrap settings", func(t *testing.T) {
		path := writeLanguages(t, t.TempDir(), `
use-grammars = { except = ["skip"] }

[language-server]
marksman = { command = "marksman", args = ["server"], timeout = 30 }
marksman.required-root-patterns = ["marksman.toml"]

[language-server.marksman.environment]
RUST_LOG = "debug"

[language-server.marksman.config.markdown]
filetypes = ["md"]

[[grammar]]
name = "markdown"
source.git = "https://example.test/markdown"
source.rev = "abc"
source.subpath = "tree-sitter-markdown"

[[grammar]]
name = "skip"
source = { path = "../skip" }

[[language]]
name = "markdown"
language-id = "markdown"
scope = "source.md"
language-servers = [
  "marksman",
  { name = "mdox", only-features = ["hover"], except-features = ["format"] },
]
file-types = ["md", { glob = "README" }]
shebangs = ["markdown"]
roots = ["marksman.toml"]
injection-regex = "md|markdown"
comment-token = "//"
block-comment-tokens = { start = "/*", end = "*/" }
indent = { tab-width = 8, unit = "  " }
text-width = 72
auto-format = true
auto-pairs = { '<' = '>' }
formatter = { command = "markdownfmt", args = ["--stdin"] }
soft-wrap.enable = true
soft-wrap.max-wrap = 18
soft-wrap.max-indent-retain = 24
soft-wrap.wrap-indicator = "» "
soft-wrap.wrap-at-text-width = true

[language.debugger]
name = "markdown-debug"
transport = "stdio"
command = "md-debug"
args = ["--stdio"]
port-arg = "--port"

[language.debugger.quirks]
absolute-paths = true

[[language.debugger.templates]]
name = "launch"
request = "launch"
completion = [
  "pid",
  { name = "file", completion = "filename", default = "README.md" },
]
args.program = "{0}"
`)

		langs, ok := language.LoadLanguages(path)

		assert.True(t, ok)
		assert.Equal(t, "markdown", langs.Languages[0].Name)
		assert.Equal(t, "markdown", langs.Languages[0].LanguageID)
		assert.Equal(t, "source.md", langs.Languages[0].Scope)
		assert.Equal(t, "marksman", langs.Languages[0].LanguageServers[0].Name)
		filtered := langs.Languages[0].LanguageServers[1]
		assert.Equal(t, "mdox", filtered.Name)
		assert.Equal(t,
			language.ServerFeature("hover"), filtered.Only[0],
		)
		assert.Equal(t,
			language.ServerFeature("format"), filtered.Excluded[0],
		)
		assert.Equal(t, "md", langs.Languages[0].FileTypes[0].Extension)
		assert.Equal(t, "*/README", langs.Languages[0].FileTypes[1].Glob)
		assert.Equal(t, "markdown", langs.Languages[0].Shebangs[0])
		assert.Equal(t, "marksman.toml", langs.Languages[0].Roots[0])
		assert.Equal(t, "md|markdown", langs.Languages[0].InjectionRegex)
		assert.Equal(t, "//", langs.Languages[0].CommentTokens[0])
		assert.Equal(t, "/*", langs.Languages[0].BlockCommentTokens[0].Start)
		assert.Equal(t, "*/", langs.Languages[0].BlockCommentTokens[0].End)
		assert.Equal(t, 8, *langs.Languages[0].Indent.TabWidth)
		assert.Equal(t, "  ", langs.Languages[0].Indent.Unit)
		assert.True(t, *langs.Languages[0].AutoFormat)
		assert.Equal(t, "markdownfmt", langs.Languages[0].Formatter.Command)
		assert.Equal(t, "--stdin", langs.Languages[0].Formatter.Args[0])
		debug := langs.Languages[0].Debugger
		assert.Equal(t, "markdown-debug", debug.Name)
		assert.Equal(t, "stdio", debug.Transport)
		assert.Equal(t, "md-debug", debug.Command)
		assert.Equal(t, "--stdio", debug.Args[0])
		assert.Equal(t, "--port", debug.PortArg)
		assert.True(t, debug.Quirks.AbsolutePaths)
		assert.Equal(t, "launch", debug.Templates[0].Name)
		assert.Equal(t, "launch", debug.Templates[0].Request)
		assert.Equal(t, "pid", debug.Templates[0].Completion[0].Name)
		assert.Equal(t, "file", debug.Templates[0].Completion[1].Name)
		assert.Equal(t, "filename",
			debug.Templates[0].Completion[1].Completion,
		)
		assert.Equal(t, "README.md",
			debug.Templates[0].Completion[1].Default,
		)
		assert.Equal(t, "{0}", debug.Templates[0].Args["program"])
		pairs, ok := langs.Languages[0].AutoPairs.AutoPairs()
		assert.True(t, ok)
		pair, ok := pairs.Get('<')
		assert.True(t, ok)
		assert.Equal(t, core.Pair{Open: '<', Close: '>'}, pair)
		assert.Equal(t, 72, *langs.Languages[0].TextWidth)
		assert.True(t, *langs.Languages[0].SoftWrap.Enable)
		assert.Equal(t, 18, *langs.Languages[0].SoftWrap.MaxWrap)
		assert.Equal(t, 24, *langs.Languages[0].SoftWrap.MaxIndentRetain)
		assert.Equal(t, "» ", *langs.Languages[0].SoftWrap.WrapIndicator)
		assert.True(t, *langs.Languages[0].SoftWrap.WrapAtTextWidth)
		server := langs.LanguageServers["marksman"]
		assert.Equal(t, "marksman", server.Command)
		assert.Equal(t, "server", server.Args[0])
		assert.Equal(t, "debug", server.Environment["RUST_LOG"])
		assert.Equal(t, int64(30), int64(server.Timeout))
		assert.Equal(t, "marksman.toml", server.RequiredRootPatterns[0])
		markdown := server.Config["markdown"].(map[string]any)
		assert.Equal(t, "md", markdown["filetypes"].([]any)[0])
		assert.Equal(t, []string{"skip"}, langs.GrammarSelection.Except)
		assert.Equal(t, "markdown", langs.Grammars[0].Name)
		assert.Equal(t,
			"https://example.test/markdown", langs.Grammars[0].Source.Git,
		)
		assert.Equal(t, "abc", langs.Grammars[0].Source.Rev)
		assert.Equal(t,
			"tree-sitter-markdown", langs.Grammars[0].Source.Subpath,
		)
		assert.Equal(t, "../skip", langs.Grammars[1].Source.Path)
		selected := langs.SelectedGrammars()
		assert.Equal(t, 1, len(selected))
		assert.Equal(t, "markdown", selected[0].Name)
	})

	t.Run("missing file returns false", func(t *testing.T) {
		_, ok := language.LoadLanguages(
			filepath.Join(t.TempDir(), "missing.toml"),
		)
		assert.False(t, ok)
	})

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
		err = config.TrustWorkspace(work)
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

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.Equal(t, "dark", cfg.Theme.Name)
		assert.False(t, cfg.Theme.Adaptive)
	})

	t.Run("loads adaptive theme", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[theme]
light = "light"
dark = "dark"
fallback = "base"
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.True(t, cfg.Theme.Adaptive)
		assert.Equal(t, "light", cfg.Theme.Choose(true))
		assert.Equal(t, "dark", cfg.Theme.Choose(false))
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

		_, ok := config.LoadConfig(path)

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

		_, ok := config.LoadConfig(path)

		assert.True(t, ok)
	})

	t.Run("uses insecure default", func(t *testing.T) {
		cfg := config.DefaultConfig()

		assert.False(t, cfg.Insecure())
	})

	t.Run("loads insecure option", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
insecure = true
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.True(t, cfg.Insecure())
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
		err = config.TrustWorkspace(work)
		assert.NoError(t, err)

		cfg, ok := config.LoadConfigForWorkspace(global, workspace, work)

		assert.True(t, ok)
		assert.Equal(t, "dracula", cfg.Theme.Name)
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

		cfg, ok := config.LoadConfigForWorkspace(global, workspace, work)

		assert.True(t, ok)
		assert.Equal(t, "mocha", cfg.Theme.Name)
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

		cfg, ok := config.LoadConfigForWorkspace(global, workspace, work)

		assert.True(t, ok)
		assert.True(t, cfg.Insecure())
		assert.Equal(t, "dracula", cfg.Theme.Name)
	})
}

func TestLanguageLookup(t *testing.T) {
	t.Run("loads by scope", func(t *testing.T) {
		root := t.TempDir()
		setUserLanguages(t, root, `
[[language]]
name = "custom"
scope = "source.custom"
`)

		lang := language.LoadLanguageForScope("source.custom")

		assert.Equal(t, "custom", lang.Name)
	})

	t.Run("finds topmost language root marker", func(t *testing.T) {
		root := t.TempDir()
		project := filepath.Join(root, "project")
		nested := filepath.Join(project, "src", "pkg")
		err := os.MkdirAll(nested, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(project, "go.mod"), []byte("module test\n"), 0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(nested, "go.work"), []byte("go 1.26\n"), 0o644,
		)
		assert.NoError(t, err)
		langDef := language.Language{Roots: []string{"go.mod", "go.work"}}

		found, ok := language.FindLanguageRoot(
			filepath.Join(nested, "main.go"), &langDef,
		)

		assert.True(t, ok)
		assert.Equal(t, project, found)
	})

	t.Run("matches glob root marker", func(t *testing.T) {
		root := t.TempDir()
		project := filepath.Join(root, "project")
		err := os.MkdirAll(project, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(project, "app.sln"), []byte(""), 0o644,
		)
		assert.NoError(t, err)
		langDef := language.Language{Roots: []string{"*.sln"}}

		found, ok := language.FindLanguageRoot(project, &langDef)

		assert.True(t, ok)
		assert.Equal(t, project, found)
	})

	t.Run("matches brace root marker", func(t *testing.T) {
		root := t.TempDir()
		project := filepath.Join(root, "project")
		err := os.MkdirAll(project, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(project, "package.json"), []byte("{}"), 0o644,
		)
		assert.NoError(t, err)
		langDef := language.Language{Roots: []string{"{package,project}.json"}}

		found, ok := language.FindLanguageRoot(project, &langDef)

		assert.True(t, ok)
		assert.Equal(t, project, found)
	})

	t.Run("finds lsp workspace with topmost marker", func(t *testing.T) {
		root := t.TempDir()
		project := filepath.Join(root, "project")
		nested := filepath.Join(project, "src", "pkg")
		err := os.MkdirAll(nested, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(project, "go.mod"), []byte("module test\n"), 0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(nested, "go.work"), []byte("go 1.26\n"), 0o644,
		)
		assert.NoError(t, err)
		langDef := language.Language{Roots: []string{"go.mod", "go.work"}}

		found, ok := language.FindLSPWorkspace(language.LSPWorkspaceArgs{
			File: filepath.Join(nested, "main.go"), Language: &langDef,
			Workspace: project,
		})

		assert.True(t, ok)
		assert.Equal(t, project, found)
	})

	t.Run("uses lsp root dir ceiling", func(t *testing.T) {
		root := t.TempDir()
		project := filepath.Join(root, "project")
		tool := filepath.Join(project, "tools", "one")
		err := os.MkdirAll(tool, 0o755)
		assert.NoError(t, err)
		langDef := language.Language{Roots: []string{"missing"}}

		found, ok := language.FindLSPWorkspace(language.LSPWorkspaceArgs{
			File: filepath.Join(tool, "main.go"), Language: &langDef,
			RootDirs: []string{"tools"}, Workspace: project,
		})

		assert.True(t, ok)
		assert.Equal(t, project, found)
	})

	t.Run("uses workspace fallback", func(t *testing.T) {
		root := t.TempDir()
		project := filepath.Join(root, "project")
		err := os.MkdirAll(project, 0o755)
		assert.NoError(t, err)
		langDef := language.Language{Roots: []string{"missing"}}

		found, ok := language.FindLSPWorkspace(language.LSPWorkspaceArgs{
			File: filepath.Join(project, "main.go"), Language: &langDef,
			Workspace: project, WorkspaceFallback: true,
		})

		assert.True(t, ok)
		assert.Equal(t, project, found)
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

		path := config.WorkspaceConfigPath(nested)

		assert.Equal(t,
			filepath.Join(work, loader.WorkspaceDirName,
				"config.toml",
			),
			path,
		)
	})

	t.Run("falls back to start directory", func(t *testing.T) {
		root := t.TempDir()
		err := os.MkdirAll(filepath.Join(root, loader.WorkspaceDirName), 0o755)
		assert.NoError(t, err)

		path := config.WorkspaceConfigPath(root)

		assert.Equal(t,
			filepath.Join(root, loader.WorkspaceDirName,
				"config.toml",
			),
			path,
		)
	})
}

func TestLogFilePath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)

	path, ok := config.LogFilePath()

	assert.True(t, ok)
	assert.Equal(t,
		filepath.Join(root, loader.DirName, loader.LogFileName),
		path,
	)
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

	err = config.TrustWorkspace(cwd)

	assert.NoError(t, err)
	path, ok := config.WorkspaceTrustPath()
	assert.True(t, ok)
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, work+"\n", string(data))

	err = config.UntrustWorkspace(cwd)

	assert.NoError(t, err)
	data, err = os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "", string(data))
}

func TestTextFormat(t *testing.T) {
	t.Run("defaults to clipping", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		format := language.TextFormatForLanguage("go", 80)

		assert.False(t, format.SoftWrap)
		assert.Equal(t, "↪ ", format.WrapIndicator)
		assert.Equal(t, 20, format.MaxWrap)
		assert.Equal(t, 32, format.MaxIndentRetain)
	})

	t.Run("markdown defaults to soft wrap", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		format := language.TextFormatForLanguage("markdown", 80)

		assert.True(t, format.SoftWrap)
		assert.Equal(t, "↪ ", format.WrapIndicator)
		assert.Equal(t, 20, format.MaxWrap)
		assert.Equal(t, 32, format.MaxIndentRetain)
	})

	t.Run("lang soft wrap settings apply", func(t *testing.T) {
		root := t.TempDir()
		writeLanguages(t, root, `
[[language]]
name = "markdown"
soft-wrap.enable = false
soft-wrap.wrap-indicator = "↳ "
`)
		t.Setenv("XDG_CONFIG_HOME", root)

		format := language.TextFormatForLanguage("markdown", 80)

		assert.False(t, format.SoftWrap)
		assert.Equal(t, "↳ ", format.WrapIndicator)
	})

	t.Run("disables soft wrap in narrow viewports", func(t *testing.T) {
		root := t.TempDir()
		writeLanguages(t, root, `
[[language]]
name = "markdown"
soft-wrap.enable = true
`)
		t.Setenv("XDG_CONFIG_HOME", root)

		format := language.TextFormatForLanguage("markdown", 10)

		assert.False(t, format.SoftWrap)
	})

	t.Run("uses soft wrap config", func(t *testing.T) {
		sw := language.SoftWrap{}
		sw.Enable = new(true)
		sw.WrapIndicator = new("» ")
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		format := language.TextFormatForLanguageWithConfig("go", nil, sw, 80)

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

		format := language.TextFormatForLanguageWithConfig("go", new(40), sw, 80)

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

		format := language.TextFormatForLanguageWithConfig("go", new(80), sw, 40)

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

func findLanguage(langs language.Languages, name string) (language.Language, bool) {
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
