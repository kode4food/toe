package core

// TextObjectParagraph selects the paragraph containing the cursor. A paragraph
// is a contiguous sequence of non-empty lines around=TextObjectAround includes
// the trailing empty lines
func TextObjectParagraph(
	doc Rope, r Range, kind TextObjectKind, count int,
) Range {
	nLines := doc.LenLines()
	line, err := doc.CharToLine(r.Cursor(doc))
	if err != nil {
		return r
	}

	isBlank := func(l int) bool {
		if l < 0 || l >= nLines {
			return true
		}
		lineRope, e := doc.Line(l)
		if e != nil {
			return true
		}
		return isBlankLine(lineRope.String())
	}

	// Walk backward to find the start of the current paragraph block
	startLine := line
	for startLine > 0 && !isBlank(startLine-1) {
		startLine--
	}

	// Walk forward through count paragraphs
	endLine := line
	for range count {
		for endLine < nLines && !isBlank(endLine) {
			endLine++
		}
		if kind == TextObjectAround {
			for endLine < nLines && isBlank(endLine) {
				endLine++
			}
		}
	}

	// Inside: trim trailing blank lines from the end
	if kind == TextObjectInside {
		for endLine > startLine && isBlank(endLine-1) {
			endLine--
		}
	}

	from, err2 := doc.LineToChar(startLine)
	if err2 != nil {
		return r
	}
	var to int
	if endLine >= nLines {
		to = doc.LenChars()
	} else {
		to, err2 = doc.LineToChar(endLine)
		if err2 != nil {
			return r
		}
	}
	return NewRange(from, to)
}

// TextObjectPairSurround selects the pair surrounding the cursor. ch,
// if non-zero, specifies which pair; zero uses the nearest pair. kind controls
// whether the delimiters themselves are included
func (r Range) TextObjectPairSurround(
	doc Rope, kind TextObjectKind, ch rune, count int,
) Range {
	sel, err := NewSelection([]Range{r}, 0)
	if err != nil {
		return r
	}
	var positions []int
	if ch != 0 {
		positions, err = GetSurroundPosFor(doc, sel, ch, count)
	} else {
		positions, err = GetSurroundPos(doc, sel, count)
	}
	if err != nil || len(positions) < 2 {
		return r
	}
	anchor, head := positions[0], positions[1]
	if kind == TextObjectInside {
		// Move one grapheme inward from each delimiter
		anchor = NextGraphemeBoundary(doc, anchor)
	} else {
		// Around: include the closing delimiter
		head = NextGraphemeBoundary(doc, head)
	}
	if r.Direction() == DirectionForward {
		return NewRange(anchor, head)
	}
	return NewRange(head, anchor)
}

func isBlankLine(s string) bool {
	for _, ch := range s {
		if ch != ' ' && ch != '\t' && ch != '\r' && ch != '\n' {
			return false
		}
	}
	return true
}
