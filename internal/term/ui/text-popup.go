package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
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
		rule    bool
	}

	popupTextRenderer struct {
		buf  *tui.Buffer
		cx   *Context
		area geom.Area
		base tui.Style
		padX int
	}
)

// popupPadX is the left/right content padding, matching the doc/prompt popups
const popupPadX = 1

// markdownLink matches inline links and images: [text](url) and ![alt](url),
// capturing the visible text so the URL and brackets can be dropped
var markdownLink = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)

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
		// collapse a run of blanks to one break and drop leading blanks; a
		// preceding rule counts as blank here, so it swallows the blank after
		if len(p.lines) == 0 || p.lines[len(p.lines)-1].text == "" {
			return
		}
		p.lines = append(p.lines, popupLine{})
		return
	}
	if popupRule(line) {
		// drop the blank (or prior) before it so it sits flush against text
		if n := len(p.lines); n > 0 && p.lines[n-1].text == "" {
			p.lines = p.lines[:n-1]
		}
		p.lines = append(p.lines, popupLine{rule: true})
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
	wrapped := core.ReflowHardWrap(text, p.width)
	for line := range strings.SplitSeq(wrapped, "\n") {
		p.lines = append(p.lines, popupLine{
			text:    line,
			heading: heading,
		})
	}
}

func (p *popupMarkdown) trim() {
	p.lines = trimPopupLines(p.lines, 0)
}

func (r *popupTextRenderer) render(lines []popupLine) {
	for i := 0; i < r.area.Height && i < len(lines); i++ {
		r.renderLine(lines[i], r.area.Y+i)
	}
}

func (r *popupTextRenderer) renderLine(line popupLine, y int) {
	if line.rule {
		// span the full box, tying into border like the picker cut-separator
		w := r.area.Width + 2*r.padX
		rule := borderML + strings.Repeat(borderH, w) + borderMR
		r.buf.SetString(geom.Point{
			X: r.area.X - 1 - r.padX,
			Y: y,
		}, rule, r.base)
		return
	}
	if line.code {
		r.renderCode(line, y)
		return
	}
	st := r.base
	if line.heading {
		st = st.Mod(tui.ModifierBold)
	}
	text := ansi.Truncate(line.text, r.area.Width, "")
	r.buf.SetString(geom.Point{X: r.area.X, Y: y}, text, st)
}

func (r *popupTextRenderer) renderCode(line popupLine, y int) {
	text := ansi.Truncate(line.text, r.area.Width, "")
	spans := highlight.Tokenize(text, line.lang)
	if len(spans) == 0 {
		r.buf.SetString(geom.Point{X: r.area.X, Y: y}, text, r.base)
		return
	}
	rs := []rune(text)
	x := r.area.X
	pos := 0
	for _, s := range spans {
		start := min(s.Start, len(rs))
		if start > pos {
			x = r.writeRun(
				geom.Point{X: x, Y: y}, string(rs[pos:start]), r.base,
			)
		}
		end := min(s.End, len(rs))
		if end < start {
			continue
		}
		st := r.highlightStyle(s.Scope)
		x = r.writeRun(geom.Point{X: x, Y: y}, string(rs[start:end]), st)
		pos = end
	}
	if pos < len(rs) {
		r.writeRun(geom.Point{X: x, Y: y}, string(rs[pos:]), r.base)
	}
}

func (r *popupTextRenderer) writeRun(
	at geom.Point, text string, st tui.Style,
) int {
	r.buf.SetString(at, text, st)
	return at.X + runewidth.StringWidth(text)
}

func (r *popupTextRenderer) highlightStyle(scope string) tui.Style {
	bg := r.cx.Theme().Get("ui.popup").BgColor()
	if st, ok := r.cx.Theme().TryGet(scope); ok {
		return inheritStyleBackground(st, bg)
	}
	st := inheritStyleBackground(highlight.DefaultStyle(scope), bg)
	return st
}

func trimPopupLines(lines []popupLine, maxVisible int) []popupLine {
	for len(lines) > 0 && lines[0].rule {
		lines = lines[1:]
	}
	if maxVisible > 0 && len(lines) > maxVisible {
		lines = lines[:maxVisible]
	}
	for len(lines) > 0 {
		last := lines[len(lines)-1]
		if last.rule || strings.TrimSpace(last.text) == "" {
			lines = lines[:len(lines)-1]
		} else {
			break
		}
	}
	return lines
}

func measureTextPopup(maxSize geom.Size, text string) ([]popupLine, geom.Size) {
	lines := popupTextLines(text, maxSize.Width-2-2*popupPadX)
	lines = trimPopupLines(lines, maxSize.Height-2)
	w := popupTextWidth(lines) + 2 + 2*popupPadX
	h := len(lines) + 2
	w = min(max(w, 2), maxSize.Width)
	h = min(max(h, 2), maxSize.Height)
	return lines, geom.Size{Width: w, Height: h}
}

func paintTextPopup(cx *Context, buf *tui.Buffer, lines []popupLine) {
	st := cx.Theme().Get("ui.popup")
	pop := popup{
		borderStyle:  st,
		contentStyle: st,
		padX:         popupPadX,
	}
	area := pop.drawInto(buf, geom.Area{Size: buf.Size})
	r := popupTextRenderer{
		buf: buf, cx: cx, area: area, base: st, padX: popupPadX,
	}
	r.render(lines)
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

func popupRule(line string) bool {
	s := strings.TrimSpace(line)
	if len(s) < 3 {
		return false
	}
	c := s[0]
	if c != '-' && c != '*' && c != '_' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] != c && s[i] != ' ' {
			return false
		}
	}
	return true
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
	text = markdownLink.ReplaceAllString(text, "$1")
	r := strings.NewReplacer("**", "", "__", "", "`", "")
	return r.Replace(text)
}
