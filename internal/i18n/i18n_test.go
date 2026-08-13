package i18n_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/i18n"
)

const testLocaleExpected = "TOE_TEST_LOCALE_EXPECTED"

func TestText(t *testing.T) {
	t.Run("interpolates named values", func(t *testing.T) {
		assert.Contains(t,
			i18n.Text(i18n.ErrorMessage, i18n.Vars{
				"message": "broken",
			}), "broken")
	})

	t.Run("returns missing key", func(t *testing.T) {
		key := i18n.Key("missing.message")
		assert.Equal(t, "missing.message", i18n.Text(key))
	})

	t.Run("formats translated error", func(t *testing.T) {
		assert.Equal(t, "error: no document",
			i18n.ErrorText(i18n.NewError(i18n.ErrorNoDocument)),
		)
		err := i18n.NewError(i18n.ErrorNoSuchCommand)
		withVars := err.WithVars(i18n.Vars{"name": "nope"})
		assert.NotSame(t, err, withVars)
		assert.Equal(t, "error: no such command: `nope`",
			i18n.ErrorText(withVars),
		)
		assert.Equal(t, "error: no such command: `nope`",
			i18n.ErrorText(fmt.Errorf("%w: context", withVars)),
		)
	})

	t.Run("formats literal error", func(t *testing.T) {
		assert.Equal(t, "error: nope",
			i18n.ErrorText(errors.New("nope")),
		)
	})
}

func TestModuleTranslations(t *testing.T) {
	key := i18n.Key("test-module.docstring")
	files := fstest.MapFS{
		"i18n/common.json": {
			Data: []byte(`{"test-module.docstring":"common"}`),
		},
		"i18n/test.en.json": {
			Data: []byte(`{"test-module.docstring":"translated"}`),
		},
	}
	tr := i18n.LoadTranslations(files)

	i18n.Register(tr)
	assert.Equal(t, "translated", i18n.Text(key))
}

func TestPlurals(t *testing.T) {
	key := i18n.Key("plural.selections")
	files := fstest.MapFS{
		"i18n/plural.en.json": {
			Data: []byte(`{"plural.selections":{` +
				`"zero":"no selections",` +
				`"one":"{count} selection","other":"{count} selections"},` +
				`"plural.other-only":{"other":"{count} rows"}}`),
		},
	}
	i18n.Register(i18n.LoadTranslations(files))

	t.Run("one for a single item", func(t *testing.T) {
		assert.Equal(t, "1 selection",
			i18n.Text(key, i18n.Vars{"count": 1}),
		)
	})

	t.Run("other for several", func(t *testing.T) {
		assert.Equal(t, "3 selections",
			i18n.Text(key, i18n.Vars{"count": 3}),
		)
	})

	t.Run("zero for none", func(t *testing.T) {
		assert.Equal(t, "no selections",
			i18n.Text(key, i18n.Vars{"count": 0}),
		)
	})

	t.Run("falls back to other without a zero", func(t *testing.T) {
		assert.Equal(t, "0 rows",
			i18n.Text("plural.other-only", i18n.Vars{"count": 0}),
		)
	})

	t.Run("falls back to other", func(t *testing.T) {
		assert.Equal(t, "1 rows",
			i18n.Text("plural.other-only", i18n.Vars{"count": 1}),
		)
	})

	t.Run("plural key without a count is missing", func(t *testing.T) {
		assert.Equal(t, "plural.selections", i18n.Text(key))
	})

	t.Run("rejects a malformed message", func(t *testing.T) {
		bad := fstest.MapFS{
			"i18n/bad.en.json": {Data: []byte(`{"plural.bad":[1,2]}`)},
		}
		assert.Panics(t, func() { i18n.LoadTranslations(bad) })
	})

	t.Run("rejects plural forms without other", func(t *testing.T) {
		bad := fstest.MapFS{
			"i18n/bad.en.json": {
				Data: []byte(`{"plural.bad":{"one":"{count} row"}}`),
			},
		}
		assert.PanicsWithError(t,
			`load translation "i18n/bad.en.json": `+
				`plural message has no "other" form: plural.bad`,
			func() { i18n.LoadTranslations(bad) },
		)
	})
}

