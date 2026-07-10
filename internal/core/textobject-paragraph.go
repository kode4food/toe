package core

// TextObjectParagraph selects the paragraph containing the cursor. A paragraph
// is a contiguous sequence of non-empty lines around=TextObjectAround includes
// the trailing empty lines
func TextObjectParagraph(
	doc Rope, r Range, kind TextObjectKind, count int,
) Range {
	nLines := doc.LenLines()
	cursor := r.Cursor(doc)
	line, err := doc.CharToLine(cursor)
	if err != nil {
		return r
	}
	if count < 1 {
		count = 1
	}

	prevEmpty := paragraphLineBlank(doc, line-1, nLines)
	currEmpty := paragraphLineBlank(doc, line, nLines)
	nextEmpty := line+1 >= nLines || paragraphLineBlank(doc, line+1, nLines)
	nextStart := paragraphLineToChar(doc, line+1)
	lastChar := PrevGraphemeBoundary(doc, nextStart) == cursor
	prevEmptyToLine := prevEmpty && !currEmpty
	currEmptyToLine := currEmpty && !nextEmpty

	lineBack := line
	if prevEmptyToLine || currEmptyToLine {
		lineBack++
	}
	if !(currEmptyToLine && lastChar) {
		for lineBack > 0 && paragraphLineBlank(doc, lineBack-1, nLines) {
			lineBack--
		}
		for lineBack > 0 && !paragraphLineBlank(doc, lineBack-1, nLines) {
			lineBack--
		}
	}

	if currEmptyToLine && lastChar {
		line++
	}
	countDone := 0
	for range count {
		done := false
		for line < nLines && !paragraphLineBlank(doc, line, nLines) {
			line++
			done = true
		}
		for line < nLines && paragraphLineBlank(doc, line, nLines) {
			line++
		}
		if done {
			countDone++
		}
	}
	if countDone != count && line >= nLines {
		for lineBack > 0 && paragraphLineBlank(doc, lineBack-1, nLines) {
			lineBack--
		}
		for lineBack > 0 && !paragraphLineBlank(doc, lineBack-1, nLines) {
			lineBack--
		}
	}
	if kind == TextObjectInside {
		for line > lineBack && paragraphLineBlank(doc, line-1, nLines) {
			line--
		}
	}

	from := paragraphLineToChar(doc, lineBack)
	to := paragraphLineToChar(doc, line)
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
		if ch != ' ' && ch != '\t' && !CharIsLineEnding(ch) {
			return false
		}
	}
	return true
}

func paragraphLineBlank(doc Rope, line, nLines int) bool {
	if line < 0 || line >= nLines {
		return true
	}
	lineRope, err := doc.Line(line)
	if err != nil {
		return true
	}
	return isBlankLine(lineRope.String())
}

func paragraphLineToChar(doc Rope, line int) int {
	if line >= doc.LenLines() {
		return doc.LenChars()
	}
	pos, err := doc.LineToChar(line)
	if err != nil {
		return doc.LenChars()
	}
	return pos
}
