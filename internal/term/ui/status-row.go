package ui

import (
	"slices"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	statusRow struct {
		at           geom.Point
		width        int
		baseStyle    tui.Style
		sectionStyle tui.Style
		edge         statusEdge
		focused      bool
		left         []statusElem
		right        []statusElem
	}

	statusElem struct {
		text    string
		style   tui.Style
		pinned  bool
		compact bool
		prose   bool
	}

	// statusEdge is one shape's dividers: the solid pair cuts between two
	// backgrounds, the thin pair divides segments sharing one
	statusEdge struct {
		solidRight string
		solidLeft  string
		thinRight  string
		thinLeft   string
	}
)

// dividerBlendPct: a thin divider sits this far from its own background toward
// the text color, so it separates without competing with the text
const dividerBlendPct = 35

// statusEdges is the glyph set for each configured shape; every one but the
// plain line needs a nerd font
var statusEdges = map[view.StatusLineEdges]statusEdge{
	view.StatusLineEdgesArrow: {
		solidRight: "\ue0b0", // '' - nf-pl-left_hard_divider
		solidLeft:  "\ue0b2", // '' - nf-pl-right_hard_divider
		thinRight:  "\ue0b1", // '' - nf-pl-left_soft_divider
		thinLeft:   "\ue0b3", // '' - nf-pl-right_soft_divider
	},
	view.StatusLineEdgesRound: {
		solidRight: "\ue0b4", // '' - nf-ple-right_half_circle_thick
		solidLeft:  "\ue0b6", // '' - nf-ple-left_half_circle_thick
		thinRight:  "\ue0b5", // '' - nf-ple-right_half_circle_thin
		thinLeft:   "\ue0b7", // '' - nf-ple-left_half_circle_thin
	},
	view.StatusLineEdgesSlant: {
		solidRight: "\ue0b8", // '' - nf-ple-lower_left_triangle
		solidLeft:  "\ue0ba", // '' - nf-ple-lower_right_triangle
		thinRight:  "\ue0b9", // '' - nf-ple-backslash_separator
		thinLeft:   "\ue0bb", // '' - nf-ple-forwardslash_separator
	},
	view.StatusLineEdgesLine: {
		solidRight: "\u258c", // '▌' - left half block
		solidLeft:  "\u2590", // '▐' - right half block
		thinRight:  "\u2502", // '│' - box drawings light vertical
		thinLeft:   "\u2502", // '│' - box drawings light vertical
	},
}

func (r statusRow) paint(buf *tui.Buffer) {
	left, right := r.left, r.right
	for r.elemsWidth(left, false)+r.elemsWidth(right, true) > r.width {
		var ok bool
		if right, ok = dropUnpinned(right, false); ok {
			continue
		}
		if left, ok = dropUnpinned(left, true); !ok {
			break
		}
	}
	buf.FillRange(r.at, r.width, r.baseStyle)
	r.writeElems(buf, left, r.at.X, false)
	r.writeElems(buf, right, r.at.X+r.width-r.elemsWidth(right, true), true)
}

// writeElems paints one side of the row; toLeft puts each divider on the
// leading edge, so the right-hand run points back toward the middle
func (r statusRow) writeElems(
	buf *tui.Buffer, elems []statusElem, x int, toLeft bool,
) {
	for i, e := range elems {
		st := r.styleFor(e)
		text, sep := r.divider(elems, i, toLeft)
		if toLeft {
			x = r.write(buf, x, text, sep)
		}
		if !e.compact {
			x = r.write(buf, x, " ", st)
		}
		x = r.write(buf, x, e.text, st)
		if !e.compact {
			x = r.write(buf, x, " ", st)
		}
		if !toLeft {
			x = r.write(buf, x, text, sep)
		}
	}
}

// divider is an element's outward edge: the solid glyph in its own background
// over the one it cuts into, or the receding thin one where they match
func (r statusRow) divider(
	elems []statusElem, idx int, toLeft bool,
) (string, tui.Style) {
	e := elems[idx]
	if e.prose {
		return "", r.baseStyle
	}
	solid, thin := r.edge.solidRight, r.edge.thinRight
	into := idx + 1
	if toLeft {
		solid, thin = r.edge.solidLeft, r.edge.thinLeft
		into = idx - 1
	}
	st := r.styleFor(e)
	bg, intoBg := st.BgColor(), r.neighborStyle(elems, into).BgColor()
	if bg == intoBg {
		return thin, st.Fg(bg.Blended(st.FgColor(), dividerBlendPct))
	}
	return solid, r.baseStyle.Fg(bg).Bg(intoBg)
}

// styleFor puts a segment on the section background, unless it brought one of
// its own or the row is unfocused, which leaves the whole row flat
func (r statusRow) styleFor(e statusElem) tui.Style {
	if e.prose || !r.focused || e.style.BgColor() != r.baseStyle.BgColor() {
		return e.style
	}
	return r.sectionStyle.Fg(e.style.FgColor())
}

func (r statusRow) neighborStyle(elems []statusElem, idx int) tui.Style {
	if idx >= 0 && idx < len(elems) {
		return r.styleFor(elems[idx])
	}
	return r.baseStyle
}

func (r statusRow) write(
	buf *tui.Buffer, x int, text string, style tui.Style,
) int {
	buf.SetString(geom.Point{X: x, Y: r.at.Y}, text, style)
	return x + runewidth.StringWidth(text)
}

// elemsWidth measures through divider, so the space reserved for an edge is
// always the glyph writeElems will paint there
func (r statusRow) elemsWidth(elems []statusElem, toLeft bool) int {
	w := 0
	for i, e := range elems {
		text, _ := r.divider(elems, i, toLeft)
		w += runewidth.StringWidth(e.text) + runewidth.StringWidth(text)
		if !e.compact {
			w += 2
		}
	}
	return w
}

func statusBadge(text string, style tui.Style) statusElem {
	return statusElem{
		text:   text,
		style:  style,
		pinned: true,
	}
}

func dropUnpinned(elems []statusElem, fromEnd bool) ([]statusElem, bool) {
	for n, i := len(elems), 0; i < n; i++ {
		idx := i
		if fromEnd {
			idx = n - 1 - i
		}
		if !elems[idx].pinned {
			return slices.Delete(elems, idx, idx+1), true
		}
	}
	return elems, false
}