func TestPluralLocales(t *testing.T) {
	key := i18n.Key("plural.copied")
	files := fstest.MapFS{
		"i18n/plural.en.json": {
			Data: []byte(`{"plural.copied":{` +
				`"one":"copied {count} line","other":"copied {count} lines"}}`),
		},
		"i18n/plural.fr.json": {
			Data: []byte(`{"plural.copied":{` +
				`"zero":"{count} ligne copiée",` +
				`"one":"{count} ligne copiée",` +
				`"other":"{count} lignes copiées"}}`),
		},
	}
	if expected := os.Getenv(testLocaleExpected); expected != "" {
		i18n.Register(i18n.LoadTranslations(files))
		assert.Equal(t, expected, i18n.Text(key, i18n.Vars{"count": 0}))
		return
	}

	tests := []struct {
		name     string
		locale   string
		expected string
	}{
		{
			name:     "a zero form for the locale",
			locale:   "fr_FR.UTF-8",
			expected: "0 ligne copiée",
		},
		{
			name:     "no zero form falls back",
			locale:   "en_US.UTF-8",
			expected: "copied 0 lines",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestPluralLocales$")
			cmd.Env = append(os.Environ(),
				"LC_ALL="+tc.locale,
				testLocaleExpected+"="+tc.expected,
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Log(string(out))
			}
			assert.NoError(t, err)
		})
	}
}

func TestLocales(t *testing.T) {
	if expected := os.Getenv(testLocaleExpected); expected != "" {
		assert.Equal(t, expected, i18n.Text(i18n.ErrorNoDocument))
		assert.Equal(t, ":", i18n.Text(i18n.PromptCommand))
		return
	}

	tests := []struct {
		name     string
		locale   string
		expected string
	}{
		{
			name:     "German in Switzerland",
			locale:   "de_CH.UTF-8",
			expected: "kein Dokument",
		},
		{
			name:     "German in Germany",
			locale:   "de_DE.UTF-8",
			expected: "kein Dokument",
		},
		{
			name:     "French in Switzerland",
			locale:   "fr_CH.UTF-8",
			expected: "aucun document",
		},
		{
			name:     "French in France",
			locale:   "fr_FR.UTF-8",
			expected: "aucun document",
		},
		{
			name:     "Italian in Switzerland",
			locale:   "it_CH.UTF-8",
			expected: "nessun documento",
		},
		{
			name:     "Italian in Italy",
			locale:   "it_IT.UTF-8",
			expected: "nessun documento",
		},
		{
			name:     "English in Britain",
			locale:   "en_GB.UTF-8",
			expected: "no document",
		},
		{
			name:     "English in the US",
			locale:   "en_US.UTF-8",
			expected: "no document",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestLocales$")
			cmd.Env = append(os.Environ(),
				"LC_ALL="+tc.locale,
				testLocaleExpected+"="+tc.expected,
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Log(string(out))
			}
			assert.NoError(t, err)
		})
	}
}

func TestTranslationFiles(t *testing.T) {
	en := readTranslations(t, "translations/en.json")
	entries, err := os.ReadDir("translations")
	assert.NoError(t, err)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || name == "common.json" ||
			name == "en.json" || !strings.HasSuffix(name, ".json") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			tr := readTranslations(t, "translations/"+name)
			regional := strings.Contains(
				strings.TrimSuffix(name, ".json"), "-",
			)
			if !regional {
				assert.Len(t, tr, len(en))
			}
			for key, value := range tr {
				fallback, ok := en[key]
				assert.True(t, ok)
				assert.Equal(t,
					placeholders(fallback), placeholders(value),
				)
			}
		})
	}
}

func readTranslations(t *testing.T, name string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(name)
	assert.NoError(t, err)
	res := map[string]string{}
	err = json.Unmarshal(data, &res)
	assert.NoError(t, err)
	return res
}

func placeholders(s string) []string {
	var res []string
	for {
		from := strings.IndexByte(s, '{')
		if from < 0 {
			return res
		}
		to := strings.IndexByte(s[from:], '}')
		if to < 0 {
			return res
		}
		to += from
		res = append(res, s[from:to+1])
		s = s[to+1:]
	}
}
