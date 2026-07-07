package syntax

import (
	"embed"
	"strings"
)

//go:embed queries
var embeddedQueryFS embed.FS

func HasHighlightQuery(lang string) bool {
	_, ok := embeddedQuery(lang)
	return ok
}

func embeddedQuery(lang string) ([]byte, bool) {
	data, err := embeddedQueryFS.ReadFile("queries/" + lang + ".scm")
	if err != nil {
		return nil, false
	}
	return data, true
}

func embeddedTextobjectQuery(lang string) ([]byte, bool) {
	return resolveTextobjectQuery(lang, map[string]bool{})
}

func resolveTextobjectQuery(lang string, seen map[string]bool) ([]byte, bool) {
	if seen[lang] {
		return nil, false
	}
	seen[lang] = true
	data, err := embeddedQueryFS.ReadFile(
		"queries/textobjects/" + lang + ".scm",
	)
	if err != nil {
		return nil, false
	}
	var out []byte
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(
			trimmed, "; inherits:",
		); ok {
			for parent := range strings.SplitSeq(after, ",") {
				parent = strings.TrimSpace(parent)
				if parent == "" {
					continue
				}
				if pb, ok := resolveTextobjectQuery(
					parent, seen,
				); ok {
					out = append(out, pb...)
					out = append(out, '\n')
				}
			}
			continue
		}
		out = append(out, line...)
		out = append(out, '\n')
	}
	return out, true
}
