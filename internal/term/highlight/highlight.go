// Package highlight provides Chroma-based syntax highlighting for the editor
// It tokenizes document text and maps Chroma token types to terminal styles
package highlight

import (
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// Span represents a highlighted region in the document
type Span struct {
	Start int // inclusive char offset
	End   int // exclusive char offset
	Scope string
}

var (
	defaultStyles = map[string]lipgloss.Style{
		"keyword":                     ansiStyle("3").Bold(true),
		"keyword.function":            ansiStyle("13").Bold(true),
		"keyword.operator":            ansiStyle("5"),
		"namespace":                   ansiStyle("13").Bold(true),
		"type":                        ansiStyle("11").Bold(true),
		"type.builtin":                ansiStyle("3").Bold(true),
		"string":                      ansiStyle("2"),
		"string.special":              ansiStyle("6"),
		"constant.character.escape":   ansiStyle("6"),
		"comment":                     ansiStyle("8").Italic(true),
		"comment.block.documentation": ansiStyle("12").Italic(true),
		"constant.numeric":            ansiStyle("14"),
		"constant":                    ansiStyle("14"),
		"operator":                    ansiStyle("5"),
		"function":                    ansiStyle("12"),
		"function.builtin":            ansiStyle("6"),
		"variable.builtin":            ansiStyle("6"),
		"attribute":                   ansiStyle("10"),
		"tag":                         ansiStyle("4").Bold(true),
		"diff.plus":                   ansiStyle("2"),
		"diff.minus":                  ansiStyle("1"),
		"markup.heading":              lipgloss.NewStyle().Bold(true),
	}

	chromaScopes = map[chroma.TokenType]string{
		chroma.Keyword:            "keyword",
		chroma.KeywordReserved:    "keyword",
		chroma.KeywordDeclaration: "keyword.function",
		chroma.KeywordNamespace:   "namespace",
		chroma.KeywordType:        "type.builtin",

		chroma.String:         "string",
		chroma.StringDoc:      "string",
		chroma.StringDouble:   "string",
		chroma.StringSingle:   "string",
		chroma.StringBacktick: "string",
		chroma.StringHeredoc:  "string",
		chroma.StringEscape:   "constant.character.escape",
		chroma.StringInterpol: "string.special",

		chroma.Comment:          "comment",
		chroma.CommentSingle:    "comment",
		chroma.CommentMultiline: "comment",
		chroma.CommentHashbang:  "comment",
		chroma.CommentSpecial:   "comment.block.documentation",

		chroma.Number:        "constant.numeric",
		chroma.NumberInteger: "constant.numeric",
		chroma.NumberFloat:   "constant.numeric",
		chroma.NumberBin:     "constant.numeric",
		chroma.NumberOct:     "constant.numeric",
		chroma.NumberHex:     "constant.numeric",

		chroma.Operator:     "operator",
		chroma.OperatorWord: "keyword.operator",

		chroma.NameFunction:      "function",
		chroma.NameFunctionMagic: "function",
		chroma.NameBuiltin:       "function.builtin",
		chroma.NameBuiltinPseudo: "variable.builtin",
		chroma.NameClass:         "type",
		chroma.NameDecorator:     "attribute",
		chroma.NameException:     "type",
		chroma.NameAttribute:     "attribute",
		chroma.NameTag:           "tag",
		chroma.NameConstant:      "constant",

		chroma.LiteralStringSymbol: "constant",

		chroma.GenericInserted: "diff.plus",
		chroma.GenericDeleted:  "diff.minus",
		chroma.GenericHeading:  "markup.heading",
	}
)

// Tokenize parses text using Chroma and returns highlight spans with theme
// scope names. Prefer calling syntax.Tokenize which tries Tree-sitter
// first and uses this as a fallback
func Tokenize(text, lang string) []Span {
	lex := lexers.Get(lang)
	if lex == nil {
		lex = lexers.Fallback
	}
	lex = chroma.Coalesce(lex)

	iter, err := lex.Tokenise(nil, text)
	if err != nil {
		return nil
	}

	var spans []Span
	pos := 0
	for tok := iter(); tok != chroma.EOF; tok = iter() {
		n := utf8.RuneCountInString(tok.Value)
		if scope, ok := scopeFor(tok.Type); ok {
			spans = append(spans, Span{Start: pos, End: pos + n, Scope: scope})
		}
		pos += n
	}
	return spans
}

// DetectLanguage returns a Chroma-compatible language name for path/content
func DetectLanguage(path, content string) string {
	if lex := lexers.Match(path); lex != nil {
		return strings.ToLower(lex.Config().Name)
	}
	if lex := lexers.Analyse(content); lex != nil {
		return strings.ToLower(lex.Config().Name)
	}
	return "text"
}

// DefaultStyle returns the fallback ANSI style for a scope name when no
// theme is active or the theme does not define the scope
func DefaultStyle(scope string) lipgloss.Style {
	// Walk up the scope hierarchy (e.g. "keyword.function" → "keyword")
	for s := scope; s != ""; {
		if style, ok := defaultStyles[s]; ok {
			return style
		}
		idx := strings.LastIndexByte(s, '.')
		if idx < 0 {
			break
		}
		s = s[:idx]
	}
	return lipgloss.NewStyle()
}

// NormalizeNewlines replaces \r\n with \n for consistent tokenization
func NormalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// scopeFor maps a Chroma token type to a theme scope name. Only token types
// that differ from plain text are returned
func scopeFor(t chroma.TokenType) (string, bool) {
	for ; t > 0; t = t.Parent() {
		if scope, ok := chromaScopes[t]; ok {
			return scope, true
		}
	}
	return "", false
}

func ansiStyle(c string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
}
