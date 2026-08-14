package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
)

type pickerItemRender struct {
	picker   *Picker
	match    pickerMatch
	width    int
	selected bool
	context  *Context
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
	count := fmt.Sprintf(
		"%d/%d", p.matchedCount(), len(p.list.items),
	)
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
	queryTUI := applyAccentStyle(styleOverlay{base: popupBg, overlay: promptSt})
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
		text := runewidth.Truncate(col, widths[i], "")
		buf.SetString(geom.Point{X: cur, Y: area.Y}, text, colTUI)
		cur += widths[i]
	}
}

func writePickerSection(
	buf *tui.Buffer, at geom.Point, args *pickerItemRender,
) {
	m := args.match
	cx := args.context
	base := pickerSectionStyle(cx)
	buf.FillRange(at, args.width, base)
	label := runewidth.Truncate(
		m.item.Display, max(args.width-pickerPadX-1, 0), "",
	)
	buf.SetString(geom.Point{X: at.X + pickerPadX, Y: at.Y}, label, base)
}

func writePickerItem(
	buf *tui.Buffer, at geom.Point, args *pickerItemRender,
) {
	p := args.picker
	m := args.match
	cx := args.context
	if m.item.Section {
		writePickerSection(buf, at, args)
		return
	}
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

	buf.FillRange(at, args.width, base)
	buf.SetString(at, marker, base)

	// Reserve 1 trailing cell for the right margin (matching the original
	// base.Width(w) right-padding that kept the highlight flush to the border)
	cellW := max(args.width-pickerMarkerW-1, 0)
	cx2 := at.X + pickerMarkerW
	cols := p.source.Columns()
	matchColumn := p.source.MatchColumn()
	fileIcon := pickerItemFileIcon(cx.Editor, p, m.item)
	iconColumn := pickerFileIconColumn(p, m.item)

	sec, secFrom := pickerSecondary(cx, base, m.item)
	if len(cols) <= 1 {
		itemBase := pickerColumnBase(cx, base, m.item.StyleScopes, 0)
		if fileIcon.glyph != "" {
			iconStyle := pickerFileIconStyle(cx.Theme(), base, fileIcon.color)
			buf.SetString(
				geom.Point{X: cx2, Y: at.Y}, fileIcon.glyph, iconStyle,
			)
			iconWidth := runewidth.StringWidth(fileIcon.glyph) + 1
			cx2 += iconWidth
			cellW = max(cellW-iconWidth, 0)
		}
		writeMatchedItem(writeMatchedItemArgs{
			buf:           buf,
			at:            geom.Point{X: cx2, Y: at.Y},
			maxWidth:      cellW,
			text:          m.item.Display,
			indices:       m.result.Indices,
			base:          itemBase,
			match:         match,
			secondary:     sec,
			secondaryFrom: secFrom,
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
			if i == iconColumn {
				val = fileIcon.glyph
			}
			colBase := pickerColumnBase(cx, base, m.item.StyleScopes, i)
			if i == iconColumn {
				colBase = pickerFileIconStyle(cx.Theme(), base, fileIcon.color)
			}
			if i == matchColumn {
				writeMatchedItem(writeMatchedItemArgs{
					buf:           buf,
					at:            geom.Point{X: cur, Y: at.Y},
					maxWidth:      widths[i],
					text:          val,
					indices:       m.result.Indices,
					base:          colBase,
					match:         match,
					secondary:     sec,
					secondaryFrom: secFrom,
				})
			} else {
				text := runewidth.Truncate(val, widths[i], "")
				buf.SetString(geom.Point{X: cur, Y: at.Y}, text, colBase)
			}
			cur += widths[i]
		}
	}
}

func pickerSecondary(
	cx *Context, base tui.Style, item *PickerItem,
) (tui.Style, int) {
	if item.SecFrom <= 0 {
		return base, 0
	}
	fg := cx.Theme().Get("ui.picker.secondary").FgColor()
	if fg.IsReset() {
		return base, 0
	}
	return base.Fg(fg), item.SecFrom
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
	switch {
	case len(ps.list.matched) > 0:
		return ""
	case ps.awaitingQuery():
		return i18n.Text(i18n.StatusPickerTypeToSearch)
	case ps.load.loading:
		return i18n.Text(i18n.StatusPickerSearching)
	default:
		return i18n.Text(i18n.StatusPickerNoResults)
	}
}

func writePickerCenteredHint(
	cx *Context, buf *tui.Buffer, area geom.Area, text string,
) {
	if text == "" || area.Height <= 0 {
		return
	}
	renderCenteredMessage(buf, area, text, pickerCountStyle(cx))
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
