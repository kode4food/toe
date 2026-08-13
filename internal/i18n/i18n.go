package i18n

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"regexp"
	"strings"

	"github.com/kode4food/toe/internal/locale"
)

type (
	// Key identifies a localized message
	Key string

	// Vars supplies named values for message interpolation
	Vars map[string]any

	// Translations contains messages resolved for the current locale
	Translations map[Key]string

	// Error identifies a localizable error and its interpolation variables
	Error struct {
		key  Key
		vars Vars
	}
)

const (
	pluralSeparator = "#"
	pluralZero      = "zero"
	pluralOne       = "one"
	pluralOther     = "other"
)

var (
	ErrMissingPluralForm = errors.New(`plural message has no "other" form`)
)

var (
	//go:embed translations/*.json
	translationFS embed.FS

	// localeRE extracts the locale from a translation file's basename, as in
	// the "de" of "i18n/config.de.json"
	localeRE = regexp.MustCompile(`(?:^|[/.])([^./]+)\.json$`)

	messages = LoadTranslations(translationFS)
)

// NewError returns an error backed by a localized message
func NewError[S ~string](key S) *Error {
	return &Error{key: Key(key)}
}

// LoadTranslations loads and resolves every locale file in files. Each embedded
// set holds exactly one module's translations, so the whole tree is taken
// rather than a caller-supplied glob
func LoadTranslations(files fs.FS) Translations {
	names, err := jsonFiles(files)
	if err != nil {
		panic(err)
	}
	if len(names) == 0 {
		panic(errors.New("no translations found"))
	}
	res := Translations{}
	loaded := map[locale.Locale]Translations{}
	for _, name := range names {
		data, err := fs.ReadFile(files, name)
		if err != nil {
			panic(err)
		}
		tr, err := parseTranslations(data)
		if err != nil {
			panic(fmt.Errorf("load translation %q: %w", name, err))
		}
		name = localeRE.FindStringSubmatch(name)[1]
		if name == "common" {
			maps.Copy(res, tr)
			continue
		}
		loaded[locale.Locale(name)] = tr
	}
	locales := locale.Environment()
	for i := len(locales) - 1; i >= 0; i-- {
		maps.Copy(res, loaded[locales[i]])
	}
	return res
}

// Register adds startup module translations
func Register(values Translations) {
	maps.Copy(messages, values)
}

// Text returns a localized message with optional named interpolation
func Text(key Key, vars ...Vars) string {
	text, _ := messages.text(key, vars...)
	return text
}

// ErrorText returns a localized error message
func ErrorText(err error) string {
	message := err.Error()
	if localized, ok := errors.AsType[*Error](err); ok {
		if text, ok := messages.text(localized.key, localized.vars); ok {
			message = text
		}
	}
	return Text(ErrorMessage, Vars{"message": message})
}

// WithVars returns a copy with interpolation variables attached
func (e *Error) WithVars(vars Vars) *Error {
	return &Error{key: e.key, vars: vars}
}

// Error returns the translated message for the current locale
func (e *Error) Error() string {
	return string(e.key)
}

func (t Translations) text(key Key, vars ...Vars) (string, bool) {
	text, ok := t.lookup(key, vars...)
	if !ok {
		return string(key), false
	}
	if len(vars) == 0 || len(vars[0]) == 0 {
		return text, true
	}
	pairs := make([]string, 0, 2*len(vars[0]))
	for k, v := range vars[0] {
		pairs = append(pairs, "{"+k+"}", fmt.Sprint(v))
	}
	return strings.NewReplacer(pairs...).Replace(text), true
}

func (t Translations) lookup(key Key, vars ...Vars) (string, bool) {
	if count, ok := pluralCount(vars); ok {
		if text, ok := t[pluralKey(key, pluralCategory(count))]; ok {
			return text, true
		}
		if text, ok := t[pluralKey(key, pluralOther)]; ok {
			return text, true
		}
	}
	text, ok := t[key]
	return text, ok
}

func pluralCategory(n int) string {
	switch n {
	case 0:
		return pluralZero
	case 1:
		return pluralOne
	}
	return pluralOther
}

func pluralCount(vars []Vars) (int, bool) {
	if len(vars) == 0 {
		return 0, false
	}
	count, ok := vars[0]["count"].(int)
	return count, ok
}

func jsonFiles(files fs.FS) ([]string, error) {
	var res []string
	err := fs.WalkDir(files, ".", func(
		name string, d fs.DirEntry, err error,
	) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(name, ".json") {
			res = append(res, name)
		}
		return nil
	})
	return res, err
}

func parseTranslations(data []byte) (Translations, error) {
	raw := map[Key]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := Translations{}
	for key, value := range raw {
		var text string
		if err := json.Unmarshal(value, &text); err == nil {
			out[key] = text
			continue
		}
		forms := map[string]string{}
		if err := json.Unmarshal(value, &forms); err != nil {
			return nil, err
		}
		if _, ok := forms[pluralOther]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrMissingPluralForm, key)
		}
		for category, text := range forms {
			out[pluralKey(key, category)] = text
		}
	}
	return out, nil
}

func pluralKey(key Key, category string) Key {
	return key + pluralSeparator + Key(category)
}
