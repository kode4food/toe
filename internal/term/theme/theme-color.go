package theme

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kode4food/toe/internal/tui"
)

var underlineStyles = map[string]tui.UnderlineStyle{
	"line":        tui.UnderlineLine,
	"curl":        tui.UnderlineCurl,
	"dashed":      tui.UnderlineDashed,
	"dotted":      tui.UnderlineDotted,
	"double_line": tui.UnderlineDoubleLine,
}

func (p palette) parseColor(value any) (colorSpec, error) {
	s, ok := value.(string)
	if !ok {
		return colorSpec{}, fmt.Errorf("%w: unrecognized value: %v",
			ErrInvalidTheme, value)
	}
	if c, ok := p[s]; ok {
		return c, nil
	}
	return parseRawColor(s)
}

func parseRawColor(s string) (colorSpec, error) {
	if strings.HasPrefix(s, "#") {
		if len(s) == 4 {
			s = "#" + strings.Repeat(s[1:2], 2) +
				strings.Repeat(s[2:3], 2) +
				strings.Repeat(s[3:4], 2)
		}
		if len(s) != 7 {
			return colorSpec{}, fmt.Errorf("%w: malformed RGB: %s",
				ErrInvalidTheme, s)
		}
		n, err := strconv.ParseUint(s[1:], 16, 32)
		if err != nil {
			return colorSpec{}, fmt.Errorf("%w: malformed RGB: %s",
				ErrInvalidTheme, s)
		}
		return colorSpec{
			color: tui.ColorRGB(uint8(n>>16), uint8(n>>8), uint8(n)),
			rgb:   true,
		}, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 255 {
		return colorSpec{}, fmt.Errorf("%w: malformed ANSI: %s",
			ErrInvalidTheme, s)
	}
	return colorSpec{color: tui.ColorANSI(uint8(n))}, nil
}

func (p palette) parseUnderline(
	style tui.Style, value any,
) (tui.Style, error) {
	m, ok := value.(map[string]any)
	if !ok {
		return style, fmt.Errorf("%w: underline must be table", ErrInvalidTheme)
	}
	seen := map[string]bool{}
	for name, value := range m {
		seen[name] = true
		switch name {
		case "color":
			c, err := p.parseColor(value)
			if err != nil {
				return style, err
			}
			style = style.UlColor(c.color)
		case "style":
			u, err := parseUnderlineStyle(value)
			if err != nil {
				return style, err
			}
			style = style.UlStyle(u)
		default:
			return style, fmt.Errorf("%w: invalid underline attribute: %s",
				ErrInvalidTheme, name)
		}
	}
	if seen["color"] && !seen["style"] {
		style = style.UlStyle(tui.UnderlineLine)
	}
	return style, nil
}

func parseUnderlineStyle(value any) (tui.UnderlineStyle, error) {
	s, ok := value.(string)
	if !ok {
		return tui.UnderlineReset,
			fmt.Errorf("%w: invalid underline style: %v",
				ErrInvalidTheme, value)
	}
	if u, ok := underlineStyles[s]; ok {
		return u, nil
	}
	return tui.UnderlineReset,
		fmt.Errorf("%w: invalid underline style: %s", ErrInvalidTheme, s)
}

func parseModifiers(style tui.Style, value any) (tui.Style, error) {
	values, ok := value.([]any)
	if !ok {
		return style, fmt.Errorf("%w: modifiers should be an array",
			ErrInvalidTheme)
	}
	for _, value := range values {
		s, ok := value.(string)
		if !ok {
			return style, fmt.Errorf("%w: invalid modifier: %v",
				ErrInvalidTheme, value)
		}
		switch s {
		case "bold":
			style = style.Mod(tui.ModifierBold)
		case "dim":
			style = style.Mod(tui.ModifierDim)
		case "italic":
			style = style.Mod(tui.ModifierItalic)
		case "slow_blink", "rapid_blink":
			style = style.Mod(tui.ModifierSlowBlink)
		case "underlined":
			style = style.UlStyle(tui.UnderlineLine)
		case "reversed":
			style = style.Mod(tui.ModifierReversed)
		case "hidden":
			continue
		case "crossed_out":
			style = style.Mod(tui.ModifierCrossedOut)
		default:
			return style, fmt.Errorf("%w: invalid modifier: %s",
				ErrInvalidTheme, s)
		}
	}
	return style, nil
}
