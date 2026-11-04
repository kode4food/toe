package ui

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/language"
)

type selectionViewport struct {
	text      core.Rope
	format    *language.TextFormat
	from      int
	to        int
	height    int
	scrolloff int
}

func (s *selectionViewport) anchor() core.VisualPos {
	if s.from < 0 || s.height <= 0 {
		return core.VisualPos{}
	}
	vf := softWrapMoveFormat(s.format)
	rows := 0
	for line := s.from; line <= s.to && rows <= s.height; line++ {
		rows += vf.VisualRows(s.text, line)
	}
	up := min(s.scrolloff, max(s.height-1, 0)/2)
	if rows <= s.height {
		up = (s.height - rows) / 2
	}
	return vf.VisualScrollUp(s.text, core.VisualPos{Line: s.from}, up)
}

// an unwrapped format keeps a zero viewport, so every line is one row
func softWrapMoveFormat(format *language.TextFormat) core.VisualMoveFormat {
	if !format.SoftWrap || format.ViewportWidth <= 0 {
		return core.VisualMoveFormat{}
	}
	return core.VisualMoveFormat{
		ViewportWidth:   format.ViewportWidth,
		TabWidth:        format.TabWidth,
		MaxWrap:         format.MaxWrap,
		MaxIndentRetain: format.MaxIndentRetain,
		WrapIndicatorWidth: runewidth.StringWidth(
			format.WrapIndicatorPrefix(),
		),
	}
}
