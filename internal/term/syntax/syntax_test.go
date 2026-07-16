package syntax_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view/language"
)

func TestSupportedLanguages(t *testing.T) {
	langs := syntax.SupportedLanguages()
	assert.NotEmpty(t, langs)
	assert.True(t, slices.Contains(langs, "go"))
	assert.True(t, slices.Contains(langs, "bash"))
	assert.True(t, slices.Contains(langs, "yaml"))

	for i := 1; i < len(langs); i++ {
		assert.LessOrEqual(t, langs[i-1], langs[i], "must be sorted")
	}
}

func TestHasHighlightQuery(t *testing.T) {
	for _, lang := range syntax.SupportedLanguages() {
		t.Run(lang, func(t *testing.T) {
			assert.True(t, syntax.HasHighlightQuery(lang))
		})
	}
}

func TestSupportedLanguageEntries(t *testing.T) {
	langs, ok := language.LoadBundledLanguages()
	assert.True(t, ok)
	names := map[string]bool{}
	for _, lang := range langs.Languages {
		names[lang.Name] = true
	}
	for _, lang := range syntax.SupportedLanguages() {
		t.Run(lang, func(t *testing.T) {
			assert.True(t, names[lang])
			assert.True(t, syntax.HasHighlightQuery(lang))
		})
	}
}

func TestHasHighlightQueryUnknown(t *testing.T) {
	assert.False(t, syntax.HasHighlightQuery("__no_such_lang__"))
}

func TestTokenize(t *testing.T) {
	cases := []struct {
		lang string
		src  string
	}{
		{"go", "package main\n\nfunc main() {}\n"},
		{"bash", "#!/bin/bash\necho hello\n"},
		{"yaml", "key: value\nlist:\n  - a\n  - b\n"},
		{"toml", "[section]\nkey = \"value\"\n"},
		{"hcl", "resource \"x\" \"y\" {\n  name = \"z\"\n}\n"},
		{"css", "body { color: red; }\n"},
		{"html", "<html><body>hi</body></html>\n"},
		{"javascript", "const f = (x) => `Hello ${x + 1}`;\n"},
		{"typescript", "const x: number = 1;\n"},
		{"tsx", "const El = () => <div className=\"foo\">{name}</div>;\n"},
		{"markdown", "# Heading\n\nParagraph text.\n"},
		{"sql", "SELECT id FROM users WHERE active = 1;\n"},
		{"makefile", "all: build\n\t$(CC) -o out main.c\n"},
		{"diff", "diff --git a/foo.txt b/foo.txt\n" +
			"index 000..111 100644\n--- a/foo.txt\n+++ b/foo.txt\n" +
			"@@ -1 +1 @@\n-old\n+new\n"},
	}

	sc := syntax.NewSyntaxCache()
	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			spans := sc.Tokenize(tc.src, tc.lang)
			assert.NotEmpty(t, spans)
			for _, sp := range spans {
				assert.Less(t, sp.Start, sp.End, "span must have Start < End")
			}
			for i := 1; i < len(spans); i++ {
				assert.LessOrEqual(t, spans[i-1].Start, spans[i].Start,
					"spans must be ordered")
			}
		})
	}
}

func TestTokenizeChromaFallback(t *testing.T) {
	// "json" has no Tree-sitter grammar in langRegistry; falls back to Chroma
	sc := syntax.NewSyntaxCache()
	spans := sc.Tokenize(`{"key": "value", "n": 42}`, "json")
	assert.NotEmpty(t, spans)
	for _, sp := range spans {
		assert.Less(t, sp.Start, sp.End)
	}
}

func TestTokenizeEmpty(t *testing.T) {
	sc := syntax.NewSyntaxCache()
	spans := sc.Tokenize("", "go")
	assert.Empty(t, spans)
}

func TestTokenizeUnknown(t *testing.T) {
	// Unknown lang must not panic and falls back to Chroma's fallback lexer
	sc := syntax.NewSyntaxCache()
	spans := sc.Tokenize("hello world", "__unknown__")
	_ = spans
}

