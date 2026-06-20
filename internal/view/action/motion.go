package action

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

var ErrNoFilePath = errors.New("no file path under cursor")

// AddNewlineAbove inserts blank lines above each selection's first line.
// Repeats count times using the document line ending
func AddNewlineAbove(e *view.Editor) {
	addNewlineImpl(e, true)
}

// AddNewlineBelow inserts blank lines below each selection's last line. Repeats
// count times using the document line ending
func AddNewlineBelow(e *view.Editor) {
	addNewlineImpl(e, false)
}

func addNewlineImpl(e *view.Editor, above bool) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	count := max(1, e.Count())
	nl := strings.Repeat(string(doc.LineEnding()), count)
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	seen := map[int]bool{}
	changes := make([]core.Change, 0, len(sel.Ranges()))
	for _, r := range sel.Ranges() {
		lr, err := r.LineRange(text)
		if err != nil {
			continue
		}
		var targetLine int
		if above {
			targetLine = lr.From
		} else {
			targetLine = lr.To + 1
		}
		pos, err := text.LineToChar(targetLine)
		if err != nil {
			continue
		}
		if seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.TextChange(pos, pos, nl))
	}
	if len(changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// AlignSelections inserts spaces before each cursor so all cursors sit at the
// same visual column (the maximum column among all cursors). Only operates
// when there are multiple selection ranges, all on different lines
func AlignSelections(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	if len(ranges) < 2 {
		return
	}

	// Compute the visual column of each cursor's anchor
	cols := make([]int, len(ranges))
	maxCol := 0
	for i, r := range ranges {
		pos := r.Cursor(text)
		line, err := text.CharToLine(pos)
		if err != nil {
			return
		}
		lineStart, err := text.LineToChar(line)
		if err != nil {
			return
		}
		col := pos - lineStart
		cols[i] = col
		if col > maxCol {
			maxCol = col
		}
	}

	// Insert spaces to bring each cursor to maxCol
	changes := make([]core.Change, 0, len(ranges))
	for i, r := range ranges {
		pad := maxCol - cols[i]
		if pad <= 0 {
			continue
		}
		pos := r.Cursor(text)
		changes = append(changes, core.TextChange(pos, pos, strings.Repeat(" ", pad)))
	}
	if len(changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// GotoFile opens the file whose path the primary cursor sits on. Returns the
// resolved path, or an error if no valid path can be found
func GotoFile(e *view.Editor) (string, error) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return "", view.ErrNoDocument
	}
	v, ok := e.FocusedView()
	if !ok {
		return "", view.ErrNoView
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	pos := sel.Primary().Cursor(text)

	// Expand outward from the cursor to capture a file-path token
	n := text.LenChars()
	from := pos
	for from > 0 {
		ch, err := text.CharAt(from - 1)
		if err != nil || isPathDelim(ch) {
			break
		}
		from--
	}
	to := pos
	for to < n {
		ch, err := text.CharAt(to)
		if err != nil || isPathDelim(ch) {
			break
		}
		to++
	}
	if from >= to {
		return "", ErrNoFilePath
	}
	slice, err := text.Slice(from, to)
	if err != nil {
		return "", err
	}
	path := slice.String()

	// Resolve relative paths against the document's directory
	if !strings.HasPrefix(path, "/") {
		base := doc.Path()
		if base != "" {
			base = base[:strings.LastIndex(base, "/")+1]
		} else {
			base = e.Cwd() + "/"
		}
		path = base + path
	}

	// Check the file exists
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("%w: '%s'", err, path)
	}
	return path, nil
}

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
				for l+1 < nLines {
					next, err := doc.Line(l + 1)
					if err != nil {
						break
					}
					if !isBlankLine(next.String()) {
						break
					}
					l++
				}
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
		// No more paragraphs: go to last line
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
		found := 0
		for l := line - 1; l >= 0; l-- {
			lr, err := doc.Line(l)
			if err != nil {
				break
			}
			if isBlankLine(lr.String()) {
				// Skip consecutive blank lines upward
				for l-1 >= 0 {
					prev, err := doc.Line(l - 1)
					if err != nil {
						break
					}
					if !isBlankLine(prev.String()) {
						break
					}
					l--
				}
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
		// No more paragraphs: go to first line
		pos, err := doc.LineToChar(0)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, pos, extend)
	})
}

// MoveLeft moves all cursors one grapheme to the left
func MoveLeft(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveHorizontally(
			doc, core.DirectionBackward, n, core.MovementMove,
		)
	})
}

