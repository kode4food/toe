package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// KillToLineEnd deletes from the cursor to the end of the current line. If the
// cursor is already at the line ending, the newline itself is deleted
func KillToLineEnd(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		line, err := text.CharToLine(pos)
		if err != nil {
			continue
		}
		lineEnd, err := text.LineEndCharIndex(line)
		if err != nil {
			continue
		}
		if pos == lineEnd {
			nextLine := line + 1
			if nextLine < text.LenLines() {
				next, err := text.LineToChar(nextLine)
				if err != nil {
					continue
				}
				changes = append(changes, core.DeleteChange(pos, next))
			}
		} else {
			changes = append(changes, core.DeleteChange(pos, lineEnd))
		}
	}
	applyDeletesAtCursor(e, applyDeletesAtCursorArgs{
		text: text, sel: sel, ranges: ranges, changes: changes,
	})
}

// KillToLineStart deletes from the cursor to the start of the current line
// If the cursor is at the start, deletes the preceding newline (joins lines)
func KillToLineStart(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		line, err := text.CharToLine(pos)
		if err != nil {
			continue
		}
		lineStart, err := text.LineToChar(line)
		if err != nil {
			continue
		}
		var head int
		if pos == lineStart {
			if line == 0 {
				continue
			}
			prevEnd, err := text.LineEndCharIndex(line - 1)
			if err != nil {
				continue
			}
			head = prevEnd
		} else {
			lineEnd, _ := text.LineEndCharIndex(line)
			firstNonWS := skipHorizontalWhitespace(text, lineStart, lineEnd)
			if firstNonWS < pos {
				head = firstNonWS
			} else {
				head = lineStart
			}
		}
		changes = append(changes, core.DeleteChange(head, pos))
	}
	applyDeletesAtCursor(e, applyDeletesAtCursorArgs{
		text: text, sel: sel, ranges: ranges, changes: changes,
	})
}
