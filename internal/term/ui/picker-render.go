package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/tui"
)

type pickerItemRender struct {
	p        *Picker
	match    pickerMatch
	w        int
	selected bool
	cx       *Context
}

const (
	pickerMarkerW = 3
	pickerPadX    = 1

	pickerSplitFrameOverhead = 3
	pickerMinSplitPaneWidth  = 20
)

func writePickerPromptRow(
	buf *tui.Buffer, x, y, w int, p *Picker, cx *Context,
) {
	th := cx.Theme()
	count := fmt.Sprintf("%d/%d", len(p.matched), len(p.items))
	cl := runewidth.StringWidth(count)

	queryArea := max(w-2*pickerPadX-1-cl, 0)

	displayQuery := p.query
	ql := runewidth.StringWidth(p.query)
	if ql > queryArea {
		runes := []rune(p.query)
		for len(runes) > 0 && runewidth.StringWidth(string(runes)) > queryArea {
			runes = runes[1:]
		}
		displayQuery = string(runes)
		ql = runewidth.StringWidth(displayQuery)
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

func writePickerHeader(buf *tui.Buffer, x, y, w int, p *Picker, cx *Context) {
	cols := p.source.Columns()
	widths := pickerColumnWidths(p, max(w-pickerMarkerW-1, 0))
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

func writePickerItem(buf *tui.Buffer, x, y, w int, args *pickerItemRender) {
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
	matchColumn := p.source.MatchColumn()

	if len(cols) <= 1 {
		itemBase := base
		fg := lipglossColorToTUI(m.item.Style.GetForeground())
		if !fg.IsReset() {
			itemBase = base.Fg(fg)
		}
		writePickerMatched(buf, writePickerMatchedArgs{
			x: cx2, y: y, maxW: cellW, text: m.item.Display,
			indices: m.indices, base: itemBase, match: match,
		})
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
			colBase := pickerColumnBase(base, m.item.StyleScopes, i, cx)
			if i == matchColumn {
				writePickerMatched(buf, writePickerMatchedArgs{
					x: cur, y: y, maxW: widths[i], text: val,
					indices: m.indices, base: colBase, match: match,
				})
			} else {
				text := ansi.Truncate(val, widths[i], "")
				buf.SetString(cur, y, text, colBase)
			}
			cur += widths[i]
		}
	}
}

func pickerColumnBase(
	base tui.Style, scopes []string, i int, cx *Context,
) tui.Style {
	if i >= len(scopes) || scopes[i] == "" {
		return base
	}
	fg := lipglossColorToTUI(cx.Theme().Get(scopes[i]).GetForeground())
	if fg.IsReset() {
		return base
	}
	return base.Fg(fg)
}

func pickerEmptyHint(ps *Picker) string {
	if len(ps.matched) > 0 {
		return ""
	}
	if _, ok := ps.source.(DynamicPickerSource); ok {
		switch {
		case ps.query == "":
			return "Type to search…"
		case ps.dynamicPending:
			return "Searching…"
		}
	}
	return "No results"
}

func writePickerCenteredHint(
	buf *tui.Buffer, x, y, w, h int, text string, cx *Context,
) {
	if text == "" || h <= 0 {
		return
	}
	style := lipglossToTUIStyle(pickerCountStyle(cx))
	hx := x + max((w-runewidth.StringWidth(text))/2, 0)
	buf.SetString(hx, y+h/2, text, style)
}

func pickerColumnWidths(p *Picker, w int) []int {
	cols := p.source.Columns()
	n := len(cols)
	widths := make([]int, n)
	if n == 0 {
		return widths
	}
	spacing := max(n-1, 0)
	available := max(w-spacing, 0)
	proportions := p.source.ColumnProportions()
	if len(proportions) != n {
		proportions = defaultColumnProportions(n)
	}
	for i, col := range cols {
		widths[i] = runewidth.StringWidth(col)
	}
	for _, m := range p.matched {
		for i := range min(len(m.item.Columns), n) {
			widths[i] = max(widths[i], runewidth.StringWidth(m.item.Columns[i]))
		}
	}
	return pickerProportionalColumnWidths(widths, proportions, available)
}

func pickerProportionalColumnWidths(
	measured []int, proportions []int, available int,
) []int {
	widths := make([]int, len(measured))
	weight := 0
	pinned := 0
	for i, proportion := range proportions {
		if proportion <= 0 {
			widths[i] = measured[i]
			pinned += widths[i]
			continue
		}
		weight += proportion
	}
	remaining := max(available-pinned, 0)
	used := 0
	if weight > 0 {
		for i, proportion := range proportions {
			if proportion <= 0 {
				continue
			}
			widths[i] = remaining * proportion / weight
			used += widths[i]
		}
		for i, proportion := range proportions {
			if used >= remaining {
				break
			}
			if proportion <= 0 {
				continue
			}
			widths[i]++
			used++
		}
	}
	total := 0
	for _, width := range widths {
		total += width
	}
	for i := len(widths) - 1; total > available && i >= 0; i-- {
		take := min(widths[i], total-available)
		widths[i] -= take
		total -= take
	}
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

func pickerSplitLeftWidth(w int, ratio float64) int {
	usable := max(w-pickerSplitFrameOverhead, 0)
	if usable == 0 {
		return 0
	}
	left := int(float64(usable)*ratio + 0.5)
	minW := min(pickerMinSplitPaneWidth, usable/2)
	if left < minW {
		return minW
	}
	if right := usable - left; right < minW {
		return usable - minW
	}
	return left
}
