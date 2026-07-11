package ui

import (
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

func documentColorSpans(colors []view.DocumentColor) []colorSpan {
	if len(colors) == 0 {
		return nil
	}
	out := make([]colorSpan, 0, len(colors))
	for _, color := range colors {
		if color.From < color.To {
			out = append(out, colorSpan{
				from:  color.From,
				to:    color.To,
				style: documentColorStyle(color),
			})
		}
	}
	return out
}

func documentColorAnnotations(colors []view.DocumentColor) []inlineAnnotation {
	if len(colors) == 0 {
		return nil
	}
	out := make([]inlineAnnotation, 0, len(colors))
	for _, color := range colors {
		if color.From < color.To {
			out = append(out, inlineAnnotation{
				pos:   color.From,
				text:  "\u25a0", // ■
				style: documentColorStyle(color),
			})
		}
	}
	return out
}

func documentColorStyle(color view.DocumentColor) tui.Style {
	bg := tui.ColorRGB(color.Red, color.Green, color.Blue)
	fg := tui.ColorWhite
	if colorLuma(color) > 128000 {
		fg = tui.ColorBlack
	}
	return tui.Style{}.Fg(fg).Bg(bg)
}

func colorLuma(color view.DocumentColor) int {
	return int(color.Red)*299 + int(color.Green)*587 + int(color.Blue)*114
}
