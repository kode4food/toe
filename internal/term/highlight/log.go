package highlight

import (
	"strings"
	"unicode/utf8"

	"github.com/kode4food/toe/internal/tui"
)

// LogError, LogWarning, LogInfo, LogCommand and LogTerminal are the tags the
// message log writes at the start of each line, naming the severity or the
// source, LogSeparator the rule drawn after them
const (
	LogError     = "ERR"
	LogWarning   = "WAR"
	LogInfo      = "INF"
	LogCommand   = "CMD"
	LogTerminal  = "TRM"
	LogSeparator = " " + tui.BorderV + " "
)

var logScopes = map[string]string{
	LogError:    "ui.log.error",
	LogWarning:  "ui.log.warning",
	LogInfo:     "ui.log.info",
	LogCommand:  "ui.log.command",
	LogTerminal: "ui.log.terminal",
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
