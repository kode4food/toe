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

// VisualRowOfOffset returns the zero-based visual row within its text line on
// which the character at charOff (relative to line start) is displayed
func (vf *VisualMoveFormat) VisualRowOfOffset(doc Rope, line, charOff int) int {
	if vf == nil || vf.ViewportWidth <= 0 {
		return 0
	}
	return newVisualLine(doc, line, vf).posOf(charOff).Y
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
				Line: args.Line, Row: args.Row - args.Up,
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
	cur := vl.posOf(cursor - lineStart)

	if dir == DirectionForward {
		total := vl.rowCount()
		remaining := count
		rowsBelow := total - 1 - cur.Y
		if remaining <= rowsBelow {
			off := vl.charAtPos(cur.Add(geom.Point{Y: remaining}))
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
				off := tl.charAtPos(geom.Point{X: cur.X, Y: remaining})
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
		off := tl.charAtPos(geom.Point{X: cur.X})
		return r.PutCursor(doc, tStart+off, move == MovementExtend)
	}

	// DirectionBackward
	remaining := count
	if remaining <= cur.Y {
		off := vl.charAtPos(cur.Sub(geom.Point{Y: remaining}))
		return r.PutCursor(doc, lineStart+off, move == MovementExtend)
	}
	remaining -= cur.Y + 1
	prevLine := line - 1
	for remaining > 0 && prevLine > 0 {
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
	off := tl.charAtPos(geom.Point{X: cur.X, Y: tRows - 1})
	return r.PutCursor(doc, tStart+off, move == MovementExtend)
}
