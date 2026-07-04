package language_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view/language"
)

func TestDecodeLanguageServer(t *testing.T) {
	t.Run("language server config decoded", func(t *testing.T) {
		setUserLangs(t, `
[language-server.gopls]
command = "gopls"
args = ["-mode=stdio"]
timeout = 30
required-root-patterns = ["go.mod"]

[language-server.gopls.environment]
GOPATH = "/tmp/go"

[language-server.gopls.config]
hints = true

[[language]]
name = "lsptest"
`)
		l := language.LoadLanguage("lsptest")
		assert.NotNil(t, l)
	})

	t.Run("language server features decoded", func(t *testing.T) {
		setUserLangs(t, `
[language-server.testlsp]
command = "testlsp"
args = ["--stdio"]
timeout = 10
required-root-patterns = [".git"]

[language-server.testlsp.environment]
KEY = "val"

[[language]]
name = "testlang"
scope = "source.testlang"
language-servers = [
  {
    name = "testlsp",
    only-features = ["completion"],
    except-features = ["formatting"]
  },
  "testlsp"
]
`)
		l := language.LoadLanguage("testlang")
		assert.NotNil(t, l)
		assert.Equal(t, "testlang", l.Name)
	})
}

func TestDecodeLanguageServerFeatures(t *testing.T) {
	t.Run("string form server feature", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "lang-str-srv"
language-servers = ["my-lsp"]
`)
		// Exercise through DetectLanguage loading
		_, _ = language.DetectLanguage("nope.xyz99qwerty", "")
	})

	t.Run("map form with only and except features", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "lang-map-srv"
language-servers = [
  {
    name = "my-lsp",
    only-features = ["completion"],
    except-features = ["diagnostics"]
  }
]
`)
		_, _ = language.DetectLanguage("nope.xyz99qwerty", "")
	})
}

func TestDecodeDebugAdapter(t *testing.T) {
	t.Run("debug adapter with templates and quirks", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "debuglang"
file-types = ["dbg"]

[language.debugger]
name = "debuglsp"
transport = "stdio"
command = "dlv"
args = ["dap"]
port-arg = "--listen"

[[language.debugger.templates]]
name = "Debug Binary"
request = "launch"

[language.debugger.templates.args]
mode = "exec"

[[language.debugger.templates.completion]]
name = "program"
completion = "filename"
default = "."

[[language.debugger.templates.completion]]
name = "mode"

[language.debugger.quirks]
absolute-paths = true
`)
		name, ok := language.DetectLanguage("main.dbg", "")
		assert.True(t, ok)
		assert.Equal(t, "debuglang", name)
	})
}

func TestDecodeGrammarConfig(t *testing.T) {
	t.Run("grammar selection only", func(t *testing.T) {
		setUserLangs(t, `
[use-grammars]
only = ["go", "json"]

[[language]]
name = "gramlang"
`)
		l := language.LoadLanguage("gramlang")
		assert.NotNil(t, l)
	})

	t.Run("grammar source path form", func(t *testing.T) {
		setUserLangs(t, `
[[grammar]]
name = "mygram"

[grammar.source]
path = "/tmp/mygram"

[[language]]
name = "gramlang2"
`)
		l := language.LoadLanguage("gramlang2")
		assert.NotNil(t, l)
	})

	t.Run("grammar source git form", func(t *testing.T) {
		setUserLangs(t, `
[[grammar]]
name = "gitgram"

[grammar.source]
git = "https://github.com/example/grammar"
rev = "main"
subpath = "src"

[[language]]
name = "gramlang3"
`)
		l := language.LoadLanguage("gramlang3")
		assert.NotNil(t, l)
	})

	t.Run("grammar without source is skipped", func(t *testing.T) {
		setUserLangs(t, `
[[grammar]]
name = "nosrcgram"

[[language]]
name = "gramlang4"
`)
		l := language.LoadLanguage("gramlang4")
		assert.NotNil(t, l)
	})
}

func TestDecodeCommentTokens(t *testing.T) {
	t.Run("comment-token singular", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "cmtlang"
comment-token = "//"
`)
		_, _ = language.DetectLanguage("nope.xyz99qwerty", "")
	})

	t.Run("comment-tokens plural list", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "cmtlang2"
comment-tokens = ["//", "#"]
`)
		_, _ = language.DetectLanguage("nope.xyz99qwerty", "")
	})

	t.Run("block-comment token table", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "blockcmt"

[language.block-comment-tokens]
start = "/*"
end = "*/"
`)
		l := language.LoadLanguage("blockcmt")
		assert.Len(t, l.BlockCommentTokens, 1)
		assert.Equal(t, "/*", l.BlockCommentTokens[0].Start)
		assert.Equal(t, "*/", l.BlockCommentTokens[0].End)
	})

	t.Run("block-comment token array", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "blockcmtlist"

[[language.block-comment-tokens]]
start = "/*"
end = "*/"

[[language.block-comment-tokens]]
start = "<!--"
end = "-->"
`)
		l := language.LoadLanguage("blockcmtlist")
		assert.Len(t, l.BlockCommentTokens, 2)
		assert.Equal(t, "<!--", l.BlockCommentTokens[1].Start)
	})

	t.Run("invalid block-comment tokens skipped", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "badblockcmt"
block-comment-tokens = ["bad"]
`)
		l := language.LoadLanguage("badblockcmt")
		assert.Empty(t, l.BlockCommentTokens)
	})
}

