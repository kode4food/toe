package ui

import (
	"regexp"
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
		rule    bool
	}

	popupTextRenderer struct {
		buf  *tui.Buffer
		cx   *Context
		area popupArea
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
	if line.rule {
		// span the full box, tying into border like the picker cut-separator
		w := r.area.w + 2*r.padX
		rule := splitLeftT + strings.Repeat(horizSplit, w) + splitRightT
		r.buf.SetString(r.area.x-1-r.padX, y, rule, r.base)
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
	lines := popupTextLines(text, maxW-2-2*popupPadX)
	w := popupTextWidth(lines) + 2 + 2*popupPadX
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
		padX:         popupPadX,
	}
	area := pop.drawInto(buf, x, y, w, h)
	r := popupTextRenderer{buf: buf, cx: cx, area: area, base: st, padX: popupPadX}
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
