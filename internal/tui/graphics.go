package tui

import (
	"math"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"

	"github.com/kode4food/toe/internal/geom"
)

type (
	Style struct {
		fg, bg, underlineColor Color
		underlineStyle         UnderlineStyle
		modifier               Modifier
	}

	Color struct {
		kind    colorKind
		r, g, b uint8
	}

	UnderlineStyle uint8
	Modifier       uint16
	colorKind      uint8
)

const (
	UnderlineReset UnderlineStyle = iota
	UnderlineLine
	UnderlineCurl
	UnderlineDotted
	UnderlineDashed
	UnderlineDoubleLine
)

// PlaceholderRune is kitty's Unicode image placeholder
const PlaceholderRune = kitty.Placeholder

const (
	ModifierBold       Modifier = 0b0000_0000_0001
	ModifierDim        Modifier = 0b0000_0000_0010
	ModifierItalic     Modifier = 0b0000_0000_0100
	ModifierSlowBlink  Modifier = 0b0000_0001_0000
	ModifierRapidBlink Modifier = 0b0000_0010_0000
	ModifierReversed   Modifier = 0b0000_0100_0000
	ModifierHidden     Modifier = 0b0000_1000_0000
	ModifierCrossedOut Modifier = 0b0001_0000_0000
)

// indices below 16 are terminal-configurable, so their nominal values are not
// reliable match targets
const (
	paletteFirst = 16
	paletteSize  = 256
)

const (
	colorReset colorKind = iota
	colorBlack
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorGray
	colorLightRed
	colorLightGreen
	colorLightYellow
	colorLightBlue
	colorLightMagenta
	colorLightCyan
	colorLightGray
	colorWhite
	colorIndexed
	colorRGB
)

var (
	ColorReset        = Color{kind: colorReset}
	ColorBlack        = Color{kind: colorBlack}
	ColorRed          = Color{kind: colorRed}
	ColorGreen        = Color{kind: colorGreen}
	ColorYellow       = Color{kind: colorYellow}
	ColorBlue         = Color{kind: colorBlue}
	ColorMagenta      = Color{kind: colorMagenta}
	ColorCyan         = Color{kind: colorCyan}
	ColorGray         = Color{kind: colorGray}
	ColorLightRed     = Color{kind: colorLightRed}
	ColorLightGreen   = Color{kind: colorLightGreen}
	ColorLightYellow  = Color{kind: colorLightYellow}
	ColorLightBlue    = Color{kind: colorLightBlue}
	ColorLightMagenta = Color{kind: colorLightMagenta}
	ColorLightCyan    = Color{kind: colorLightCyan}
	ColorLightGray    = Color{kind: colorLightGray}
	ColorWhite        = Color{kind: colorWhite}

	ansiColors = [...]Color{
		ColorBlack, ColorRed, ColorGreen, ColorYellow,
		ColorBlue, ColorMagenta, ColorCyan, ColorLightGray,
		ColorGray, ColorLightRed, ColorLightGreen, ColorLightYellow,
		ColorLightBlue, ColorLightMagenta, ColorLightCyan, ColorWhite,
	}
)

// ColorIndexed returns a color naming a 256-color palette entry
func ColorIndexed(idx uint8) Color {
	return Color{kind: colorIndexed, r: idx}
}

// ColorANSI returns the terminal color for an ANSI palette index
func ColorANSI(idx uint8) Color {
	if idx < 16 {
		return ansiColors[idx]
	}
	return ColorIndexed(idx)
}

// ColorRGB returns a 24-bit color
func ColorRGB(r, g, b uint8) Color {
	return Color{kind: colorRGB, r: r, g: g, b: b}
}

// ImageColor encodes a 24-bit image id as a terminal color
func ImageColor(id uint32) Color {
	return ColorRGB(uint8(id>>16), uint8(id>>8), uint8(id))
}

// PlaceholderSymbol builds the cell content for image row and column
func PlaceholderSymbol(at geom.Point) string {
	return string([]rune{
		kitty.Placeholder, kitty.Diacritic(at.Y), kitty.Diacritic(at.X),
	})
}

