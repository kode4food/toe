package i18n

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kode4food/toe/internal/locale"
)

type (
	// Key identifies a localized message
	Key string

	// Vars supplies named values for message interpolation
	Vars map[string]any

	// Error identifies a localizable error and its interpolation variables
	Error struct {
		key  Key
		vars Vars
	}

	catalog      []translations
	translations map[Key]string
)

//go:embed translations/*.json
var translationFiles embed.FS

// Catalog data is loaded once from embedded files and never mutated
var (
	defaultCatalog = resolve(locale.Environment()...)

	commonTranslations = loadTranslation("translations/common.json")
	localeTranslations = loadTranslations()
)

// NewError returns an error backed by a localized message
func NewError[S ~string](key S) *Error {
	return &Error{key: Key(key)}
}

// Text returns a localized message with optional named interpolation
func Text(key Key, vars ...Vars) string {
	return defaultCatalog.text(key, vars...)
}

// ErrorText returns a localized error message
func ErrorText(err error) string {
	key := Key(err.Error())
	message := err.Error()
	var vars Vars
	if localized, ok := errors.AsType[*Error](err); ok {
		key = localized.key
		vars = localized.vars
	}
	for _, tr := range defaultCatalog {
		if _, ok := tr[key]; ok {
			message = Text(key, vars)
			break
		}
	}
	return Text(ErrorMessage, Vars{"message": message})
}

// WithVars returns a copy with interpolation variables attached
func (e *Error) WithVars(vars Vars) *Error {
	return &Error{key: e.key, vars: vars}
}

func (e *Error) Error() string {
	return string(e.key)
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
		return text
	}
	pairs := make([]string, 0, 2*len(vars[0]))
	for k, v := range vars[0] {
		pairs = append(pairs, "{"+k+"}", fmt.Sprint(v))
	}
	return strings.NewReplacer(pairs...).Replace(text)
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
