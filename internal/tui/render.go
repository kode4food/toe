package tui

import "strings"

// RenderToANSI serialises the buffer as rows joined by '\n', emitting style
// escapes only on changes — used to bridge into the string-based render path
func (b *Buffer) RenderToANSI() string {
	if b.Width == 0 || b.Height == 0 {
		return ""
	}
	var sb strings.Builder
	sb.Grow(b.Width*b.Height + max(b.Height-1, 0))
	e := &ansiEmitter{w: &sb}
	style := Style{}
	for y := range b.Height {
		if y > 0 {
			sb.WriteByte('\n')
		}
		style = emitRow(e, b.cells[y*b.Width:(y+1)*b.Width], style)
	}
	return sb.String()
}

// RenderRowsToANSI serialises each buffer row as an independent ANSI string,
// resetting style at the end of any row that left styling active. Unlike
// RenderToANSI it does not carry style across row boundaries, so the rows can
// be composed individually (for example inside a lipgloss box) without a row's
// trailing colors bleeding into the next
func (b *Buffer) RenderRowsToANSI() []string {
	rows := make([]string, b.Height)
	for y := range b.Height {
		var sb strings.Builder
		e := &ansiEmitter{w: &sb}
		style := emitRow(e, b.cells[y*b.Width:(y+1)*b.Width], Style{})
		if style != (Style{}) {
			sb.WriteString("\x1b[m")
		}
		rows[y] = sb.String()
	}
	return rows
}

type ansiEmitter struct {
	w        *strings.Builder
	fg, bg   Color
	ulColor  Color
	ulStyle  UnderlineStyle
	modifier Modifier
}

func (a *ansiEmitter) emitStyle(s Style) {
	a.emitModifiers(s.modifier)
	a.emitColors(s.fg, s.bg)
	a.emitUnderline(s.underlineColor, s.underlineStyle)
}

func (a *ansiEmitter) emitModifiers(m Modifier) {
	if m == a.modifier {
		return
	}
	removed := a.modifier &^ m
	added := m &^ a.modifier
	if removed.has(ModifierReversed) {
		_, _ = a.w.WriteString("\x1b[27m")
	}
	if removed.has(ModifierBold) {
		_, _ = a.w.WriteString("\x1b[22m")
		if m.has(ModifierDim) {
			_, _ = a.w.WriteString("\x1b[2m")
		}
	}
	if removed.has(ModifierItalic) {
		_, _ = a.w.WriteString("\x1b[23m")
	}
	if removed.has(ModifierDim) {
		_, _ = a.w.WriteString("\x1b[22m")
	}
	if removed.has(ModifierCrossedOut) {
		_, _ = a.w.WriteString("\x1b[29m")
	}
	if removed.has(ModifierSlowBlink) || removed.has(ModifierRapidBlink) {
		_, _ = a.w.WriteString("\x1b[25m")
	}
	if removed.has(ModifierHidden) {
		_, _ = a.w.WriteString("\x1b[28m")
	}
	if added.has(ModifierReversed) {
		_, _ = a.w.WriteString("\x1b[7m")
	}
	if added.has(ModifierBold) {
		_, _ = a.w.WriteString("\x1b[1m")
	}
	if added.has(ModifierItalic) {
		_, _ = a.w.WriteString("\x1b[3m")
	}
	if added.has(ModifierDim) {
		_, _ = a.w.WriteString("\x1b[2m")
	}
	if added.has(ModifierCrossedOut) {
		_, _ = a.w.WriteString("\x1b[9m")
	}
	if added.has(ModifierSlowBlink) {
		_, _ = a.w.WriteString("\x1b[5m")
	}
	if added.has(ModifierRapidBlink) {
		_, _ = a.w.WriteString("\x1b[6m")
	}
	if added.has(ModifierHidden) {
		_, _ = a.w.WriteString("\x1b[8m")
	}
	a.modifier = m
}

func (a *ansiEmitter) emitColors(fg, bg Color) {
	if fg != a.fg {
		emitFgColor(a.w, fg)
		a.fg = fg
	}
	if bg != a.bg {
		emitBgColor(a.w, bg)
		a.bg = bg
	}
}

func (a *ansiEmitter) emitUnderline(uc Color, us UnderlineStyle) {
	if uc != a.ulColor {
		emitUlColor(a.w, uc)
		a.ulColor = uc
	}
	if us == a.ulStyle {
		return
	}
	switch us {
	case UnderlineReset:
		_, _ = a.w.WriteString("\x1b[24m")
	case UnderlineLine:
		_, _ = a.w.WriteString("\x1b[4m")
	case UnderlineCurl:
		_, _ = a.w.WriteString("\x1b[4:3m")
	case UnderlineDotted:
		_, _ = a.w.WriteString("\x1b[4:4m")
	case UnderlineDashed:
		_, _ = a.w.WriteString("\x1b[4:5m")
	case UnderlineDoubleLine:
		_, _ = a.w.WriteString("\x1b[21m")
	}
	a.ulStyle = us
}

