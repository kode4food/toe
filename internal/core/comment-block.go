package core

import (
	"cmp"
	"slices"
	"unicode/utf8"
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
			startLen := utf8.RuneCountInString(tok.Start)
			endLen := utf8.RuneCountInString(tok.End)
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
