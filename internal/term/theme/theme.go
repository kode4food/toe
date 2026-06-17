package theme

import (
	"errors"
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/loader"
)

type (
	Theme struct {
		name          string
		styles        map[string]lipgloss.Style
		scopes        []string
		rainbowLength int
		rgb           bool
	}

	palette map[string]colorSpec

	colorSpec struct {
		color color.Color
		rgb   bool
	}
)

var (
	ErrMissingSelection = errors.New("missing required ui.selection scope")
	ErrInvalidTheme     = errors.New("invalid theme")
)

func Decode(data map[string]any) (*Theme, []string) {
	pal, warnings := decodePalette(data["palette"])
	styles := map[string]lipgloss.Style{}
	var scopes []string
	var rgb bool
	rainbow, rainbowRGB, err := pal.parseStyleArray(data["rainbow"])
	if err != nil {
		warnings = append(warnings, err.Error())
		rainbow = defaultRainbow()
		rainbowRGB = false
	}
	rgb = rainbowRGB
	for i, style := range rainbow {
		name := fmt.Sprintf("rainbow.%d", i)
		styles[name] = style
		scopes = append(scopes, name)
	}
	for name, value := range data {
		if name == "palette" || name == "inherits" || name == "rainbow" {
			continue
		}
		style, styleRGB, err := pal.parseStyle(value)
		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("failed to parse style for key %q: %v", name, err),
			)
		}
		rgb = rgb || styleRGB
		styles[name] = style
		scopes = append(scopes, name)
	}
	return &Theme{
		styles:        styles,
		scopes:        scopes,
		rainbowLength: len(rainbow),
		rgb:           rgb,
	}, warnings
}

func Load(name string) (*Theme, []string, error) {
	data, err := loader.LoadThemeTOML(name)
	if err != nil {
		return nil, nil, err
	}
	th, warnings := Decode(data)
	th.name = name
	if err := th.Validate(); err != nil {
		return nil, warnings, err
	}
	return th, warnings, nil
}

func Default() (*Theme, []string, error) {
	return Load("mocha")
}

func (t *Theme) Name() string {
	return t.name
}

func (t *Theme) Get(scope string) lipgloss.Style {
	style, _ := t.TryGet(scope)
	return style
}

func (t *Theme) TryGet(scope string) (lipgloss.Style, bool) {
	for s := scope; s != ""; {
		style, ok := t.styles[s]
		if ok {
			return style, true
		}
		idx := strings.LastIndexByte(s, '.')
		if idx < 0 {
			break
		}
		s = s[:idx]
	}
	return lipgloss.Style{}, false
}

func (t *Theme) TryGetExact(scope string) (lipgloss.Style, bool) {
	style, ok := t.styles[scope]
	return style, ok
}

func (t *Theme) Scopes() []string {
	return slices.Clone(t.scopes)
}

func (t *Theme) Is16Color() bool {
	return !t.rgb
}

func (t *Theme) RainbowLength() int {
	return t.rainbowLength
}

func (t *Theme) Validate() error {
	if _, ok := t.TryGetExact("ui.selection"); !ok {
		return ErrMissingSelection
	}
	return nil
}

func decodePalette(value any) (palette, []string) {
	p := defaultPalette()
	m, ok := value.(map[string]any)
	if !ok {
		return p, nil
	}
	next := defaultPalette()
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

func defaultPalette() palette {
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
		return defaultRainbow(), false, nil
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

func defaultRainbow() []lipgloss.Style {
	return []lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
	}
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
	switch s {
	case "line":
		return lipgloss.UnderlineSingle, nil
	case "curl":
		return lipgloss.UnderlineCurly, nil
	case "dashed":
		return lipgloss.UnderlineDashed, nil
	case "dotted":
		return lipgloss.UnderlineDotted, nil
	case "double_line":
		return lipgloss.UnderlineDouble, nil
	default:
		return lipgloss.UnderlineNone,
			fmt.Errorf("%w: invalid underline style: %s", ErrInvalidTheme, s)
	}
}

func parseModifiers(
	style lipgloss.Style, value any,
) (lipgloss.Style, error) {
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
