package language

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
)

func decodeStringOrSlice(value any) []string {
	if s, ok := value.(string); ok {
		return []string{s}
	}
	return decodeStringSlice(value)
}

func decodeBlockCommentTokens(value any) []core.BlockCommentToken {
	if token, ok := decodeBlockCommentToken(value); ok {
		return []core.BlockCommentToken{token}
	}
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]core.BlockCommentToken, 0, len(values))
	for _, value := range values {
		if token, ok := decodeBlockCommentToken(value); ok {
			out = append(out, token)
		}
	}
	return out
}

func decodeBlockCommentToken(value any) (core.BlockCommentToken, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return core.BlockCommentToken{}, false
	}
	start, ok := m["start"].(string)
	if !ok {
		return core.BlockCommentToken{}, false
	}
	end, ok := m["end"].(string)
	if !ok {
		return core.BlockCommentToken{}, false
	}
	return core.BlockCommentToken{Start: start, End: end}, true
}

func decodeStringSlice(value any) []string {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func anySlice(value any) ([]any, bool) {
	switch values := value.(type) {
	case []any:
		return values, true
	case []map[string]any:
		out := make([]any, len(values))
		for i, value := range values {
			out[i] = value
		}
		return out, true
	case []string:
		out := make([]any, len(values))
		for i, value := range values {
			out[i] = value
		}
		return out, true
	default:
		return nil, false
	}
}

func decodeStringMap(value any) map[string]string {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, value := range m {
		if s, ok := value.(string); ok {
			out[k] = s
		}
	}
	return out
}

func decodeAnyMap(value any) map[string]any {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

// Low-level helpers

func boolValue(lang, editor *bool, fallback bool) bool {
	if lang != nil {
		return *lang
	}
	if editor != nil {
		return *editor
	}
	return fallback
}

func intValue(lang, editor *int, fallback int) int {
	if lang != nil {
		return *lang
	}
	if editor != nil {
		return *editor
	}
	return fallback
}

func stringValue(lang, editor *string, fallback string) string {
	if lang != nil {
		return *lang
	}
	if editor != nil {
		return *editor
	}
	return fallback
}

func stringValueFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func intValueFromMap(m map[string]any, key string, fallback int) int {
	if n, ok := loader.IntPtr(m[key]); ok {
		return *n
	}
	return fallback
}
