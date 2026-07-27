package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type (
	ansiEmitter struct {
		w        *strings.Builder
		fg, bg   Color
		ulColor  Color
		ulStyle  UnderlineStyle
		modifier Modifier
	}

	colorEsc struct {
		named   namedEsc
		indexed string
		rgb     string
	}

	namedEsc [colorWhite + 1]string
)

const (
	csi           = "\x1b["
	sgrTerminator = 'm'

	sgrBold            = csi + "1m"
	sgrDim             = csi + "2m"
	sgrItalic          = csi + "3m"
	sgrSlowBlink       = csi + "5m"
	sgrRapidBlink      = csi + "6m"
	sgrReversed        = csi + "7m"
	sgrHidden          = csi + "8m"
	sgrCrossedOut      = csi + "9m"
	sgrNoItalic        = csi + "23m"
	sgrNoBlink         = csi + "25m"
	sgrNoReversed      = csi + "27m"
	sgrNoHidden        = csi + "28m"
	sgrNoCrossedOut    = csi + "29m"
	sgrNormalIntensity = csi + "22m" // clears bold and dim together

	sgrForegroundIndexed = csi + "38;5;"
	sgrForegroundRGB     = csi + "38;2;"
	sgrBackgroundIndexed = csi + "48;5;"
	sgrBackgroundRGB     = csi + "48;2;"

	sgrUnderlineIndexed = csi + "58:5:"
	// the trailing separator is the empty color-space field RGB requires
	sgrUnderlineRGB          = csi + "58:2::"
	sgrUnderlineColorDefault = csi + "59m"

	sgrNoUnderline         = csi + "24m"
	sgrUnderlineLine       = csi + "4m"
	sgrUnderlineCurl       = csi + "4:3m"
	sgrUnderlineDotted     = csi + "4:4m"
	sgrUnderlineDashed     = csi + "4:5m"
	sgrUnderlineDoubleLine = csi + "4:2m"
)

var (
	underlineEsc = [UnderlineDoubleLine + 1]string{
		UnderlineReset:      sgrNoUnderline,
		UnderlineLine:       sgrUnderlineLine,
		UnderlineCurl:       sgrUnderlineCurl,
		UnderlineDotted:     sgrUnderlineDotted,
		UnderlineDashed:     sgrUnderlineDashed,
		UnderlineDoubleLine: sgrUnderlineDoubleLine,
	}

	fgColorEsc = colorEsc{
		named:   buildNamedEsc(0),
		indexed: sgrForegroundIndexed,
		rgb:     sgrForegroundRGB,
	}

	bgColorEsc = colorEsc{
		named:   buildNamedEsc(10),
		indexed: sgrBackgroundIndexed,
		rgb:     sgrBackgroundRGB,
	}
)

// RenderToANSI serializes the buffer as rows joined by '\n', emitting style
// escapes only on changes — used to bridge into the string-based render path
func (b *Buffer) RenderToANSI() string {
	if b.Empty() {
		return ""
	}
	var sb strings.Builder
	sb.Grow(max(b.lastANSILen, b.Width*b.Height+max(b.Height-1, 0)))
	e := &ansiEmitter{w: &sb}
	style := Style{}
	for y := range b.Height {
		if y > 0 {
			sb.WriteByte('\n')
		}
		style = emitRow(e, b.cells[y*b.Width:(y+1)*b.Width], style)
	}
	out := sb.String()
	b.lastANSILen = len(out)
	return out
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
		_, _ = a.w.WriteString(sgrNoReversed)
	}
	if removed.has(ModifierBold) {
		_, _ = a.w.WriteString(sgrNormalIntensity)
		if m.has(ModifierDim) {
			_, _ = a.w.WriteString(sgrDim)
		}
	}
	if removed.has(ModifierItalic) {
		_, _ = a.w.WriteString(sgrNoItalic)
	}
	if removed.has(ModifierDim) {
		_, _ = a.w.WriteString(sgrNormalIntensity)
	}
	if removed.has(ModifierCrossedOut) {
		_, _ = a.w.WriteString(sgrNoCrossedOut)
	}
	if removed.has(ModifierSlowBlink) || removed.has(ModifierRapidBlink) {
		_, _ = a.w.WriteString(sgrNoBlink)
	}
	if removed.has(ModifierHidden) {
		_, _ = a.w.WriteString(sgrNoHidden)
	}
	if added.has(ModifierReversed) {
		_, _ = a.w.WriteString(sgrReversed)
	}
	if added.has(ModifierBold) {
		_, _ = a.w.WriteString(sgrBold)
	}
	if added.has(ModifierItalic) {
		_, _ = a.w.WriteString(sgrItalic)
	}
	if added.has(ModifierDim) {
		_, _ = a.w.WriteString(sgrDim)
	}
	if added.has(ModifierCrossedOut) {
		_, _ = a.w.WriteString(sgrCrossedOut)
	}
	if added.has(ModifierSlowBlink) {
		_, _ = a.w.WriteString(sgrSlowBlink)
	}
	if added.has(ModifierRapidBlink) {
		_, _ = a.w.WriteString(sgrRapidBlink)
	}
	if added.has(ModifierHidden) {
		_, _ = a.w.WriteString(sgrHidden)
	}
	a.modifier = m
}

