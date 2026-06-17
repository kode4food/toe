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

		langs, ok := config.LoadLanguages(path)

		assert.True(t, ok)
		assert.Equal(t, "markdown", langs.Languages[0].Name)
		assert.Equal(t, "markdown", langs.Languages[0].LanguageID)
		assert.Equal(t, "source.md", langs.Languages[0].Scope)
		assert.Equal(t, "marksman", langs.Languages[0].LanguageServers[0].Name)
		filtered := langs.Languages[0].LanguageServers[1]
		assert.Equal(t, "mdox", filtered.Name)
		assert.Equal(t,
			config.LanguageServerFeature("hover"), filtered.Only[0],
		)
		assert.Equal(t,
			config.LanguageServerFeature("format"), filtered.Excluded[0],
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
		_, ok := config.LoadLanguages(
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

		langs, ok := config.LoadLanguagesForWorkspace(global, workspace, work)

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

		langs, ok := config.LoadLanguagesForWorkspace(global, workspace, work)

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

		langs, ok := config.LoadLanguagesForWorkspace(global, workspace, root)

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

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.Equal(t, 72, *cfg.Editor.TextWidth)
		assert.True(t, *cfg.Editor.SoftWrap.Enable)
		assert.Equal(t, 18, *cfg.Editor.SoftWrap.MaxWrap)
		assert.Equal(t, 24, *cfg.Editor.SoftWrap.MaxIndentRetain)
		assert.Equal(t, "» ", *cfg.Editor.SoftWrap.WrapIndicator)
		assert.True(t, *cfg.Editor.SoftWrap.WrapAtTextWidth)
	})

	t.Run("loads editor auto pair bool", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
auto-pairs = false
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		_, ok = cfg.AutoPairs()
		assert.False(t, ok)
	})

	t.Run("loads editor auto pair table", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.auto-pairs]
'(' = ')'
'<' = '>'
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		pairs, ok := cfg.AutoPairs()
		assert.True(t, ok)
		pair, ok := pairs.Get('<')
		assert.True(t, ok)
		assert.Equal(t, core.Pair{Open: '<', Close: '>'}, pair)
	})

	t.Run("loads editor auto save bool", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
auto-save = true
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.True(t, cfg.AutoSaveFocusLost())
		assert.False(t, cfg.AutoSaveAfterDelay())
		assert.Equal(t, config.DefaultAutoSaveDelay,
			cfg.AutoSaveDelayTimeout(),
		)
	})

	t.Run("loads editor auto save table", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.auto-save]
focus-lost = true

[editor.auto-save.after-delay]
enable = true
timeout = 1200
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.True(t, cfg.AutoSaveFocusLost())
		assert.True(t, cfg.AutoSaveAfterDelay())
		assert.Equal(t, 1200, cfg.AutoSaveDelayTimeout())
	})

	t.Run("loads cursor shape settings", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.cursor-shape]
normal = "block"
insert = "bar"
select = "underline"
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.Equal(t, config.CursorKindBlock,
			cfg.CursorShapeForMode("normal"),
		)
		assert.Equal(t, config.CursorKindBar,
			cfg.CursorShapeForMode("insert"),
		)
		assert.Equal(t, config.CursorKindUnderline,
			cfg.CursorShapeForMode("select"),
		)
	})

	t.Run("rejects invalid cursor shape", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.cursor-shape]
normal = "bogus"
`)

		_, ok := config.LoadConfig(path)

		assert.False(t, ok)
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

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.Equal(t, "|", cfg.StatusLineSeparator())
		assert.Equal(t, []config.StatusLineElement{
			config.StatusLineMode,
			config.StatusLineFileName,
		}, cfg.StatusLineLeft())
		assert.Equal(t, []config.StatusLineElement{
			config.StatusLineSpacer,
		}, cfg.StatusLineCenter())
		assert.Equal(t, []config.StatusLineElement{
			config.StatusLinePosition,
			config.StatusLineFileEncoding,
		}, cfg.StatusLineRight())
		assert.Equal(t, "NORMAL", cfg.ModeNameForMode("normal"))
		assert.Equal(t, "INSERT", cfg.ModeNameForMode("insert"))
		assert.Equal(t, "SELECT", cfg.ModeNameForMode("select"))
	})

	t.Run("rejects invalid statusline element", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.statusline]
left = ["bogus"]
`)

		_, ok := config.LoadConfig(path)

		assert.False(t, ok)
	})

	t.Run("loads editor search settings", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.search]
