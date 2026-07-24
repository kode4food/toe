package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
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
	cx *Context, buf *tui.Buffer, area geom.Area, p *Picker,
) {
	th := cx.Theme()
	count := fmt.Sprintf("%d/%d", len(p.list.matched), len(p.list.items))
	cl := runewidth.StringWidth(count)

	queryArea := max(area.Width-2*pickerPadX-1-cl, 0)

	displayQuery := p.list.query
	ql := runewidth.StringWidth(p.list.query)
	if ql > queryArea {
		runes := []rune(p.list.query)
		for len(runes) > 0 && runewidth.StringWidth(string(runes)) > queryArea {
			runes = runes[1:]
		}
		displayQuery = string(runes)
		ql = runewidth.StringWidth(displayQuery)
	}
	gap := max(queryArea-ql, 0)

	popup := th.Get("ui.popup")
	popupBg := tui.Style{}.Bg(popup.BgColor())
	promptSt := th.Get("ui.prompt")

	bgTUI := popupBg
	queryTUI := applyAccentStyle(popupBg, promptSt)
	cursorTUI := tui.Style{}.
		Fg(popupBg.BgColor()).
		Bg(promptSt.FgColor())
	countTUI := pickerCountStyle(cx)

	buf.FillRange(area.Point, area.Width, bgTUI)
	buf.SetString(geom.Point{
		X: area.X + pickerPadX,
		Y: area.Y,
	}, displayQuery, queryTUI)
	buf.SetString(geom.Point{
		X: area.X + pickerPadX + ql,
		Y: area.Y,
	}, " ", cursorTUI)
	buf.SetString(geom.Point{
		X: area.X + pickerPadX + ql + 1 + gap,
		Y: area.Y,
	}, count, countTUI)
}

func writePickerHeader(
	cx *Context, buf *tui.Buffer, area geom.Area, p *Picker,
) {
	cols := p.source.Columns()
	widths := pickerColumnWidths(p, max(area.Width-pickerMarkerW-1, 0))
	bgTUI := pickerHeaderStyle(cx)
	underlineColor := cx.Theme().Get("ui.text.inactive").FgColor()
	colTUI := pickerHeaderStyle(cx).
		UlStyle(tui.UnderlineLine).
		UlColor(underlineColor)
	buf.FillRange(area.Point, area.Width, bgTUI)
	cur := area.X + pickerMarkerW
	for i, col := range cols {
		if i > 0 {
			cur++
		}
		text := ansi.Truncate(col, widths[i], "")
		buf.SetString(geom.Point{X: cur, Y: area.Y}, text, colTUI)
		cur += widths[i]
	}
}

func writePickerItem(
	buf *tui.Buffer, at geom.Point, args *pickerItemRender,
) {
	p := args.p
	m := args.match
	cx := args.cx
	var marker string
	var base, match tui.Style
	if args.selected {
		marker = " > "
		base = pickerSelStyle(cx)
		match = pickerSelMatchStyle(cx)
	} else {
		marker = strings.Repeat(" ", pickerMarkerW)
		base = pickerItemStyle(cx)
		match = pickerMatchStyle(cx)
	}

	buf.FillRange(at, args.w, base)
	buf.SetString(at, marker, base)

	// Reserve 1 trailing cell for the right margin (matching the original
	// base.Width(w) right-padding that kept the highlight flush to the border)
	cellW := max(args.w-pickerMarkerW-1, 0)
	cx2 := at.X + pickerMarkerW
	cols := p.source.Columns()
	matchColumn := p.source.MatchColumn()

	if len(cols) <= 1 {
		itemBase := pickerColumnBase(cx, base, m.item.StyleScopes, 0)
		writePickerMatched(buf, writePickerMatchedArgs{
			at:      geom.Point{X: cx2, Y: at.Y},
			maxW:    cellW,
			text:    m.item.Display,
			indices: m.indices,
			base:    itemBase,
			match:   match,
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
			colBase := pickerColumnBase(cx, base, m.item.StyleScopes, i)
			if i == matchColumn {
				writePickerMatched(buf, writePickerMatchedArgs{
					at:      geom.Point{X: cur, Y: at.Y},
					maxW:    widths[i],
					text:    val,
					indices: m.indices,
					base:    colBase,
					match:   match,
				})
			} else {
				text := ansi.Truncate(val, widths[i], "")
				buf.SetString(geom.Point{X: cur, Y: at.Y}, text, colBase)
			}
			cur += widths[i]
		}
	}
}

func pickerColumnBase(
	cx *Context, base tui.Style, scopes []string, i int,
) tui.Style {
	if i >= len(scopes) || scopes[i] == "" {
		return base
	}
	fg := cx.Theme().Get(scopes[i]).FgColor()
	if fg.IsReset() {
		return base
	}
	return base.Fg(fg)
}

func pickerEmptyHint(ps *Picker) string {
	if len(ps.list.matched) > 0 {
		return ""
	}
	if _, ok := ps.source.(DynamicPickerSource); ok {
		switch {
		case ps.list.query == "":
			return i18n.Text(i18n.StatusPickerTypeToSearch)
		case ps.load.dynamicPending:
			return i18n.Text(i18n.StatusPickerSearching)
		}
	}
	return i18n.Text(i18n.StatusPickerNoResults)
}

func writePickerCenteredHint(
	cx *Context, buf *tui.Buffer, area geom.Area, text string,
) {
	if text == "" || area.Height <= 0 {
		return
	}
	style := pickerCountStyle(cx)
	hx := area.X + max((area.Width-runewidth.StringWidth(text))/2, 0)
	buf.SetString(geom.Point{
		X: hx,
		Y: area.Y + area.Height/2,
	}, text, style)
}

func pickerOverlaySize(screen geom.Size) geom.Size {
	return geom.Size{
		Width:  screen.Width * 90 / 100,
		Height: max((screen.Height-2)*90/100, 0),
	}
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