func (a *ansiEmitter) emitColors(fg, bg Color) {
	if fg != a.fg {
		fgColorEsc.emit(a.w, fg)
		a.fg = fg
	}
	if bg != a.bg {
		bgColorEsc.emit(a.w, bg)
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
	if us < UnderlineStyle(len(underlineEsc)) {
		_, _ = a.w.WriteString(underlineEsc[us])
	}
	a.ulStyle = us
}

func (c *colorEsc) emit(w *strings.Builder, col Color) {
	if col.kind <= colorWhite {
		_, _ = w.WriteString(c.named[col.kind])
		return
	}
	switch col.kind {
	case colorIndexed:
		w.WriteString(c.indexed)
		writeUint8(w, col.r)
		w.WriteByte(sgrTerminator)
	case colorRGB:
		w.WriteString(c.rgb)
		writeUint8(w, col.r)
		w.WriteByte(';')
		writeUint8(w, col.g)
		w.WriteByte(';')
		writeUint8(w, col.b)
		w.WriteByte(sgrTerminator)
	default:
	}
}

func emitUlColor(w *strings.Builder, c Color) {
	if c.kind == colorReset {
		_, _ = w.WriteString(sgrUnderlineColorDefault)
		return
	}
	switch c.kind {
	case colorIndexed:
		w.WriteString(sgrUnderlineIndexed)
		writeUint8(w, c.r)
		w.WriteByte(sgrTerminator)
	case colorRGB:
		w.WriteString(sgrUnderlineRGB)
		writeUint8(w, c.r)
		w.WriteByte(':')
		writeUint8(w, c.g)
		w.WriteByte(':')
		writeUint8(w, c.b)
		w.WriteByte(sgrTerminator)
	default:
		_, _ = w.WriteString(sgrUnderlineColorDefault)
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
		e.w.WriteString(c.Symbol)
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

// offset 10 shifts the whole foreground set to its background counterpart
func buildNamedEsc(offset int) namedEsc {
	codes := [colorWhite + 1]ansi.Attr{
		colorReset:        ansi.AttrDefaultForegroundColor,
		colorBlack:        ansi.AttrBlackForegroundColor,
		colorRed:          ansi.AttrRedForegroundColor,
		colorGreen:        ansi.AttrGreenForegroundColor,
		colorYellow:       ansi.AttrYellowForegroundColor,
		colorBlue:         ansi.AttrBlueForegroundColor,
		colorMagenta:      ansi.AttrMagentaForegroundColor,
		colorCyan:         ansi.AttrCyanForegroundColor,
		colorGray:         ansi.AttrBrightBlackForegroundColor,
		colorLightRed:     ansi.AttrBrightRedForegroundColor,
		colorLightGreen:   ansi.AttrBrightGreenForegroundColor,
		colorLightYellow:  ansi.AttrBrightYellowForegroundColor,
		colorLightBlue:    ansi.AttrBrightBlueForegroundColor,
		colorLightMagenta: ansi.AttrBrightMagentaForegroundColor,
		colorLightCyan:    ansi.AttrBrightCyanForegroundColor,
		colorLightGray:    ansi.AttrWhiteForegroundColor,
		colorWhite:        ansi.AttrBrightWhiteForegroundColor,
	}
	var t namedEsc
	for i, c := range codes {
		t[i] = ansi.SGR(c + offset)
	}
	return t
}
