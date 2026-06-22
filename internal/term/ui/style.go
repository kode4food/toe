package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view/language"
)

var ansiBasicColors = [16]tui.Color{
	tui.ColorBlack, tui.ColorRed, tui.ColorGreen, tui.ColorYellow,
	tui.ColorBlue, tui.ColorMagenta, tui.ColorCyan, tui.ColorLightGray,
	tui.ColorGray, tui.ColorLightRed, tui.ColorLightGreen, tui.ColorLightYellow,
	tui.ColorLightBlue, tui.ColorLightMagenta, tui.ColorLightCyan, tui.ColorWhite,
}

func searchMatchStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("3")).
		Foreground(lipgloss.Color("0"))
}

func pickerContentStyle(cx *Context) lipgloss.Style {
	popup := cx.Theme().Get("ui.popup")
	return lipgloss.NewStyle().Background(popup.GetBackground())
}

func pickerCountStyle(cx *Context) lipgloss.Style {
	th := cx.Theme()
	bg := lipgloss.NewStyle().Background(th.Get("ui.popup").GetBackground())
	return bg.Foreground(th.Get("ui.text.inactive").GetForeground())
}

func pickerItemStyle(cx *Context) lipgloss.Style {
	return cx.Theme().Get("ui.menu")
}

func pickerSelStyle(cx *Context) lipgloss.Style {
	return cx.Theme().Get("ui.menu.selected")
}

func pickerMatchStyle(cx *Context) lipgloss.Style {
	s := cx.Theme().Get("ui.picker.match")
	return pickerItemStyle(cx).Foreground(s.GetForeground()).Bold(true)
}

func pickerSelMatchStyle(cx *Context) lipgloss.Style {
	s := cx.Theme().Get("ui.picker.match")
	return pickerSelStyle(cx).Foreground(s.GetForeground()).Bold(true)
}

func pickerFrameStyle(cx *Context) lipgloss.Style {
	popup := cx.Theme().Get("ui.popup")
	return lipgloss.NewStyle().
		Foreground(popup.GetForeground()).
		Background(popup.GetBackground())
}

func softWrapPrefix(format *language.TextFormat, indent int) string {
	if indent > format.MaxIndentRetain {
		indent = 0
	}
	return strings.Repeat(" ", indent) + format.WrapIndicator
}

// lipglossToTUIStyle converts a lipgloss style at the per-cell boundary
func lipglossToTUIStyle(s lipgloss.Style) tui.Style {
	st := tui.Style{}
	st = st.Fg(lipglossColorToTUI(s.GetForeground()))
	st = st.Bg(lipglossColorToTUI(s.GetBackground()))
	st = st.UlColor(lipglossColorToTUI(s.GetUnderlineColor()))
	st = st.UlStyle(lipglossUnderlineToTUI(s.GetUnderlineStyle()))
	var m tui.Modifier
	if s.GetBold() {
		m |= tui.ModifierBold
	}
	if s.GetFaint() {
		m |= tui.ModifierDim
	}
	if s.GetItalic() {
		m |= tui.ModifierItalic
	}
	if s.GetBlink() {
		m |= tui.ModifierSlowBlink
	}
	if s.GetReverse() {
		m |= tui.ModifierReversed
	}
	if s.GetStrikethrough() {
		m |= tui.ModifierCrossedOut
	}
	if m != 0 {
		st = st.Mod(m)
	}
	return st
}

func lipglossColorToTUI(c color.Color) tui.Color {
	if c == nil {
		return tui.ColorReset
	}
	switch v := c.(type) {
	case lipgloss.NoColor:
		return tui.ColorReset
	case ansi.BasicColor:
		return basicTUIColor(uint8(v))
	case ansi.IndexedColor:
		return tui.ColorIndexed(uint8(v))
	default:
		r, g, b, _ := c.RGBA()
		return tui.ColorRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
}

func basicTUIColor(idx uint8) tui.Color {
	if idx < 16 {
		return ansiBasicColors[idx]
	}
	return tui.ColorIndexed(idx)
}

func lipglossUnderlineToTUI(u lipgloss.Underline) tui.UnderlineStyle {
	switch u {
	case lipgloss.UnderlineSingle:
		return tui.UnderlineLine
	case lipgloss.UnderlineDouble:
		return tui.UnderlineDoubleLine
	case lipgloss.UnderlineCurly:
		return tui.UnderlineCurl
	case lipgloss.UnderlineDotted:
		return tui.UnderlineDotted
	case lipgloss.UnderlineDashed:
		return tui.UnderlineDashed
	default:
		return tui.UnderlineReset
	}
}
