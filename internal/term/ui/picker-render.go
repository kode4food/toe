package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/tui"
)

type (
	pickerBoxFrame struct {
		border       lipgloss.Border
		borderStyle  tui.Style
		contentStyle tui.Style
	}

	pickerBoxAreas struct {
		left  popupArea
		right popupArea
	}

	pickerItemRender struct {
		p        *Picker
		match    pickerMatch
		w        int
		selected bool
		cx       *Context
	}
)

const (
	pickerMarkerW = 3
	pickerPadX    = 1
)

func (f pickerBoxFrame) drawSplit(
	buf *tui.Buffer, x, y, w, h, lw, cutY int,
) pickerBoxAreas {
	rw := max(w-lw-3, 0)
	for dy := range h {
		buf.FillRange(x, y+dy, w, f.contentStyle)
	}
	if w < 2 || h < 2 {
		return pickerBoxAreas{}
	}
	top := f.border.TopLeft +
		strings.Repeat(f.border.Top, lw) +
		f.border.MiddleTop +
		strings.Repeat(f.border.Top, rw) +
		f.border.TopRight
	bot := f.border.BottomLeft +
		strings.Repeat(f.border.Bottom, lw) +
		f.border.MiddleBottom +
		strings.Repeat(f.border.Bottom, rw) +
		f.border.BottomRight
	buf.SetString(x, y, top, f.borderStyle)
	buf.SetString(x, y+h-1, bot, f.borderStyle)
	for i := 0; i < h-2; i++ {
		ry := y + 1 + i
		if cutY > 0 && i == cutY-1 {
			cut := f.border.MiddleLeft +
				strings.Repeat(f.border.Top, lw) +
				f.border.MiddleRight
			buf.SetString(x, ry, cut, f.borderStyle)
			buf.SetString(x+w-1, ry, f.border.Right, f.borderStyle)
		} else {
			buf.SetString(x, ry, f.border.Left, f.borderStyle)
			buf.SetString(x+1+lw, ry, f.border.Left, f.borderStyle)
			buf.SetString(x+w-1, ry, f.border.Right, f.borderStyle)
		}
	}
	return pickerBoxAreas{
		left:  popupArea{x: x + 1, y: y + 1, w: lw, h: h - 2},
		right: popupArea{x: x + 2 + lw, y: y + 1, w: rw, h: h - 2},
	}
}

func (f pickerBoxFrame) drawSingle(
	buf *tui.Buffer, x, y, w, h, cutY int,
) popupArea {
	innerW := max(w-2, 0)
	for dy := range h {
		buf.FillRange(x, y+dy, w, f.contentStyle)
	}
	if w < 2 || h < 2 {
		return popupArea{}
	}
	top := f.border.TopLeft +
		strings.Repeat(f.border.Top, innerW) +
		f.border.TopRight
	bot := f.border.BottomLeft +
		strings.Repeat(f.border.Bottom, innerW) +
		f.border.BottomRight
	buf.SetString(x, y, top, f.borderStyle)
	buf.SetString(x, y+h-1, bot, f.borderStyle)
	for i := 0; i < h-2; i++ {
		ry := y + 1 + i
		if cutY > 0 && i == cutY-1 {
			cut := f.border.MiddleLeft +
				strings.Repeat(f.border.Top, innerW) +
				f.border.MiddleRight
			buf.SetString(x, ry, cut, f.borderStyle)
		} else {
			buf.SetString(x, ry, f.border.Left, f.borderStyle)
			buf.SetString(x+w-1, ry, f.border.Right, f.borderStyle)
		}
	}
	return popupArea{x: x + 1, y: y + 1, w: innerW, h: h - 2}
}

func writePickerPromptRow(
	buf *tui.Buffer, x, y, w int, p *Picker, cx *Context,
) {
	th := cx.Theme()
	count := fmt.Sprintf("%d/%d", len(p.matched), len(p.items))
	cl := ansi.StringWidth(count)

	queryArea := max(w-2*pickerPadX-1-cl, 0)

	displayQuery := p.query
	ql := ansi.StringWidth(p.query)
	if ql > queryArea {
		runes := []rune(p.query)
		for len(runes) > 0 && ansi.StringWidth(string(runes)) > queryArea {
			runes = runes[1:]
		}
		displayQuery = string(runes)
		ql = ansi.StringWidth(displayQuery)
	}
	gap := max(queryArea-ql, 0)

	popup := th.Get("ui.popup")
	popupBg := lipgloss.NewStyle().Background(popup.GetBackground())
	promptFg := lipgloss.NewStyle().Foreground(
		th.Get("ui.picker.match").GetForeground(),
	)

	bgTUI := lipglossToTUIStyle(popupBg)
	queryTUI := lipglossToTUIStyle(
		popupBg.Foreground(promptFg.GetForeground()),
	)
	cursorTUI := lipglossToTUIStyle(lipgloss.NewStyle().
		Foreground(popupBg.GetBackground()).
		Background(promptFg.GetForeground()))
	countTUI := lipglossToTUIStyle(pickerCountStyle(cx))

	buf.FillRange(x, y, w, bgTUI)
	buf.SetString(x+pickerPadX, y, displayQuery, queryTUI)
	buf.SetString(x+pickerPadX+ql, y, " ", cursorTUI)
	buf.SetString(x+pickerPadX+ql+1+gap, y, count, countTUI)
}