smart-case = false
wrap-around = false
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.False(t, cfg.SearchSmartCase())
		assert.False(t, cfg.SearchWrapAround())
	})

	t.Run("loads default line ending", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
default-line-ending = "crlf"
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.Equal(t, core.LineEndingCRLF,
			cfg.Editor.DefaultLineEnding,
		)
	})

	t.Run("rejects invalid default line ending", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
default-line-ending = "bogus"
`)

		_, ok := config.LoadConfig(path)

		assert.False(t, ok)
	})

	t.Run("uses save option defaults", func(t *testing.T) {
		cfg := config.DefaultConfig()

		assert.True(t, cfg.InsertFinalNewline())
		assert.False(t, cfg.TrimFinalNewlines())
		assert.False(t, cfg.TrimTrailingWhitespace())
		assert.True(t, cfg.AtomicSave())
		assert.True(t, cfg.ContinueComments())
		assert.Equal(t, config.CursorKindBlock,
			cfg.CursorShapeForMode("insert"),
		)
		assert.Equal(t, "NOR", cfg.ModeNameForMode("normal"))
		assert.Equal(t, "│", cfg.StatusLineSeparator())
		assert.Equal(t, []config.StatusLineElement{
			config.StatusLineMode,
			config.StatusLineSpinner,
			config.StatusLineFileName,
			config.StatusLineReadOnly,
			config.StatusLineModified,
		}, cfg.StatusLineLeft())
		assert.Empty(t, cfg.StatusLineCenter())
		assert.Equal(t, []config.StatusLineElement{
			config.StatusLineDiagnostics,
			config.StatusLineSelections,
			config.StatusLineRegister,
			config.StatusLinePosition,
			config.StatusLineFileEncoding,
		}, cfg.StatusLineRight())
	})

	t.Run("loads save options", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
insert-final-newline = false
trim-final-newlines = true
trim-trailing-whitespace = true
atomic-save = false
insecure = true
editor-config = false
continue-comments = false
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.False(t, cfg.InsertFinalNewline())
		assert.True(t, cfg.TrimFinalNewlines())
		assert.True(t, cfg.TrimTrailingWhitespace())
		assert.False(t, cfg.AtomicSave())
		assert.True(t, cfg.Insecure())
		assert.False(t, cfg.EditorConfig())
		assert.False(t, cfg.ContinueComments())
	})

	t.Run("merges trusted workspace config", func(t *testing.T) {
		root := t.TempDir()
		global := writeConfig(t, root, `
[editor]
text-width = 80

[editor.soft-wrap]
enable = true
wrap-indicator = "↪ "
`)
		work := filepath.Join(root, "work")
		workspaceDir := filepath.Join(work,
			loader.WorkspaceDirName,
		)
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(workspaceDir, 0o755)
		assert.NoError(t, err)
		workspace := filepath.Join(workspaceDir, "config.toml")
		err = os.WriteFile(workspace, []byte(`
[editor]
text-width = 72

[editor.soft-wrap]
wrap-indicator = "» "
`), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		err = config.TrustWorkspace(work)
		assert.NoError(t, err)

		cfg, ok := config.LoadConfigForWorkspace(global, workspace, work)

		assert.True(t, ok)
		assert.Equal(t, 72, *cfg.Editor.TextWidth)
		assert.True(t, *cfg.Editor.SoftWrap.Enable)
		assert.Equal(t, "» ", *cfg.Editor.SoftWrap.WrapIndicator)
	})

	t.Run("ignores untrusted workspace config", func(t *testing.T) {
		root := t.TempDir()
		global := writeConfig(t, root, `
[editor]
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
		workspace := filepath.Join(workspaceDir, "config.toml")
		err = os.WriteFile(workspace, []byte(`
[editor]
text-width = 72
`), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_DATA_HOME", t.TempDir())

		cfg, ok := config.LoadConfigForWorkspace(global, workspace, work)

		assert.True(t, ok)
		assert.Equal(t, 80, *cfg.Editor.TextWidth)
	})

	t.Run("insecure global enables workspace config", func(t *testing.T) {
		root := t.TempDir()
		global := writeConfig(t, root, `
[editor]
text-width = 80
insecure = true
`)
		work := filepath.Join(root, "work")
		workspaceDir := filepath.Join(work,
			loader.WorkspaceDirName,
		)
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(workspaceDir, 0o755)
		assert.NoError(t, err)
		workspace := filepath.Join(workspaceDir, "config.toml")
		err = os.WriteFile(workspace, []byte(`
[editor]
text-width = 72
`), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_DATA_HOME", t.TempDir())

		cfg, ok := config.LoadConfigForWorkspace(global, workspace, work)

		assert.True(t, ok)
		assert.Equal(t, 72, *cfg.Editor.TextWidth)
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

		lang := config.LoadLanguageForScope("source.custom")

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
		lang := config.Language{Roots: []string{"go.mod", "go.work"}}

		found, ok := config.FindLanguageRoot(
			filepath.Join(nested, "main.go"), &lang,
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
		lang := config.Language{Roots: []string{"*.sln"}}

		found, ok := config.FindLanguageRoot(project, &lang)

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
		lang := config.Language{Roots: []string{"{package,project}.json"}}

		found, ok := config.FindLanguageRoot(project, &lang)

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
		lang := config.Language{Roots: []string{"go.mod", "go.work"}}

		found, ok := config.FindLSPWorkspace(config.LSPWorkspaceArgs{
			File: filepath.Join(nested, "main.go"), Language: &lang,
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
		lang := config.Language{Roots: []string{"missing"}}

		found, ok := config.FindLSPWorkspace(config.LSPWorkspaceArgs{
			File: filepath.Join(tool, "main.go"), Language: &lang,
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
		lang := config.Language{Roots: []string{"missing"}}

		found, ok := config.FindLSPWorkspace(config.LSPWorkspaceArgs{
			File: filepath.Join(project, "main.go"), Language: &lang,
			Workspace: project, WorkspaceFallback: true,
		})

		assert.True(t, ok)
		assert.Equal(t, project, found)
	})
}

func TestWhitespaceConfig(t *testing.T) {
	t.Run("defaults to none for all character types", func(t *testing.T) {
		cfg := config.DefaultConfig()
		ws := cfg.Whitespace()

		assert.Equal(t, config.WhitespaceRenderNone, ws.Render.SpaceRender())
		assert.Equal(t, config.WhitespaceRenderNone, ws.Render.TabRender())
		assert.Equal(t, config.WhitespaceRenderNone, ws.Render.NewlineRender())
		assert.Equal(t, config.WhitespaceRenderNone, ws.Render.NbspRender())
		assert.Equal(t, config.WhitespaceRenderNone, ws.Render.NnbspRender())
	})

	t.Run("defaults render characters match reference", func(t *testing.T) {
		cfg := config.DefaultConfig()
		chars := cfg.Whitespace().Characters

		assert.Equal(t, config.DefaultWSSpace, chars.SpaceRune())
		assert.Equal(t, config.DefaultWSTab, chars.TabRune())
		assert.Equal(t, config.DefaultWSTabpad, chars.TabpadRune())
		assert.Equal(t, config.DefaultWSNewline, chars.NewlineRune())
		assert.Equal(t, config.DefaultWSNbsp, chars.NbspRune())
		assert.Equal(t, config.DefaultWSNnbsp, chars.NnbspRune())
	})

	t.Run("loads basic render value from toml", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.whitespace]
render = "all"
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		ws := cfg.Whitespace()
		assert.Equal(t, config.WhitespaceRenderAll, ws.Render.SpaceRender())
		assert.Equal(t, config.WhitespaceRenderAll, ws.Render.TabRender())
		assert.Equal(t, config.WhitespaceRenderAll, ws.Render.NewlineRender())
	})

	t.Run("loads specific render values from toml", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.whitespace.render]
default = "all"
space = "none"
tab = "all"
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		ws := cfg.Whitespace()
		assert.Equal(t, config.WhitespaceRenderNone, ws.Render.SpaceRender())
		assert.Equal(t, config.WhitespaceRenderAll, ws.Render.TabRender())
		assert.Equal(t, config.WhitespaceRenderAll, ws.Render.NewlineRender())
	})

	t.Run("loads custom characters from toml", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.whitespace.characters]
space = "."
tab = ">"
newline = "$"
tabpad = "-"
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		chars := cfg.Whitespace().Characters
		assert.Equal(t, '.', chars.SpaceRune())
		assert.Equal(t, '>', chars.TabRune())
		assert.Equal(t, '$', chars.NewlineRune())
		assert.Equal(t, '-', chars.TabpadRune())
	})
}

