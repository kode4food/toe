package tui

import (
	"github.com/rivo/uniseg"

	"github.com/kode4food/toe/internal/geom"
)

// Fill sets every cell in the buffer to a space with the given style
func (b *Buffer) Fill(style Style) {
	c := Cell{Symbol: " ", Style: style}
	for i := range b.cells {
		b.cells[i] = c
	}
}

// FillRange fills width cells with a space in the given style. A style with no
// background is transparent: each cell keeps its existing background so a
// pre-painted layer (ruler, cursorline) shows through
func (b *Buffer) FillRange(p geom.Point, width int, style Style) {
	x, y := p.X, p.Y
	if y < 0 || y >= b.Height || width <= 0 || x >= b.Width {
		return
	}
	if x < 0 {
		width += x
		x = 0
	}
	end := min(x+width, b.Width)
	keepBg := style.BgColor().IsReset()
	for i := x; i < end; i++ {
		idx := y*b.Width + i
		st := style
		if keepBg {
			st = style.Bg(b.cells[idx].Style.BgColor())
		}
		b.cells[idx] = Cell{Symbol: " ", Style: st}
	}
}

// SetRightAlignedInt writes n as decimal digits right-aligned in a field of
// width starting at p, padding left with spaces
func (b *Buffer) SetRightAlignedInt(
	p geom.Point, width, n int, style Style,
) {
	x, y := p.X, p.Y
	if y < 0 || y >= b.Height || width <= 0 || x >= b.Width {
		return
	}
	var tmp [20]byte
	i := len(tmp)
	if n <= 0 {
		i--
		tmp[i] = '0'
	} else {
		for n > 0 {
			i--
			tmp[i] = byte('0' + n%10)
			n /= 10
		}
	}
	col := x
	for pad := width - (len(tmp) - i); pad > 0; pad-- {
		b.setASCIICell(geom.Point{X: col, Y: y}, ' ', style)
		col++
	}
	for ; i < len(tmp); i++ {
		b.setASCIICell(geom.Point{X: col, Y: y}, tmp[i], style)
		col++
	}
}

// SetString writes graphemes of s starting at p, advancing x by display width.
// A style with no background is transparent, keeping each cell's existing
// background
func (b *Buffer) SetString(p geom.Point, s string, style Style) {
	x, y := p.X, p.Y
	if y < 0 || y >= b.Height || x >= b.Width {
		return
	}
	keepBg := style.BgColor().IsReset()
	nx, rest := b.setASCIIString(p, s, style, keepBg)
	if rest == "" {
		return
	}
	x = nx
	s = rest
	state := -1
	for len(s) > 0 {
		var cluster string
		cluster, s, _, state = uniseg.FirstGraphemeClusterInString(s, state)
		w := uniseg.StringWidth(cluster)
		if w == 0 {
			continue
		}
		if x >= b.Width {
			break
		}
		if x+w > b.Width {
			break
		}
		st := style
		if keepBg {
			st = style.Bg(b.cells[y*b.Width+x].Style.BgColor())
		}
		b.Set(geom.Point{X: x, Y: y}, Cell{Symbol: cluster, Style: st})
		for i := 1; i < w && x+i < b.Width; i++ {
			b.Set(geom.Point{X: x + i, Y: y}, Cell{Skip: true, Style: st})
		}
		x += w
	}
}

func (b *Buffer) setASCIICell(p geom.Point, ch byte, style Style) {
	x, y := p.X, p.Y
	if x < 0 || x >= b.Width {
		return
	}
	idx := y*b.Width + x
	if style.BgColor().IsReset() {
		style = style.Bg(b.cells[idx].Style.BgColor())
	}
	b.cells[idx] = Cell{Symbol: asciiTable[ch : ch+1], Style: style}
}

func (b *Buffer) setASCIIString(
	p geom.Point, s string, style Style, overBg bool,
) (int, string) {
	x, y := p.X, p.Y
	for i := range len(s) {
		ch := s[i]
		if ch < ' ' || ch >= 0x7f {
			return x, s[i:]
		}
		if x >= b.Width {
			return x, ""
		}
		if x >= 0 {
			idx := y*b.Width + x
			st := style
			if overBg {
				st = style.Bg(b.cells[idx].Style.BgColor())
			}
			b.cells[idx] = Cell{
				Symbol: asciiTable[ch : ch+1],
				Style:  st,
			}
		}
		x++
	}
	return x, ""
}
