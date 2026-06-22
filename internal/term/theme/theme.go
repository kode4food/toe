package theme

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/loader"
)

type Theme struct {
	name          string
	styles        map[string]lipgloss.Style
	scopes        []string
	rainbowLength int
	rgb           bool
}

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