// RGBA returns the color's red, green, blue, and alpha values
func (c Color) RGBA() (uint32, uint32, uint32, uint32) {
	switch c.kind {
	case colorReset:
		return 0, 0, 0, 0
	case colorIndexed:
		return ansi.IndexedColor(c.r).RGBA()
	case colorRGB:
		return uint32(c.r) * 0x101, uint32(c.g) * 0x101,
			uint32(c.b) * 0x101, 0xffff
	default:
		return ansi.BasicColor(c.kind - 1).RGBA()
	}
}

// Darkened scales the color toward black by pct/100 (pct<100 darkens). Reset
// stays reset; named and indexed colors resolve to rgb first
func (c Color) Darkened(pct int) Color {
	if c.IsReset() {
		return c
	}
	r, g, b, _ := c.RGBA()
	scale := func(v uint32) uint8 {
		return uint8(v >> 8 * uint32(pct) / 100)
	}
	return ColorRGB(scale(r), scale(g), scale(b))
}

// Quantized returns the nearest 256-color palette entry, by true distance
// rather than the per-channel rounding a terminal library would apply
func (c Color) Quantized() Color {
	if c.kind != colorRGB {
		return c
	}
	r, g, b, _ := c.RGBA()
	best := paletteFirst
	bestDist := math.MaxInt
	for i := paletteFirst; i < paletteSize; i++ {
		pr, pg, pb, _ := ansi.IndexedColor(i).RGBA()
		dr := int(r>>8) - int(pr>>8)
		dg := int(g>>8) - int(pg>>8)
		db := int(b>>8) - int(pb>>8)
		if d := dr*dr + dg*dg + db*db; d < bestDist {
			best = i
			bestDist = d
		}
	}
	return ColorIndexed(uint8(best))
}

// Fg returns a copy with the foreground set
func (s Style) Fg(c Color) Style {
	s.fg = c
	return s
}

// Bg returns a copy with the background set
func (s Style) Bg(c Color) Style {
	s.bg = c
	return s
}

// UlColor returns a copy with the underline color set
func (s Style) UlColor(c Color) Style {
	s.underlineColor = c
	return s
}

// UlStyle returns a copy with the underline variant set
func (s Style) UlStyle(u UnderlineStyle) Style {
	s.underlineStyle = u
	return s
}

// Mod returns a copy with the given modifier bits added
func (s Style) Mod(m Modifier) Style {
	s.modifier |= m
	return s
}

// Darkened darkens the foreground, background, and underline colors so an
// unfocused pane recedes
func (s Style) Darkened(pct int) Style {
	s.fg = s.fg.Darkened(pct)
	s.bg = s.bg.Darkened(pct)
	s.underlineColor = s.underlineColor.Darkened(pct)
	return s
}

// Quantized snaps the foreground, background, and underline colors onto the
// 256-color palette
func (s Style) Quantized() Style {
	s.fg = s.fg.Quantized()
	s.bg = s.bg.Quantized()
	s.underlineColor = s.underlineColor.Quantized()
	return s
}

// FgColor returns the style foreground color
func (s Style) FgColor() Color {
	return s.fg
}

// BgColor returns the style background color
func (s Style) BgColor() Color {
	return s.bg
}

// UnderlineColor returns the style underline color
func (s Style) UnderlineColor() Color {
	return s.underlineColor
}

// UnderlineStyle returns the style underline variant
func (s Style) UnderlineStyle() UnderlineStyle {
	return s.underlineStyle
}

// Modifier returns the style modifier bits
func (s Style) Modifier() Modifier {
	return s.modifier
}

// HasMod reports whether every bit in m is set
func (s Style) HasMod(m Modifier) bool {
	return s.modifier&m == m
}

// IsReset reports whether the color defers to the terminal default
func (c Color) IsReset() bool {
	return c.kind == colorReset
}

func (m Modifier) has(bit Modifier) bool {
	return m&bit != 0
}
