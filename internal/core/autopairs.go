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

var defaultAutoPairRunes = [][2]rune{
	{'(', ')'},
	{'{', '}'},
	{'[', ']'},
	{'\'', '\''},
	{'"', '"'},
	{'`', '`'},
}

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
	return NewAutoPairs(defaultAutoPairRunes)
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
