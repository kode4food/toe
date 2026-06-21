package action

import (
	"strings"
	"unicode"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// GotoLineEndNewline moves each cursor to the end of its current line,
// landing on the newline character (for use in insert mode)
func GotoLineEndNewline(e *view.Editor) {
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
		return r.PutCursor(doc, lineEnd, false)
	})
}

// ExtendToLineEndNewline extends each selection to the end of its current line,
// landing on the newline character
func ExtendToLineEndNewline(e *view.Editor) {
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
		return r.PutCursor(doc, lineEnd, true)
	})
}

// TrimSelections trims leading and trailing whitespace from each selection
// range. Empty or all-whitespace ranges are dropped. When all ranges are
// dropped the selection falls back to a single cursor at the primary position
func TrimSelections(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	oldPrimary := sel.Primary()
	out := make([]core.Range, 0, len(sel.Ranges()))
	for _, r := range sel.Ranges() {
		from, to := r.From(), r.To()
		// drop empty or all-whitespace ranges entirely
		if from == to {
			continue
		}
		allSpace := true
		for i := from; i < to; i++ {
			ch, err := text.CharAt(i)
			if err != nil || !unicode.IsSpace(ch) {
				allSpace = false
				break
			}
		}
		if allSpace {
			continue
		}
		for from < to {
			ch, _ := text.CharAt(from)
			if !unicode.IsSpace(ch) {
				break
			}
			from++
		}
		for to > from {
			ch, _ := text.CharAt(to - 1)
			if !unicode.IsSpace(ch) {
				break
			}
			to--
		}
		out = append(out, core.NewRange(from, to).WithDirection(r.Direction()))
	}
	if len(out) == 0 {
		// all ranges were empty/whitespace: collapse to primary cursor
		cursor := oldPrimary.Cursor(text)
		newSel, err := core.NewSelection(
			[]core.Range{core.NewRange(cursor, cursor)}, 0)
		if err != nil {
			return
		}
		doc.SetSelectionFor(v.ID(), newSel)
		return
	}
	// set primary to first surviving range that overlaps old primary, else last
	primary := len(out) - 1
	for i, r := range out {
		if r.Overlaps(oldPrimary) {
			primary = i
			break
		}
	}
	newSel, err := core.NewSelection(out, primary)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// JoinSelections joins the lines within each selected line range by replacing
// each line ending (and surrounding whitespace) with nothing
func JoinSelections(e *view.Editor) {
	joinSelectionsImpl(e, false)
}

// JoinSelectionsSpace joins the lines within each selected line range by
// replacing each line ending (and surrounding whitespace) with a single space
func JoinSelectionsSpace(e *view.Editor) {
	joinSelectionsImpl(e, true)
}

// ReindentSelections normalizes the leading whitespace on each selected
// line to use the document's current indent style at the same depth
// Lines with mixed indentation (tabs and spaces) are converted
func ReindentSelections(e *view.Editor) {
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
	unit := doc.IndentStyle().AsStr()
	tabW := doc.TabWidth()

	lines := selectionLines(text, sel)
	changes := make([]core.Change, 0, len(lines))
	for _, line := range lines {
		lineStart, err := text.LineToChar(line)
		if err != nil {
			continue
		}
		lineEnd, err := text.LineEndCharIndex(line)
		if err != nil {
			continue
		}
		// Measure existing leading whitespace in columns
		cols := 0
		wsEnd := lineStart
	wsLoop:
		for i := lineStart; i < lineEnd; i++ {
			ch, err2 := text.CharAt(i)
			if err2 != nil {
				break
			}
			switch ch {
			case ' ':
				cols++
				wsEnd = i + 1
			case '\t':
				cols = (cols/tabW + 1) * tabW
				wsEnd = i + 1
			default:
				break wsLoop
			}
		}
		// Rebuild indentation using current style
		var depth int
		if unit == "\t" {
			depth = cols / tabW
		} else {
			depth = cols / max(len(unit), 1)
		}
		newWS := strings.Repeat(unit, depth)
		// Collect old whitespace for comparison
		var sb strings.Builder
		for i := lineStart; i < wsEnd; i++ {
			ch, _ := text.CharAt(i)
			sb.WriteRune(ch)
		}
		if sb.String() == newWS {
			continue
		}
		changes = append(changes, core.TextChange(lineStart, wsEnd, newWS))
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
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}

// RotateSelectionsForward rotates the primary selection index forward by
// count steps (wrapping around)
func RotateSelectionsForward(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	n := len(sel.Ranges())
	if n == 0 {
		return
	}
	count := max(e.Count(), 1)
	newSel, err := core.NewSelection(sel.Ranges(), (sel.PrimaryIndex()+count)%n)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// RotateSelectionsBackward rotates the primary selection index backward by
// count steps (wrapping around)
func RotateSelectionsBackward(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	n := len(sel.Ranges())
	if n == 0 {
		return
	}
	count := max(e.Count(), 1)
	prev := (sel.PrimaryIndex() + n - count%n) % n
	newSel, err := core.NewSelection(sel.Ranges(), prev)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// RotateContentsForward rotates the text content of each selection
// range forward by count steps
func RotateContentsForward(e *view.Editor) {
	rotateSelectionContents(e, true)
}

// RotateContentsBackward rotates the text content of each selection
// range backward by count steps
func RotateContentsBackward(e *view.Editor) {
	rotateSelectionContents(e, false)
}

// SaveSelection pushes the current cursor position to the view's jump list
func SaveSelection(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	v.PushJump(v.DocID(), sel.Primary().Cursor(text))
}

// CommitUndoCheckpoint explicitly commits any pending insert-mode changes to
// history, creating an undo boundary mid-session
func CommitUndoCheckpoint(e *view.Editor) {
	e.CommitInsertHistory()
}

// JumpBackward navigates to the previous position in the view's jump list
func JumpBackward(e *view.Editor) {
	jumpTo(e, (*view.View).JumpBackward)
}

// JumpForward navigates to the next position in the view's jump list
func JumpForward(e *view.Editor) {
	jumpTo(e, (*view.View).JumpForward)
}

func jumpTo(e *view.Editor, fn func(*view.View) (view.DocumentId, int, bool)) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	_, pos, ok := fn(v)
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	newSel, err := core.NewSelection([]core.Range{core.PointRange(pos)}, 0)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// GotoLastModification moves each cursor to the position of the most recent
// committed change in the current document
func GotoLastModification(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	pos := doc.LastEditPos()
	text := doc.Text()
	extend := e.Mode() == view.ModeSelect
	SaveSelection(e)
	newSel, err := core.NewSelection(
		[]core.Range{core.PointRange(pos).PutCursor(text, pos, extend)},
		0,
	)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

func copySelectionOnLine(e *view.Editor, forward bool) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	n := max(e.Count(), 1)
	nLines := text.LenLines()

	primary := sel.PrimaryIndex()
	ranges := sel.Ranges()
	out := make([]core.Range, len(ranges))
	copy(out, ranges)
	newPrimary := primary

	for i, r := range ranges {
		anchorLine, err := text.CharToLine(r.From())
		if err != nil {
			continue
		}
		headLine, err := text.CharToLine(r.To())
		if err != nil {
			continue
		}
		// height is the number of lines spanned by the selection (min 1)
		height := headLine - anchorLine + 1

		// Column offsets within each line
		anchorLineStart, _ := text.LineToChar(anchorLine)
		headLineStart, _ := text.LineToChar(headLine)
		anchorCol := r.From() - anchorLineStart
		headCol := r.To() - headLineStart

		added := 0
		for step := 1; added < n; step++ {
			offset := step * height
			var destAnchorLine, destHeadLine int
			if forward {
				destAnchorLine = anchorLine + offset
				destHeadLine = headLine + offset
			} else {
				destAnchorLine = anchorLine - offset
				destHeadLine = headLine - offset
			}
			if destAnchorLine < 0 || destHeadLine < 0 ||
				destAnchorLine >= nLines || destHeadLine >= nLines {
				break
			}
			destAnchorStart, err := text.LineToChar(destAnchorLine)
			if err != nil {
				break
			}
			destHeadStart, err := text.LineToChar(destHeadLine)
			if err != nil {
				break
			}
			// Clamp column to line length
			destAnchorLineEnd, _ := text.LineEndCharIndex(destAnchorLine)
			destHeadLineEnd, _ := text.LineEndCharIndex(destHeadLine)
			newAnchor := min(destAnchorStart+anchorCol, destAnchorLineEnd)
			newHead := min(destHeadStart+headCol, destHeadLineEnd)

			newRange := core.NewRange(newAnchor, newHead)
			if hasDuplicateHead(out, newRange) {
				break
			}
			out = append(out, newRange)
			if i == primary {
				newPrimary = len(out) - 1
			}
			added++
		}
	}

	newSel, err := core.NewSelection(out, newPrimary)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

func hasDuplicateHead(ranges []core.Range, r core.Range) bool {
	for _, existing := range ranges {
		if existing.Head == r.Head {
			return true
		}
	}
	return false
}

func selectionLines(text core.Rope, sel core.Selection) []int {
	seen := map[int]bool{}
	var lines []int
	for _, r := range sel.Ranges() {
		lr, err := r.LineRange(text)
		if err != nil {
			continue
		}
		for l := lr.From; l <= lr.To; l++ {
			if !seen[l] {
				seen[l] = true
				lines = append(lines, l)
			}
		}
	}
	return lines
}

func isBlankLine(s string) bool {
	for _, ch := range s {
		if ch != ' ' && ch != '\t' && ch != '\r' && ch != '\n' {
			return false
		}
	}
	return true
}

// selectionIsLinewise returns true when every range in sel spans at least
// two lines and starts/ends exactly on line boundaries (start of a line and
// start of the next line, i.e., covers whole lines including newlines)
func selectionIsLinewise(text core.Rope, sel core.Selection) bool {
	nLines := text.LenLines()
	for _, r := range sel.Ranges() {
		lr, err := r.LineRange(text)
		if err != nil {
			return false
		}
		startLine, endLine := lr.From, lr.To
		if endLine <= startLine {
			return false
		}
		start, err := text.LineToChar(startLine)
		if err != nil {
			return false
		}
		endLineNext := min(endLine+1, nLines)
		end, err := text.LineToChar(endLineNext)
		if err != nil {
			return false
		}
		if r.From() != start || r.To() != end {
			return false
		}
	}
	return true
}

// applyMove applies fn to every range in the focused selection and
// replaces the selection with the transformed result
func applyMove(e *view.Editor, fn func(core.Rope, core.Range) core.Range) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	for i, r := range ranges {
		ranges[i] = fn(text, r)
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}
