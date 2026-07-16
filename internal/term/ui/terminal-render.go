package ui

import (
	"fmt"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/tui"
)

type tuiScreen struct {
	buf      *tui.Buffer
	originX  int
	originY  int
	w, h     int
	styleIn  uv.Style
	styleOut tui.Style
	styleOk  bool
}

func (s *tuiScreen) SetCell(x, y int, c *uv.Cell) {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return
	}
	if c == nil {
		s.buf.Set(s.originX+x, s.originY+y, tui.Cell{Symbol: " "})
		return
	}
	content := c.Content
	if content == "" {
		content = " "
	}
	s.buf.Set(s.originX+x, s.originY+y, tui.Cell{
		Symbol: content, Style: s.styleFor(c.Style),
	})
	for i := 1; i < c.Width; i++ {
		s.buf.Set(s.originX+x+i, s.originY+y, tui.Cell{Skip: true})
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
	bg := r.activeTheme().Get("ui.background").GetBackground()
	emu.SetBackgroundColor(bg)
	scr := &tuiScreen{
		buf: buf, originX: a.X, originY: y0 + a.Y,
		w: a.Width, h: contentH,
	}
	drawViewport(scr, tp, tp.ScrollOffset(), a.Width, contentH)
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
	st := lipglossToTUIStyle(th.Get(statusKey))
	y := y0 + a.Y + a.Height - 1
	buf.FillRange(a.X, y, a.Width, st)

	modeSt := st
	if focused {
		modeSt = lipglossToTUIStyle(th.Get("ui.statusline.terminal"))
	}
	label := " TRM "
	if tp.ConsumeBell(focused) && !focused {
		label = " TRM* "
	}
	buf.SetString(a.X, y, label, modeSt)

	title := tp.Title()
	if title == "" {
		title = "terminal"
	}
	if n := tp.ScrollOffset(); n > 0 {
		title = fmt.Sprintf("%s [scrollback -%d]", title, n)
	}
	buf.SetString(a.X+runewidth.StringWidth(label), y, " "+title, st)
}

func highlightSelection(scr *tuiScreen, tp *TerminalPane) {
	sel := tp.selection()
	if !sel.active {
		return
	}
	// span is in absolute (scrollback+screen) rows; translate to the rows
	// currently visible in this viewport
	start := tp.viewStart(scr.h)
	sp := sel.span
	y0, y1 := sp.start.Y-start, sp.end.Y-start
	for y := max(y0, 0); y <= y1 && y < scr.h; y++ {
		startX, endX := 0, scr.w-1
		if y == y0 {
			startX = sp.start.X
		}
		if y == y1 {
			endX = sp.end.X
		}
		for x := max(startX, 0); x <= endX && x < scr.w; x++ {
			c := scr.buf.Get(scr.originX+x, scr.originY+y)
			c.Style = c.Style.Mod(tui.ModifierReversed)
			scr.buf.Set(scr.originX+x, scr.originY+y, c)
		}
	}
}

func uvStyleToTUI(st uv.Style) tui.Style {
	out := tui.Style{}
	out = out.Fg(lipglossColorToTUI(st.Fg))
	out = out.Bg(lipglossColorToTUI(st.Bg))
	out = out.UlColor(lipglossColorToTUI(st.UnderlineColor))
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
	switch u {
	case ansi.UnderlineSingle:
		return tui.UnderlineLine
	case ansi.UnderlineDouble:
		return tui.UnderlineDoubleLine
	case ansi.UnderlineCurly:
		return tui.UnderlineCurl
	case ansi.UnderlineDotted:
		return tui.UnderlineDotted
	case ansi.UnderlineDashed:
		return tui.UnderlineDashed
	default:
		return tui.UnderlineReset
	}
}

func drawViewport(scr *tuiScreen, tp *TerminalPane, n, w, h int) {
	emu := tp.Emulator()
	bg := emu.BackgroundColor()
	blank := uv.EmptyCell
	blank.Style.Bg = bg

	sb := emu.Scrollback()
	sbLen := sb.Len()
	total := sbLen + emu.Height()
	start := max(total-h-n, 0)
	for row := range h {
		line := start + row
		for x := 0; x < w; {
			var cell *uv.Cell
			if line < total {
				if line < sbLen {
					cell = sb.CellAt(x, line)
				} else {
					cell = emu.CellAt(x, line-sbLen)
				}
			}
			if cell == nil {
				scr.SetCell(x, row, &blank)
				x++
				continue
			}
			if cell.Style.Bg == nil && bg != nil {
				cell = cell.Clone()
				cell.Style.Bg = bg
			}
			scr.SetCell(x, row, cell)
			x += max(cell.Width, 1)
		}
	}
}
