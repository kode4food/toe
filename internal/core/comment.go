package core

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode"
)

type (
	// BlockCommentToken holds the start and end delimiters for a block comment
	BlockCommentToken struct {
		Start string
		End   string
	}

	// CommentChange describes the comment action for one selection range
	CommentChange struct {
		Kind        CommentChangeKind
		Range       Range
		StartPos    int
		EndPos      int
		StartMargin bool
		EndMargin   bool
		StartToken  string
		EndToken    string
	}

	// CommentChangeKind identifies which variant a CommentChange represents
	CommentChangeKind int
)

const (
	CommentChangeCommented CommentChangeKind = iota
	CommentChangeUncommented
	CommentChangeWhitespace
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
	tokenLen := len([]rune(token))

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

// FindBlockComments inspects each range in sel to determine whether it is
// already block-commented and returns the per-range actions
func FindBlockComments(
	tokens []BlockCommentToken, text Rope, sel Selection,
) (bool, []CommentChange, error) {
	if len(tokens) == 0 {
		tokens = []BlockCommentToken{DefaultBlockCommentToken()}
	}

	sorted := slices.Clone(tokens)
	slices.SortFunc(sorted, func(a, b BlockCommentToken) int {
		if len(a.Start) != len(b.Start) {
			return cmp.Compare(len(b.Start), len(a.Start))
		}
		return cmp.Compare(len(b.End), len(a.End))
	})

	defaultTok := tokens[0]
	commented := true
	onlyWhitespace := true
	changes := make([]CommentChange, 0, len(sel.Ranges()))

	for _, r := range sel.Ranges() {
		slice, err := r.Slice(text)
		if err != nil {
			return false, nil, err
		}
		startPos, ok1 := commentFirstNonWhitespace(slice)
		endPos, ok2 := commentLastNonWhitespace(slice)
		if !ok1 || !ok2 {
			changes = append(changes, CommentChange{
				Kind: CommentChangeWhitespace, Range: r,
			})
			continue
		}

		onlyWhitespace = false
		lineCommented := false

		for _, tok := range sorted {
			startLen := len([]rune(tok.Start))
			endLen := len([]rune(tok.End))
			afterStart := startPos + startLen
			beforeEnd := max(0, endPos-endLen)
			contentLen := (endPos + 1) - startPos

			if contentLen < startLen+endLen {
				continue
			}
			startFrag, err1 := slice.Slice(startPos, afterStart)
			endFrag, err2 := slice.Slice(beforeEnd+1, endPos+1)
			if err1 != nil || err2 != nil {
				continue
			}
			startMatch := startFrag.String() == tok.Start
			endMatch := endFrag.String() == tok.End
			if !startMatch || !endMatch {
				continue
			}
			startMargin := false
			if ch, err := slice.CharAt(afterStart); err == nil {
				startMargin = ch == ' '
			}
			endMargin := false
			if afterStart != beforeEnd {
				if ch, err := slice.CharAt(beforeEnd); err == nil {
					endMargin = ch == ' '
				}
			}
			changes = append(changes, CommentChange{
				Kind:        CommentChangeCommented,
				Range:       r,
				StartPos:    startPos,
				EndPos:      endPos,
				StartMargin: startMargin,
				EndMargin:   endMargin,
				StartToken:  tok.Start,
				EndToken:    tok.End,
			})
			lineCommented = true
			break
		}

		if !lineCommented {
			changes = append(changes, CommentChange{
				Kind:       CommentChangeUncommented,
				Range:      r,
				StartPos:   startPos,
				EndPos:     endPos,
				StartToken: defaultTok.Start,
				EndToken:   defaultTok.End,
			})
			commented = false
		}
	}

	if onlyWhitespace {
		commented = false
	}
	return commented, changes, nil
}

// DefaultBlockCommentToken returns the fallback block comment delimiters
func DefaultBlockCommentToken() BlockCommentToken {
	return BlockCommentToken{Start: "/*", End: "*/"}
}

// CreateBlockCommentTransaction builds the transaction and updated ranges for a
// block comment toggle operation
func CreateBlockCommentTransaction(
	doc Rope, commented bool, commentChanges []CommentChange,
) (Transaction, []Range, error) {
	changes := make([]Change, 0, len(commentChanges)*2)
	var ranges []Range
	offs := 0

	for _, ch := range commentChanges {
		from := ch.Range.From()
		if commented {
			if ch.Kind != CommentChangeCommented {
				continue
			}
			startLen := len([]rune(ch.StartToken))
			endLen := len([]rune(ch.EndToken))
			sm := 0
			if ch.StartMargin {
				sm = 1
			}
			em := 0
			if ch.EndMargin {
				em = 1
			}
			changes = append(changes,
				DeleteChange(
					from+ch.StartPos,
					from+ch.StartPos+startLen+sm,
				),
				DeleteChange(
					from+ch.EndPos-endLen-em+1,
					from+ch.EndPos+1,
				),
			)
		} else {
			switch ch.Kind {
			case CommentChangeUncommented:
				startLen := len([]rune(ch.StartToken))
				endLen := len([]rune(ch.EndToken))
				sp := from + ch.StartPos
				ep := from + ch.EndPos
				changes = append(changes,
					TextChange(sp, sp, ch.StartToken+" "),
					TextChange(ep+1, ep+1, " "+ch.EndToken),
				)
				offset := startLen + endLen + 2
				rng := NewRange(
					from+offs, from+offs+ch.EndPos+1+offset,
				).WithDirection(ch.Range.Direction())
				ranges = append(ranges, rng)
				offs += offset
			default:
				f, t := ch.Range.From()+offs, ch.Range.To()+offs
				ranges = append(ranges, NewRange(f, t))
			}
		}
	}

	cs, err := NewChangeSetFromChanges(doc, changes)
	if err != nil {
		return Transaction{}, nil, err
	}
	return NewTransaction(doc).WithChanges(cs), ranges, nil
}

// ToggleBlockComments toggles block comments on all ranges in sel
func ToggleBlockComments(
	doc Rope, sel Selection, tokens []BlockCommentToken,
) (Transaction, error) {
	commented, commentChanges, err := FindBlockComments(tokens, doc, sel)
	if err != nil {
		return Transaction{}, err
	}
	tx, ranges, err := CreateBlockCommentTransaction(
		doc, commented, commentChanges,
	)
	if err != nil {
		return Transaction{}, err
	}
	if !commented && len(ranges) > 0 {
		pri := min(sel.PrimaryIndex(), len(ranges)-1)
		newSel, err := NewSelection(ranges, pri)
		if err != nil {
			return Transaction{}, err
		}
		tx = tx.WithSelection(newSel)
	}
	return tx, nil
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
	tokenLen := len([]rune(token))

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
