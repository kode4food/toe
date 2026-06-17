package core

type (
	// AutoPairs holds the set of active bracket pairs, keyed by both
	// opener and closer
	AutoPairs map[rune]Pair

	// Pair describes one opener/closer bracket pair
	Pair struct {
		Open  rune
		Close rune
	}
)

// NewAutoPairs constructs AutoPairs from [open, close] pairs
func NewAutoPairs(pairs [][2]rune) AutoPairs {
	m := make(map[rune]Pair, len(pairs)*2)
	for _, p := range pairs {
		pair := Pair{Open: p[0], Close: p[1]}
		m[pair.Open] = pair
		if pair.Open != pair.Close {
			m[pair.Close] = pair
		}
	}
	return m
}

// DefaultAutoPairs returns an AutoPairs with the standard bracket/quote pairs
func DefaultAutoPairs() AutoPairs {
	return NewAutoPairs(defaultAutoPairRunes())
}

// HookInsert returns a Change and updated Range when ch should trigger an
// auto-pair action, or ok=false when no action is needed
func HookInsert(
	doc Rope, r Range, ch rune, pairs AutoPairs,
) (Change, Range, bool) {
	pair, ok := pairs.Get(ch)
	if !ok {
		if autoPairIsWhitespace(ch) {
			return hookInsertWhitespace(doc, r, ch, pairs)
		}
		return Change{}, Range{}, false
	}
	if pair.Same() {
		return handleInsertSame(doc, r, pair)
	}
	if pair.Open == ch {
		return handleInsertOpen(doc, r, pair)
	}
	if pair.Close == ch {
		return handleInsertClose(doc, r, pair)
	}
	return Change{}, Range{}, false
}

// HookDelete returns a Deletion and updated Range when backspace should erase
// an auto-inserted pair, or ok=false when no action is needed
func HookDelete(doc Rope, r Range, pairs AutoPairs) (Deletion, Range, bool) {
	cursor := r.Cursor(doc)

	cur, ok := autoPairCharAt(doc, cursor)
	if !ok {
		return Deletion{}, Range{}, false
	}
	prev, ok := autoPairPrevChar(doc, cursor)
	if !ok {
		return Deletion{}, Range{}, false
	}

	if doc.LenChars() >= 4 &&
		autoPairIsWhitespace(prev) && autoPairIsWhitespace(cur) {
		secondPrev, ok1 := autoPairCharAt(
			doc, NthPrevGraphemeBoundary(doc, cursor, 2),
		)
		secondNext, ok2 := autoPairCharAt(
			doc, NextGraphemeBoundary(doc, cursor),
		)
		if ok1 && ok2 {
			if pair, ok := pairs.Get(secondPrev); ok {
				if pair.Open == secondPrev && pair.Close == secondNext {
					return handleDeletePair(doc, r)
				}
			}
		}
	}

	pair, ok := pairs.Get(cur)
	if !ok {
		return Deletion{}, Range{}, false
	}
	if pair.Open != prev || pair.Close != cur {
		return Deletion{}, Range{}, false
	}
	return handleDeletePair(doc, r)
}

// Get returns the Pair for ch, or false if ch is not a registered opener/closer
func (a AutoPairs) Get(ch rune) (Pair, bool) {
	p, ok := a[ch]
	return p, ok
}

// Same reports whether the pair's open and close characters are identical
func (p Pair) Same() bool {
	return p.Open == p.Close
}

// ShouldClose reports whether typing this pair's close character at range
// should insert the closing character rather than skip past an existing one
func (p Pair) ShouldClose(doc Rope, r Range) bool {
	ok := nextIsNotAlphaPair(doc, r)
	if p.Same() {
		ok = ok && prevIsNotAlphaPair(doc, r)
	}
	return ok
}

func handleDeletePair(doc Rope, r Range) (Deletion, Range, bool) {
	cursor := r.Cursor(doc)
	endNext := NextGraphemeBoundary(doc, cursor)
	endPrev := PrevGraphemeBoundary(doc, cursor)
	sizeDelete := endNext - endPrev

	nextHead := NextGraphemeBoundary(doc, r.Head) - sizeDelete

	var nextAnchor int
	single := r.IsSingleGrapheme(doc)
	switch {
	case r.Direction() == DirectionForward && single:
		nextAnchor = r.Anchor - (endNext - cursor)
	case r.Direction() == DirectionBackward && single:
		nextAnchor = r.Anchor - (cursor - endPrev)
	case r.Direction() == DirectionForward:
		nextAnchor = r.Anchor
	default:
		nextAnchor = r.Anchor - sizeDelete
	}

	del := Deletion{From: endPrev, To: endNext}
	return del, NewRange(nextAnchor, nextHead), true
}

func handleInsertOpen(doc Rope, r Range, pair Pair) (Change, Range, bool) {
	cursor := r.Cursor(doc)
	if !pair.ShouldClose(doc, r) {
		if _, ok := autoPairCharAt(doc, cursor); ok {
			return Change{}, Range{}, false
		}
	}
	text := string([]rune{pair.Open, pair.Close})
	change := TextChange(cursor, cursor, text)
	return change, autoPairNextRange(doc, r, 2), true
}

