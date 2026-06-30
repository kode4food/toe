package loader

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	ErrThemeNotFound = errors.New("theme not found")
	ErrThemeCycle    = errors.New("theme inheritance cycle")
)

var (
	//go:embed assets/themes
	embeddedThemes embed.FS

	supportedThemeNames = slices.Sorted(slices.Values([]string{
		"frappe",
		"latte",
		"macchiato",
		"mocha",
	}))
)

func ThemeNames() []string {
	return slices.Clone(supportedThemeNames)
}

func LoadThemeTOML(name string) (map[string]any, error) {
	return loadThemeTOML(name, map[string]bool{})
}

func loadThemeTOML(name string, seen map[string]bool) (map[string]any, error) {
	path, err := themeFileForLoad(name, seen)
	if err != nil {
		return nil, err
	}
	var data []byte
	if embPath, ok := strings.CutPrefix(path, "embed:"); ok {
		data, err = embeddedThemes.ReadFile(embPath)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	theme, err := decodeThemeTOML(string(data))
	if err != nil {
		return nil, err
	}
	return resolveInherits(theme, name, seen)
}

func resolveInherits(
	theme map[string]any, _ string, seen map[string]bool,
) (map[string]any, error) {
	parent, ok := theme["inherits"].(string)
	if !ok {
		return theme, nil
	}
	parentTheme, err := loadThemeTOML(parent, seen)
	if err != nil {
		return nil, err
	}
	return mergeThemes(parentTheme, theme), nil
}

func themeFileForLoad(name string, seen map[string]bool) (string, error) {
	if !supportedThemeName(name) {
		return "", fmt.Errorf("%w: %s", ErrThemeNotFound, name)
	}
	embPath := "assets/themes/" + name + ".toml"
	if _, err := embeddedThemes.Open(embPath); err == nil {
		key := "embed:" + embPath
		if seen[key] {
			return "", fmt.Errorf("%w: %s", ErrThemeCycle, name)
		}
		seen[key] = true
		return "embed:" + embPath, nil
	}
	return "", fmt.Errorf("%w: %s", ErrThemeNotFound, name)
}

func mergeThemes(parent, theme map[string]any) map[string]any {
	palette := mergeThemePalette(parent["palette"], theme["palette"])
	merged, ok := MergeTOMLValues(parent, theme, 1).(map[string]any)
	if !ok {
		return theme
	}
	merged["palette"] = palette
	return merged
}

func mergeThemePalette(parent, theme any) any {
	switch {
	case parent != nil && theme != nil:
		return MergeTOMLValues(parent, theme, 2)
	case parent != nil:
		return parent
	case theme != nil:
		return theme
	default:
		return map[string]any{}
	}
}

func supportedThemeName(name string) bool {
	_, found := slices.BinarySearch(supportedThemeNames, name)
	return found
}

func decodeThemeTOML(text string) (map[string]any, error) {
	var theme map[string]any
	if _, err := toml.Decode(text, &theme); err != nil {
		return nil, err
	}
	return theme, nil
}
