package locale

import (
	"os"
	"slices"
	"strings"
)

// Locale identifies a language and optional regional variant
type Locale string

const (
	// En identifies English
	En Locale = "en"

	// EnUS identifies US English
	EnUS Locale = "en-US"
)

// Environment returns the locale fallback chain selected by the environment
func Environment() []Locale {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); value != "" {
			return parse(value)
		}
	}
	return []Locale{EnUS, En}
}

func parse(value string) []Locale {
	value = strings.SplitN(value, ".", 2)[0]
	value = strings.SplitN(value, "@", 2)[0]
	if value == "" || strings.EqualFold(value, "C") ||
		strings.EqualFold(value, "POSIX") {
		return []Locale{EnUS, En}
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(parts) == 0 || parts[0] == "" {
		return []Locale{EnUS, En}
	}
	lang := strings.ToLower(parts[0])
	out := []Locale{Locale(lang)}
	if len(parts) > 1 && parts[1] != "" {
		region := strings.ToUpper(parts[1])
		out = []Locale{Locale(lang + "-" + region), Locale(lang)}
	}
	for _, loc := range []Locale{EnUS, En} {
		if !slices.Contains(out, loc) {
			out = append(out, loc)
		}
	}
	return out
}
