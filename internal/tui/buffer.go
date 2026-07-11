package tui

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

// Blit copies src's cells into b at offset (ox, oy), clipped to b's bounds
func (b *Buffer) Blit(src *Buffer, ox, oy int) {
	for sy := range src.Height {
		dy := oy + sy
		if dy < 0 || dy >= b.Height {
			continue
		}
		for sx := range src.Width {
			dx := ox + sx
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
