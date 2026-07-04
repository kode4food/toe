package tui

import "github.com/rivo/uniseg"

type (
	Cell struct {
		Symbol string
		Style  Style
		Skip   bool
	}

	Buffer struct {
		cells []Cell
		// lastANSILen pre-sizes the next RenderToANSI output
		lastANSILen int
		Width       int
		Height      int
	}
)

var (
	defaultCell = Cell{Symbol: " "}

	// asciiTable holds bytes 0x00..0x7f in order; asciiTable[ch:ch+1] yields a
	// single-byte string that shares this backing array, so cell symbols for
	// ASCII content are produced without allocation
	asciiTable = func() string {
		var b [128]byte
		for i := range b {
			b[i] = byte(i)
		}
		return string(b[:])
	}()
)

func NewBuffer(w, h int) *Buffer {
	n := max(w*h, 0)
	cells := make([]Cell, n)
	for i := range cells {
		cells[i] = defaultCell
	}
	return &Buffer{cells: cells, Width: w, Height: h}
}

func (b *Buffer) Set(x, y int, c Cell) {
	if x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return
	}
	b.cells[y*b.Width+x] = c
}

// PatchBg sets only the background color of the cell at (x, y), preserving its
// symbol and foreground. Used to overlay rulers behind already-rendered text
func (b *Buffer) PatchBg(x, y int, bg Color) {
	if x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return
	}
	i := y*b.Width + x
	b.cells[i].Style = b.cells[i].Style.Bg(bg)
}

// PatchBgRange sets only the background color of width cells starting at
// (x, y), preserving each cell's symbol and foreground. Used to overlay a row
// highlight behind already-rendered text
func (b *Buffer) PatchBgRange(x, y, width int, bg Color) {
	for i := range width {
		b.PatchBg(x+i, y, bg)
	}
}

func (b *Buffer) Get(x, y int) Cell {
	if x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return defaultCell
	}
	return b.cells[y*b.Width+x]
}

func (b *Buffer) Clear() {
	for i := range b.cells {
		b.cells[i] = defaultCell
	}
}

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
func (b *Buffer) FillRange(x, y, width int, style Style) {
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

// SetRightAlignedInt writes the non-negative integer n as decimal digits
// right-aligned within a field of the given width starting at (x, y), padding
// the left with spaces. It uses the ASCII cell table and a stack scratch, so
// it allocates nothing. Columns outside the buffer are clipped
func (b *Buffer) SetRightAlignedInt(x, y, width, n int, style Style) {
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
		b.setASCIICell(col, y, ' ', style)
		col++
	}
	for ; i < len(tmp); i++ {
		b.setASCIICell(col, y, tmp[i], style)
		col++
	}
}

// SetString writes graphemes of s starting at (x, y), advancing x by display
// width. Wide graphemes reserve their trailing columns without emitting
// additional printable cells. A style with no background is transparent: each
// written cell keeps its existing background so a pre-painted layer (ruler,
// cursorline) shows through the glyphs
func (b *Buffer) SetString(x, y int, s string, style Style) {
	if y < 0 || y >= b.Height || x >= b.Width {
		return
	}
	keepBg := style.BgColor().IsReset()
	nx, rest := b.setASCIIString(x, y, s, style, keepBg)
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
		b.Set(x, y, Cell{Symbol: cluster, Style: st})
		for i := 1; i < w && x+i < b.Width; i++ {
			b.Set(x+i, y, Cell{Skip: true, Style: st})
		}
		x += w
	}
}

// setASCIICell writes a single printable ASCII byte to (x, y) using the shared
// asciiTable, avoiding a per-call string allocation. The caller is responsible
// for validating y; x is clipped to the buffer width. A style with no
// background is transparent: the cell keeps its existing background
func (b *Buffer) setASCIICell(x, y int, ch byte, style Style) {
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
	x, y int, s string, style Style, overBg bool,
) (int, string) {
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
