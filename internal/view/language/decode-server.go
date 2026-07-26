package language

import "github.com/kode4food/toe/internal/loader"

func decodeLanguageServers(value any) map[string]Server {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]Server, len(m))
	for name, value := range m {
		if cfg, ok := decodeLanguageServer(value); ok {
			out[name] = cfg
		}
	}
	return out
}

func decodeLanguageServer(value any) (Server, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return Server{}, false
	}
	if cmd, ok := m["command"].(string); ok {
		return Server{
			Command:      cmd,
			Args:         decodeStringSlice(m["args"]),
			Environment:  decodeStringMap(m["environment"]),
			Config:       decodeAnyMap(m["config"]),
			Timeout:      intValueFromMap(m, "timeout", 20),
			RootPatterns: decodeStringSlice(m["required-root-patterns"]),
		}, true
	}
	return Server{}, false
}

func decodeLanguageServerFeatures(value any) []ServerFeatures {
	values, ok := loader.AnySlice(value)
	if !ok {
		return nil
	}
	out := make([]ServerFeatures, 0, len(values))
	for _, value := range values {
		if features, ok := decodeLanguageServerFeature(value); ok {
			out = append(out, features)
		}
	}
	return out
}

func decodeLanguageServerFeature(value any) (ServerFeatures, bool) {
	switch v := value.(type) {
	case string:
		return ServerFeatures{Name: v}, true
	case map[string]any:
		if name, ok := v["name"].(string); ok {
			return ServerFeatures{Name: name}, true
		}
		return ServerFeatures{}, false
	default:
		return ServerFeatures{}, false
	}
}

func decodeFormatter(value any) (Formatter, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return Formatter{}, false
	}
	if cmd, ok := m["command"].(string); ok {
		return Formatter{
			Command: cmd,
			Args:    decodeStringSlice(m["args"]),
		}, true
	}
	return Formatter{}, false
}
