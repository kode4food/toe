package toml

import (
	"maps"
	"slices"
)

// Overlay is a base value and the value layered on top of it
type Overlay[T any] struct {
	Base T
	Over T
}

// MergeValues overlays Over onto Base, merging maps only while depth remains.
// Below that Over replaces Base outright
func MergeValues(values Overlay[any], depth int) any {
	right := values.Over
	switch l := values.Base.(type) {
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
				out[key] = MergeValues(Overlay[any]{
					Base: lv,
					Over: rv,
				}, depth-1)
				continue
			}
			out[key] = rv
		}
		return out
	case []map[string]any:
		if r, ok := AnySlice(right); ok {
			return mergeArrays(Overlay[[]any]{
				Base: mapSliceToAny(l),
				Over: r,
			}, depth)
		}
		return right
	case []any:
		if r, ok := AnySlice(right); ok {
			return mergeArrays(Overlay[[]any]{Base: l, Over: r}, depth)
		}
		return right
	default:
		return right
	}
}

// LoadMergedWithBase merges TOML files onto an already decoded base map
func LoadMergedWithBase(
	base map[string]any, paths []string, depth int,
) (map[string]any, bool) {
	merged := any(base)
	loaded := base != nil
	for _, path := range paths {
		var next map[string]any
		if err := DecodeFile(path, &next); err != nil {
			continue
		}
		if !loaded {
			merged = next
			loaded = true
			continue
		}
		merged = MergeValues(Overlay[any]{
			Base: merged,
			Over: next,
		}, depth)
	}
	if !loaded {
		return nil, false
	}
	out, ok := merged.(map[string]any)
	return out, ok
}

// LoadMerged overlays each readable path onto the previous ones
func LoadMerged(paths []string, depth int) (map[string]any, bool) {
	return LoadMergedWithBase(nil, paths, depth)
}

func mergeArrays(values Overlay[[]any], depth int) []any {
	if depth <= 0 {
		return values.Over
	}
	out := slices.Clone(values.Base)
	for _, rv := range values.Over {
		idx := -1
		if name, ok := valueName(rv); ok {
			idx = namedValueIndex(out, name)
		}
		if idx >= 0 {
			lv := out[idx]
			out = slices.Delete(out, idx, idx+1)
			out = append(out, MergeValues(Overlay[any]{
				Base: lv,
				Over: rv,
			}, depth-1))
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

func namedValueIndex(values []any, name string) int {
	for i, value := range values {
		if n, ok := valueName(value); ok && n == name {
			return i
		}
	}
	return -1
}

func valueName(value any) (string, bool) {
	if m, ok := value.(map[string]any); ok {
		name, ok := m["name"].(string)
		return name, ok
	}
	return "", false
}