func TestGutterConfig(t *testing.T) {
	t.Run("defaults include line-numbers", func(t *testing.T) {
		cfg := config.DefaultConfig()
		g := cfg.Gutters()

		assert.True(t, g.HasGutterType(config.GutterTypeLineNumbers))
		assert.Equal(t,
			config.DefaultGutterLineNumberMinWidth, g.LineNumberMinWidth(),
		)
	})

	t.Run("loads layout as array", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
gutters = ["line-numbers", "spacer", "diff"]
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		g := cfg.Gutters()
		assert.True(t, g.HasGutterType(config.GutterTypeLineNumbers))
		assert.True(t, g.HasGutterType(config.GutterTypeSpacer))
		assert.True(t, g.HasGutterType(config.GutterTypeDiff))
		assert.False(t, g.HasGutterType(config.GutterTypeDiagnostics))
	})

	t.Run("loads layout as table with line-numbers config", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.gutters]
layout = ["line-numbers"]

[editor.gutters.line-numbers]
min-width = 5
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		g := cfg.Gutters()
		assert.True(t, g.HasGutterType(config.GutterTypeLineNumbers))
		assert.False(t, g.HasGutterType(config.GutterTypeSpacer))
		assert.Equal(t, 5, g.LineNumberMinWidth())
	})

	t.Run("empty layout hides line numbers", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
gutters = []
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		g := cfg.Gutters()
		assert.False(t, g.HasGutterType(config.GutterTypeLineNumbers))
	})
}