func writePickerHeader(
	buf *tui.Buffer, x, y, w int, p *Picker, cx *Context,
) {
	cols := p.source.Columns()
	widths := pickerColumnWidths(p, max(w-pickerMarkerW, 0))
	colTUI := lipglossToTUIStyle(pickerCountStyle(cx))
	buf.FillRange(x, y, w, colTUI)
	cur := x + pickerMarkerW
	for i, col := range cols {
		if i > 0 {
			cur++
		}
		text := ansi.Truncate(col, widths[i], "")
		buf.SetString(cur, y, text, colTUI)
		cur += widths[i]
	}
}

func writePickerItem(
	buf *tui.Buffer, x, y, w int, args *pickerItemRender,
) {
	p := args.p
	m := args.match
	cx := args.cx
	var marker string
	var base, match tui.Style
	if args.selected {
		marker = " > "
		base = lipglossToTUIStyle(pickerSelStyle(cx))
		match = lipglossToTUIStyle(pickerSelMatchStyle(cx))
	} else {
		marker = strings.Repeat(" ", pickerMarkerW)
		base = lipglossToTUIStyle(pickerItemStyle(cx))
		match = lipglossToTUIStyle(pickerMatchStyle(cx))
	}

	buf.FillRange(x, y, w, base)
	buf.SetString(x, y, marker, base)

	// Reserve 1 trailing cell for the right margin (matching the original
	// base.Width(w) right-padding that kept the highlight flush to the border)
	cellW := max(w-pickerMarkerW-1, 0)
	cx2 := x + pickerMarkerW
	cols := p.source.Columns()
	primary := p.source.Primary()

	if len(cols) <= 1 {
		itemBase := base
		fg := lipglossColorToTUI(m.item.Style.GetForeground())
		if !fg.IsReset() {
			itemBase = base.Fg(fg)
		}
		writePickerMatchedTextInto(
			buf, cx2, y, cellW, m.item.Display, m.indices, itemBase, match,
		)
	} else {
		widths := pickerColumnWidths(p, cellW)
		cur := cx2
		for i := range cols {
			if i > 0 {
				cur++
			}
			var val string
			if i < len(m.item.Columns) {
				val = m.item.Columns[i]
			}
			if i == primary {
				writePickerMatchedTextInto(
					buf, cur, y, widths[i], val, m.indices, base, match,
				)
			} else {
				text := ansi.Truncate(val, widths[i], "")
				buf.SetString(cur, y, text, base)
			}
			cur += widths[i]
		}
	}
}

func writePickerMatchedTextInto(
	buf *tui.Buffer, x, y, maxW int,
	text string, indices []int,
	base, match tui.Style,
) {
	if maxW <= 0 {
		return
	}
	runes := []rune(text)
	ptr := 0
	col := x
	budget := maxW
	for i := 0; i < len(runes) && budget > 0; {
		isMatch := ptr < len(indices) && indices[ptr] == i
		j := i + 1
		if isMatch {
			ptr++
			for j < len(runes) && ptr < len(indices) && indices[ptr] == j {
				ptr++
				j++
			}
		} else {
			for j < len(runes) &&
				!(ptr < len(indices) && indices[ptr] == j) {
				j++
			}
		}
		run := string(runes[i:j])
		rw := ansi.StringWidth(run)
		if rw > budget {
			run = ansi.Truncate(run, budget, "")
			rw = ansi.StringWidth(run)
		}
		st := base
		if isMatch {
			st = match
		}
		buf.SetString(col, y, run, st)
		col += rw
		budget -= rw
		i = j
	}
}

func pickerColumnWidths(p *Picker, w int) []int {
	cols := p.source.Columns()
	primary := p.source.Primary()
	n := len(cols)
	widths := make([]int, n)
	if n == 0 {
		return widths
	}
	spacing := max(n-1, 0)
	available := max(w-spacing, 0)
	for i, col := range cols {
		widths[i] = ansi.StringWidth(col)
	}
	for _, m := range p.matched {
		for i := range min(len(m.item.Columns), n) {
			widths[i] = max(widths[i], ansi.StringWidth(m.item.Columns[i]))
		}
	}
	total := 0
	for _, width := range widths {
		total += width
	}
	if total <= available {
		widths[n-1] += available - total
		return widths
	}
	fixed := 0
	for i := range n {
		if i != primary {
			fixed += widths[i]
		}
	}
	widths[primary] = max(available-fixed, 1)
	return widths
}

func clipPad(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = ansi.Truncate(s, w, "")
	if n := ansi.StringWidth(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func pickerOverlaySize(w, h int) (int, int) {
	areaW := w * 90 / 100
	areaH := max((h-2)*90/100, 0)
	return areaW, areaH
}
