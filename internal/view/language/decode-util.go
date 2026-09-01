package language

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/toml"
)

// settingValueArgs is a value that a language definition may override, falling
// back to the editor-wide settingValueArgs when the language leaves it unset
type settingValueArgs[T any] struct {
	lang   *T
	editor *T
}

func decodeBlockCommentTokens(value any) []core.BlockCommentToken {
	if token, ok := decodeBlockCommentToken(value); ok {
		return []core.BlockCommentToken{token}
	}
	values, ok := toml.AnySlice(value)
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
	if end, ok := m["end"].(string); ok {
		return core.BlockCommentToken{Start: start, End: end}, true
	}
	return core.BlockCommentToken{}, false
}

func settingValue[T any](s settingValueArgs[T], fallback T) T {
	if s.lang != nil {
		return *s.lang
	}
	if s.editor != nil {
		return *s.editor
	}
	return fallback
}
