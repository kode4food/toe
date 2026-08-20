package syntax

import (
	"embed"
	"strings"
)

//go:embed queries
var queryFS embed.FS

// HasHighlightQuery reports whether a highlight query ships for lang
func HasHighlightQuery(lang string) bool {
	_, ok := embeddedQuery(lang)
	return ok
}

func embeddedQuery(lang string) ([]byte, bool) {
	data, err := queryFS.ReadFile("queries/" + lang + ".scm")
	if err != nil {
		return nil, false
	}
	return data, true
}

func embeddedInjectionQuery(lang string) ([]byte, bool) {
	return resolveQueryDir(resolveQueryDirArgs{
		dir:  "queries/injections",
		lang: lang,
	}, map[string]bool{})
}

func embeddedTextobjectQuery(lang string) ([]byte, bool) {
	return resolveQueryDir(resolveQueryDirArgs{
		dir:  "queries/textobjects",
		lang: lang,
	}, map[string]bool{})
}

type resolveQueryDirArgs struct {
	dir  string
	lang string
}

func resolveQueryDir(
	ref resolveQueryDirArgs, seen map[string]bool,
) ([]byte, bool) {
	lang := ref.lang
	if seen[lang] {
		return nil, false
	}
	seen[lang] = true
	data, err := queryFS.ReadFile(ref.dir + "/" + lang + ".scm")
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
				pb, ok := resolveQueryDir(resolveQueryDirArgs{
					dir:  ref.dir,
					lang: parent,
				}, seen)
				if ok {
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
