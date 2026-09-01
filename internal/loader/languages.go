package loader

import (
	_ "embed"
	"sync"

	"github.com/kode4food/toe/internal/toml"
)

//go:embed assets/languages.toml
var defaultLanguageAssets string

var decodeDefaults = sync.OnceValues(func() (map[string]any, bool) {
	var out map[string]any
	if err := toml.Decode(defaultLanguageAssets, &out); err != nil {
		return nil, false
	}
	return out, true
})

// DefaultLanguages returns the cached bundled language defaults. Do not mutate
func DefaultLanguages() (map[string]any, bool) {
	return decodeDefaults()
}
