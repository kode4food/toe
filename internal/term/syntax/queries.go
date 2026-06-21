package syntax

import "embed"

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
