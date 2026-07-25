package core

import "errors"

type surroundPosFinder func(Range) (Range, error)

var (
	ErrPairNotFound = errors.New(
		"surround pair not found around all cursors",
	)
	ErrSurroundCursorOverlap = errors.New(
		"cursors overlap for a single surround pair",
	)
	ErrRangeExceedsText      = errors.New("cursor range exceeds text length")
	ErrCursorOnAmbiguousPair = errors.New("cursor on ambiguous surround pair")
)

// FindNthClosestPairsPos finds the nth-nearest surrounding bracket pair around
// r (plaintext only). The returned Range keeps the source range's direction
func FindNthClosestPairsPos(
	doc Rope, r Range, skip int,
) (Range, error) {
	pos := r.From()
	closePos := pos
	if pos > 0 {
		closePos = pos - 1
	}

	var stack []rune

	for ; closePos < doc.LenChars(); closePos++ {
		ch, e := doc.CharAt(closePos)
		if e != nil {
			break
		}
		if IsOpenBracket(ch) {
			stack = append(stack, ch)
			continue
		}
		if !IsCloseBracket(ch) {
			continue
		}
		open, _ := GetPair(ch)
		if len(stack) > 0 && stack[len(stack)-1] == open {
			stack = stack[:len(stack)-1]
			continue
		}
		openPos, ok := doc.surroundFindNthOpen(open, ch, closePos, 1)
		if !ok {
			continue
		}
		if openPos > pos || closePos < r.To()-1 {
			continue
		}
		if skip > 1 {
			skip--
			continue
		}
		if r.Direction() == DirectionForward {
			return NewRange(openPos, closePos), nil
		}
		return NewRange(closePos, openPos), nil
	}
	return Range{}, ErrPairNotFound
}

// FindNthPairsPos finds the nth surrounding pair for character ch around r
// (plaintext only). ch may be either opening or closing
func FindNthPairsPos(
	doc Rope, ch rune, r Range, n int,
) (Range, error) {
	if doc.LenChars() < 2 {
		return Range{}, ErrPairNotFound
	}
	if r.To() >= doc.LenChars() {
		return Range{}, ErrRangeExceedsText
	}
	openCh, closeCh := GetPair(ch)
	pos := r.Cursor(doc)

	var openPos, closePos int
	var openOk, closeOk bool

	if openCh == closeCh {
		cur, e := doc.CharAt(pos)
		if e == nil && cur == openCh {
			match, ok := FindMatchingBracket(doc, pos)
			if !ok {
				return Range{}, ErrCursorOnAmbiguousPair
			}
			if match > pos {
				openPos, closePos = pos, match
			} else {
				openPos, closePos = match, pos
			}
			openOk, closeOk = true, true
		} else {
			openPos, openOk = doc.FindNthChar(
				n, openCh, pos, DirectionBackward,
			)
			closePos, closeOk = doc.FindNthChar(
				n, closeCh, pos, DirectionForward,
			)
		}
	} else {
		openPos, openOk = doc.surroundFindNthOpen(openCh, closeCh, pos, n)
		closePos, closeOk = doc.surroundFindNthClose(openCh, closeCh, pos, n)
	}

	if !openOk || !closeOk {
		return Range{}, ErrPairNotFound
	}
	if r.Direction() == DirectionForward {
		return NewRange(openPos, closePos), nil
	}
	return NewRange(closePos, openPos), nil
}

// GetSurroundPos returns flat pairs of [open, close] positions for every
// range in sel, auto-detecting the nearest pair. skip controls how many
// pairs to step over
func GetSurroundPos(doc Rope, sel Selection, skip int) ([]int, error) {
	return collectSurroundPos(sel, func(r Range) (Range, error) {
		return FindNthClosestPairsPos(doc, r, skip)
	})
}

// GetSurroundPosFor returns flat pairs of [open, close] positions for every
// range in sel, searching for the pair matching ch. skip controls how many
// pairs to step over
func GetSurroundPosFor(
	doc Rope, sel Selection, ch rune, skip int,
) ([]int, error) {
	return collectSurroundPos(sel, func(r Range) (Range, error) {
		return FindNthPairsPos(doc, ch, r, skip)
	})
}

func collectSurroundPos(sel Selection, find surroundPosFinder) ([]int, error) {
	var positions []int
	for _, r := range sel.Ranges() {
		pair, err := find(r)
		if err != nil {
			return nil, err
		}
		anchor := pair.From()
		head := pair.To()
		for _, p := range positions {
			if p == anchor || p == head {
				return nil, ErrSurroundCursorOverlap
			}
		}
		positions = append(positions, anchor, head)
	}
	return positions, nil
}
