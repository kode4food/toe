package core

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
