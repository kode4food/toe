package language_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/language"
)

func TestDecodeLanguage(t *testing.T) {
	t.Run("loads full language definition", func(t *testing.T) {
		path := writeLangToml(t, `
[[language]]
name = "test-lang"
language-id = "testlang"
scope = "source.test"
injection-regex = "test|tst"
file-types = ["test", { glob = "*.test" }]
shebangs = ["testsh"]
roots = ["test.toml"]
comment-token = "//"
block-comment-tokens = { start = "/*", end = "*/" }
indent = { tab-width = 2, unit = "  " }
auto-pairs = { '(' = ')' }
auto-format = true
formatter = { command = "testfmt", args = ["--stdin"] }
soft-wrap.enable = true
soft-wrap.wrap-indicator = ">> "
text-width = 100

[language.debugger]
name = "test-debug"
transport = "stdio"
command = "dbg"
args = ["--debug"]
port-arg = "--port"

[language.debugger.quirks]
absolute-paths = true

[[language.debugger.templates]]
name = "launch"
request = "launch"
completion = [
  "pid",
  { name = "file", completion = "filename", default = "main.test" },
]
args.program = "{0}"

[[language]]
name = "other"
language-servers = [
  "test-server",
  { name = "lsp2", only-features = ["hover"], except-features = ["format"] },
]

[language-server]
test-server = { command = "test-lsp", args = ["--stdio"], timeout = 10 }

[language-server.test-server.environment]
LOG_LEVEL = "debug"

[language-server.test-server.config.test]
key = "value"

[[grammar]]
name = "test-grammar"
source.git = "https://example.test/grammar"
source.rev = "abc123"
source.subpath = "tree-sitter-test"
`)
		langs, ok := language.LoadLanguages(path)

		assert.True(t, ok)
		assert.Len(t, langs.Languages, 2)

		lang := langs.Languages[0]
		assert.Equal(t, "test-lang", lang.Name)
		assert.Equal(t, "testlang", lang.LanguageID)
		assert.Equal(t, "source.test", lang.Scope)
		assert.Equal(t, "test|tst", lang.InjectionRegex)
		assert.Equal(t, "test", lang.FileTypes[0].Extension)
		assert.Contains(t, lang.FileTypes[1].Glob, ".test")
		assert.Equal(t, "testsh", lang.Shebangs[0])
		assert.Equal(t, "test.toml", lang.Roots[0])
		assert.Equal(t, "//", lang.CommentTokens[0])
		assert.Equal(t, "/*", lang.BlockCommentTokens[0].Start)
		assert.Equal(t, "*/", lang.BlockCommentTokens[0].End)
		assert.Equal(t, 2, *lang.Indent.TabWidth)
		assert.Equal(t, "  ", lang.Indent.Unit)
		assert.True(t, *lang.AutoFormat)
		assert.Equal(t, "testfmt", lang.Formatter.Command)
		assert.Equal(t, "--stdin", lang.Formatter.Args[0])
		assert.True(t, *lang.SoftWrap.Enable)
		assert.Equal(t, ">> ", *lang.SoftWrap.WrapIndicator)
		assert.Equal(t, 100, *lang.TextWidth)

		dbg := lang.Debugger
		assert.Equal(t, "test-debug", dbg.Name)
		assert.Equal(t, "stdio", dbg.Transport)
		assert.Equal(t, "dbg", dbg.Command)
		assert.Equal(t, "--debug", dbg.Args[0])
		assert.Equal(t, "--port", dbg.PortArg)
		assert.True(t, dbg.Quirks.AbsolutePaths)
		assert.Equal(t, "launch", dbg.Templates[0].Name)
		assert.Equal(t, "launch", dbg.Templates[0].Request)
		assert.Equal(t, "pid", dbg.Templates[0].Completion[0].Name)
		assert.Equal(t, "file", dbg.Templates[0].Completion[1].Name)
		assert.Equal(t, "filename", dbg.Templates[0].Completion[1].Completion)
		assert.Equal(t, "main.test", dbg.Templates[0].Completion[1].Default)
		assert.Equal(t, "{0}", dbg.Templates[0].Args["program"])

		pairs, ok := lang.AutoPairs.AutoPairs()
		assert.True(t, ok)
		p, ok := pairs.Get('(')
		assert.True(t, ok)
		assert.Equal(t, ')', p.Close)

		other := langs.Languages[1]
		assert.Equal(t, "test-server", other.LanguageServers[0].Name)
		lsp2 := other.LanguageServers[1]
		assert.Equal(t, "lsp2", lsp2.Name)
		assert.Equal(t, language.ServerFeature("hover"), lsp2.Only[0])
		assert.Equal(t, language.ServerFeature("format"), lsp2.Excluded[0])

		srv := langs.LanguageServers["test-server"]
		assert.Equal(t, "test-lsp", srv.Command)
		assert.Equal(t, "--stdio", srv.Args[0])
		assert.Equal(t, "debug", srv.Environment["LOG_LEVEL"])
		assert.Equal(t, 10, srv.Timeout)

		assert.Equal(t, "test-grammar", langs.Grammars[0].Name)
		assert.Equal(t,
			"https://example.test/grammar", langs.Grammars[0].Source.Git)
		assert.Equal(t, "abc123", langs.Grammars[0].Source.Rev)
		assert.Equal(t, "tree-sitter-test", langs.Grammars[0].Source.Subpath)
	})

	t.Run("comment-tokens plural takes precedence", func(t *testing.T) {
		path := writeLangToml(t, `
[[language]]
name = "multi-comment"
comment-token = "//"
comment-tokens = ["#", "//"]
`)
		langs, ok := language.LoadLanguages(path)
		assert.True(t, ok)
		assert.Equal(t, "#", langs.Languages[0].CommentTokens[0])
	})

	t.Run("block-comment-tokens as slice", func(t *testing.T) {
		path := writeLangToml(t, `
[[language]]
name = "multi-block"
block-comment-tokens = [
  { start = "/*", end = "*/" },
  { start = "<!--", end = "-->" },
]
`)
		langs, ok := language.LoadLanguages(path)
		assert.True(t, ok)
		assert.Len(t, langs.Languages[0].BlockCommentTokens, 2)
	})
}

func TestDecodeUseGrammars(t *testing.T) {
	t.Run("use-grammars only selects subset", func(t *testing.T) {
		path := writeLangToml(t, `
use-grammars = { only = ["go"] }

[[language]]
name = "dummy"

[[grammar]]
name = "go"
source.git = "https://example.test/go"
source.rev = "v1"

[[grammar]]
name = "rust"
source.git = "https://example.test/rust"
source.rev = "v2"
`)
		langs, ok := language.LoadLanguages(path)
		assert.True(t, ok)
		selected := langs.SelectedGrammars()
		assert.Len(t, selected, 1)
		assert.Equal(t, "go", selected[0].Name)
	})
}

func TestDecodeGrammarSourcePath(t *testing.T) {
	t.Run("grammar with local path", func(t *testing.T) {
		path := writeLangToml(t, `
[[language]]
name = "dummy"

[[grammar]]
name = "local"
source = { path = "../local-grammar" }
`)
		langs, ok := language.LoadLanguages(path)
		assert.True(t, ok)
		assert.Equal(t, "../local-grammar", langs.Grammars[0].Source.Path)
	})
}