func TestIndentGuidesConfig(t *testing.T) {
	t.Run("defaults to disabled with reference character", func(t *testing.T) {
		cfg := config.DefaultConfig()
		ig := cfg.IndentGuides()

		assert.False(t, ig.Render)
		assert.Equal(t, config.DefaultIndentGuideChar, ig.CharRune())
		assert.Equal(t, 0, ig.GetSkipLevels())
	})

	t.Run("loads from toml", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor.indent-guides]
render = true
character = "┊"
skip-levels = 1
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		ig := cfg.IndentGuides()
		assert.True(t, ig.Render)
		assert.Equal(t, '┊', ig.CharRune())
		assert.Equal(t, 1, ig.GetSkipLevels())
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

		format := config.TextFormatForLanguage("go", 80)

		assert.False(t, format.SoftWrap)
		assert.Equal(t, "↪ ", format.WrapIndicator)
		assert.Equal(t, 20, format.MaxWrap)
		assert.Equal(t, 32, format.MaxIndentRetain)
	})

	t.Run("markdown defaults to soft wrap", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		format := config.TextFormatForLanguage("markdown", 80)

		assert.True(t, format.SoftWrap)
		assert.Equal(t, "↪ ", format.WrapIndicator)
		assert.Equal(t, 20, format.MaxWrap)
		assert.Equal(t, 32, format.MaxIndentRetain)
	})

	t.Run("uses editor soft wrap config", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `
[editor.soft-wrap]
enable = true
wrap-indicator = "» "
`)
		t.Setenv("XDG_CONFIG_HOME", root)

		format := config.TextFormatForLanguage("markdown", 80)

		assert.True(t, format.SoftWrap)
		assert.Equal(t, "» ", format.WrapIndicator)
	})

	t.Run("lang soft wrap config overrides editor config", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `
[editor.soft-wrap]
enable = true
wrap-indicator = "» "
`)
		writeLanguages(t, root, `
[[language]]
name = "markdown"
soft-wrap.enable = false
soft-wrap.wrap-indicator = "↳ "
`)
		t.Setenv("XDG_CONFIG_HOME", root)

		format := config.TextFormatForLanguage("markdown", 80)

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

		format := config.TextFormatForLanguage("markdown", 10)

		assert.False(t, format.SoftWrap)
	})

	t.Run("wraps at width when configured and narrower", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `
[editor]
text-width = 40

[editor.soft-wrap]
enable = true
wrap-at-text-width = true
max-wrap = 20
max-indent-retain = 40
`)
		t.Setenv("XDG_CONFIG_HOME", root)

		format := config.TextFormatForLanguage("markdown", 80)

		assert.True(t, format.SoftWrapAtTextWidth)
		assert.Equal(t, 40, format.ViewportWidth)
		assert.Equal(t, 10, format.MaxWrap)
		assert.Equal(t, 16, format.MaxIndentRetain)
	})

	t.Run("ignores text width when viewport is narrower", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `
[editor]
text-width = 80

[editor.soft-wrap]
enable = true
wrap-at-text-width = true
`)
		t.Setenv("XDG_CONFIG_HOME", root)

		format := config.TextFormatForLanguage("markdown", 40)

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

		lang, ok := config.DetectLanguage(filepath.Join(root, "main.foo"), "")

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

		lang, ok := config.DetectLanguage(
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

		lang, ok := config.DetectLanguage(path, "")

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

		lang, ok := config.DetectLanguage(
			filepath.Join(root, "custom-addon", "views", "main.hbs"), "",
		)

		assert.True(t, ok)
		assert.Equal(t, "config", lang)
		lang, ok = config.DetectLanguage(
			filepath.Join(root, "myjsconfig.json"), "",
		)
		assert.True(t, ok)
		assert.Equal(t, "config", lang)
	})

	t.Run("matches extension brace glob", func(t *testing.T) {
		root := t.TempDir()
		setUserLanguages(t, root, `
[[language]]
name = "conf"
file-types = [{ glob = "myconf/*/*.{inc,conf}" }]
`)
		path := filepath.Join(root, "myconf", "machine", "default.inc")

		lang, ok := config.DetectLanguage(path, "")

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

		lang, ok := config.DetectLanguage(
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

		lang, ok := config.DetectLanguage(
			"no-language-match", "embedded foo-bar marker",
		)

		assert.True(t, ok)
		assert.Equal(t, "long", lang)
	})
}

