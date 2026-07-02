package highlight_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/highlight"
)

func TestNormalizeNewlines(t *testing.T) {
	assert.Equal(t, "a\nb\nc\n", highlight.NormalizeNewlines("a\r\nb\r\nc\r\n"))
	assert.Equal(t, "a\nb", highlight.NormalizeNewlines("a\nb"))
	assert.Equal(t, "", highlight.NormalizeNewlines(""))
}

func TestTokenizeUnknown(t *testing.T) {
	// Unknown lang falls back to Chroma's Fallback lexer, so spans may be empty
	// but we should not panic
	spans := highlight.Tokenize("hello world", "totally-unknown-lang-xyzzy")
	// We don't assert a specific result — just that it doesn't panic
	_ = spans
}

func TestTokenizeGo(t *testing.T) {
	src := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	spans := highlight.Tokenize(src, "go")
	assert.NotEmpty(t, spans)

	// Every span must have Start < End
	for _, sp := range spans {
		assert.Less(t, sp.Start, sp.End)
	}

	// Spans must be ordered by Start
	for i := 1; i < len(spans); i++ {
		assert.LessOrEqual(t, spans[i-1].Start, spans[i].Start,
			"spans must be ordered")
	}
}

func TestTokenizeEmpty(t *testing.T) {
	spans := highlight.Tokenize("", "go")
	// Should return without panicking; empty text has no tokens
	assert.Empty(t, spans)
}

func TestTokenizePython(t *testing.T) {
	src := "def foo(x):\n    return x + 1\n"
	spans := highlight.Tokenize(src, "python")
	assert.NotEmpty(t, spans)
	for _, sp := range spans {
		assert.Less(t, sp.Start, sp.End)
	}
}

func TestTokenizeLanguages(t *testing.T) {
	cases := []struct {
		lang string
		src  string
	}{
		{"go", `package main
// doc comment
import "fmt"
func main() { fmt.Println("hello", 42, 3.14) }
`},
		{"python", `# comment
#: special doc comment
@property
def bar(self):
    """docstring"""
    x = "hello\nworld"
    y = True and False or None
    raise ValueError('err')
`},
		{"javascript", `// comment
function greet(name) {
    const msg = ` + "`" + `Hello ${name + 1}` + "`" + `;
    return msg;
}
`},
		{"ruby", `:symbol\nx = :foo\ny = \"hello\"\n`},
		{"html", `<!DOCTYPE html>
<html lang="en">
  <head><title>Test</title></head>
  <body class="main">Hello</body>
</html>
`},
		{"diff", `--- a/file.go
+++ b/file.go
@@ -1,3 +1,3 @@
-old line
+new line
 context
`},
		{"markdown", "# Heading\n\n**bold** _italic_\n\n" +
			"```go\npkg main\n```\n"},
		{"yaml", "key: value\nlist:\n  - item1\n  - item2\n"},
		{"json", `{"name": "toe", "version": 1, "active": true}`},
	}

	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			spans := highlight.Tokenize(tc.src, tc.lang)
			// Some langs may produce no highlights — just verify no panic
			for _, sp := range spans {
				assert.Less(t, sp.Start, sp.End)
			}
			// Spans must be ordered
			for i := 1; i < len(spans); i++ {
				assert.LessOrEqual(t, spans[i-1].Start, spans[i].Start)
			}
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	t.Run("detects by path", func(t *testing.T) {
		lang := highlight.DetectLanguage("main.go", "")
		assert.Equal(t, "go", lang)
	})

	t.Run("detects by content shebang", func(t *testing.T) {
		lang := highlight.DetectLanguage(
			"script.xyz_unknown_ext",
			"#!/usr/bin/env python3\nprint('hi')\n",
		)
		assert.NotEmpty(t, lang)
	})

	t.Run("unknown returns text", func(t *testing.T) {
		lang := highlight.DetectLanguage("", "")
		assert.Equal(t, "text", lang)
	})
}

func TestDefaultStyle(t *testing.T) {
	t.Run("known scope returns non-empty style", func(t *testing.T) {
		s := highlight.DefaultStyle("keyword")
		assert.NotEqual(t, s, highlight.DefaultStyle("__unknown__scope__"))
	})

	t.Run("parent scope fallback", func(t *testing.T) {
		// "keyword.function" has its own entry; "keyword.unknown" should
		// fall back to "keyword"
		full := highlight.DefaultStyle("keyword.function")
		fallback := highlight.DefaultStyle("keyword.unknown")
		parent := highlight.DefaultStyle("keyword")
		assert.Equal(t, parent, fallback)
		assert.NotEqual(t, full, parent)
	})

	t.Run("unknown scope returns empty style", func(t *testing.T) {
		s := highlight.DefaultStyle("__no_such_scope__")
		assert.Equal(t, s, highlight.DefaultStyle(""))
	})
}
