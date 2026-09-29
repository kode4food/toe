package core

import "github.com/kode4food/toe/internal/geom"

// VisualRows returns the number of visual (soft-wrapped) rows the given text
// line occupies. It returns 1 when soft-wrap is inactive
func (vf *VisualMoveFormat) VisualRows(doc Rope, line int) int {
	if vf == nil || vf.ViewportWidth <= 0 {
		return 1
	}
	return newVisualLine(doc, line, vf).rowCount()
}

// VisualRowOfOffsetArgs is a text line and a character offset within it
type VisualRowOfOffsetArgs struct {
	Doc     Rope
	Line    int
	CharOff int
}

// VisualRowOfOffset returns the zero-based visual row within its text line on
// which the character at CharOff (relative to line start) is displayed
func (vf *VisualMoveFormat) VisualRowOfOffset(args VisualRowOfOffsetArgs) int {
	if vf == nil || vf.ViewportWidth <= 0 {
		return 0
	}
	return newVisualLine(args.Doc, args.Line, vf).posOf(args.CharOff).Y
}

type (
	// VisualScrollUpArgs identifies a visual row and upward distance
	VisualScrollUpArgs struct {
		Doc  Rope
		Line int
		Row  int
		Up   int
	}

	// VisualScrollUpRes identifies the resulting line and visual row
	VisualScrollUpRes struct {
		Line int
		Row  int
	}
)

// VisualScrollUp moves upward from a visual row, clamped at the document start
func (vf *VisualMoveFormat) VisualScrollUp(
	args VisualScrollUpArgs,
) VisualScrollUpRes {
	for args.Up > 0 {
		if args.Row >= args.Up {
			return VisualScrollUpRes{
				Line: args.Line,
				Row:  args.Row - args.Up,
			}
		}
		args.Up -= args.Row + 1
		if args.Line == 0 {
			return VisualScrollUpRes{}
		}
		args.Line--
		args.Row = vf.VisualRows(args.Doc, args.Line) - 1
	}
	return VisualScrollUpRes{Line: args.Line, Row: args.Row}
}

func (v visualMover) moveVertically(
	doc Rope, r Range, mv VerticalMove,
) (Range, int) {
	vf := v.format
	if vf == nil || vf.ViewportWidth <= 0 {
		return r.MoveVertically(doc, mv)
	}
	count := mv.Count
	extend := mv.Movement == MovementExtend

	cursor := r.Cursor(doc)
	line, err := doc.CharToLine(cursor)
	if err != nil {
		return r, mv.GoalColumn
	}
	lineStart, err := doc.LineToChar(line)
	if err != nil {
		return r, mv.GoalColumn
	}

	vl := newVisualLine(doc, line, vf)
	cur := vl.posOf(cursor - lineStart)
	// a short row clamped the cursor, but the goal column is not forgotten
	cur.X = max(mv.GoalColumn, cur.X)
	goal := cur.X

	if mv.Dir == DirectionForward {
		total := vl.rowCount()
		remaining := count
		rowsBelow := total - 1 - cur.Y
		if remaining <= rowsBelow {
			off := vl.charAtPos(cur.Add(geom.Point{Y: remaining}))
			return r.PutCursor(doc, lineStart+off, extend), goal
		}
		remaining -= rowsBelow + 1
		nextLine := line + 1
		nLines := doc.LenLines()
		for nextLine < nLines {
			tStart, err := doc.LineToChar(nextLine)
			if err != nil {
				break
			}
			tl := newVisualLine(doc, nextLine, vf)
			tRows := tl.rowCount()
			if remaining < tRows {
				off := tl.charAtPos(geom.Point{X: cur.X, Y: remaining})
				return r.PutCursor(doc, tStart+off, extend), goal
			}
			remaining -= tRows
			nextLine++
		}
		// ran past the end, so clamp to the last row of the last line
		tLine := min(nextLine, nLines-1)
		tStart, err := doc.LineToChar(tLine)
		if err != nil {
			return r, goal
		}
		tl := newVisualLine(doc, tLine, vf)
		off := tl.charAtPos(geom.Point{X: cur.X, Y: tl.rowCount() - 1})
		return r.PutCursor(doc, tStart+off, extend), goal
	}

	// DirectionBackward
	remaining := count
	if remaining <= cur.Y {
		off := vl.charAtPos(cur.Sub(geom.Point{Y: remaining}))
		return r.PutCursor(doc, lineStart+off, extend), goal
	}
	remaining -= cur.Y + 1
	prevLine := line - 1
	for prevLine >= 0 {
		tStart, err := doc.LineToChar(prevLine)
		if err != nil {
			break
		}
		tl := newVisualLine(doc, prevLine, vf)
		tRows := tl.rowCount()
		if remaining < tRows {
			off := tl.charAtPos(geom.Point{
				X: cur.X, Y: tRows - 1 - remaining,
			})
			return r.PutCursor(doc, tStart+off, extend), goal
		}
		remaining -= tRows
		prevLine--
	}
	// ran past the start, so clamp to the first row of the first line
	tStart, err := doc.LineToChar(0)
	if err != nil {
		return r, goal
	}
	tl := newVisualLine(doc, 0, vf)
	off := tl.charAtPos(geom.Point{X: cur.X})
	return r.PutCursor(doc, tStart+off, extend), goal
}
