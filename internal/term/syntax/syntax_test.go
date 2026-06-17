package syntax_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/syntax"
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
		{"css", "body { color: red; }\n"},
		{"html", "<html><body>hi</body></html>\n"},
		{"javascript", "function f() { return 1; }\n"},
		{"typescript", "const x: number = 1;\n"},
		{"markdown", "# Heading\n\nParagraph text.\n"},
		{"sql", "SELECT id FROM users WHERE active = 1;\n"},
	}

	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			spans := syntax.Tokenize(tc.src, tc.lang)
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
	spans := syntax.Tokenize(`{"key": "value", "n": 42}`, "json")
	assert.NotEmpty(t, spans)
	for _, sp := range spans {
		assert.Less(t, sp.Start, sp.End)
	}
}

func TestTokenizeEmpty(t *testing.T) {
	spans := syntax.Tokenize("", "go")
	assert.Empty(t, spans)
}

func TestTokenizeUnknown(t *testing.T) {
	// Unknown lang must not panic and falls back to Chroma's fallback lexer
	spans := syntax.Tokenize("hello world", "__unknown__")
	_ = spans
}

func TestTokenizeGoScopes(t *testing.T) {
	src := "package main\n\nfunc main() {}\n"
	spans := syntax.Tokenize(src, "go")

	scopes := make(map[string]bool)
	for _, sp := range spans {
		scopes[sp.Scope] = true
	}
	// "package" and "func" are keywords — expect a keyword scope
	assert.True(t, scopes["keyword"] || scopes["keyword.function"],
		"expected keyword scope in go source")
}

func TestTokenizeNonOverlapping(t *testing.T) {
	src := "package main\n\nimport \"fmt\"\n\n" +
		"func main() {\n\tfmt.Println(\"hello\")\n}\n"
	spans := syntax.Tokenize(src, "go")

	for i := 1; i < len(spans); i++ {
		assert.LessOrEqual(t,
			spans[i-1].End, spans[i].Start, "spans must not overlap")
	}
}
