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

// IndentForNewlineArgs carries the document position and style used to compute
// indentation for a new line
type IndentForNewlineArgs struct {
	Text  core.Rope
	Lang  string
	Line  int
	Pos   int
	Style core.IndentStyle
}

// IndentForNewline returns syntax-aware indentation for a newline after Pos
func IndentForNewline(args IndentForNewlineArgs) (string, bool) {
	language := languageFor(args.Lang)
	if language == nil {
		return "", false
	}
	src := args.Text.String()
	at := core.LinePos{Line: args.Line, Pos: args.Pos}
	indent := leadingIndent(src, args.Text, args.Line)
	body := strings.TrimSpace(linePrefix(src, args.Text, at))
	if outdentsLine(outdentLineArgs{lang: args.Lang, body: body}) {
		indent = dropIndent(dropIndentArgs{
			indent: indent,
			unit:   args.Style.AsStr(),
		})
	}
	ch, chPos, ok := lastCodeChar(src, args.Text, at)
	if !ok || !indentsAfter(ch) || hasMatchingCloseAt(src, chPos+1, ch) {
		return indent, true
	}
	if inStringOrComment(src, language, chPos) {
		return indent, true
	}
	return indent + args.Style.AsStr(), true
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

func linePrefix(src string, text core.Rope, at core.LinePos) string {
	start, err := text.LineToChar(at.Line)
	if err != nil || at.Pos < start {
		return ""
	}
	runes := []rune(src)
	pos := min(at.Pos, len(runes))
	return string(runes[start:pos])
}

func lastCodeChar(
	src string, text core.Rope, at core.LinePos,
) (rune, int, bool) {
	start, err := text.LineToChar(at.Line)
	if err != nil {
		return 0, 0, false
	}
	runes := []rune(src)
	pos := min(at.Pos, len(runes))
	for i := pos - 1; i >= start; i-- {
		ch := runes[i]
		if ch != ' ' && ch != '\t' {
			return ch, i, true
		}
	}
	return 0, 0, false
}

func indentsAfter(ch rune) bool {
	switch ch {
	case '(', '[', '{', ',', '.', ':', '+', '-', '*', '/', '%', '&', '|',
		'^', '=', '<', '>', '?', '\\':
		return true
	default:
		return false
	}
}

func hasMatchingCloseAt(src string, pos int, open rune) bool {
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

func inStringOrComment(src string, language *sitter.Language, pos int) bool {
	tree, ok := parseTree([]byte(src), language)
	if !ok {
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

type outdentLineArgs struct {
	lang string
	body string
}

func outdentsLine(args outdentLineArgs) bool {
	word := firstWord(args.body)
	switch args.lang {
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

type dropIndentArgs struct {
	indent string
	unit   string
}

func dropIndent(args dropIndentArgs) string {
	indent := args.indent
	unit := args.unit
	if unit == "" || indent == "" {
		return indent
	}
	if strings.HasSuffix(indent, unit) {
		return indent[:len(indent)-len(unit)]
	}
	return strings.TrimRight(indent, " \t")
}
