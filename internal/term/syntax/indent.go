package syntax

import (
	"slices"
	"strings"
	"unicode"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/kode4food/toe/internal/core"
)

var (
	pythonOutdents  = []string{"elif", "else", "except", "finally"}
	cLikeOutdents   = []string{"else", "case", "default"}
	bashOutdents    = []string{"else", "elif", "fi", "done", "esac"}
	defaultOutdents = []string{"else", "elseif", "end"}
)

// IndentForNewline returns syntax-aware indentation for a newline after pos
func IndentForNewline(
	text core.Rope, lang string, line, pos int, style core.IndentStyle,
) (string, bool) {
	language, ok := languageFor(lang)
	if !ok {
		return "", false
	}
	src := text.String()
	indent := leadingIndent(src, text, line)
	body := strings.TrimSpace(linePrefix(src, text, line, pos))
	if outdentLine(lang, body) {
		indent = dropIndent(indent, style.AsStr())
	}
	ch, chPos, ok := lastCodeChar(src, text, line, pos)
	if !ok || !indentAfter(ch) || matchingCloseAt(src, chPos+1, ch) {
		return indent, true
	}
	if syntaxStringOrComment(src, language, chPos) {
		return indent, true
	}
	return indent + style.AsStr(), true
}

func leadingIndent(src string, text core.Rope, line int) string {
	start, err := text.LineToChar(line)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, ch := range []rune(src)[start:] {
		if ch != ' ' && ch != '\t' {
			break
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func linePrefix(src string, text core.Rope, line, pos int) string {
	start, err := text.LineToChar(line)
	if err != nil || pos < start {
		return ""
	}
	runes := []rune(src)
	pos = min(pos, len(runes))
	return string(runes[start:pos])
}

func lastCodeChar(
	src string, text core.Rope, line, pos int,
) (rune, int, bool) {
	start, err := text.LineToChar(line)
	if err != nil {
		return 0, 0, false
	}
	runes := []rune(src)
	pos = min(pos, len(runes))
	for i := pos - 1; i >= start; i-- {
		ch := runes[i]
		if ch != ' ' && ch != '\t' {
			return ch, i, true
		}
	}
	return 0, 0, false
}

func indentAfter(ch rune) bool {
	switch ch {
	case '(', '[', '{', ',', '.', ':', '+', '-', '*', '/', '%', '&', '|',
		'^', '=', '<', '>', '?', '\\':
		return true
	default:
		return false
	}
}

func matchingCloseAt(src string, pos int, open rune) bool {
	runes := []rune(src)
	if pos < 0 || pos >= len(runes) {
		return false
	}
	switch open {
	case '(':
		return runes[pos] == ')'
	case '[':
		return runes[pos] == ']'
	case '{':
		return runes[pos] == '}'
	default:
		return false
	}
}

func syntaxStringOrComment(
	src string, language *sitter.Language, pos int,
) bool {
	p := sitter.NewParser()
	defer p.Close()
	if err := p.SetLanguage(language); err != nil {
		return false
	}
	tree := p.Parse([]byte(src), nil)
	if tree == nil {
		return false
	}
	defer tree.Close()

	c2b := buildCharToByte(src)
	if pos < 0 || pos >= len(c2b) {
		return false
	}
	b := uint(c2b[pos])
	n := tree.RootNode().NamedDescendantForByteRange(b, b+1)
	for n != nil {
		kind := n.Kind()
		if strings.Contains(kind, "comment") ||
			strings.Contains(kind, "string") {
			return true
		}
		n = n.Parent()
	}
	return false
}

func outdentLine(lang, body string) bool {
	word := firstWord(body)
	switch lang {
	case "python":
		return slices.Contains(pythonOutdents, word)
	case "javascript", "typescript", "tsx":
		return slices.Contains(cLikeOutdents, word)
	case "bash":
		return slices.Contains(bashOutdents, word)
	default:
		return slices.Contains(defaultOutdents, word)
	}
}

func firstWord(s string) string {
	for i, ch := range s {
		if !unicode.IsLetter(ch) {
			return s[:i]
		}
	}
	return s
}

func dropIndent(indent, unit string) string {
	if unit == "" || indent == "" {
		return indent
	}
	if strings.HasSuffix(indent, unit) {
		return indent[:len(indent)-len(unit)]
	}
	return strings.TrimRight(indent, " \t")
}
