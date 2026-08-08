package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// GotoNextParagraph moves (or extends in select mode) each cursor to the start
// of the next paragraph. A paragraph boundary is a blank line
func GotoNextParagraph(e *view.Editor) {
	e.SetLastMotion(GotoNextParagraph)
	n := countOrOne(e)
	extend := e.Mode() == view.ModeSelect
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		nLines := doc.LenLines()
		found := 0
		for l := line + 1; l < nLines; l++ {
			lr, err := doc.Line(l)
			if err != nil {
				break
			}
			if isBlankLine(lr.String()) {
				l = skipConsecutiveBlanks(doc, skipBlanksArgs{
					line:      l,
					lineCount: nLines,
					step:      1,
				})
				l++
				found++
				if found >= n || l >= nLines {
					target := min(l, nLines-1)
					if pos, err := doc.LineToChar(target); err == nil {
						return r.PutCursor(doc, pos, extend)
					}
					return r
				}
			}
		}
		if pos, err := doc.LineToChar(nLines - 1); err == nil {
			return r.PutCursor(doc, pos, extend)
		}
		return r
	})
}

// GotoPrevParagraph moves (or extends in select mode) each cursor to the start
// of the previous paragraph
func GotoPrevParagraph(e *view.Editor) {
	e.SetLastMotion(GotoPrevParagraph)
	n := countOrOne(e)
	extend := e.Mode() == view.ModeSelect
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		nLines := doc.LenLines()
		found := 0
		for l := line - 1; l >= 0; l-- {
			lr, err := doc.Line(l)
			if err != nil {
				break
			}
			if isBlankLine(lr.String()) {
				l = skipConsecutiveBlanks(doc, skipBlanksArgs{
					line:      l,
					lineCount: nLines,
					step:      -1,
				})
				found++
				if found >= n || l <= 0 {
					target := max(l-1, 0)
					if pos, err := doc.LineToChar(target); err == nil {
						return r.PutCursor(doc, pos, extend)
					}
					return r
				}
			}
		}
		if pos, err := doc.LineToChar(0); err == nil {
			return r.PutCursor(doc, pos, extend)
		}
		return r
	})
}

type skipBlanksArgs struct {
	line      int
	lineCount int
	step      int
}

func skipConsecutiveBlanks(doc core.Rope, args skipBlanksArgs) int {
	l := args.line
	for {
		next := l + args.step
		if next < 0 || next >= args.lineCount {
			break
		}
		lr, err := doc.Line(next)
		if err != nil {
			break
		}
		if !isBlankLine(lr.String()) {
			break
		}
		l = next
	}
	return l
}
