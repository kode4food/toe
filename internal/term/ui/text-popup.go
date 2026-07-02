package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/tui"
)

type (
	popupMarkdown struct {
		lines []popupLine
		width int
		code  bool
		lang  string
	}

	popupLine struct {
		text    string
		lang    string
		code    bool
		heading bool
	}

	popupTextRenderer struct {
		buf  *tui.Buffer
		cx   *Context
		area popupArea
		base tui.Style
	}
)

func (p *popupMarkdown) parse(text string) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	for line := range strings.SplitSeq(text, "\n") {
		p.parseLine(line)
	}
	p.trim()
}

func (p *popupMarkdown) parseLine(line string) {
	if lang, ok := popupFence(line); ok {
		p.code = !p.code
		p.lang = lang
		if !p.code {
			p.lang = ""
		}
		return
	}
	if p.code {
		p.lines = append(p.lines, popupLine{
			text: line,
			lang: p.lang,
			code: true,
		})
		return
	}
	line = strings.TrimRight(line, " \t")
	if strings.TrimSpace(line) == "" {
		p.lines = append(p.lines, popupLine{})
		return
	}
	if text, ok := popupHeading(line); ok {
		p.appendWrapped(text, true)
		return
	}
	p.appendWrapped(popupInlineText(line), false)
}

func (p *popupMarkdown) appendWrapped(text string, heading bool) {
	if p.width <= 0 {
		return
	}
	for line := range strings.SplitSeq(lipgloss.Wrap(text, p.width, ""), "\n") {
		p.lines = append(p.lines, popupLine{
			text:    line,
			heading: heading,
		})
	}
}

func (p *popupMarkdown) trim() {
	for len(p.lines) > 0 && p.lines[len(p.lines)-1].text == "" {
		p.lines = p.lines[:len(p.lines)-1]
	}
}

func (r *popupTextRenderer) render(lines []popupLine) {
	for i := 0; i < r.area.h && i < len(lines); i++ {
		r.renderLine(lines[i], r.area.y+i)
	}
}

func (r *popupTextRenderer) renderLine(line popupLine, y int) {
	if line.code {
		r.renderCode(line, y)
		return
	}
	st := r.base
	if line.heading {
		st = st.Mod(tui.ModifierBold)
	}
	text := ansi.Truncate(line.text, r.area.w, "")
	r.buf.SetString(r.area.x, y, text, st)
}

func (r *popupTextRenderer) renderCode(line popupLine, y int) {
	text := ansi.Truncate(line.text, r.area.w, "")
	spans := highlight.Tokenize(text, line.lang)
	if len(spans) == 0 {
		r.buf.SetString(r.area.x, y, text, r.base)
		return
	}
	rs := []rune(text)
	x := r.area.x
	pos := 0
	for _, s := range spans {
		start := min(s.Start, len(rs))
		if start > pos {
			x = r.writeRun(x, y, string(rs[pos:start]), r.base)
		}
		end := min(s.End, len(rs))
		if end < start {
			continue
		}
		st := r.highlightStyle(s.Scope)
		x = r.writeRun(x, y, string(rs[start:end]), st)
		pos = end
	}
	if pos < len(rs) {
		r.writeRun(x, y, string(rs[pos:]), r.base)
	}
}

func (r *popupTextRenderer) writeRun(x, y int, text string, st tui.Style) int {
	r.buf.SetString(x, y, text, st)
	return x + runewidth.StringWidth(text)
}

func (r *popupTextRenderer) highlightStyle(scope string) tui.Style {
	bg := r.cx.Theme().Get("ui.popup").GetBackground()
	if st, ok := r.cx.Theme().TryGet(scope); ok {
		return lipglossToTUIStyle(inheritStyleBackground(st, bg))
	}
	st := inheritStyleBackground(highlight.DefaultStyle(scope), bg)
	return lipglossToTUIStyle(st)
}

func drawTextPopup(
	buf *tui.Buffer, x, y, maxW, maxH int, text string, cx *Context,
) popupArea {
	lines := popupTextLines(text, maxW-2)
	w := popupTextWidth(lines) + 2
	h := len(lines) + 2
	w = min(max(w, 2), maxW)
	h = min(max(h, 2), maxH)
	if x+w > buf.Width {
		x = max(buf.Width-w, 0)
	}
	if y+h > buf.Height {
		y = max(buf.Height-h, 0)
	}
	st := lipglossToTUIStyle(cx.Theme().Get("ui.popup"))
	pop := popup{
		border:       lipgloss.RoundedBorder(),
		borderStyle:  st,
		contentStyle: st,
		padX:         0,
	}
	area := pop.drawInto(buf, x, y, w, h)
	r := popupTextRenderer{buf: buf, cx: cx, area: area, base: st}
	r.render(lines)
	return area
}

func popupTextLines(text string, w int) []popupLine {
	if w <= 0 {
		return nil
	}
	p := popupMarkdown{width: w}
	p.parse(text)
	return p.lines
}

func popupTextWidth(lines []popupLine) int {
	w := 0
	for _, line := range lines {
		w = max(w, runewidth.StringWidth(line.text))
	}
	return w
}

func popupFence(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "```") {
		return "", false
	}
	lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
	if idx := strings.IndexAny(lang, " \t"); idx >= 0 {
		lang = lang[:idx]
	}
	return lang, true
}

func popupHeading(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	if i == 0 || i > 6 || i >= len(trimmed) || trimmed[i] != ' ' {
		return "", false
	}
	return popupInlineText(strings.TrimSpace(trimmed[i:])), true
}

func popupInlineText(text string) string {
	r := strings.NewReplacer("**", "", "__", "", "`", "")
	return r.Replace(text)
}
