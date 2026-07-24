package ui

import (
	"fmt"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type tuiScreen struct {
	buf      *tui.Buffer
	area     geom.Area
	styleIn  uv.Style
	styleOut tui.Style
	styleOk  bool
}

var ansiUnderlineTUIStyles = [...]tui.UnderlineStyle{
	ansi.UnderlineNone:   tui.UnderlineReset,
	ansi.UnderlineSingle: tui.UnderlineLine,
	ansi.UnderlineDouble: tui.UnderlineDoubleLine,
	ansi.UnderlineCurly:  tui.UnderlineCurl,
	ansi.UnderlineDotted: tui.UnderlineDotted,
	ansi.UnderlineDashed: tui.UnderlineDashed,
}

func (s *tuiScreen) SetCell(at geom.Point, c *uv.Cell) {
	if !s.area.Size.Contains(at) {
		return
	}
	dst := s.area.Point.Add(at)
	if c == nil {
		s.buf.Set(dst, tui.Cell{Symbol: " "})
		return
	}
	content := c.Content
	if content == "" {
		content = " "
	}
	s.buf.Set(dst, tui.Cell{
		Symbol: content,
		Style:  s.styleFor(c.Style),
	})
	for i := 1; i < c.Width; i++ {
		s.buf.Set(dst.Add(geom.Point{X: i}), tui.Cell{Skip: true})
	}
}

func (s *tuiScreen) styleFor(st uv.Style) tui.Style {
	if s.styleOk && s.styleIn == st {
		return s.styleOut
	}
	out := uvStyleToTUI(st)
	s.styleIn, s.styleOut, s.styleOk = st, out, true
	return out
}

func (r *renderPass) renderTerminalPane(
	buf *tui.Buffer, tp *TerminalPane, y0 int, focused bool,
) {
	a := tp.Area()
	contentH := max(a.Height-1, 0)
	emu := tp.Emulator()
	bg := r.activeTheme().Get("ui.background").BgColor()
	if emu.BackgroundColor() != bg {
		emu.SetBackgroundColor(bg)
	}
	scr := &tuiScreen{
		buf: buf,
		area: geom.Area{
			Point: geom.Point{X: a.X, Y: y0 + a.Y},
			Size:  geom.Size{Width: a.Width, Height: contentH},
		},
	}
	drawViewport(scr, tp, tp.ScrollOffset(), geom.Size{
		Width: a.Width, Height: contentH,
	})
	highlightSelection(scr, tp)
	r.renderTerminalStatus(buf, tp, y0, focused)
}

func (r *renderPass) renderTerminalStatus(
	buf *tui.Buffer, tp *TerminalPane, y0 int, focused bool,
) {
	a := tp.Area()
	th := r.activeTheme()
	statusKey := "ui.statusline.inactive"
	if focused {
		statusKey = "ui.statusline"
	}
	st := th.Get(statusKey)
	y := y0 + a.Bottom()
	buf.FillRange(geom.Point{X: a.X, Y: y}, a.Width, st)

	modeSt := st
	if focused {
		modeSt = th.Get("ui.statusline.terminal")
	}
	label := " TRM "
	if tp.ConsumeBell(focused) && !focused {
		label = " TRM* "
	}
	buf.SetString(geom.Point{X: a.X, Y: y}, label, modeSt)

	title := tp.Title()
	if title == "" {
		title = "terminal"
	}
	if n := tp.ScrollOffset(); n > 0 {
		title = fmt.Sprintf("%s [scrollback -%d]", title, n)
	}
	buf.SetString(geom.Point{
		X: a.X + runewidth.StringWidth(label),
		Y: y,
	}, " "+title, st)
}

func highlightSelection(scr *tuiScreen, tp *TerminalPane) {
	sp, ok := tp.selectedSpan()
	if !ok {
		return
	}
	// span is in absolute (scrollback+screen) rows; translate to the rows
	// currently visible in this viewport
	start := tp.viewStart(scr.area.Height)
	y0, y1 := sp.start.Y-start, sp.end.Y-start
	for y := max(y0, 0); y <= y1 && y < scr.area.Height; y++ {
		startX, endX := 0, scr.area.Width-1
		if y == y0 {
			startX = sp.start.X
		}
		if y == y1 {
			endX = sp.end.X
		}
		for x := max(startX, 0); x <= endX && x < scr.area.Width; x++ {
			p := scr.area.Point.Add(geom.Point{X: x, Y: y})
			c := scr.buf.Get(p)
			c.Style = c.Style.Mod(tui.ModifierReversed)
			scr.buf.Set(p, c)
		}
	}
}

func uvStyleToTUI(st uv.Style) tui.Style {
	out := tui.Style{}
	out = out.Fg(colorToTUI(st.Fg))
	out = out.Bg(colorToTUI(st.Bg))
	out = out.UlColor(colorToTUI(st.UnderlineColor))
	out = out.UlStyle(ansiUnderlineToTUI(st.Underline))
	var m tui.Modifier
	if st.Attrs&uv.AttrBold != 0 {
		m |= tui.ModifierBold
	}
	if st.Attrs&uv.AttrFaint != 0 {
		m |= tui.ModifierDim
	}
	if st.Attrs&uv.AttrItalic != 0 {
		m |= tui.ModifierItalic
	}
	if st.Attrs&uv.AttrBlink != 0 {
		m |= tui.ModifierSlowBlink
	}
	if st.Attrs&uv.AttrReverse != 0 {
		m |= tui.ModifierReversed
	}
	if st.Attrs&uv.AttrStrikethrough != 0 {
		m |= tui.ModifierCrossedOut
	}
	if m != 0 {
		out = out.Mod(m)
	}
	return out
}

func ansiUnderlineToTUI(u ansi.Underline) tui.UnderlineStyle {
	if int(u) >= len(ansiUnderlineTUIStyles) {
		return tui.UnderlineReset
	}
	return ansiUnderlineTUIStyles[u]
}

func drawViewport(
	scr *tuiScreen, tp *TerminalPane, n int, size geom.Size,
) {
	emu := tp.Emulator()
	bg := emu.BackgroundColor()
	blank := uv.EmptyCell
	blank.Style.Bg = bg

	sb := emu.Scrollback()
	sbLen := sb.Len()
	total := sbLen + emu.Height()
	start := max(total-size.Height-n, 0)
	for row := range size.Height {
		line := start + row
		for x := 0; x < size.Width; {
			var cell *uv.Cell
			if line < total {
				if line < sbLen {
					cell = sb.CellAt(x, line)
				} else {
					cell = emu.CellAt(x, line-sbLen)
				}
			}
			if cell == nil {
				scr.SetCell(geom.Point{X: x, Y: row}, &blank)
				x++
				continue
			}
			if cell.Style.Bg == nil && bg != nil {
				cell = cell.Clone()
				cell.Style.Bg = bg
			}
			scr.SetCell(geom.Point{X: x, Y: row}, cell)
			x += max(cell.Width, 1)
		}
	}
}
