package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/tui"
)

var (
	ansiBasicColors = [16]tui.Color{
		tui.ColorBlack, tui.ColorRed, tui.ColorGreen, tui.ColorYellow,
		tui.ColorBlue, tui.ColorMagenta, tui.ColorCyan, tui.ColorLightGray,
		tui.ColorGray, tui.ColorLightRed, tui.ColorLightGreen,
		tui.ColorLightYellow, tui.ColorLightBlue, tui.ColorLightMagenta,
		tui.ColorLightCyan, tui.ColorWhite,
	}

	completionKindStyleScopes = map[string]string{
		"function":    "function",
		"method":      "function",
		"constructor": "function",
		"field":       "variable.other.member",
		"property":    "variable.other.member",
		"reference":   "variable.other.member",
		"variable":    "variable.parameter",
		"class":       "type",
		"interface":   "type",
		"struct":      "type",
		"type_param":  "type",
		"unit":        "type",
		"module":      "namespace",
		"value":       "constant",
		"enum":        "constant",
		"enum_member": "constant",
		"constant":    "constant",
		"keyword":     "keyword",
		"operator":    "operator",
		"snippet":     "string",
		"text":        "string",
		"file":        "string",
		"folder":      "string",
		"color":       "string",
		"event":       "string",
	}

	underlineTUIStyles = [...]tui.UnderlineStyle{
		lipgloss.UnderlineNone:   tui.UnderlineReset,
		lipgloss.UnderlineSingle: tui.UnderlineLine,
		lipgloss.UnderlineDouble: tui.UnderlineDoubleLine,
		lipgloss.UnderlineCurly:  tui.UnderlineCurl,
		lipgloss.UnderlineDotted: tui.UnderlineDotted,
		lipgloss.UnderlineDashed: tui.UnderlineDashed,
	}
)

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

func pickerHeaderStyle(cx *Context) lipgloss.Style {
	th := cx.Theme()
	bg := lipgloss.NewStyle().Background(th.Get("ui.popup").GetBackground())
	return bg.Foreground(th.Get("ui.text.focus").GetForeground()).Bold(true)
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

func completionIconStyle(
	cx *Context, kind string, selected bool,
) lipgloss.Style {
	base := completionBaseStyle(cx, selected)
	scope := completionKindStyleScope(kind)
	icon, ok := cx.Theme().TryGet(scope)
	if !ok {
		icon = cx.Theme().Get("ui.text.inactive")
	}
	return applyAccentStyle(base, icon)
}

func completionInfoStyle(cx *Context, selected bool) lipgloss.Style {
	base := completionBaseStyle(cx, selected)
	info, ok := cx.Theme().TryGet("comment")
	if !ok {
		info = cx.Theme().Get("ui.text.inactive")
	}
	return applyAccentStyle(base, info)
}

func completionBaseStyle(cx *Context, selected bool) lipgloss.Style {
	base := pickerItemStyle(cx)
	if selected {
		base = pickerSelStyle(cx)
	}
	return base
}

func applyAccentStyle(base, accent lipgloss.Style) lipgloss.Style {
	if fg := accent.GetForeground(); fg != nil {
		base = base.Foreground(fg)
	}
	if accent.GetBold() {
		base = base.Bold(true)
	}
	if accent.GetFaint() {
		base = base.Faint(true)
	}
	if accent.GetItalic() {
		base = base.Italic(true)
	}
	return base
}

func completionKindStyleScope(kind string) string {
	if scope, ok := completionKindStyleScopes[kind]; ok {
		return scope
	}
	return "ui.text.inactive"
}

func pickerFrameStyle(cx *Context) lipgloss.Style {
	popup := cx.Theme().Get("ui.popup")
	return lipgloss.NewStyle().
		Foreground(popup.GetForeground()).
		Background(popup.GetBackground())
}

func styleToTUI(s lipgloss.Style) tui.Style {
	st := tui.Style{}
	st = st.Fg(colorToTUI(s.GetForeground()))
	st = st.Bg(colorToTUI(s.GetBackground()))
	st = st.UlColor(colorToTUI(s.GetUnderlineColor()))
	st = st.UlStyle(underlineToTUI(s.GetUnderlineStyle()))
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

func colorToTUI(c color.Color) tui.Color {
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

func underlineToTUI(u lipgloss.Underline) tui.UnderlineStyle {
	if int(u) >= len(underlineTUIStyles) {
		return tui.UnderlineReset
	}
	return underlineTUIStyles[u]
}
