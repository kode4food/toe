package theme

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/tui"
)

type Theme struct {
	name          string
	styles        map[string]tui.Style
	scopes        []string
	rainbowLength int
	rgb           bool
}

const defaultThemeName = "mocha"

var (
	ErrMissingSelection = errors.New("missing required ui.selection scope")
	ErrInvalidTheme     = errors.New("invalid theme")
)

// Decode builds a theme from raw TOML, returning warnings for entries it could
// not parse rather than failing
func Decode(data map[string]any) (*Theme, []string) {
	pal, warnings := decodePalette(data["palette"])
	styles := map[string]tui.Style{}
	var scopes []string
	var rgb bool
	rainbow, rainbowRGB, err := pal.parseStyleArray(data["rainbow"])
	if err != nil {
		warnings = append(warnings, err.Error())
		rainbow = defaultRainbow
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

// Load reads an embedded theme by name, resolving its inherits chain
func Load(name string) (*Theme, error) {
	data, err := loader.LoadThemeTOML(name)
	if err != nil {
		return nil, err
	}
	th, _ := Decode(data)
	th.name = name
	return th, nil
}

// Default loads the theme used when none is configured
func Default() (*Theme, error) {
	return Load(defaultThemeName)
}

// Name is the theme's identifier
func (t *Theme) Name() string {
	return t.name
}

// Dimmed returns a copy of the theme with every style darkened by pct,
// for rendering unfocused panes
func (t *Theme) Dimmed(pct int) *Theme {
	if pct >= 100 {
		return t
	}
	styles := make(map[string]tui.Style, len(t.styles))
	for scope, style := range t.styles {
		styles[scope] = style.Darkened(pct)
	}
	dimmed := *t
	dimmed.styles = styles
	return &dimmed
}

// Quantized returns a copy of the theme with every style snapped onto the
// 256-color palette, for terminals without true color
func (t *Theme) Quantized() *Theme {
	styles := make(map[string]tui.Style, len(t.styles))
	for scope, style := range t.styles {
		styles[scope] = style.Quantized()
	}
	quantized := *t
	quantized.styles = styles
	return &quantized
}

// Get resolves a scope, falling back to its dotted ancestors, and yields the
// zero style when nothing matches
func (t *Theme) Get(scope string) tui.Style {
	style, _ := t.TryGet(scope)
	return style
}

// TryGet resolves a scope, falling back to its dotted ancestors
func (t *Theme) TryGet(scope string) (tui.Style, bool) {
	for s := scope; s != ""; {
		if style, ok := t.styles[s]; ok {
			return style, true
		}
		idx := strings.LastIndexByte(s, '.')
		if idx < 0 {
			break
		}
		s = s[:idx]
	}
	return tui.Style{}, false
}

// TryGetExact resolves a scope without ancestor fallback
func (t *Theme) TryGetExact(scope string) (tui.Style, bool) {
	style, ok := t.styles[scope]
	return style, ok
}

// Scopes lists every scope the theme defines
func (t *Theme) Scopes() []string {
	return slices.Clone(t.scopes)
}

// Is16Color reports whether the theme names only palette colors, so it renders
// correctly without true color
func (t *Theme) Is16Color() bool {
	return !t.rgb
}

// RainbowLength is the number of rainbow bracket colors defined
func (t *Theme) RainbowLength() int {
	return t.rainbowLength
}

// Validate reports whether the theme defines the scopes rendering needs
func (t *Theme) Validate() error {
	if _, ok := t.TryGetExact("ui.selection"); !ok {
		return ErrMissingSelection
	}
	return nil
}
