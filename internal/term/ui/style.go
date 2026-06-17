package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/view/config"
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

func softWrapPrefix(format *config.TextFormat, indent int) string {
	if indent > format.MaxIndentRetain {
		indent = 0
	}
	return strings.Repeat(" ", indent) + format.WrapIndicator
}
