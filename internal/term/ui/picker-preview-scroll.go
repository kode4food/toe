package ui

import "github.com/kode4food/toe/internal/core"

// previewScroll is the preview's vertical position: it follows the renderer's
// anchor until the wheel or the scrollbar moves it
type previewScroll struct {
	top      core.VisualPos
	end      core.VisualPos
	delta    int
	scrolled bool
}

// resolve applies the wheel's pending rows and keeps the top at or before end,
// the top of the last full page
func (s *previewScroll) resolve(
	anchor core.VisualPos, walk core.VisualRowWalk, height int,
) core.VisualPos {
	s.end = lastPageTop(walk, height)
	top := anchor
	if s.scrolled {
		top = s.top
	}
	switch {
	case s.delta < 0:
		top = walk.Up(top, -s.delta)
	case s.delta > 0:
		top = walk.Down(top, s.delta)
	}
	if s.delta != 0 {
		s.delta = 0
		s.scrolled = true
	}
	if top.Line > s.end.Line || top.Line == s.end.Line && top.Row > s.end.Row {
		top = s.end
	}
	top.Row = min(top.Row, walk.RowsAt(top.Line)-1)
	if s.scrolled {
		s.top = top
	}
	return top
}

func lastPageTop(walk core.VisualRowWalk, height int) core.VisualPos {
	line := max(walk.Lines-1, 0)
	last := core.VisualPos{Line: line, Row: walk.RowsAt(line) - 1}
	return walk.Up(last, height-1)
}
