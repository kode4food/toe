package core

// paragraphLines is a document and its line count, for blank-line tests
type paragraphLines struct {
	doc       Rope
	lineCount int
}

// TextObjectParagraph selects the paragraph containing the cursor. A paragraph
// is a contiguous sequence of non-empty lines around=TextObjectAround includes
// the trailing empty lines
func TextObjectParagraph(
	doc Rope, r Range, kind TextObjectKind, count int,
) Range {
	lines := paragraphLines{doc: doc, lineCount: doc.LenLines()}
	cursor := r.Cursor(doc)
	line, err := doc.CharToLine(cursor)
	if err != nil {
		return r
	}
	if count < 1 {
		count = 1
	}

	prevEmpty := lines.isBlank(line - 1)
	currEmpty := lines.isBlank(line)
	nextEmpty := line+1 >= lines.lineCount || lines.isBlank(line+1)
	nextStart := paragraphLineToChar(doc, line+1)
	lastChar := PrevGraphemeBoundary(doc, nextStart) == cursor
	prevEmptyToLine := prevEmpty && !currEmpty
	currEmptyToLine := currEmpty && !nextEmpty

	lineBack := line
	if prevEmptyToLine || currEmptyToLine {
		lineBack++
	}
	if !(currEmptyToLine && lastChar) {
		for lineBack > 0 && lines.isBlank(lineBack-1) {
			lineBack--
		}
		for lineBack > 0 && !lines.isBlank(lineBack-1) {
			lineBack--
		}
	}

	if currEmptyToLine && lastChar {
		line++
	}
	countDone := 0
	for range count {
		done := false
		for line < lines.lineCount && !lines.isBlank(line) {
			line++
			done = true
		}
		for line < lines.lineCount && lines.isBlank(line) {
			line++
		}
		if done {
			countDone++
		}
	}
	if countDone != count && line >= lines.lineCount {
		for lineBack > 0 && lines.isBlank(lineBack-1) {
			lineBack--
		}
		for lineBack > 0 && !lines.isBlank(lineBack-1) {
			lineBack--
		}
	}
	if kind == TextObjectInside {
		for line > lineBack && lines.isBlank(line-1) {
			line--
		}
	}

	from := paragraphLineToChar(doc, lineBack)
	to := paragraphLineToChar(doc, line)
	return Range{Anchor: from, Head: to}
}

// TextObjectPairSurround selects the pair surrounding the cursor. ch, if
// non-zero, specifies which pair. Zero uses the nearest pair. kind controls
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
		return Range{Anchor: anchor, Head: head}
	}
	return Range{Anchor: head, Head: anchor}
}

func (p paragraphLines) isBlank(line int) bool {
	if line < 0 || line >= p.lineCount {
		return true
	}
	if lineRope, err := p.doc.Line(line); err == nil {
		return isBlankLine(lineRope.String())
	}
	return true
}

func isBlankLine(s string) bool {
	for _, ch := range s {
		if ch != ' ' && ch != '\t' && !CharIsLineEnding(ch) {
			return false
		}
	}
	return true
}

func paragraphLineToChar(doc Rope, line int) int {
	if line >= doc.LenLines() {
		return doc.LenChars()
	}
	if pos, err := doc.LineToChar(line); err == nil {
		return pos
	}
	return doc.LenChars()
}