func TestTokenizeGoScopes(t *testing.T) {
	src := "package main\n\nfunc main() {}\n"
	sc := syntax.NewSyntaxCache()
	spans := sc.Tokenize(src, "go")

	scopes := make(map[string]bool)
	for _, sp := range spans {
		scopes[sp.Scope] = true
	}
	// "package" and "func" are keywords — expect a keyword scope
	assert.True(t, scopes["keyword"] || scopes["keyword.function"],
		"expected keyword scope in go source")
}

func TestTokenizeHTMLInjections(t *testing.T) {
	src := `<style>body { color: red; }</style>` + "\n" +
		`<script>const answer = 42;</script>`
	sc := syntax.NewSyntaxCache()
	spans := sc.Tokenize(src, "html")

	assert.Equal(t, "variable.other.member", scopeAt(spans, src, "color"))
	assert.Equal(t, "keyword.storage.modifier", scopeAt(spans, src, "const"))
}

func TestTokenizeGoRich(t *testing.T) {
	// Rich source to trigger overlapping tree-sitter captures at different
	// start positions (exercises buildSpans c.end <= pos branch)
	src := `package main

import (
	"fmt"
	"strings"
)

// Greet returns a greeting string
func Greet(name string) string {
	if name == "" {
		name = "world"
	}
	return fmt.Sprintf("Hello, %s!", strings.TrimSpace(name))
}

type Config struct {
	Host string
	Port int
}

const DefaultPort = 8080
var _ = DefaultPort
`
	sc := syntax.NewSyntaxCache()
	spans := sc.Tokenize(src, "go")
	assert.NotEmpty(t, spans)
	for i := 1; i < len(spans); i++ {
		assert.LessOrEqual(t, spans[i-1].End, spans[i].Start,
			"spans must not overlap")
	}
}

func scopeAt(spans []highlight.Span, src, needle string) string {
	pos := strings.Index(src, needle)
	if pos < 0 {
		return ""
	}
	for _, sp := range spans {
		if sp.Start <= pos && pos < sp.End {
			return sp.Scope
		}
	}
	return ""
}

func TestTokenizeCached(t *testing.T) {
	t.Run("highlight cache hit", func(t *testing.T) {
		// Second call hits rawQuery and langCache
		src := "package main\n"
		sc := syntax.NewSyntaxCache()
		spans1 := sc.Tokenize(src, "go")
		spans2 := sc.Tokenize(src, "go")
		assert.Equal(t, len(spans1), len(spans2))
	})

	t.Run("injection cache hit", func(t *testing.T) {
		// Second call hits rawInject cache
		src := "<script>const x = 1;</script>\n"
		sc := syntax.NewSyntaxCache()
		spans1 := sc.Tokenize(src, "html")
		spans2 := sc.Tokenize(src, "html")
		assert.Equal(t, len(spans1), len(spans2))
	})
}

func TestTokenizeEscapeOverlap(t *testing.T) {
	t.Run("escape in string skips nested capture", func(t *testing.T) {
		src := "package main\nconst s = \"hello\\nworld\"\n"
		sc := syntax.NewSyntaxCache()
		spans := sc.Tokenize(src, "go")
		assert.NotEmpty(t, spans)
		for i := 1; i < len(spans); i++ {
			assert.LessOrEqual(t, spans[i-1].End, spans[i].Start)
		}
	})
}

func TestTokenizeNonOverlapping(t *testing.T) {
	src := "package main\n\nimport \"fmt\"\n\n" +
		"func main() {\n\tfmt.Println(\"hello\")\n}\n"
	sc := syntax.NewSyntaxCache()
	spans := sc.Tokenize(src, "go")

	for i := 1; i < len(spans); i++ {
		assert.LessOrEqual(t,
			spans[i-1].End, spans[i].Start, "spans must not overlap")
	}
}
