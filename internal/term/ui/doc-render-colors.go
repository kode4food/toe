package ui

import (
	"unicode/utf8"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

const (
	documentColorIcon     = "\uf0c8" // '' - nf-fa-square
	documentColorFallback = "■"

	// rec. 601 luma coefficients, scaled to stay in integer math
	lumaRed   = 299
	lumaGreen = 587
	lumaBlue  = 114
	lumaScale = 1000
)

func (dc *docRenderCache) visibleHexColors(
	rawText string, lineIdx []lineIndexEntry, lines core.Span,
) []view.DocumentColor {
	if len(lineIdx) == 0 {
		return nil
	}
	last := len(lineIdx) - 1
	from := min(max(lines.From, 0), last)
	to := min(max(lines.To, from), last)
	start := lineIdx[from]
	dc.hexScratch = hexColorSpans(
		dc.hexScratch[:0],
		rawText[start.byteStart:lineIdx[to].byteStart], start.charStart,
	)
	return dc.hexScratch
}

func appendColorAnnotations(
	out []inlineAnnotation, colors []view.DocumentColor, nerd bool,
) []inlineAnnotation {
	if len(colors) == 0 {
		return out
	}
	icon := documentColorFallback
	if nerd {
		icon = documentColorIcon
	}
	for _, color := range colors {
		if color.From < color.To {
			out = append(out, inlineAnnotation{
				pos:   color.From,
				text:  icon,
				style: documentColorStyle(color),
			})
		}
	}
	return out
}

func documentColorStyle(color view.DocumentColor) tui.Style {
	return tui.Style{}.Fg(tui.ColorRGB(color.Red, color.Green, color.Blue))
}

func colorLuma(c tui.Color) int {
	r, g, b, _ := c.RGBA()
	return lumaRed*int(r>>8) + lumaGreen*int(g>>8) + lumaBlue*int(b>>8)
}

func hexColorSpans(
	out []view.DocumentColor, text string, baseChar int,
) []view.DocumentColor {
	var prev rune
	charPos := baseChar
	for bytePos, ch := range text {
		from, before := charPos, prev
		charPos++
		prev = ch
		if ch != '#' || core.CharIsWord(before) {
			continue
		}
		// hex digits are ASCII, so byte counts are character counts
		digits := text[bytePos+1:]
		n := 0
		for n < len(digits) && hexDigit(rune(digits[n])) >= 0 {
			n++
		}
		if n != 3 && n != 4 && n != 6 && n != 8 {
			continue
		}
		next, _ := utf8.DecodeRuneInString(digits[n:])
		if core.CharIsWord(next) {
			continue
		}
		out = append(out, hexColor(digits[:n], core.Span{
			From: from,
			To:   charPos + n,
		}))
	}
	return out
}

func hexColor(digits string, at core.Span) view.DocumentColor {
	color := view.DocumentColor{From: at.From, To: at.To}
	// 3 and 4 digit forms repeat each nibble, #abc meaning #aabbcc
	if len(digits) <= 4 {
		color.Red = uint8(hexDigit(rune(digits[0])) * 17)
		color.Green = uint8(hexDigit(rune(digits[1])) * 17)
		color.Blue = uint8(hexDigit(rune(digits[2])) * 17)
		return color
	}
	color.Red = uint8(hexDigit(rune(digits[0]))*16 + hexDigit(rune(digits[1])))
	color.Green = uint8(hexDigit(rune(digits[2]))*16 +
		hexDigit(rune(digits[3])))
	color.Blue = uint8(hexDigit(rune(digits[4]))*16 +
		hexDigit(rune(digits[5])))
	return color
}

func hexDigit(ch rune) int {
	switch {
	case ch >= '0' && ch <= '9':
		return int(ch - '0')
	case ch >= 'a' && ch <= 'f':
		return int(ch-'a') + 10
	case ch >= 'A' && ch <= 'F':
		return int(ch-'A') + 10
	default:
		return -1
	}
}
