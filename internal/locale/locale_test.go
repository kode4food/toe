package locale_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/locale"
)

func TestEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		all      string
		messages string
		lang     string
		expected []locale.Locale
	}{
		{
			name:     "LC_ALL takes precedence",
			all:      "de_CH.UTF-8",
			messages: "fr_CH",
			lang:     "it_CH",
			expected: []locale.Locale{
				"de-CH", "de", locale.EnUS, locale.En,
			},
		},
		{
			name:     "LC_MESSAGES before LANG",
			messages: "fr_CH",
			lang:     "it_CH",
			expected: []locale.Locale{
				"fr-CH", "fr", locale.EnUS, locale.En,
			},
		},
		{
			name: "LANG fallback",
			lang: "it_CH@euro",
			expected: []locale.Locale{
				"it-CH", "it", locale.EnUS, locale.En,
			},
		},
		{
			name: "strips English region",
			lang: "en_GB.UTF-8",
			expected: []locale.Locale{
				"en-GB", locale.En, locale.EnUS,
			},
		},
		{
			name:     "US English uses baked default",
			lang:     "en_US.UTF-8",
			expected: []locale.Locale{locale.EnUS, locale.En},
		},
		{
			name:     "C uses English",
			lang:     "C.UTF-8",
			expected: []locale.Locale{locale.EnUS, locale.En},
		},
		{
			name:     "missing uses English",
			expected: []locale.Locale{locale.EnUS, locale.En},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LC_ALL", tc.all)
			t.Setenv("LC_MESSAGES", tc.messages)
			t.Setenv("LANG", tc.lang)
			assert.Equal(t, tc.expected, locale.Environment())
		})
	}
}
