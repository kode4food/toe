package highlight

import (
	"strings"
	"unicode/utf8"

	"github.com/kode4food/toe/internal/tui"
)

// LogError, LogWarning and LogInfo are the severity tags the message log writes
// at the start of each line, LogSeparator the rule drawn after them
const (
	LogError     = "ERR"
	LogWarning   = "WAR"
	LogInfo      = "INF"
	LogSeparator = " " + tui.BorderV + " "
)

var logScopes = map[string]string{
	LogError:   "error",
	LogWarning: "warning",
	LogInfo:    "info",
}

func tokenizeLog(text string) []Span {
	var spans []Span
	pos := 0
	for line := range strings.SplitSeq(text, "\n") {
		spans = append(spans, logLineSpans(line, pos)...)
		pos += utf8.RuneCountInString(line) + 1
	}
	return spans
}

// the separator stays unscoped, so it draws in the document's text style
func logLineSpans(line string, pos int) []Span {
	tag, rest, ok := strings.Cut(line, LogSeparator)
	if !ok || rest == "" {
		return nil
	}
	scope, known := logScopes[tag]
	if !known {
		return nil
	}
	tagEnd := pos + utf8.RuneCountInString(tag)
	restAt := tagEnd + utf8.RuneCountInString(LogSeparator)
	return []Span{
		{Start: pos, End: tagEnd, Scope: scope},
		{
			Start: restAt,
			End:   restAt + utf8.RuneCountInString(rest),
			Scope: scope,
		},
	}
}
