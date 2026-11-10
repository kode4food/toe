package tui

import (
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/kode4food/toe/internal/geom"
)

type (
	// Screen tracks the frame the terminal last showed, so each new frame writes only the cells
	// that changed
	Screen struct {
		prev    *Frame
		pool    sync.Pool
		lastLen int
		base    [][]Cell
		blank   []Cell
	}

	// Frame is an immutable Buffer snapshot, written to the terminal by the Screen that took it
	Frame struct {
		screen *Screen
		buf    Buffer
	}

	// scrollMatch is a run of new rows top..bottom matching previous frame's rows shifted by shift
	scrollMatch struct {
		top    int
		bottom int
		shift  int
		gain   int
	}
)

const (
	sgrReset = csi + "m"

	csiCursorPosition = 'H'
	csiScrollRegion   = 'r'
	csiScrollUp       = 'S'
	csiScrollDown     = 'T'

	// minScrollGain is the fewest row rewrites a terminal scroll must save
	minScrollGain = 4
)

// Snapshot copies b into a frame that stays valid while b is redrawn
func (s *Screen) Snapshot(b *Buffer) *Frame {
	var cells []Cell
	if p, ok := s.pool.Get().(*[]Cell); ok {
		cells = *p
	}
	if cap(cells) < len(b.cells) {
		cells = make([]Cell, len(b.cells))
	}
	cells = cells[:len(b.cells)]
	copy(cells, b.cells)
	return &Frame{
		screen: s,
		buf: Buffer{
			cells: cells,
			Size:  b.Size,
		},
	}
}

// WriteFrame writes the cells that differ from the last frame written, or every cell when nothing
// valid was written before
func (f *Frame) WriteFrame(w io.Writer) error {
	s := f.screen
	prev := s.prev
	repaint := prev == nil || prev.buf.Size != f.buf.Size
	var sb strings.Builder
	sb.Grow(s.lastLen)
	if !repaint {
		s.resetBase(prev)
		s.scroll(&sb, f)
	}
	e := &ansiEmitter{out: &sb}
	var style Style
	for y := range f.buf.Height {
		at := -1
		for x, c := range bufferRow(&f.buf, y) {
			if c.Skip || !repaint && c == s.base[y][x] {
				continue
			}
			if at != x {
				writeCursorPosition(&sb, geom.Point{
					X: x,
					Y: y,
				})
			}
			if c.Style != style {
				e.emitStyle(c.Style)
				style = c.Style
			}
			sb.WriteString(c.Symbol)
			at = x + 1
		}
	}
	if style != (Style{}) {
		sb.WriteString(sgrReset)
	}
	s.lastLen = sb.Len()
	if prev != nil && prev != f {
		s.pool.Put(&prev.buf.cells)
	}
	s.prev = f
	_, err := io.WriteString(w, sb.String())
	return err
}

// Invalidate forgets what the terminal shows, so the next write covers every cell
func (f *Frame) Invalidate() {
	f.screen.prev = nil
}

// String renders the frame as ANSI rows joined by '\n'
func (f *Frame) String() string {
	return f.buf.RenderToANSI()
}

func (s *Screen) resetBase(prev *Frame) {
	s.base = s.base[:0]
	for y := range prev.buf.Height {
		s.base = append(s.base, bufferRow(&prev.buf, y))
	}
	if len(s.blank) != prev.buf.Width {
		s.blank = make([]Cell, prev.buf.Width)
		for i := range s.blank {
			s.blank[i] = defaultCell
		}
	}
}

// scroll finds rows the terminal already shows elsewhere and moves them with a scroll region,
// leaving s.base matching what the terminal then displays
func (s *Screen) scroll(sb *strings.Builder, f *Frame) {
	first := 0
	for first < f.buf.Height && s.unchanged(f, first) {
		first++
	}
	if first == f.buf.Height {
		return
	}
	last := f.buf.Height - 1
	for s.unchanged(f, last) {
		last--
	}
	if last-first+1 < minScrollGain {
		return
	}
	// a scroll worth doing spans most of the changed rows, so one of these anchors lands inside it
	// even when a status or tab line changed too
	best := scrollMatch{bottom: -1}
	for _, y := range [...]int{first, (first + last) / 2, last} {
		if y >= best.top && y <= best.bottom {
			continue
		}
		if m := s.bestScroll(f, y); m.gain > best.gain {
			best = m
		}
	}
	if best.gain >= minScrollGain {
		s.applyScroll(sb, best)
	}
}

