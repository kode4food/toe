package ui

import (
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

// rec. 601 luma coefficients, scaled to stay in integer math
const (
	lumaRed      = 299
	lumaGreen    = 587
	lumaBlue     = 114
	lumaScale    = 1000
	lumaMidpoint = 128 * lumaScale
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
	if colorLuma(bg) > lumaMidpoint {
		fg = tui.ColorBlack
	}
	return tui.Style{}.Fg(fg).Bg(bg)
}

func colorLuma(c tui.Color) int {
	r, g, b, _ := c.RGBA()
	return lumaRed*int(r>>8) + lumaGreen*int(g>>8) + lumaBlue*int(b>>8)
}
