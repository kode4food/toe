package core

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

// DefaultCommentToken is used for line comments when no token is configured
const DefaultCommentToken = "#"

// GetCommentToken returns the longest token matching the start of the
// non-whitespace content on the given line
func GetCommentToken(text Rope, tokens []string, lineNum int) (string, bool) {
	line, err := text.Line(lineNum)
	if err != nil {
		return "", false
	}
	start, ok := commentFirstNonWhitespace(line)
	if !ok {
		return "", false
	}
	sub, err := line.Slice(start, line.LenChars())
	if err != nil {
		return "", false
	}
	subStr := sub.String()
	var best string
	for _, tok := range tokens {
		if len(tok) > len(best) && strings.HasPrefix(subStr, tok) {
			best = tok
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}

// ToggleLineComments builds a transaction that toggles line comments on every
// non-blank line in sel using token (DefaultCommentToken when empty)
func ToggleLineComments(
	doc Rope, sel Selection, token string,
) (Transaction, error) {
	if token == "" {
		token = DefaultCommentToken
	}
	comment := token + " "

	lineRanges, err := sel.LineRanges(doc)
	if err != nil {
		return Transaction{}, err
	}

	var lines []int
	for _, lr := range lineRanges {
		end := min(lr.To+1, doc.LenLines())
		for l := lr.From; l < end; l++ {
			lines = append(lines, l)
		}
	}

	found := findLineComment(token, doc, lines)
	tokenLen := utf8.RuneCountInString(token)

	changes := make([]Change, 0, len(found.lines))
	for _, lineNum := range found.lines {
		lineStart, err := doc.LineToChar(lineNum)
		if err != nil {
			return Transaction{}, err
		}
		pos := lineStart + found.minCol
		if !found.commented {
			changes = append(changes, TextChange(pos, pos, comment))
		} else {
			changes = append(changes,
				DeleteChange(pos, pos+tokenLen+found.margin),
			)
		}
	}

	cs, err := NewChangeSetFromChanges(doc, changes)
	if err != nil {
		return Transaction{}, err
	}
	return NewTransaction(doc).WithChanges(cs), nil
}

// SplitLinesOfSelection returns a new selection where each range covers exactly
// one line within the original selection spans
func SplitLinesOfSelection(text Rope, sel Selection) (Selection, error) {
	var ranges []Range
	for _, r := range sel.Ranges() {
		lr, err := r.LineRange(text)
		if err != nil {
			return Selection{}, err
		}
		for lineNum := lr.From; lineNum <= lr.To; lineNum++ {
			start, err := text.LineToChar(lineNum)
			if err != nil {
				return Selection{}, err
			}
			end := commentLineEndChar(text, lineNum)
			ranges = append(ranges, NewRange(start, end))
		}
	}
	if len(ranges) == 0 {
		return sel, nil
	}
	return NewSelection(ranges, 0)
}

type lineCommentRes struct {
	commented bool
	lines     []int
	minCol    int
	margin    int
}

func findLineComment(
	token string, text Rope, lines []int,
) lineCommentRes {
	res := lineCommentRes{
		commented: true,
		minCol:    math.MaxInt,
		margin:    1,
	}
	tokenLen := utf8.RuneCountInString(token)

	for _, lineNum := range lines {
		line, err := text.Line(lineNum)
		if err != nil {
			continue
		}
		pos, ok := commentFirstNonWhitespace(line)
		if !ok {
			continue
		}
		lineLen := line.LenChars()
		if pos < res.minCol {
			res.minCol = pos
		}
		fragEnd := min(pos+tokenLen, lineLen)
		frag, err := line.Slice(pos, fragEnd)
		if err != nil || frag.String() != token {
			res.commented = false
		}
		if ch, err := line.CharAt(pos + tokenLen); err != nil || ch != ' ' {
			res.margin = 0
		}
		res.lines = append(res.lines, lineNum)
	}

	if res.minCol == math.MaxInt {
		res.minCol = 0
	}
	return res
}

func commentFirstNonWhitespace(r Rope) (int, bool) {
	n := r.LenChars()
	for i := range n {
		ch, err := r.CharAt(i)
		if err != nil {
			return 0, false
		}
		if !unicode.IsSpace(ch) {
			return i, true
		}
	}
	return 0, false
}

func commentLastNonWhitespace(r Rope) (int, bool) {
	n := r.LenChars()
	for i := n - 1; i >= 0; i-- {
		ch, err := r.CharAt(i)
		if err != nil {
			return 0, false
		}
		if !unicode.IsSpace(ch) {
			return i, true
		}
	}
	return 0, false
}

func commentLineEndChar(text Rope, lineNum int) int {
	next := lineNum + 1
	if next >= text.LenLines() {
		return text.LenChars()
	}
	pos, err := text.LineToChar(next)
	if err != nil {
		return text.LenChars()
	}
	return pos
}