// bestScroll tries each old row equal to new row y as part of a shifted run, walking up to where
// that run starts
func (s *Screen) bestScroll(f *Frame, y int) scrollMatch {
	row := bufferRow(&f.buf, y)
	best := scrollMatch{}
	for j, old := range s.base {
		if j == y || !slices.Equal(row, old) {
			continue
		}
		shift := j - y
		top := y
		for top > 0 && top-1+shift >= 0 &&
			slices.Equal(bufferRow(&f.buf, top-1), s.base[top-1+shift]) {
			top--
		}
		seed := scrollMatch{
			top:   top,
			shift: shift,
		}
		if m := s.scrollRun(f, seed); m.gain > best.gain {
			best = m
		}
	}
	return best
}

// scrollRun extends seed down over new rows matching old rows shift away, crediting rows it
// recovers and charging unchanged rows the scroll blanks
func (s *Screen) scrollRun(f *Frame, seed scrollMatch) scrollMatch {
	m := seed
	m.bottom = m.top - 1
	for y := m.top; y < f.buf.Height && y+m.shift < f.buf.Height; y++ {
		row := bufferRow(&f.buf, y)
		if !slices.Equal(row, s.base[y+m.shift]) {
			break
		}
		m.bottom = y
		if !s.unchanged(f, y) {
			m.gain++
		}
	}
	from := m.bottom + 1
	to := m.bottom + m.shift
	if m.shift < 0 {
		from = m.top + m.shift
		to = m.top - 1
	}
	for y := from; y <= to; y++ {
		if s.unchanged(f, y) {
			m.gain--
		}
	}
	return m
}

func (s *Screen) unchanged(f *Frame, y int) bool {
	return slices.Equal(bufferRow(&f.buf, y), s.base[y])
}

func (s *Screen) applyScroll(sb *strings.Builder, m scrollMatch) {
	writeScroll(sb, m)
	if m.shift > 0 {
		for y := m.top; y <= m.bottom; y++ {
			s.base[y] = s.base[y+m.shift]
		}
		for y := m.bottom + 1; y <= m.bottom+m.shift; y++ {
			s.base[y] = s.blank
		}
	} else {
		for y := m.bottom; y >= m.top; y-- {
			s.base[y] = s.base[y+m.shift]
		}
		for y := m.top + m.shift; y < m.top; y++ {
			s.base[y] = s.blank
		}
	}
}

func bufferRow(b *Buffer, y int) []Cell {
	return b.cells[y*b.Width : (y+1)*b.Width]
}

func writeCursorPosition(sb *strings.Builder, p geom.Point) {
	sb.WriteString(csi)
	writeInt(sb, p.Y+1)
	sb.WriteByte(';')
	writeInt(sb, p.X+1)
	sb.WriteByte(csiCursorPosition)
}

// writeScroll limits scrolling to the rows m moves, scrolls them by its shift, then restores the
// full-screen scroll region
func writeScroll(sb *strings.Builder, m scrollMatch) {
	top := m.top
	bottom := m.bottom + m.shift
	rows := m.shift
	final := byte(csiScrollUp)
	if m.shift < 0 {
		top = m.top + m.shift
		bottom = m.bottom
		rows = -m.shift
		final = csiScrollDown
	}
	sb.WriteString(csi)
	writeInt(sb, top+1)
	sb.WriteByte(';')
	writeInt(sb, bottom+1)
	sb.WriteByte(csiScrollRegion)
	sb.WriteString(csi)
	writeInt(sb, rows)
	sb.WriteByte(final)
	sb.WriteString(csi)
	sb.WriteByte(csiScrollRegion)
}

func writeInt(sb *strings.Builder, n int) {
	var tmp [20]byte
	sb.Write(strconv.AppendInt(tmp[:0], int64(n), 10))
}
