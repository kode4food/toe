package language

import (
	"path/filepath"
	"strings"

	"github.com/kode4food/toe/internal/loader"
)

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
	values, ok := loader.AnySlice(value)
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
	if n, ok := loader.IntPtr(m["text-width"]); ok {
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
	if formatter, ok := decodeFormatter(m["formatter"]); ok {
		l.Formatter = &formatter
	}
	if soft, ok := m["soft-wrap"].(map[string]any); ok {
		l.SoftWrap = decodeSoftWrap(soft)
	}
	return l, l.Name != ""
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
		TabWidth: loader.IntPtrOrNil(m["tab-width"]),
		Unit:     stringValueFromMap(m, "unit"),
	}
}

func decodeCommentTokens(m map[string]any) []string {
	if tokens := decodeStringOrSlice(m["comment-tokens"]); len(tokens) > 0 {
		return tokens
	}
	return decodeStringOrSlice(m["comment-token"])
}

func decodeFileTypes(value any) []FileType {
	values, ok := loader.AnySlice(value)
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

func decodeSoftWrap(m map[string]any) SoftWrap {
	return SoftWrap{
		Enable:          loader.BoolPtr(m["enable"]),
		MaxWrap:         loader.IntPtrOrNil(m["max-wrap"]),
		MaxIndentRetain: loader.IntPtrOrNil(m["max-indent-retain"]),
		WrapIndicator:   loader.StringPtr(m["wrap-indicator"]),
		WrapAtTextWidth: loader.BoolPtr(m["wrap-at-text-width"]),
	}
}

func normalizeGlob(g string) string {
	if filepath.IsAbs(g) || strings.HasPrefix(g, "*/") {
		return g
	}
	return "*/" + g
}
