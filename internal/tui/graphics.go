package tui

import (
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

func ColorRGB(r, g, b uint8) Color {
	return Color{kind: colorRGB, r: r, g: g, b: b}
}

// ImageColor encodes a 24-bit image id as a terminal colour
func ImageColor(id uint32) Color {
	return ColorRGB(uint8(id>>16), uint8(id>>8), uint8(id))
}

// PlaceholderSymbol builds the cell content for image row and column
func PlaceholderSymbol(at geom.Point) string {
	return string([]rune{
		kitty.Placeholder, kitty.Diacritic(at.Y), kitty.Diacritic(at.X),
	})
}

func (s Style) Fg(c Color) Style {
	s.fg = c
	return s
}

func (s Style) Bg(c Color) Style {
	s.bg = c
	return s
}

func (s Style) UlColor(c Color) Style {
	s.underlineColor = c
	return s
}

func (s Style) UlStyle(u UnderlineStyle) Style {
	s.underlineStyle = u
	return s
}

func (s Style) Mod(m Modifier) Style {
	s.modifier |= m
	return s
}

func (s Style) FgColor() Color {
	return s.fg
}

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

func (s Style) HasMod(m Modifier) bool {
	return s.modifier&m == m
}

func (c Color) IsReset() bool {
	return c.kind == colorReset
}

func (m Modifier) has(bit Modifier) bool {
	return m&bit != 0
}
