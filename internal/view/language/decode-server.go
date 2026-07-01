package language

func decodeLanguageServers(value any) map[string]Server {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]Server, len(m))
	for name, value := range m {
		cfg, ok := decodeLanguageServer(value)
		if ok {
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
	cmd, ok := m["command"].(string)
	if !ok {
		return Server{}, false
	}
	return Server{
		Command:              cmd,
		Args:                 decodeStringSlice(m["args"]),
		Environment:          decodeStringMap(m["environment"]),
		Config:               decodeAnyMap(m["config"]),
		Timeout:              intValueFromMap(m, "timeout", 20),
		RequiredRootPatterns: decodeStringSlice(m["required-root-patterns"]),
	}, true
}

func decodeLanguageServerFeatures(value any) []ServerFeatures {
	values, ok := anySlice(value)
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

func decodeLanguageServerFeature(
	value any,
) (ServerFeatures, bool) {
	switch v := value.(type) {
	case string:
		return ServerFeatures{Name: v}, true
	case map[string]any:
		name, ok := v["name"].(string)
		if !ok {
			return ServerFeatures{}, false
		}
		return ServerFeatures{Name: name}, true
	default:
		return ServerFeatures{}, false
	}
}

func decodeFormatter(value any) (Formatter, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return Formatter{}, false
	}
	cmd, ok := m["command"].(string)
	if !ok {
		return Formatter{}, false
	}
	return Formatter{
		Command: cmd,
		Args:    decodeStringSlice(m["args"]),
	}, true
}
