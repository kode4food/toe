package loader

import (
	_ "embed"
	"maps"
	"slices"
	"sync"

	"github.com/BurntSushi/toml"
)

//go:embed assets/languages.toml
var defaultLanguagesTOML string

var defaultLanguages = sync.OnceValues(func() (map[string]any, bool) {
	var out map[string]any
	if _, err := toml.Decode(defaultLanguagesTOML, &out); err != nil {
		return nil, false
	}
	return out, true
})

// LoadDefaultLanguagesTOML returns the cached bundled defaults; do not mutate
func LoadDefaultLanguagesTOML() (map[string]any, bool) {
	return defaultLanguages()
}

func MergeTOMLValues(left, right any, depth int) any {
	switch l := left.(type) {
	case map[string]any:
		r, ok := right.(map[string]any)
		if !ok {
			return right
		}
		if depth <= 0 {
			return r
		}
		out := maps.Clone(l)
		for key, rv := range r {
			if lv, ok := out[key]; ok {
				out[key] = MergeTOMLValues(lv, rv, depth-1)
				continue
			}
			out[key] = rv
		}
		return out
	case []map[string]any:
		r, ok := AnySlice(right)
		if !ok {
			return right
		}
		return mergeTOMLArrays(mapSliceToAny(l), r, depth)
	case []any:
		r, ok := AnySlice(right)
		if !ok {
			return right
		}
		return mergeTOMLArrays(l, r, depth)
	default:
		return right
	}
}

// LoadMergedTOMLWithBase merges TOML files onto an already decoded base map
func LoadMergedTOMLWithBase(
	base map[string]any, paths []string, depth int,
) (map[string]any, bool) {
	merged := any(base)
	loaded := base != nil
	for _, path := range paths {
		var next map[string]any
		if _, err := toml.DecodeFile(path, &next); err != nil {
			continue
		}
		if !loaded {
			merged = next
			loaded = true
			continue
		}
		merged = MergeTOMLValues(merged, next, depth)
	}
	if !loaded {
		return nil, false
	}
	out, ok := merged.(map[string]any)
	return out, ok
}

func LoadMergedTOML(paths []string, depth int) (map[string]any, bool) {
	return LoadMergedTOMLWithBase(nil, paths, depth)
}

// BoolPtr converts a TOML any value to *bool, returning nil for non-bool
func BoolPtr(value any) *bool {
	v, ok := value.(bool)
	if !ok {
		return nil
	}
	return &v
}

// IntPtr converts a TOML any value to (*int, bool)
func IntPtr(value any) (*int, bool) {
	switch v := value.(type) {
	case int:
		return &v, true
	case int64:
		return new(int(v)), true
	default:
		return nil, false
	}
}

// IntPtrOrNil converts a TOML any value to *int, returning nil for non-int
func IntPtrOrNil(value any) *int {
	v, _ := IntPtr(value)
	return v
}

// StringPtr converts a TOML any value to *string, returning nil for non-string
func StringPtr(value any) *string {
	v, ok := value.(string)
	if !ok {
		return nil
	}
	return &v
}

func mergeTOMLArrays(left, right []any, depth int) []any {
	if depth <= 0 {
		return right
	}
	out := slices.Clone(left)
	for _, rv := range right {
		name, ok := valueName(rv)
		idx := -1
		if ok {
			idx = namedValueIndex(out, name)
		}
		if idx >= 0 {
			lv := out[idx]
			out = slices.Delete(out, idx, idx+1)
			out = append(out, MergeTOMLValues(lv, rv, depth-1))
			continue
		}
		out = append(out, rv)
	}
	return out
}

func mapSliceToAny(values []map[string]any) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

// AnySlice coerces common TOML slice types to []any
func AnySlice(value any) ([]any, bool) {
	switch v := value.(type) {
	case []any:
		return v, true
	case []map[string]any:
		return mapSliceToAny(v), true
	case []string:
		out := make([]any, len(v))
		for i, s := range v {
			out[i] = s
		}
		return out, true
	default:
		return nil, false
	}
}

func namedValueIndex(values []any, name string) int {
	for i, value := range values {
		if n, ok := valueName(value); ok && n == name {
			return i
		}
	}
	return -1
}

func valueName(value any) (string, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return "", false
	}
	name, ok := m["name"].(string)
	return name, ok
}
