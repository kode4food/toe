package toml

// BoolPtr converts a TOML any value to *bool, returning nil for non-bool
func BoolPtr(value any) *bool {
	if v, ok := value.(bool); ok {
		return &v
	}
	return nil
}

// IntPtr converts a TOML any value to *int, returning nil for non-int
func IntPtr(value any) *int {
	switch v := value.(type) {
	case int:
		return &v
	case int64:
		return new(int(v))
	default:
		return nil
	}
}

// StringPtr converts a TOML any value to *string, returning nil for non-string
func StringPtr(value any) *string {
	if v, ok := value.(string); ok {
		return &v
	}
	return nil
}

// StringValue returns the string at key, or the empty string when the key is
// missing or holds another type
func StringValue(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// IntValue returns the int at key, falling back when the key is missing or
// holds another type
func IntValue(m map[string]any, key string, fallback int) int {
	if n := IntPtr(m[key]); n != nil {
		return *n
	}
	return fallback
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

// StringOrSlice accepts either a bare string or a slice of them
func StringOrSlice(value any) []string {
	if s, ok := value.(string); ok {
		return []string{s}
	}
	return StringSlice(value)
}

// StringSlice keeps the string elements of a TOML slice, dropping the rest
func StringSlice(value any) []string {
	values, ok := AnySlice(value)
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

// StringMap keeps the string-valued entries of a TOML table, dropping the rest
func StringMap(value any) map[string]string {
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

// AnyMap returns value as a TOML table, or nil when it is not one
func AnyMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return nil
}
