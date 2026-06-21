package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// ScrollUp scrolls the view up by count lines without moving the cursor
func ScrollUp(e *view.Editor) {
	scrollView(e, max(e.Count(), 1), true)
}

// ScrollDown scrolls the view down by count lines without moving the cursor
func ScrollDown(e *view.Editor) {
	scrollView(e, max(e.Count(), 1), false)
}

// PageUp moves the cursor and scrolls the view up by one page
func PageUp(e *view.Editor) {
	h := max(e.ViewHeight(), 1)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveVertically(
			doc, core.DirectionBackward, h, core.MovementMove,
		)
	})
	scrollView(e, h, true)
}

// PageDown moves the cursor and scrolls the view down by one page
func PageDown(e *view.Editor) {
	h := max(e.ViewHeight(), 1)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveVertically(
			doc, core.DirectionForward, h, core.MovementMove,
		)
	})
	scrollView(e, h, false)
}

// PageCursorHalfUp moves the cursor and scrolls the view up by half a page
func PageCursorHalfUp(e *view.Editor) {
	half := max(max(e.ViewHeight(), 1)/2, 1)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveVertically(
			doc, core.DirectionBackward, half, core.MovementMove,
		)
	})
	scrollView(e, half, true)
}

// PageCursorHalfDown moves the cursor and scrolls the view down by half a page
func PageCursorHalfDown(e *view.Editor) {
	half := max(max(e.ViewHeight(), 1)/2, 1)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveVertically(
			doc, core.DirectionForward, half, core.MovementMove,
		)
	})
	scrollView(e, half, false)
}

// HalfPageUp scrolls the view up by half a page without moving the cursor
func HalfPageUp(e *view.Editor) {
	scrollView(e, max(max(e.ViewHeight(), 1)/2, 1), true)
}

// HalfPageDown scrolls the view down by half a page without moving the cursor
func HalfPageDown(e *view.Editor) {
	scrollView(e, max(max(e.ViewHeight(), 1)/2, 1), false)
}

// AlignViewTop scrolls the viewport so the cursor line is at the top
func AlignViewTop(e *view.Editor) {
	alignViewImpl(e, 0)
}

// AlignViewCenter scrolls the viewport so the cursor line is at the center
func AlignViewCenter(e *view.Editor) {
	alignViewImpl(e, (max(e.ViewHeight(), 1)-1)/2)
}

// AlignViewBottom scrolls the viewport so the cursor line is at the bottom
func AlignViewBottom(e *view.Editor) {
	alignViewImpl(e, max(e.ViewHeight(), 1)-1)
}

// GotoWindowTop moves the cursor to the top of the viewport
func GotoWindowTop(e *view.Editor) {
	gotoWindowImpl(e, 0)
}

// GotoWindowBottom moves the cursor to the last visible line in the viewport
func GotoWindowBottom(e *view.Editor) {
	gotoWindowImpl(e, 2)
}

// GotoWindowCenter moves the cursor to the center visible line
func GotoWindowCenter(e *view.Editor) {
	gotoWindowImpl(e, 1)
}

func alignViewImpl(e *view.Editor, relOffset int) {
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
	primary := sel.Ranges()[sel.PrimaryIndex()]
	cursor := primary.Cursor(text)
	cursorLine, err := text.CharToLine(cursor)
	if err != nil {
		return
	}
	firstLine := max(0, cursorLine-relOffset)
	anchor, err := text.LineToChar(firstLine)
	if err != nil {
		return
	}
	offset := v.Offset()
	offset.Anchor = anchor
	v.SetOffset(offset)
}

// gotoWindowImpl moves to the top, center, or bottom of the viewport.
// align: 0=top, 1=center, 2=bottom. Respects scrolloff and count offset.
// In select mode the selection is extended rather than moved
func gotoWindowImpl(e *view.Editor, align int) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	anchorLine, err := text.CharToLine(v.Offset().Anchor)
	if err != nil {
		anchorLine = 0
	}
	height := e.ViewHeight()
	if height <= 0 {
		height = 1
	}
	scrolloff := e.Options().ScrollOff
	offset := max(e.Count()-1, 0)
	e.ResetCount()
	lastLine := min(anchorLine+height-1, text.LenLines()-1)
	var targetLine int
	switch align {
	case 0: // top
		targetLine = anchorLine + scrolloff + offset
	case 1: // center
		targetLine = anchorLine + height/2
	default: // bottom
		targetLine = lastLine - scrolloff - offset
	}
	targetLine = max(targetLine, anchorLine+scrolloff)
	targetLine = min(targetLine, lastLine-scrolloff)
	targetLine = max(0, min(targetLine, text.LenLines()-1))
	pos, err := text.LineToChar(targetLine)
	if err != nil {
		return
	}
	extend := e.Mode() == view.ModeSelect
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.PutCursor(doc, pos, extend)
	})
}
