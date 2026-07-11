package ui

import "github.com/kode4food/toe/internal/tui"

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
	x, y      int
	fillStyle tui.Style
	width     int
	startCol  int
}

func (r *renderedRow) writeToBuffer(args rowWriteArgs) {
	cx := writeCellsWindowed(
		args.buf, r.cells, args.x, args.y, args.width, args.startCol,
		r.colStart,
	)
	r.writeFillToBuffer(rowFillArgs{
		buf: args.buf, x: cx, y: args.y,
		width: max(args.x+args.width-cx, 0), style: args.fillStyle,
	})
}

type rowFillArgs struct {
	buf   *tui.Buffer
	x, y  int
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
	args.buf.FillRange(args.x, args.y, args.width, args.style)
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

func writeCellsWindowed(
	buf *tui.Buffer, cells []renderedCell, x, y, width, startCol, cellsCol int,
) int {
	col := cellsCol
	end := startCol + width
	cx := x
	for _, c := range cells {
		if col >= end {
			break
		}
		cutOff := startCol - col
		switch {
		case cutOff <= 0 && col+c.width <= end:
			sx := x + col - startCol
			buf.SetString(sx, y, c.text, c.style)
			cx = sx + c.width
		case cutOff > 0 && cutOff < c.width:
			// straddles the left edge: the visible remainder of a tab or wide
			// rune is drawn as styled blanks
			visW := c.width - cutOff
			buf.FillRange(x, y, visW, c.style)
			cx = x + visW
		}
		// else: fully off-screen, or straddles the right edge — drawn as
		// nothing, leaving the column for the trailing fill
		col += c.width
	}
	return cx
}

// rulers are 1-based content columns
func applyRulers(
	buf *tui.Buffer, contentX, y0, width, height, hOff int, rulers []int,
	rulerBg tui.Color,
) {
	for _, ruler := range rulers {
		rel := ruler - 1 - hOff
		if rel < 0 || rel >= width {
			continue
		}
		sx := contentX + rel
		for y := y0; y < y0+height; y++ {
			buf.PatchBg(sx, y, rulerBg)
		}
	}
}
