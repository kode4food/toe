package loader

import (
	"embed"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	ErrThemeNotFound = errors.New("theme not found")
	ErrThemeCycle    = errors.New("theme inheritance cycle")
)

//go:embed assets/themes
var themeFS embed.FS

// ThemeNames lists the embedded themes, sorted
func ThemeNames() []string {
	entries, err := themeFS.ReadDir("assets/themes")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(name, ".toml") {
			names = append(names, strings.TrimSuffix(name, ".toml"))
		}
	}
	slices.Sort(names)
	return names
}

// LoadThemeTOML reads an embedded theme, resolving its inherits chain
func LoadThemeTOML(name string) (map[string]any, error) {
	return loadThemeTOML(name, map[string]bool{})
}

func loadThemeTOML(name string, seen map[string]bool) (map[string]any, error) {
	if seen[name] {
		return nil, fmt.Errorf("%w: %s", ErrThemeCycle, name)
	}
	seen[name] = true
	data, err := themeFS.ReadFile("assets/themes/" + name + ".toml")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrThemeNotFound, name)
	}
	theme, err := decodeThemeTOML(string(data))
	if err != nil {
		return nil, err
	}
	return resolveInherits(theme, seen)
}

func resolveInherits(
	theme map[string]any, seen map[string]bool,
) (map[string]any, error) {
	parent, ok := theme["inherits"].(string)
	if !ok {
		return theme, nil
	}
	parentTheme, err := loadThemeTOML(parent, seen)
	if err != nil {
		return nil, err
	}
	return mergeThemes(Overlay[map[string]any]{
		Base: parentTheme,
		Over: theme,
	}), nil
}

func mergeThemes(themes Overlay[map[string]any]) map[string]any {
	parent := themes.Base
	theme := themes.Over
	palette := mergeThemePalette(Overlay[any]{
		Base: parent["palette"],
		Over: theme["palette"],
	})
	merged, ok := MergeTOMLValues(Overlay[any]{
		Base: parent,
		Over: theme,
	}, 1).(map[string]any)
	if !ok {
		return theme
	}
	merged["palette"] = palette
	return merged
}

func mergeThemePalette(palettes Overlay[any]) any {
	parent := palettes.Base
	theme := palettes.Over
	switch {
	case parent != nil && theme != nil:
		return MergeTOMLValues(palettes, 2)
	case parent != nil:
		return parent
	case theme != nil:
		return theme
	default:
		return map[string]any{}
	}
}

func decodeThemeTOML(text string) (map[string]any, error) {
	var theme map[string]any
	if _, err := toml.Decode(text, &theme); err != nil {
		return nil, err
	}
	return theme, nil
}
