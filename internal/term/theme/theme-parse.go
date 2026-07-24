package theme

import (
	"fmt"

	"github.com/kode4food/toe/internal/tui"
)

type (
	palette map[string]colorSpec

	colorSpec struct {
		color tui.Color
		rgb   bool
	}
)

var defaultRainbow = []tui.Style{
	tui.Style{}.Fg(tui.ColorANSI(1)),
	tui.Style{}.Fg(tui.ColorANSI(3)),
	tui.Style{}.Fg(tui.ColorANSI(2)),
	tui.Style{}.Fg(tui.ColorANSI(4)),
	tui.Style{}.Fg(tui.ColorANSI(6)),
	tui.Style{}.Fg(tui.ColorANSI(5)),
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
		"default":       {color: tui.ColorReset},
		"black":         {color: tui.ColorANSI(0)},
		"red":           {color: tui.ColorANSI(1)},
		"green":         {color: tui.ColorANSI(2)},
		"yellow":        {color: tui.ColorANSI(3)},
		"blue":          {color: tui.ColorANSI(4)},
		"magenta":       {color: tui.ColorANSI(5)},
		"cyan":          {color: tui.ColorANSI(6)},
		"gray":          {color: tui.ColorANSI(8)},
		"light-red":     {color: tui.ColorANSI(9)},
		"light-green":   {color: tui.ColorANSI(10)},
		"light-yellow":  {color: tui.ColorANSI(11)},
		"light-blue":    {color: tui.ColorANSI(12)},
		"light-magenta": {color: tui.ColorANSI(13)},
		"light-cyan":    {color: tui.ColorANSI(14)},
		"light-gray":    {color: tui.ColorANSI(7)},
		"white":         {color: tui.ColorANSI(15)},
	}
}

func (p palette) parseStyle(value any) (tui.Style, bool, error) {
	style := tui.Style{}
	m, ok := value.(map[string]any)
	if !ok {
		c, err := p.parseColor(value)
		if err != nil {
			return style, false, err
		}
		return style.Fg(c.color), c.rgb, nil
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
			style = style.Fg(c.color)
		case "bg":
			c, err := p.parseColor(value)
			if err != nil {
				return style, rgb, err
			}
			rgb = rgb || c.rgb
			style = style.Bg(c.color)
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

func (p palette) parseStyleArray(value any) ([]tui.Style, bool, error) {
	if value == nil {
		return defaultRainbow, false, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, false, fmt.Errorf(
			"%w: could not parse value as an array: %v", ErrInvalidTheme, value)
	}
	styles := make([]tui.Style, 0, len(values))
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
