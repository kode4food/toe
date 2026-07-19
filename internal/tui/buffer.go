package tui

import "github.com/kode4food/toe/internal/geom"

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
		geom.Size
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

func NewBuffer(size geom.Size) *Buffer {
	n := max(size.Width*size.Height, 0)
	cells := make([]Cell, n)
	for i := range cells {
		cells[i] = defaultCell
	}
	return &Buffer{cells: cells, Size: size}
}

func (b *Buffer) Set(p geom.Point, c Cell) {
	if !b.Size.Contains(p) {
		return
	}
	b.cells[p.Y*b.Width+p.X] = c
}

// PatchBg sets only the background color of the cell at p, preserving its
// symbol and foreground. Used to overlay rulers behind already-rendered text
func (b *Buffer) PatchBg(p geom.Point, bg Color) {
	if !b.Size.Contains(p) {
		return
	}
	i := p.Y*b.Width + p.X
	b.cells[i].Style = b.cells[i].Style.Bg(bg)
}

// PatchBgRange sets only the background color of width cells starting at p,
// preserving each cell's symbol and foreground. Used to overlay a row
// highlight behind already-rendered text
func (b *Buffer) PatchBgRange(p geom.Point, width int, bg Color) {
	for i := range width {
		b.PatchBg(geom.Point{X: p.X + i, Y: p.Y}, bg)
	}
}

// Blit copies src's cells into b at offset at, clipped to b's bounds
func (b *Buffer) Blit(src *Buffer, at geom.Point) {
	for sy := range src.Height {
		dy := at.Y + sy
		if dy < 0 || dy >= b.Height {
			continue
		}
		for sx := range src.Width {
			dx := at.X + sx
			if dx < 0 || dx >= b.Width {
				continue
			}
			c := src.cells[sy*src.Width+sx]
			// blank a wide glyph whose trailing Skip cell got clipped, so it
			// doesn't render with no room for its second column
			if !c.Skip && dx == b.Width-1 && sx+1 < src.Width &&
				src.cells[sy*src.Width+sx+1].Skip {
				c = Cell{Symbol: " ", Style: c.Style}
			}
			di := dy*b.Width + dx
			// reset bg means transparent: keep whatever's already at di
			if c.Style.BgColor().IsReset() {
				c.Style = c.Style.Bg(b.cells[di].Style.BgColor())
			}
			b.cells[di] = c
		}
	}
}

func (b *Buffer) Get(p geom.Point) Cell {
	if !b.Size.Contains(p) {
		return defaultCell
	}
	return b.cells[p.Y*b.Width+p.X]
}

func (b *Buffer) Clear() {
	for i := range b.cells {
		b.cells[i] = defaultCell
	}
}
