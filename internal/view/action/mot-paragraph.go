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
				l = skipConsecutiveBlanks(doc, l, nLines, 1)
				l++
				found++
				if found >= n || l >= nLines {
					target := min(l, nLines-1)
					pos, err := doc.LineToChar(target)
					if err != nil {
						return r
					}
					return r.PutCursor(doc, pos, extend)
				}
			}
		}
		pos, err := doc.LineToChar(nLines - 1)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, pos, extend)
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
				l = skipConsecutiveBlanks(doc, l, nLines, -1)
				found++
				if found >= n || l <= 0 {
					target := max(l-1, 0)
					pos, err := doc.LineToChar(target)
					if err != nil {
						return r
					}
					return r.PutCursor(doc, pos, extend)
				}
			}
		}
		pos, err := doc.LineToChar(0)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, pos, extend)
	})
}

func skipConsecutiveBlanks(doc core.Rope, l, nLines, step int) int {
	for {
		next := l + step
		if next < 0 || next >= nLines {
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
