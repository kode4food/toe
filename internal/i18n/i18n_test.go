package i18n_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

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
}

func TestLocales(t *testing.T) {
	if expected := os.Getenv(testLocaleExpected); expected != "" {
		assert.Equal(t, expected, i18n.Text(i18n.StatusWritten))
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
			expected: "gespeichert",
		},
		{
			name:     "German in Germany",
			locale:   "de_DE.UTF-8",
			expected: "gespeichert",
		},
		{
			name:     "French in Switzerland",
			locale:   "fr_CH.UTF-8",
			expected: "enregistré",
		},
		{
			name:     "French in France",
			locale:   "fr_FR.UTF-8",
			expected: "enregistré",
		},
		{
			name:     "Italian in Switzerland",
			locale:   "it_CH.UTF-8",
			expected: "salvato",
		},
		{
			name:     "Italian in Italy",
			locale:   "it_IT.UTF-8",
			expected: "salvato",
		},
		{
			name:     "English in Britain",
			locale:   "en_GB.UTF-8",
			expected: "written",
		},
		{
			name:     "English in the US",
			locale:   "en_US.UTF-8",
			expected: "written",
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