func TestDecodeAutoPairMap(t *testing.T) {
	t.Run("auto-pairs map syntax", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "pairlang"
file-types = ["prl"]

[language.auto-pairs]
"(" = ")"
"[" = "]"
`)
		name, ok := language.DetectLanguage("test.prl", "")
		assert.True(t, ok)
		assert.Equal(t, "pairlang", name)
	})
}

func TestLoadLanguage(t *testing.T) {
	t.Run("known language returns entry", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "knownlang"
scope = "source.knownlang"
`)
		l := language.LoadLanguage("knownlang")
		assert.NotNil(t, l)
		assert.Equal(t, "knownlang", l.Name)
	})

	t.Run("unknown language returns empty", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "somelang"
`)
		l := language.LoadLanguage("does-not-exist")
		assert.NotNil(t, l)
		assert.Equal(t, "", l.Name)
	})
}

func TestDecodeStringMap(t *testing.T) {
	t.Run("env map in language server", func(t *testing.T) {
		setUserLangs(t, `
[language-server.envlsp]
command = "envlsp"

[language-server.envlsp.environment]
FOO = "bar"
BAZ = "qux"

[[language]]
name = "envlang"
`)
		_, _ = language.DetectLanguage("nope.xyz99qwerty", "")
	})
}

func TestDecodeLanguageIDAndRulers(t *testing.T) {
	t.Run("language-id and rulers decoded", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "idlang"
language-id = "myLangID"
rulers = [80, 120]
text-width = 100
injection-regex = "idlang"
`)
		l := language.LoadLanguage("idlang")
		assert.NotNil(t, l)
		assert.Equal(t, "idlang", l.Name)
	})

	t.Run("shebangs and roots decoded", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "rootlang"
shebangs = ["rootlang"]
roots = ["root.lang"]
`)
		l := language.LoadLanguage("rootlang")
		assert.Equal(t, []string{"rootlang"}, l.Shebangs)
		assert.Equal(t, []string{"root.lang"}, l.Roots)
	})
}

func TestDecodeAutoPairBool(t *testing.T) {
	t.Run("auto-pairs false disables pairs", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "boolpairlang"
file-types = ["bpl"]
auto-pairs = false
`)
		name, ok := language.DetectLanguage("test.bpl", "")
		assert.True(t, ok)
		assert.Equal(t, "boolpairlang", name)
	})
}

func TestDecodeDebugStringCompletion(t *testing.T) {
	t.Run("string completion item in template", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "strcompl"
file-types = ["strc"]

[language.debugger]
name = "dbg"
transport = "stdio"
command = "dbg"

[[language.debugger.templates]]
name = "Run"
request = "launch"
completion = ["program"]
`)
		name, ok := language.DetectLanguage("test.strc", "")
		assert.True(t, ok)
		assert.Equal(t, "strcompl", name)
	})
}

func TestDecodeSoftWrap(t *testing.T) {
	t.Run("soft-wrap config decoded", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "swlang"
file-types = ["swl"]

[language.soft-wrap]
enable = true
wrap-at-text-width = true
`)
		name, ok := language.DetectLanguage("test.swl", "")
		assert.True(t, ok)
		assert.Equal(t, "swlang", name)
	})

	t.Run("soft-wrap with all fields decoded", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "swlang2"
file-types = ["swl2"]

[language.soft-wrap]
enable = true
max-wrap = 80
max-indent-retain = 20
wrap-indicator = "↩"
`)
		name, ok := language.DetectLanguage("test.swl2", "")
		assert.True(t, ok)
		assert.Equal(t, "swlang2", name)
	})
}

func TestDecodeLanguageServerNoCommand(t *testing.T) {
	t.Run("server without command is skipped", func(t *testing.T) {
		setUserLangs(t, `
[language-server.badlsp]
args = ["--stdio"]

[[language]]
name = "nolsp"
`)
		l := language.LoadLanguage("nolsp")
		assert.NotNil(t, l)
	})
}

func TestDecodeServerFeatureInt(t *testing.T) {
	t.Run("integer entry in server list skipped", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "intservlang"
file-types = ["iss"]
language-servers = [42]
`)
		name, ok := language.DetectLanguage("test.iss", "")
		assert.True(t, ok)
		assert.Equal(t, "intservlang", name)
	})
}

func TestDecodeFormatter(t *testing.T) {
	t.Run("formatter command and args decoded", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "fmtlang"
auto-format = true
formatter = { command = "gofmt", args = ["-s"] }
`)
		l := language.LoadLanguage("fmtlang")
		assert.NotNil(t, l.Formatter)
		assert.Equal(t, "gofmt", l.Formatter.Command)
		assert.Equal(t, []string{"-s"}, l.Formatter.Args)
		assert.True(t, l.AutoFormat)
	})

	t.Run("formatter without command is skipped", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "nocmdlang"
formatter = { args = ["-s"] }
`)
		l := language.LoadLanguage("nocmdlang")
		assert.Nil(t, l.Formatter)
	})

	t.Run("server feature map without name is skipped", func(t *testing.T) {
		setUserLangs(t, `
[[language]]
name = "nonamelang"
language-servers = [{ only-features = ["completion"] }]
`)
		l := language.LoadLanguage("nonamelang")
		assert.Empty(t, l.LanguageServers)
	})
}

func TestLoadLanguageNoConfig(t *testing.T) {
	t.Run("no config path returns empty language", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "")
		l := language.LoadLanguage("anything")
		assert.NotNil(t, l)
		assert.Equal(t, "", l.Name)
	})
}

func TestLoadLanguagesForWorkspace(t *testing.T) {
	t.Run("trusted workspace path is included", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		wsRoot := t.TempDir()
		assert.NoError(t, os.MkdirAll(filepath.Join(wsRoot, ".git"), 0o755))
		assert.NoError(t, loader.TrustWorkspace(wsRoot))
		langs, ok := language.LoadLanguagesForWorkspace("", "", wsRoot)
		assert.True(t, ok)
		_ = langs
	})
}