func handleInsertClose(doc Rope, r Range, pair Pair) (Change, Range, bool) {
	cursor := r.Cursor(doc)
	ch, ok := autoPairCharAt(doc, cursor)
	if !ok || ch != pair.Close {
		return Change{}, Range{}, false
	}
	change := TextChange(cursor, cursor, "")
	return change, autoPairNextRange(doc, r, 0), true
}

func handleInsertSame(doc Rope, r Range, pair Pair) (Change, Range, bool) {
	cursor := r.Cursor(doc)
	ch, ok := autoPairCharAt(doc, cursor)
	if ok && ch == pair.Open {
		change := TextChange(cursor, cursor, "")
		return change, autoPairNextRange(doc, r, 0), true
	}
	if !pair.ShouldClose(doc, r) {
		return Change{}, Range{}, false
	}
	text := string([]rune{pair.Open, pair.Close})
	return TextChange(cursor, cursor, text), autoPairNextRange(doc, r, 2), true
}

func hookInsertWhitespace(
	doc Rope, r Range, ch rune, pairs AutoPairs,
) (Change, Range, bool) {
	cursor := r.Cursor(doc)
	cur, ok1 := autoPairCharAt(doc, cursor)
	prev, ok2 := autoPairPrevChar(doc, cursor)
	if !ok1 || !ok2 {
		return Change{}, Range{}, false
	}
	pair, ok := pairs.Get(cur)
	if !ok || pair.Open != prev || pair.Close != cur {
		return Change{}, Range{}, false
	}
	wsPair := Pair{Open: ch, Close: ch}
	return handleInsertSame(doc, r, wsPair)
}

// autoPairNextRange computes the resulting range after an auto-pair insertion
// of lenInserted chars
func autoPairNextRange(doc Rope, start Range, lenInserted int) Range {
	n := doc.LenChars()
	if start.Head == n && start.Anchor == n {
		return NewRange(start.Anchor+1, start.Head+1)
	}

	single := start.IsSingleGrapheme(doc)

	if lenInserted == 0 {
		var endAnchor int
		if single {
			endAnchor = NextGraphemeBoundary(doc, start.Anchor)
		} else {
			endAnchor = start.Anchor
		}
		return NewRange(
			endAnchor, NextGraphemeBoundary(doc, start.Head),
		)
	}

	if lenInserted == 1 {
		var endAnchor int
		if single || start.Direction() == DirectionBackward {
			endAnchor = start.Anchor + 1
		} else {
			endAnchor = start.Anchor
		}
		return NewRange(endAnchor, start.Head+1)
	}

	var endHead int
	if start.Head == 0 || start.Direction() == DirectionBackward {
		endHead = start.Head + 1
	} else {
		prevBound := PrevGraphemeBoundary(doc, start.Head)
		endHead = prevBound + lenInserted
	}

	var endAnchor int
	switch {
	case start.Len() == 0:
		endAnchor = endHead
	case start.Len() == 1 && start.Direction() == DirectionForward:
		endAnchor = endHead - 1
	case start.Len() == 1 && start.Direction() == DirectionBackward:
		endAnchor = endHead + 1
	case start.Direction() == DirectionForward:
		if single {
			endAnchor = PrevGraphemeBoundary(doc, start.Head) + 1
		} else {
			endAnchor = start.Anchor
		}
	default:
		if single {
			b := PrevGraphemeBoundary(doc, start.Anchor)
			endAnchor = b + lenInserted
		} else {
			endAnchor = start.Anchor + lenInserted
		}
	}

	return NewRange(endAnchor, endHead)
}

func defaultAutoPairRunes() [][2]rune {
	return [][2]rune{
		{'(', ')'},
		{'{', '}'},
		{'[', ']'},
		{'\'', '\''},
		{'"', '"'},
		{'`', '`'},
	}
}

func nextIsNotAlphaPair(doc Rope, r Range) bool {
	cursor := r.Cursor(doc)
	ch, ok := autoPairCharAt(doc, cursor)
	if !ok {
		return true
	}
	return !autoPairIsAlphanumeric(ch)
}

func prevIsNotAlphaPair(doc Rope, r Range) bool {
	cursor := r.Cursor(doc)
	ch, ok := autoPairPrevChar(doc, cursor)
	if !ok {
		return true
	}
	return !autoPairIsAlphanumeric(ch)
}

func autoPairCharAt(doc Rope, pos int) (rune, bool) {
	ch, err := doc.CharAt(pos)
	if err != nil {
		return 0, false
	}
	return ch, true
}

func autoPairPrevChar(doc Rope, pos int) (rune, bool) {
	if pos == 0 {
		return 0, false
	}
	return autoPairCharAt(doc, pos-1)
}

func autoPairIsAlphanumeric(ch rune) bool {
	return CharIsWord(ch)
}

func autoPairIsWhitespace(ch rune) bool {
	return CharIsWhitespace(ch) || CharIsLineEnding(ch)
}
