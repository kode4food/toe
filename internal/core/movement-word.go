package core

func newCharIterAt(doc Rope, charIdx int) *charIter {
	return &charIter{runes: []rune(doc.String()), pos: charIdx}
}

func (c *charIter) next() (rune, bool) {
	if c.rev {
		if c.pos == 0 {
			return 0, false
		}
		c.pos--
		return c.runes[c.pos], true
	}
	if c.pos >= len(c.runes) {
		return 0, false
	}
	ch := c.runes[c.pos]
	c.pos++
	return ch, true
}

func (c *charIter) prev() (rune, bool) {
	if c.rev {
		if c.pos >= len(c.runes) {
			return 0, false
		}
		ch := c.runes[c.pos]
		c.pos++
		return ch, true
	}
	if c.pos == 0 {
		return 0, false
	}
	c.pos--
	return c.runes[c.pos], true
}

func (c *charIter) reverse() {
	c.rev = !c.rev
}

func wordMove(doc Rope, r Range, count int, target WordMotionTarget) Range {
	prev := isPrevWordMotion(target)

	n := doc.LenChars()
	if prev && r.Head == 0 {
		return r
	}
	if !prev && r.Head == n {
		return r
	}

	var start Range
	if prev {
		if r.Anchor < r.Head {
			start = NewRange(r.Head, PrevGraphemeBoundary(doc, r.Head))
		} else {
			start = NewRange(NextGraphemeBoundary(doc, r.Head), r.Head)
		}
	} else {
		if r.Anchor < r.Head {
			start = NewRange(PrevGraphemeBoundary(doc, r.Head), r.Head)
		} else {
			start = NewRange(r.Head, NextGraphemeBoundary(doc, r.Head))
		}
	}

	cur := start
	for range count {
		next := rangeToTarget(doc, cur, target)
		if next == cur {
			break
		}
		cur = next
	}
	return cur
}

// rangeToTarget extends or moves origin to reach the given word motion target
func rangeToTarget(doc Rope, origin Range, target WordMotionTarget) Range {
	prev := isPrevWordMotion(target)

	it := newCharIterAt(doc, origin.Head)
	if prev {
		it.reverse()
	}

	var advance func(int) int
	if prev {
		advance = func(idx int) int {
			if idx > 0 {
				return idx - 1
			}
			return idx
		}
	} else {
		advance = func(idx int) int { return idx + 1 }
	}

	anchor := origin.Anchor
	head := origin.Head

	// Peek at the character before head for context
	var prevCh rune
	hasPrev := false
	if ch, ok := it.prev(); ok {
		prevCh = ch
		hasPrev = true
		_, _ = it.next()
	}

	// Skip initial newline characters
	for {
		ch, ok := it.next()
		if !ok {
			break
		}
		if !CharIsLineEnding(ch) {
			_, _ = it.prev()
			break
		}
		prevCh = ch
		hasPrev = true
		head = advance(head)
	}
	if hasPrev && CharIsLineEnding(prevCh) {
		anchor = head
	}

	// Find the target position
	headStart := head
	for {
		nextCh, ok := it.next()
		if !ok {
			break
		}
		if !hasPrev || reachedTarget(target, prevCh, nextCh) {
			if head == headStart {
				anchor = head
			} else {
				break
			}
		}
		prevCh = nextCh
		hasPrev = true
		head = advance(head)
	}

	return NewRange(anchor, head)
}

func reachedTarget(target WordMotionTarget, prev, next rune) bool {
	switch target {
	case WordMotionNextWordStart, WordMotionPrevWordEnd:
		return isWordBoundary(prev, next) && atWordStartPos(next)
	case WordMotionNextWordEnd, WordMotionPrevWordStart:
		return isWordBoundary(prev, next) && atWordEndPos(prev, next)
	case WordMotionNextLongWordStart, WordMotionPrevLongWordEnd:
		return isLongWordBoundary(prev, next) && atWordStartPos(next)
	case WordMotionNextLongWordEnd, WordMotionPrevLongWordStart:
		return isLongWordBoundary(prev, next) && atWordEndPos(prev, next)
	case WordMotionNextSubWordStart:
		return isSubWordBoundary(prev, next, DirectionForward) &&
			atSubWordStartPos(next)
	case WordMotionPrevSubWordEnd:
		return isSubWordBoundary(prev, next, DirectionBackward) &&
			atSubWordStartPos(next)
	case WordMotionNextSubWordEnd:
		return isSubWordBoundary(prev, next, DirectionForward) &&
			atSubWordEndPos(prev, next)
	case WordMotionPrevSubWordStart:
		return isSubWordBoundary(prev, next, DirectionBackward) &&
			atSubWordEndPos(prev, next)
	default:
		return false
	}
}