func TestBufferLine(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := config.DefaultConfig()

		assert.Equal(t, config.BufferLineNever, cfg.GetBufferLine())
	})

	t.Run("loads from toml", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
bufferline = "always"
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.Equal(t, config.BufferLineAlways, cfg.GetBufferLine())
	})
}

func TestEditorMiscOptions(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := config.DefaultConfig()

		assert.False(t, cfg.Cursorcolumn())
		assert.True(t, cfg.Mouse())
	})

	t.Run("loads from toml", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
cursorcolumn = true
mouse = false
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.True(t, cfg.Cursorcolumn())
		assert.False(t, cfg.Mouse())
	})
}

func TestRulersConfig(t *testing.T) {
	t.Run("defaults to empty", func(t *testing.T) {
		cfg := config.DefaultConfig()

		assert.Empty(t, cfg.Rulers())
	})

	t.Run("loads from toml", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
rulers = [80, 120]
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.Equal(t, []int{80, 120}, cfg.Rulers())
	})

}

func TestShellConfig(t *testing.T) {
	t.Run("defaults to sh -c", func(t *testing.T) {
		cfg := config.DefaultConfig()

		sh := cfg.Shell()

		assert.Equal(t, "sh", sh[0])
		assert.Equal(t, "-c", sh[1])
	})

	t.Run("loads shell from toml", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), `
[editor]
shell = ["bash", "--norc", "-c"]
`)

		cfg, ok := config.LoadConfig(path)

		assert.True(t, ok)
		assert.Equal(t, []string{"bash", "--norc", "-c"}, cfg.Shell())
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

func findLanguage(langs config.Languages, name string) (config.Language, bool) {
	var found config.Language
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