// MoveRight moves all cursors one grapheme to the right
func MoveRight(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveHorizontally(
			doc, core.DirectionForward, n, core.MovementMove,
		)
	})
}

// MoveUp moves all cursors up one visual line, respecting soft-wrap
func MoveUp(e *view.Editor) {
	n := countOrOne(e)
	vf := visualMoveFormat(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return vf.MoveVerticallyVisual(doc, r, core.DirectionBackward, n)
	})
}

// MoveDown moves all cursors down one visual line, respecting soft-wrap
func MoveDown(e *view.Editor) {
	n := countOrOne(e)
	vf := visualMoveFormat(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return vf.MoveVerticallyVisual(doc, r, core.DirectionForward, n)
	})
}

// MoveWordForward moves all cursors to the start of the next word
func MoveWordForward(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextWordStart(doc, r, n)
	})
}

// MoveWordBackward moves all cursors to the start of the previous word
func MoveWordBackward(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevWordStart(doc, r, n)
	})
}

// MoveWordEnd moves all cursors to the end of the current or next word
func MoveWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextWordEnd(doc, r, n)
	})
}

// MoveLongWordForward moves all cursors to the start of the next WORD
func MoveLongWordForward(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextLongWordStart(doc, r, n)
	})
}

// MoveLongWordBackward moves all cursors to the start of the previous WORD
func MoveLongWordBackward(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevLongWordStart(doc, r, n)
	})
}

// MoveLongWordEnd moves all cursors to the end of the current or next WORD
func MoveLongWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextLongWordEnd(doc, r, n)
	})
}

// MovePrevWordEnd moves all cursors to the end of the previous word
func MovePrevWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevWordEnd(doc, r, n)
	})
}

// MovePrevLongWordEnd moves all cursors to the end of the previous WORD
func MovePrevLongWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevLongWordEnd(doc, r, n)
	})
}

// MoveNextSubWordStart moves to the start of the next sub-word
func MoveNextSubWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextSubWordStart(doc, r, n)
	})
}

// MovePrevSubWordStart moves to the start of the previous sub-word
func MovePrevSubWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevSubWordStart(doc, r, n)
	})
}

// MoveNextSubWordEnd moves to the end of the next sub-word
func MoveNextSubWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextSubWordEnd(doc, r, n)
	})
}

// MovePrevSubWordEnd moves to the end of the previous sub-word
func MovePrevSubWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevSubWordEnd(doc, r, n)
	})
}

// MoveLineStart moves all cursors to the start of their current line
func MoveLineStart(e *view.Editor) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		start, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, start, false)
	})
}

// MoveLineEnd moves all cursors to the last non-newline character of
// their current line
func MoveLineEnd(e *view.Editor) {
	moveLineEnd(e, false)
}

// MoveLineNonWhitespace moves all cursors to the first non-whitespace
// character of their current line
func MoveLineNonWhitespace(e *view.Editor) {
	moveToNonWhitespace(e, e.Mode() == view.ModeSelect)
}

// ExtendToNonWhitespace extends the selection to the first non-whitespace
// character on the current line
func ExtendToNonWhitespace(e *view.Editor) {
	moveToNonWhitespace(e, true)
}

// MoveFileStart moves all cursors to the start of the document
func MoveFileStart(e *view.Editor) {
	SaveSelection(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.PutCursor(doc, 0, false)
	})
}

// MoveFileEnd moves all cursors to the start of the last non-blank line
func MoveFileEnd(e *view.Editor) {
	moveFileEnd(e, false)
}

// ExtendCharLeft extends the selection one grapheme to the left
func ExtendCharLeft(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveHorizontally(
			doc, core.DirectionBackward, n, core.MovementExtend,
		)
	})
}

// ExtendCharRight extends the selection one grapheme to the right
func ExtendCharRight(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveHorizontally(
			doc, core.DirectionForward, n, core.MovementExtend,
		)
	})
}

// ExtendLineUp extends the selection up one visual line, respecting soft-wrap
func ExtendLineUp(e *view.Editor) {
	n := countOrOne(e)
	vf := visualMoveFormat(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return vf.ExtendVerticallyVisual(doc, r, core.DirectionBackward, n)
	})
}

// ExtendLineDown extends the selection down one visual line, respecting
// soft-wrap
func ExtendLineDown(e *view.Editor) {
	n := countOrOne(e)
	vf := visualMoveFormat(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return vf.ExtendVerticallyVisual(doc, r, core.DirectionForward, n)
	})
}

