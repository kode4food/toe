package theme

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

var underlineStyles = map[string]lipgloss.Underline{
	"line":        lipgloss.UnderlineSingle,
	"curl":        lipgloss.UnderlineCurly,
	"dashed":      lipgloss.UnderlineDashed,
	"dotted":      lipgloss.UnderlineDotted,
	"double_line": lipgloss.UnderlineDouble,
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
		if _, err := strconv.ParseUint(s[1:], 16, 32); err != nil {
			return colorSpec{}, fmt.Errorf("%w: malformed RGB: %s",
				ErrInvalidTheme, s)
		}
		return colorSpec{color: lipgloss.Color(s), rgb: true}, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 255 {
		return colorSpec{}, fmt.Errorf("%w: malformed ANSI: %s",
			ErrInvalidTheme, s)
	}
	return colorSpec{color: lipgloss.Color(s)}, nil
}

func (p palette) parseUnderline(
	style lipgloss.Style, value any,
) (lipgloss.Style, error) {
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
			style = style.UnderlineColor(c.color)
		case "style":
			u, err := parseUnderlineStyle(value)
			if err != nil {
				return style, err
			}
			style = style.UnderlineStyle(u)
		default:
			return style, fmt.Errorf("%w: invalid underline attribute: %s",
				ErrInvalidTheme, name)
		}
	}
	if seen["color"] && !seen["style"] {
		style = style.Underline(true)
	}
	return style, nil
}

func parseUnderlineStyle(value any) (lipgloss.Underline, error) {
	s, ok := value.(string)
	if !ok {
		return lipgloss.UnderlineNone,
			fmt.Errorf("%w: invalid underline style: %v",
				ErrInvalidTheme, value)
	}
	if u, ok := underlineStyles[s]; ok {
		return u, nil
	}
	return lipgloss.UnderlineNone,
		fmt.Errorf("%w: invalid underline style: %s", ErrInvalidTheme, s)
}

func parseModifiers(style lipgloss.Style, value any) (lipgloss.Style, error) {
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
			style = style.Bold(true)
		case "dim":
			style = style.Faint(true)
		case "italic":
			style = style.Italic(true)
		case "slow_blink", "rapid_blink":
			style = style.Blink(true)
		case "underlined":
			style = style.Underline(true)
		case "reversed":
			style = style.Reverse(true)
		case "hidden":
			continue
		case "crossed_out":
			style = style.Strikethrough(true)
		default:
			return style, fmt.Errorf("%w: invalid modifier: %s",
				ErrInvalidTheme, s)
		}
	}
	return style, nil
}
