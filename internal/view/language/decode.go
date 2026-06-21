package language

import "github.com/kode4food/toe/internal/core"

func decodeLanguagesMap(m map[string]any) (Languages, bool) {
	values, ok := languageValues(m["language"])
	if !ok {
		return Languages{}, false
	}
	langs := Languages{
		Languages:        make([]Language, 0, len(values)),
		LanguageServers:  decodeLanguageServers(m["language-server"]),
		GrammarSelection: decodeGrammarSelection(m["use-grammars"]),
		Grammars:         decodeGrammars(m["grammar"]),
	}
	for _, value := range values {
		l, ok := decodeLanguage(value)
		if !ok {
			return Languages{}, false
		}
		langs.Languages = append(langs.Languages, l)
	}
	return langs, true
}

func decodeGrammarSelection(value any) GrammarSelection {
	m, ok := value.(map[string]any)
	if !ok {
		return GrammarSelection{}
	}
	return GrammarSelection{
		Only:   decodeStringSlice(m["only"]),
		Except: decodeStringSlice(m["except"]),
	}
}

func decodeGrammars(value any) []Grammar {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]Grammar, 0, len(values))
	for _, value := range values {
		m, ok := value.(map[string]any)
		if !ok {
			continue
		}
		source, ok := decodeGrammarSource(m["source"])
		if !ok {
			continue
		}
		out = append(out, Grammar{
			Name:   stringValueFromMap(m, "name"),
			Source: source,
		})
	}
	return out
}

func decodeGrammarSource(value any) (GrammarSource, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return GrammarSource{}, false
	}
	if path, ok := m["path"].(string); ok {
		return GrammarSource{Path: path}, true
	}
	git, ok := m["git"].(string)
	if !ok {
		return GrammarSource{}, false
	}
	return GrammarSource{
		Git:     git,
		Rev:     stringValueFromMap(m, "rev"),
		Subpath: stringValueFromMap(m, "subpath"),
	}, true
}

func languageValues(value any) ([]map[string]any, bool) {
	switch values := value.(type) {
	case []map[string]any:
		return values, true
	case []any:
		out := make([]map[string]any, 0, len(values))
		for _, value := range values {
			m, ok := value.(map[string]any)
			if !ok {
				return nil, false
			}
			out = append(out, m)
		}
		return out, true
	default:
		return nil, false
	}
}

func decodeLanguage(m map[string]any) (Language, bool) {
	var l Language
	if name, ok := m["name"].(string); ok {
		l.Name = name
	}
	if id, ok := m["language-id"].(string); ok {
		l.LanguageID = id
	}
	if scope, ok := m["scope"].(string); ok {
		l.Scope = scope
	}
	if injection, ok := m["injection-regex"].(string); ok {
		l.InjectionRegex = injection
	}
	if n, ok := intPtr(m["text-width"]); ok {
		l.TextWidth = n
	}
	l.FileTypes = decodeFileTypes(m["file-types"])
	l.Shebangs = decodeStringSlice(m["shebangs"])
	l.Roots = decodeStringSlice(m["roots"])
	l.LanguageServers = decodeLanguageServerFeatures(m["language-servers"])
	l.CommentTokens = decodeCommentTokens(m)
	l.BlockCommentTokens = decodeBlockCommentTokens(m["block-comment-tokens"])
	if indent, ok := m["indent"].(map[string]any); ok {
		l.Indent = decodeIndent(indent)
	}
	if pairs, ok := decodeAutoPairConfig(m["auto-pairs"]); ok {
		l.AutoPairs = pairs
	}
	l.AutoFormat = boolPtr(m["auto-format"])
	if formatter, ok := decodeFormatter(m["formatter"]); ok {
		l.Formatter = &formatter
	}
	if debug, ok := decodeDebugAdapter(m["debugger"]); ok {
		l.Debugger = &debug
	}
	if soft, ok := m["soft-wrap"].(map[string]any); ok {
		l.SoftWrap = decodeSoftWrap(soft)
	}
	return l, l.Name != ""
}

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

func decodeAutoPairConfig(value any) (AutoPairConfig, bool) {
	switch v := value.(type) {
	case nil:
		return AutoPairConfig{}, false
	case bool:
		return AutoPairConfig{Present: true, Enable: &v}, true
	case map[string]any:
		return decodeAutoPairMap(v)
	case map[string]string:
		m := make(map[string]any, len(v))
		for k, value := range v {
			m[k] = value
		}
		return decodeAutoPairMap(m)
	default:
		return AutoPairConfig{}, false
	}
}

func decodeAutoPairMap(m map[string]any) (AutoPairConfig, bool) {
	pairs := make([][2]rune, 0, len(m))
	for k, value := range m {
		v, ok := value.(string)
		if !ok {
			return AutoPairConfig{}, false
		}
		openRunes := []rune(k)
		closeRunes := []rune(v)
		if len(openRunes) != 1 || len(closeRunes) != 1 {
			return AutoPairConfig{}, false
		}
		pairs = append(pairs, [2]rune{openRunes[0], closeRunes[0]})
	}
	return AutoPairConfig{Present: true, Pairs: pairs}, true
}

func decodeIndent(m map[string]any) Indent {
	return Indent{
		TabWidth: intPtrOrNil(m["tab-width"]),
		Unit:     stringValueFromMap(m, "unit"),
	}
}

func decodeCommentTokens(m map[string]any) []string {
	if tokens := decodeStringOrSlice(m["comment-tokens"]); len(tokens) > 0 {
		return tokens
	}
	return decodeStringOrSlice(m["comment-token"])
}

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

func decodeFileTypes(value any) []FileType {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]FileType, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case string:
			out = append(out, FileType{Extension: v})
		case map[string]any:
			if glob, ok := v["glob"].(string); ok {
				out = append(out, FileType{Glob: normalizeGlob(glob)})
			}
		}
	}
	return out
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

func decodeSoftWrap(m map[string]any) SoftWrap {
	return SoftWrap{
		Enable:          boolPtr(m["enable"]),
		MaxWrap:         intPtrOrNil(m["max-wrap"]),
		MaxIndentRetain: intPtrOrNil(m["max-indent-retain"]),
		WrapIndicator:   stringPtr(m["wrap-indicator"]),
		WrapAtTextWidth: boolPtr(m["wrap-at-text-width"]),
	}
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

func boolPtr(value any) *bool {
	v, ok := value.(bool)
	if !ok {
		return nil
	}
	return &v
}

func intPtr(value any) (*int, bool) {
	switch v := value.(type) {
	case int:
		return &v, true
	case int64:
		return new(int(v)), true
	default:
		return nil, false
	}
}

func intPtrOrNil(value any) *int {
	v, _ := intPtr(value)
	return v
}

func stringPtr(value any) *string {
	v, ok := value.(string)
	if !ok {
		return nil
	}
	return &v
}

func stringValueFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func intValueFromMap(m map[string]any, key string, fallback int) int {
	if n, ok := intPtr(m[key]); ok {
		return *n
	}
	return fallback
}

func boolValueFromMap(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}
