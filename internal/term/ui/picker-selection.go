package ui

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	selectionViewport struct {
		text      core.Rope
		format    *language.TextFormat
		from      int
		to        int
		height    int
		scrolloff int
	}

	selectionAnchor struct {
		line   int
		offset int
	}
)

func (s *selectionViewport) anchor() selectionAnchor {
	if s.from < 0 || s.height <= 0 {
		return selectionAnchor{}
	}
	vf := &core.VisualMoveFormat{}
	if s.format.SoftWrap && s.format.ViewportWidth > 0 {
		vf = &core.VisualMoveFormat{
			ViewportWidth:   s.format.ViewportWidth,
			TabWidth:        s.format.TabWidth,
			MaxWrap:         s.format.MaxWrap,
			MaxIndentRetain: s.format.MaxIndentRetain,
			WrapIndicatorWidth: runewidth.StringWidth(
				s.format.WrapIndicatorPrefix(),
			),
		}
	}
	rows := 0
	for line := s.from; line <= s.to && rows <= s.height; line++ {
		rows += vf.VisualRows(s.text, line)
	}
	up := min(s.scrolloff, max(s.height-1, 0)/2)
	if rows <= s.height {
		up = (s.height - rows) / 2
	}
	res := vf.VisualScrollUp(core.VisualScrollUpArgs{
		Doc: s.text, Line: s.from, Up: up,
	})
	return selectionAnchor{
		line:   res.Line,
		offset: res.Row,
	}
}
