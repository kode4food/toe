package ui

import (
	"image/color"

	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/tui"
)

var (
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
)

func searchMatchStyle() tui.Style {
	return tui.Style{}.Bg(tui.ColorANSI(3)).Fg(tui.ColorANSI(0))
}

func pickerContentStyle(cx *Context) tui.Style {
	popup := cx.Theme().Get("ui.popup")
	return tui.Style{}.Bg(popup.BgColor())
}

func pickerCountStyle(cx *Context) tui.Style {
	th := cx.Theme()
	bg := tui.Style{}.Bg(th.Get("ui.popup").BgColor())
	return bg.Fg(th.Get("ui.text.inactive").FgColor())
}

func pickerHeaderStyle(cx *Context) tui.Style {
	th := cx.Theme()
	bg := tui.Style{}.Bg(th.Get("ui.popup").BgColor())
	return bg.Fg(th.Get("ui.text.focus").FgColor()).Mod(tui.ModifierBold)
}

func pickerItemStyle(cx *Context) tui.Style {
	return cx.Theme().Get("ui.menu")
}

func pickerSelStyle(cx *Context) tui.Style {
	return cx.Theme().Get("ui.menu.selected")
}

func pickerMatchStyle(cx *Context) tui.Style {
	s := cx.Theme().Get("ui.picker.match")
	return pickerItemStyle(cx).Fg(s.FgColor()).Mod(tui.ModifierBold)
}

func pickerSelMatchStyle(cx *Context) tui.Style {
	s := cx.Theme().Get("ui.picker.match")
	return pickerSelStyle(cx).Fg(s.FgColor()).Mod(tui.ModifierBold)
}

func completionIconStyle(
	cx *Context, kind string, selected bool,
) tui.Style {
	base := completionBaseStyle(cx, selected)
	scope := completionKindStyleScope(kind)
	icon, ok := cx.Theme().TryGet(scope)
	if !ok {
		icon = cx.Theme().Get("ui.text.inactive")
	}
	return applyAccentStyle(base, icon)
}

func completionInfoStyle(cx *Context, selected bool) tui.Style {
	base := completionBaseStyle(cx, selected)
	info, ok := cx.Theme().TryGet("comment")
	if !ok {
		info = cx.Theme().Get("ui.text.inactive")
	}
	return applyAccentStyle(base, info)
}

func completionBaseStyle(cx *Context, selected bool) tui.Style {
	base := pickerItemStyle(cx)
	if selected {
		base = pickerSelStyle(cx)
	}
	return base
}

func applyAccentStyle(base, accent tui.Style) tui.Style {
	if fg := accent.FgColor(); !fg.IsReset() {
		base = base.Fg(fg)
	}
	if accent.HasMod(tui.ModifierBold) {
		base = base.Mod(tui.ModifierBold)
	}
	if accent.HasMod(tui.ModifierDim) {
		base = base.Mod(tui.ModifierDim)
	}
	if accent.HasMod(tui.ModifierItalic) {
		base = base.Mod(tui.ModifierItalic)
	}
	return base
}

func completionKindStyleScope(kind string) string {
	if scope, ok := completionKindStyleScopes[kind]; ok {
		return scope
	}
	return "ui.text.inactive"
}

func pickerFrameStyle(cx *Context) tui.Style {
	popup := cx.Theme().Get("ui.popup")
	return tui.Style{}.Fg(popup.FgColor()).Bg(popup.BgColor())
}

func colorToTUI(c color.Color) tui.Color {
	if c == nil {
		return tui.ColorReset
	}
	switch v := c.(type) {
	case tui.Color:
		return v
	case ansi.BasicColor:
		return tui.ColorANSI(uint8(v))
	case ansi.IndexedColor:
		return tui.ColorIndexed(uint8(v))
	default:
		r, g, b, _ := c.RGBA()
		return tui.ColorRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
}
