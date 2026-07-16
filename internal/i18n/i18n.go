package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kode4food/toe/internal/locale"
)

type (
	// Key identifies a localized message
	Key string

	// Vars supplies named values for message interpolation
	Vars map[string]any

	catalog      []translations
	translations map[Key]string
)

//go:embed translations/*.json
var translationFiles embed.FS

// Catalog data is loaded once from embedded files and never mutated
var (
	commonCatalog  = catalog{commonTranslations}
	defaultCatalog = resolve(locale.Environment()...)
	enUSCatalog    = resolve(locale.EnUS, locale.En)

	commonTranslations = loadTranslation("translations/common.json")
	localeTranslations = loadTranslations()
)

// Text returns a localized message with optional named interpolation
func Text(key Key, vars ...Vars) string {
	return defaultCatalog.text(key, vars...)
}

// CommonText returns a locale-independent message
func CommonText(key Key, vars ...Vars) string {
	return commonCatalog.text(key, vars...)
}

// EnglishText returns an English message
func EnglishText(key Key, vars ...Vars) string {
	return enUSCatalog.text(key, vars...)
}

func (c catalog) text(key Key, vars ...Vars) string {
	var text string
	found := false
	for _, tr := range c {
		if next, ok := tr[key]; ok {
			text = next
			found = true
			break
		}
	}
	if !found {
		return string(key)
	}
	if len(vars) == 0 || len(vars[0]) == 0 {
		return string(text)
	}
	pairs := make([]string, 0, 2*len(vars[0]))
	for k, v := range vars[0] {
		pairs = append(pairs, "{"+k+"}", fmt.Sprint(v))
	}
	return strings.NewReplacer(pairs...).Replace(string(text))
}

func resolve(locales ...locale.Locale) catalog {
	res := make(catalog, 0, len(locales)+1)
	for _, loc := range locales {
		if tr, ok := localeTranslations[loc]; ok {
			res = append(res, tr)
		}
	}
	return append(res, commonTranslations)
}

func loadTranslations() map[locale.Locale]translations {
	res := map[locale.Locale]translations{}
	entries, err := translationFiles.ReadDir("translations")
	if err != nil {
		return res
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || name == "common.json" ||
			!strings.HasSuffix(name, ".json") {
			continue
		}
		loc := locale.Locale(strings.TrimSuffix(name, ".json"))
		res[loc] = loadTranslation("translations/" + name)
	}
	return res
}

func loadTranslation(name string) translations {
	res := translations{}
	data, err := translationFiles.ReadFile(name)
	if err != nil {
		return res
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return translations{}
	}
	return res
}
