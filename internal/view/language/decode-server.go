package language

import "github.com/kode4food/toe/internal/toml"

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
			Args:         toml.StringSlice(m["args"]),
			Environment:  toml.StringMap(m["environment"]),
			Config:       toml.AnyMap(m["config"]),
			Timeout:      toml.IntValue(m, "timeout", 20),
			RootPatterns: toml.StringSlice(m["required-root-patterns"]),
		}, true
	}
	return Server{}, false
}

func decodeLanguageServerNames(value any) []string {
	values, ok := toml.AnySlice(value)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		switch value := value.(type) {
		case string:
			out = append(out, value)
		case map[string]any:
			if name, ok := value["name"].(string); ok {
				out = append(out, name)
			}
		}
	}
	return out
}

func decodeFormatter(value any) (Formatter, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return Formatter{}, false
	}
	if cmd, ok := m["command"].(string); ok {
		return Formatter{
			Command: cmd,
			Args:    toml.StringSlice(m["args"]),
		}, true
	}
	return Formatter{}, false
}
