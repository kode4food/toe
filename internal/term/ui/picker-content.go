package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
)

const (
	pickerMarkerW = 3

	pickerSplitFrameOverhead = 3
	pickerMinSplitPaneWidth  = 20

	defaultPickerScale = 0.9
)

func (p *pickerRender) writePromptRow(area geom.Area) geom.Point {
	ps := p.component.state
	buf := p.buffer
	th := p.theme
	count := fmt.Sprintf(
		"%d/%d", ps.matchedCount(), len(ps.list.items),
	)
	cl := runewidth.StringWidth(count)

	queryArea := max(area.Width-2*overlayPadX-1-cl, 0)

	displayQuery := ps.list.query
	ql := runewidth.StringWidth(ps.list.query)
	if ql > queryArea {
		runes := []rune(ps.list.query)
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
	countTUI := p.count

	buf.FillRange(area.Point, area.Width, bgTUI)
	buf.SetString(geom.Point{
		X: area.X + overlayPadX,
		Y: area.Y,
	}, displayQuery, queryTUI)
	buf.SetString(geom.Point{
		X: area.X + overlayPadX + ql + 1 + gap,
		Y: area.Y,
	}, count, countTUI)
	return geom.Point{X: area.X + overlayPadX + ql, Y: area.Y}
}

func (p *pickerRender) writeHeader(area geom.Area) {
	buf := p.buffer
	cols := p.columns
	widths := p.widths
	bgTUI := pickerHeaderStyle(p.context)
	underlineColor := p.theme.Get("ui.text.inactive").FgColor()
	colTUI := bgTUI.
		UlStyle(tui.UnderlineLine).
		UlColor(underlineColor)
	buf.FillRange(area.Point, area.Width, bgTUI)
	cur := area.X + pickerMarkerW
	for i, col := range cols {
		if widths[i] == 0 {
			continue
		}
		if cur > area.X+pickerMarkerW {
			cur++
		}
		text := runewidth.Truncate(col, widths[i], "")
		buf.SetString(geom.Point{X: cur, Y: area.Y}, text, colTUI)
		cur += widths[i]
	}
}

func (p *pickerRender) writeSection(at geom.Point, m pickerMatch) {
	buf := p.buffer
	base := p.section
	buf.FillRange(at, p.width, base)
	label := runewidth.Truncate(
		m.item.Display, max(p.width-overlayPadX-1, 0), "",
	)
	buf.SetString(geom.Point{X: at.X + overlayPadX, Y: at.Y}, label, base)
}

func (p *pickerRender) writeItem(
	at geom.Point, m pickerMatch, selected bool,
) {
	ps := p.component.state
	buf := p.buffer
	if m.item.Section {
		p.writeSection(at, m)
		return
	}
	var marker string
	var base, match tui.Style
	if selected {
		marker = " > "
		base = p.selectStyle
		match = p.selectMatch
	} else {
		marker = strings.Repeat(" ", pickerMarkerW)
		base = p.itemStyle
		match = p.matchStyle
	}

	buf.FillRange(at, p.width, base)
	buf.SetString(at, marker, base)

	// Reserve 1 trailing cell for the right margin (matching the original
	// base.Width(w) right-padding that kept the highlight flush to the border)
	cellW := max(p.width-pickerMarkerW-1, 0)
	cx2 := at.X + pickerMarkerW
	cols := p.columns
	matchColumn := p.matchColumn
	fileIcon, iconColumn := pickerItemFileIcon(ps, m.item)

	sec, secFrom := p.secondary(base, m.item)
	if len(cols) <= 1 {
		itemBase := p.columnBase(base, m.item.StyleScopes, 0)
		if fileIcon.glyph != "" {
			iconStyle := pickerFileIconStyle(p.theme, base, fileIcon.color)
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
		widths := p.widths
		cur := cx2
		for i := range cols {
			if widths[i] == 0 {
				continue
			}
			if cur > cx2 {
				cur++
			}
			var val string
			if i < len(m.item.Columns) {
				val = m.item.Columns[i]
			}
			colBase := p.columnBase(base, m.item.StyleScopes, i)
			if i == iconColumn {
				val = fileIcon.glyph
				colBase = pickerFileIconStyle(p.theme, base, fileIcon.color)
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

func (p *pickerRender) secondary(
	base tui.Style, item *PickerItem,
) (tui.Style, int) {
	if item.SecFrom <= 0 {
		return base, 0
	}
	fg := p.theme.Get("ui.picker.secondary").FgColor()
	if fg.IsReset() {
		return base, 0
	}
	return base.Fg(fg), item.SecFrom
}

func (p *pickerRender) columnBase(
	base tui.Style, scopes []string, i int,
) tui.Style {
	if i >= len(scopes) || scopes[i] == "" {
		return base
	}
	fg := p.theme.Get(scopes[i]).FgColor()
	if fg.IsReset() {
		return base
	}
	return base.Fg(fg)
}

func (p *pickerRender) writeCenteredHint(area geom.Area, text string) {
	if text == "" || area.Height <= 0 {
		return
	}
	renderCenteredMessage(p.buffer, area, text, p.count)
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

func pickerOverlaySize(cx *Context, screen geom.Size, id string) geom.Size {
	opts := cx.pickerLayout
	return geom.Size{
		Width: scaleExtent(
			screen.Width, opts.widthScale(id, defaultPickerScale),
		),
		Height: scaleExtent(
			pickerAvailHeight(screen), opts.heightScale(id, defaultPickerScale),
		),
	}
}

func pickerAvailHeight(screen geom.Size) int {
	return max(screen.Height-overlayKeepClear, 0)
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
