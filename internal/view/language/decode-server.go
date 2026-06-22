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
		return ServerFeatures{
			Name:     name,
			Only:     decodeLanguageServerFeatureNames(v["only-features"]),
			Excluded: decodeLanguageServerFeatureNames(v["except-features"]),
		}, true
	default:
		return ServerFeatures{}, false
	}
}

func decodeLanguageServerFeatureNames(value any) []ServerFeature {
	names := decodeStringSlice(value)
	out := make([]ServerFeature, len(names))
	for i, name := range names {
		out[i] = ServerFeature(name)
	}
	return out
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

func decodeDebugAdapter(value any) (DebugAdapter, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return DebugAdapter{}, false
	}
	name, ok := m["name"].(string)
	if !ok {
		return DebugAdapter{}, false
	}
	transport, ok := m["transport"].(string)
	if !ok {
		return DebugAdapter{}, false
	}
	return DebugAdapter{
		Name:      name,
		Transport: transport,
		Command:   stringValueFromMap(m, "command"),
		Args:      decodeStringSlice(m["args"]),
		PortArg:   stringValueFromMap(m, "port-arg"),
		Templates: decodeDebugTemplates(m["templates"]),
		Quirks:    decodeDebuggerQuirks(m["quirks"]),
	}, true
}

func decodeDebugTemplates(value any) []DebugTemplate {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]DebugTemplate, 0, len(values))
	for _, value := range values {
		m, ok := value.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, DebugTemplate{
			Name:       stringValueFromMap(m, "name"),
			Request:    stringValueFromMap(m, "request"),
			Completion: decodeDebugCompletions(m["completion"]),
			Args:       decodeAnyMap(m["args"]),
		})
	}
	return out
}

func decodeDebugCompletions(value any) []DebugCompletion {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]DebugCompletion, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case string:
			out = append(out, DebugCompletion{Name: v})
		case map[string]any:
			out = append(out, DebugCompletion{
				Name:       stringValueFromMap(v, "name"),
				Completion: stringValueFromMap(v, "completion"),
				Default:    stringValueFromMap(v, "default"),
			})
		}
	}
	return out
}

func decodeDebuggerQuirks(value any) DebuggerQuirks {
	m, ok := value.(map[string]any)
	if !ok {
		return DebuggerQuirks{}
	}
	return DebuggerQuirks{AbsolutePaths: boolValueFromMap(m, "absolute-paths")}
}
