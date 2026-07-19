package ui

import (
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type (
	renderedRow struct {
		cells    []renderedCell
		width    int
		offset   int
		colStart int
	}

	// renderedCell stores plain text + tui.Style rather than pre-rendered ANSI,
	// keeping lipgloss.Style.Render() out of the per-rune character loop
	renderedCell struct {
		text  string
		width int
		style tui.Style
	}
)

type rowWriteArgs struct {
	buf       *tui.Buffer
	at        geom.Point
	fillStyle tui.Style
	width     int
	startCol  int
}

func (r *renderedRow) writeToBuffer(args rowWriteArgs) {
	cx := writeCellsWindowed(writeCellsArgs{
		buf: args.buf, cells: r.cells, at: args.at,
		width: args.width, startCol: args.startCol, cellsCol: r.colStart,
	})
	r.writeFillToBuffer(rowFillArgs{
		buf: args.buf, at: geom.Point{X: cx, Y: args.at.Y},
		width: max(args.at.X+args.width-cx, 0), style: args.fillStyle,
	})
}

type rowFillArgs struct {
	buf   *tui.Buffer
	at    geom.Point
	width int
	style tui.Style
}

func (r *renderedRow) writeFillToBuffer(args rowFillArgs) {
	if args.width <= 0 {
		return
	}
	// fg-only spaces are invisible over the base fill; skip them
	s := args.style
	if s.BgColor().IsReset() && s.Modifier() == 0 &&
		s.UnderlineStyle() == tui.UnderlineReset {
		return
	}
	args.buf.FillRange(args.at, args.width, args.style)
}

func (r *renderedRow) empty() bool {
	return len(r.cells) == 0
}

func (r *renderedRow) write(text string, width int, style tui.Style) {
	if text == "" || width <= 0 {
		return
	}
	r.cells = append(r.cells, renderedCell{
		text: text, width: width, style: style,
	})
	r.width += width
}

func (r *renderedRow) append(other renderedRow) {
	r.cells = append(r.cells, other.cells...)
	r.width += other.width
}

type writeCellsArgs struct {
	buf      *tui.Buffer
	cells    []renderedCell
	at       geom.Point
	width    int
	startCol int
	cellsCol int
}

func writeCellsWindowed(a writeCellsArgs) int {
	col := a.cellsCol
	end := a.startCol + a.width
	cx := a.at.X
	for _, c := range a.cells {
		if col >= end {
			break
		}
		cutOff := a.startCol - col
		switch {
		case cutOff <= 0 && col+c.width <= end:
			sx := a.at.X + col - a.startCol
			a.buf.SetString(geom.Point{X: sx, Y: a.at.Y}, c.text, c.style)
			cx = sx + c.width
		case cutOff > 0 && cutOff < c.width:
			// straddles the left edge: the visible remainder of a tab or wide
			// rune is drawn as styled blanks
			visW := c.width - cutOff
			a.buf.FillRange(a.at, visW, c.style)
			cx = a.at.X + visW
		}
		// else: fully off-screen, or straddles the right edge — drawn as
		// nothing, leaving the column for the trailing fill
		col += c.width
	}
	return cx
}

// rulers are 1-based content columns
type applyRulersArgs struct {
	buf     *tui.Buffer
	at      geom.Point
	size    geom.Size
	hOff    int
	rulers  []int
	rulerBg tui.Color
}

func applyRulers(a applyRulersArgs) {
	for _, ruler := range a.rulers {
		rel := ruler - 1 - a.hOff
		if rel < 0 || rel >= a.size.Width {
			continue
		}
		sx := a.at.X + rel
		for y := a.at.Y; y < a.at.Y+a.size.Height; y++ {
			a.buf.PatchBg(geom.Point{X: sx, Y: y}, a.rulerBg)
		}
	}
}
