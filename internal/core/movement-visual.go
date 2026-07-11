package core

// VisualRows returns the number of visual (soft-wrapped) rows the given text
// line occupies. It returns 1 when soft-wrap is inactive
func (vf *VisualMoveFormat) VisualRows(doc Rope, line int) int {
	if vf == nil || vf.ViewportWidth <= 0 {
		return 1
	}
	return newVisualLine(doc, line, vf).rowCount()
}

// VisualRowOfOffset returns the zero-based visual row within its text line on
// which the character at charOff (relative to line start) is displayed
func (vf *VisualMoveFormat) VisualRowOfOffset(doc Rope, line, charOff int) int {
	if vf == nil || vf.ViewportWidth <= 0 {
		return 0
	}
	row, _ := newVisualLine(doc, line, vf).posOf(charOff)
	return row
}

// VisualScrollUp moves up by up visual rows from (line, row), clamped at the
// start of the document
func (vf *VisualMoveFormat) VisualScrollUp(
	doc Rope, line, row, up int,
) (int, int) {
	for up > 0 {
		if row >= up {
			return line, row - up
		}
		up -= row + 1
		if line == 0 {
			return 0, 0
		}
		line--
		row = vf.VisualRows(doc, line) - 1
	}
	return line, row
}

func (v visualMover) moveVertically(
	doc Rope, r Range, dir Direction, count int,
) Range {
	vf := v.format
	move := v.movement
	if vf == nil || vf.ViewportWidth <= 0 {
		return r.MoveVertically(doc, dir, count, move)
	}

	cursor := r.Cursor(doc)
	line, err := doc.CharToLine(cursor)
	if err != nil {
		return r
	}
	lineStart, err := doc.LineToChar(line)
	if err != nil {
		return r
	}

	vl := newVisualLine(doc, line, vf)
	curRow, curCol := vl.posOf(cursor - lineStart)

	if dir == DirectionForward {
		total := vl.rowCount()
		remaining := count
		rowsBelow := total - 1 - curRow
		if remaining <= rowsBelow {
			off := vl.charAtPos(curRow+remaining, curCol)
			return r.PutCursor(doc, lineStart+off, move == MovementExtend)
		}
		remaining -= rowsBelow + 1
		nextLine := line + 1
		nLines := doc.LenLines()
		for remaining > 0 && nextLine < nLines-1 {
			tStart, err := doc.LineToChar(nextLine)
			if err != nil {
				break
			}
			tl := newVisualLine(doc, nextLine, vf)
			tRows := tl.rowCount()
			if remaining < tRows {
				off := tl.charAtPos(remaining, curCol)
				return r.PutCursor(doc, tStart+off, move == MovementExtend)
			}
			remaining -= tRows
			nextLine++
		}
		tLine := min(nextLine, nLines-1)
		tStart, err := doc.LineToChar(tLine)
		if err != nil {
			return r
		}
		tl := newVisualLine(doc, tLine, vf)
		off := tl.charAtPos(0, curCol)
		return r.PutCursor(doc, tStart+off, move == MovementExtend)
	}

	// DirectionBackward
	remaining := count
	if remaining <= curRow {
		off := vl.charAtPos(curRow-remaining, curCol)
		return r.PutCursor(doc, lineStart+off, move == MovementExtend)
	}
	remaining -= curRow + 1
	prevLine := line - 1
	for remaining > 0 && prevLine > 0 {
		tStart, err := doc.LineToChar(prevLine)
		if err != nil {
			break
		}
		tl := newVisualLine(doc, prevLine, vf)
		tRows := tl.rowCount()
		if remaining < tRows {
			off := tl.charAtPos(tRows-1-remaining, curCol)
			return r.PutCursor(doc, tStart+off, move == MovementExtend)
		}
		remaining -= tRows
		prevLine--
	}
	tLine := max(prevLine, 0)
	tStart, err := doc.LineToChar(tLine)
	if err != nil {
		return r
	}
	tl := newVisualLine(doc, tLine, vf)
	tRows := tl.rowCount()
	off := tl.charAtPos(tRows-1, curCol)
	return r.PutCursor(doc, tStart+off, move == MovementExtend)
}
