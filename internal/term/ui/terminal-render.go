package ui

import (
	"fmt"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/tui"
)

// tuiScreen adapts a rectangular region of a *tui.Buffer to uv.Screen so a
// vt.Emulator can Draw directly into it
type tuiScreen struct {
	buf       *tui.Buffer
	originX   int
	originY   int
	w, h      int
	widthMeth uv.WidthMethod
}

var _ uv.Screen = (*tuiScreen)(nil)

func (s *tuiScreen) Bounds() uv.Rectangle {
	return uv.Rect(0, 0, s.w, s.h)
}

func (s *tuiScreen) CellAt(x, y int) *uv.Cell {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return nil
	}
	c := s.buf.Get(s.originX+x, s.originY+y)
	return &uv.Cell{Content: c.Symbol, Width: 1}
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
		Symbol: content, Style: uvStyleToTUI(c.Style),
	})
	for i := 1; i < c.Width; i++ {
		s.buf.Set(s.originX+x+i, s.originY+y, tui.Cell{Skip: true})
	}
}

func (s *tuiScreen) WidthMethod() uv.WidthMethod {
	return s.widthMeth
}

// renderTerminalPane paints a terminal pane's emulator screen into buf at
// its assigned area, reserving the bottom row for the status line
func (r *renderPass) renderTerminalPane(
	buf *tui.Buffer, tp *TerminalPane, y0 int, focused bool,
) {
	a := tp.Area()
	contentH := max(a.Height-1, 0)
	emu := tp.Emulator()
	scr := &tuiScreen{
		buf: buf, originX: a.X, originY: y0 + a.Y,
		w: a.Width, h: contentH, widthMeth: emu.WidthMethod(),
	}
	if n := tp.ScrollOffset(); n > 0 {
		drawScrollback(scr, tp, n, a.Width, contentH)
	} else {
		emu.Draw(scr, uv.Rect(0, 0, a.Width, contentH))
	}
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

// drawScrollback paints the h lines ending n lines back from live output,
// combining scrollback with the live screen for the most recent lines
func drawScrollback(scr *tuiScreen, tp *TerminalPane, n, w, h int) {
	emu := tp.Emulator()
	sb := emu.Scrollback()
	sbLen := sb.Len()
	total := sbLen + emu.Height()
	start := max(total-h-n, 0)
	for row := range h {
		line := start + row
		if line >= total {
			continue
		}
		for x := range w {
			if line < sbLen {
				scr.SetCell(x, row, sb.CellAt(x, line))
			} else {
				scr.SetCell(x, row, emu.CellAt(x, line-sbLen))
			}
		}
	}
}