var (
	fgNamedEsc = [colorWhite + 1]string{
		colorReset:        "\x1b[39m",
		colorBlack:        "\x1b[30m",
		colorRed:          "\x1b[31m",
		colorGreen:        "\x1b[32m",
		colorYellow:       "\x1b[33m",
		colorBlue:         "\x1b[34m",
		colorMagenta:      "\x1b[35m",
		colorCyan:         "\x1b[36m",
		colorGray:         "\x1b[90m",
		colorLightRed:     "\x1b[91m",
		colorLightGreen:   "\x1b[92m",
		colorLightYellow:  "\x1b[93m",
		colorLightBlue:    "\x1b[94m",
		colorLightMagenta: "\x1b[95m",
		colorLightCyan:    "\x1b[96m",
		colorLightGray:    "\x1b[37m",
		colorWhite:        "\x1b[97m",
	}
	bgNamedEsc = [colorWhite + 1]string{
		colorReset:        "\x1b[49m",
		colorBlack:        "\x1b[40m",
		colorRed:          "\x1b[41m",
		colorGreen:        "\x1b[42m",
		colorYellow:       "\x1b[43m",
		colorBlue:         "\x1b[44m",
		colorMagenta:      "\x1b[45m",
		colorCyan:         "\x1b[46m",
		colorGray:         "\x1b[100m",
		colorLightRed:     "\x1b[101m",
		colorLightGreen:   "\x1b[102m",
		colorLightYellow:  "\x1b[103m",
		colorLightBlue:    "\x1b[104m",
		colorLightMagenta: "\x1b[105m",
		colorLightCyan:    "\x1b[106m",
		colorLightGray:    "\x1b[47m",
		colorWhite:        "\x1b[107m",
	}
)

func emitFgColor(w *strings.Builder, c Color) {
	emitColorTo(w, c, &fgNamedEsc, "\x1b[38;5;", "\x1b[38;2;")
}

func emitBgColor(w *strings.Builder, c Color) {
	emitColorTo(w, c, &bgNamedEsc, "\x1b[48;5;", "\x1b[48;2;")
}

func emitColorTo(
	w *strings.Builder, c Color, named *[colorWhite + 1]string,
	indexedPfx, rgbPfx string,
) {
	if c.kind <= colorWhite {
		_, _ = w.WriteString(named[c.kind])
		return
	}
	switch c.kind {
	case colorIndexed:
		w.WriteString(indexedPfx)
		writeUint8(w, c.r)
		w.WriteString("m")
	case colorRGB:
		w.WriteString(rgbPfx)
		writeUint8(w, c.r)
		w.WriteByte(';')
		writeUint8(w, c.g)
		w.WriteByte(';')
		writeUint8(w, c.b)
		w.WriteByte('m')
	default:
	}
}

func emitUlColor(w *strings.Builder, c Color) {
	if c.kind == colorReset {
		_, _ = w.WriteString("\x1b[59m")
		return
	}
	switch c.kind {
	case colorIndexed:
		w.WriteString("\x1b[58:5:")
		writeUint8(w, c.r)
		w.WriteByte('m')
	case colorRGB:
		w.WriteString("\x1b[58:2::")
		writeUint8(w, c.r)
		w.WriteByte(':')
		writeUint8(w, c.g)
		w.WriteByte(':')
		writeUint8(w, c.b)
		w.WriteByte('m')
	default:
		_, _ = w.WriteString("\x1b[59m")
	}
}

func emitRow(e *ansiEmitter, row []Cell, style Style) Style {
	for _, c := range row {
		if c.Skip {
			continue
		}
		if c.Style != style {
			e.emitStyle(c.Style)
			style = c.Style
		}
		sym := c.Symbol
		if sym == "" {
			sym = " "
		}
		e.w.WriteString(sym)
	}
	return style
}

func writeUint8(w *strings.Builder, n uint8) {
	if n >= 100 {
		w.WriteByte('0' + n/100)
		n %= 100
		w.WriteByte('0' + n/10)
		w.WriteByte('0' + n%10)
		return
	}
	if n >= 10 {
		w.WriteByte('0' + n/10)
		w.WriteByte('0' + n%10)
		return
	}
	w.WriteByte('0' + n)
}
