// Package highlight provides Chroma-based syntax highlighting for the editor
// It tokenizes document text and maps Chroma token types to terminal styles
package highlight

import (
	"strings"

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
		n := len([]rune(tok.Value))
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

// SpanAt returns the scope name for the character at pos, or ("", false)
func SpanAt(spans []Span, pos int) (string, bool) {
	lo, hi := 0, len(spans)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		s := spans[mid]
		if pos < s.Start {
			hi = mid - 1
		} else if pos >= s.End {
			lo = mid + 1
		} else {
			return s.Scope, true
		}
	}
	return "", false
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

// scopeFor maps a Chroma token type to a theme scope name. Only token types
// that differ from plain text are returned
func scopeFor(t chroma.TokenType) (string, bool) {
	for ; t > 0; t = t.Parent() {
		switch t {
		case chroma.Keyword, chroma.KeywordReserved:
			return "keyword", true
		case chroma.KeywordDeclaration:
			return "keyword.function", true
		case chroma.KeywordNamespace:
			return "namespace", true
		case chroma.KeywordType:
			return "type.builtin", true

		case chroma.String, chroma.StringDoc, chroma.StringDouble,
			chroma.StringSingle, chroma.StringBacktick, chroma.StringHeredoc:
			return "string", true
		case chroma.StringEscape:
			return "constant.character.escape", true
		case chroma.StringInterpol:
			return "string.special", true

		case chroma.Comment, chroma.CommentSingle,
			chroma.CommentMultiline, chroma.CommentHashbang:
			return "comment", true
		case chroma.CommentSpecial:
			return "comment.block.documentation", true

		case chroma.Number, chroma.NumberInteger, chroma.NumberFloat,
			chroma.NumberBin, chroma.NumberOct, chroma.NumberHex:
			return "constant.numeric", true

		case chroma.Operator:
			return "operator", true
		case chroma.OperatorWord:
			return "keyword.operator", true

		case chroma.NameFunction, chroma.NameFunctionMagic:
			return "function", true
		case chroma.NameBuiltin:
			return "function.builtin", true
		case chroma.NameBuiltinPseudo:
			return "variable.builtin", true
		case chroma.NameClass:
			return "type", true
		case chroma.NameDecorator:
			return "attribute", true
		case chroma.NameException:
			return "type", true
		case chroma.NameAttribute:
			return "attribute", true
		case chroma.NameTag:
			return "tag", true
		case chroma.NameConstant:
			return "constant", true

		case chroma.LiteralStringSymbol:
			return "constant", true

		case chroma.GenericInserted:
			return "diff.plus", true
		case chroma.GenericDeleted:
			return "diff.minus", true
		case chroma.GenericHeading:
			return "markup.heading", true

		default:
			// fall through to parent
		}
	}
	return "", false
}

var defaultStyles = map[string]lipgloss.Style{
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

// NormalizeNewlines replaces \r\n with \n for consistent tokenization
func NormalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func ansiStyle(c string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
}