// ExtendNextWordStart extends the selection to the start of the next word
func ExtendNextWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevWordStart extends the selection to the start of the previous word
func ExtendPrevWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextWordEnd extends the selection to the end of the next word
func ExtendNextWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextLongWordStart extends the selection to the start of the next WORD
func ExtendNextLongWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextLongWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevLongWordStart extends to the start of the previous WORD
func ExtendPrevLongWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevLongWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextLongWordEnd extends the selection to the end of the next WORD
func ExtendNextLongWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextLongWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevWordEnd extends the selection to the end of the previous word
func ExtendPrevWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevLongWordEnd extends the selection to the end of the previous WORD
func ExtendPrevLongWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevLongWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextSubWordStart extends to the start of the next sub-word
func ExtendNextSubWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextSubWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevSubWordStart extends to the start of the previous sub-word
func ExtendPrevSubWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevSubWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextSubWordEnd extends to the end of the next sub-word
func ExtendNextSubWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextSubWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevSubWordEnd extends to the end of the previous sub-word
func ExtendPrevSubWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevSubWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendToLineStart extends the selection to the start of the current line
func ExtendToLineStart(e *view.Editor) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		start, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, start, true)
	})
}

// ExtendToLineEnd extends the selection to the end of the current line
func ExtendToLineEnd(e *view.Editor) {
	moveLineEnd(e, true)
}

// ExtendToFileStart extends the selection to the beginning of the document
func ExtendToFileStart(e *view.Editor) {
	SaveSelection(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.PutCursor(doc, 0, true)
	})
}

// ExtendToLastLine extends to the start of the last non-blank line
func ExtendToLastLine(e *view.Editor) {
	moveFileEnd(e, true)
}

// GotoFileEnd moves all cursors to the absolute end of the document
// (past all characters, including any trailing newline)
func GotoFileEnd(e *view.Editor) {
	SaveSelection(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.PutCursor(doc, doc.LenChars(), false)
	})
}

// ExtendToFileEnd extends all selections to the absolute end of the document
func ExtendToFileEnd(e *view.Editor) {
	SaveSelection(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.PutCursor(doc, doc.LenChars(), true)
	})
}

func moveLineEnd(e *view.Editor, extend bool) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		lineEnd, err := doc.LineEndCharIndex(line)
		if err != nil {
			return r
		}
		pos := max(lineEnd-1, 0)
		start, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		if pos < start {
			pos = start
		}
		return r.PutCursor(doc, pos, extend)
	})
}

func moveToNonWhitespace(e *view.Editor, extend bool) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		lineEnd, err := doc.LineEndCharIndex(line)
		if err != nil {
			return r
		}
		start, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		pos := start
		for pos < lineEnd {
			ch, err := doc.CharAt(pos)
			if err != nil {
				break
			}
			if ch != ' ' && ch != '\t' {
				break
			}
			pos++
		}
		return r.PutCursor(doc, pos, extend)
	})
}

func moveFileEnd(e *view.Editor, extend bool) {
	SaveSelection(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		nLines := doc.LenLines()
		lineIdx := nLines - 1
		if lineIdx > 0 {
			lastStart, err := doc.LineToChar(lineIdx)
			if err == nil && lastStart >= doc.LenChars() {
				lineIdx--
			}
		}
		pos, err := doc.LineToChar(lineIdx)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, pos, extend)
	})
}

func isPathDelim(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
		ch == '"' || ch == '\'' || ch == '(' || ch == ')' ||
		ch == '[' || ch == ']' || ch == '{' || ch == '}'
}

// countOrOne returns the pending count or 1 if none is set
func countOrOne(e *view.Editor) int {
	if n := e.Count(); n > 0 {
		return n
	}
	return 1
}

// visualMoveFormat builds a VisualMoveFormat for the focused document if
// soft-wrap is active, returning a zero value otherwise
func visualMoveFormat(e *view.Editor) *core.VisualMoveFormat {
	w := e.ViewContentWidth()
	if w <= 0 {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	format := doc.TextFormatForConfig(w, e.Options())
	if !format.SoftWrap {
		return nil
	}
	return &core.VisualMoveFormat{
		ViewportWidth:    format.ViewportWidth,
		TabWidth:         format.TabWidth,
		MaxWrap:          format.MaxWrap,
		MaxIndentRetain:  format.MaxIndentRetain,
		WrapIndicatorLen: ansi.StringWidth(format.WrapIndicator),
	}
}
