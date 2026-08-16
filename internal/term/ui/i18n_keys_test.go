package ui_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/i18n"
)

var uiKeyRE = regexp.MustCompile(`i18n\.Key = "([^"]+)"`)

func TestPackageKeys(t *testing.T) {
	// a key whose text lives in another package's module resolves only once
	// that module registers, which leaves this package's messages untranslated
	t.Run("resolve without other modules", func(t *testing.T) {
		for _, key := range packageKeys(t) {
			t.Run(key, func(t *testing.T) {
				assert.NotEqual(t, key, i18n.Text(i18n.Key(key)))
			})
		}
	})
}

func packageKeys(t *testing.T) []string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	assert.True(t, ok)
	dir := filepath.Dir(file)
	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)

	var out []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		assert.NoError(t, err)
		for _, m := range uiKeyRE.FindAllStringSubmatch(string(data), -1) {
			out = append(out, m[1])
		}
	}
	return out
}
