package theme

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

type (
	palette map[string]colorSpec

	colorSpec struct {
		color color.Color
		rgb   bool
	}
)

var defaultRainbow = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
}

func decodePalette(value any) (palette, []string) {
	p := basePalette()
	m, ok := value.(map[string]any)
	if !ok {
		return p, nil
	}
	next := basePalette()
	for name, value := range m {
		s, ok := value.(string)
		if !ok {
			return p, []string{fmt.Sprintf("invalid palette color %q", name)}
		}
		c, err := parseRawColor(s)
		if err != nil {
			return p, []string{
				fmt.Sprintf("invalid palette color %q: %v", name, err),
			}
		}
		next[name] = c
	}
	return next, nil
}

func basePalette() palette {
	return palette{
		"default":       {color: lipgloss.NoColor{}},
		"black":         {color: lipgloss.Color("0")},
		"red":           {color: lipgloss.Color("1")},
		"green":         {color: lipgloss.Color("2")},
		"yellow":        {color: lipgloss.Color("3")},
		"blue":          {color: lipgloss.Color("4")},
		"magenta":       {color: lipgloss.Color("5")},
		"cyan":          {color: lipgloss.Color("6")},
		"gray":          {color: lipgloss.Color("8")},
		"light-red":     {color: lipgloss.Color("9")},
		"light-green":   {color: lipgloss.Color("10")},
		"light-yellow":  {color: lipgloss.Color("11")},
		"light-blue":    {color: lipgloss.Color("12")},
		"light-magenta": {color: lipgloss.Color("13")},
		"light-cyan":    {color: lipgloss.Color("14")},
		"light-gray":    {color: lipgloss.Color("7")},
		"white":         {color: lipgloss.Color("15")},
	}
}

func (p palette) parseStyle(value any) (lipgloss.Style, bool, error) {
	style := lipgloss.NewStyle()
	m, ok := value.(map[string]any)
	if !ok {
		c, err := p.parseColor(value)
		if err != nil {
			return style, false, err
		}
		return style.Foreground(c.color), c.rgb, nil
	}
	var rgb bool
	for name, value := range m {
		switch name {
		case "fg":
			c, err := p.parseColor(value)
			if err != nil {
				return style, rgb, err
			}
			rgb = rgb || c.rgb
			style = style.Foreground(c.color)
		case "bg":
			c, err := p.parseColor(value)
			if err != nil {
				return style, rgb, err
			}
			rgb = rgb || c.rgb
			style = style.Background(c.color)
		case "underline":
			next, err := p.parseUnderline(style, value)
			if err != nil {
				return style, rgb, err
			}
			style = next
		case "modifiers":
			next, err := parseModifiers(style, value)
			if err != nil {
				return style, rgb, err
			}
			style = next
		default:
			return style, rgb, fmt.Errorf("invalid style attribute: %s", name)
		}
	}
	return style, rgb, nil
}

func (p palette) parseStyleArray(value any) ([]lipgloss.Style, bool, error) {
	if value == nil {
		return defaultRainbow, false, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, false, fmt.Errorf(
			"%w: could not parse value as an array: %v", ErrInvalidTheme, value)
	}
	styles := make([]lipgloss.Style, 0, len(values))
	var rgb bool
	for _, value := range values {
		style, styleRGB, err := p.parseStyle(value)
		if err != nil {
			return nil, rgb, err
		}
		rgb = rgb || styleRGB
		styles = append(styles, style)
	}
	return styles, rgb, nil
}
