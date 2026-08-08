package ui

import (
	"strings"

	"github.com/kode4food/toe/internal/fuzzy"
)

type matcher struct {
	fields      map[int]*fuzzy.Matcher
	matchColumn int
}

func newMatcher(query string, columns []string, matchColumn int) *matcher {
	fields := parsePickerQuery(columns, matchColumn, query)
	patterns := make(map[int]*fuzzy.Matcher, len(fields))
	for col, pat := range fields {
		patterns[col] = fuzzy.NewMatcher(pat)
	}
	return &matcher{fields: patterns, matchColumn: matchColumn}
}

func (m *matcher) match(item *PickerItem) (MatchResult, bool) {
	var out MatchResult
	for col, fm := range m.fields {
		key := item.columnText(col)
		res, ok := fm.Match(key)
		if !ok {
			return MatchResult{}, false
		}
		out.Score += res.Score
		if col == m.matchColumn {
			out.Indices = res.Indices
		}
	}
	return out, true
}

type fuzzyMatchArgs struct {
	pattern string
	text    string
}

func fuzzyMatch(args fuzzyMatchArgs) (MatchResult, bool) {
	res, ok := fuzzy.Match(fuzzy.MatchArgs{
		Pattern: args.pattern, Text: args.text,
	})
	return MatchResult{Score: res.Score, Indices: res.Indices}, ok
}

func parsePickerQuery(
	columns []string, matchColumn int, input string,
) map[int]string {
	fields := map[int]string{}
	if input == "" {
		fields[matchColumn] = ""
		return fields
	}
	field := matchColumn
	var fieldText strings.Builder
	var text strings.Builder
	escaped := false
	inField := false
	finish := func() {
		pat := strings.TrimSuffix(text.String(), " ")
		if pat != "" {
			if prev := fields[field]; prev != "" {
				fields[field] = prev + " " + pat
			} else {
				fields[field] = pat
			}
		}
		text.Reset()
	}
	for _, ch := range input {
		switch {
		case escaped:
			if ch != '%' {
				text.WriteRune('\\')
			}
			text.WriteRune(ch)
			escaped = false
		case ch == '\\':
			escaped = true
		case ch == '%':
			if text.Len() > 0 {
				finish()
			}
			field = matchColumn
			fieldText.Reset()
			inField = true
		case ch == ' ' && inField:
			text.Reset()
			inField = false
		case inField:
			fieldText.WriteRune(ch)
			if idx, ok := matchPickerColumn(columns, fieldText.String()); ok {
				field = idx
			}
		default:
			text.WriteRune(ch)
		}
	}
	if !inField && text.Len() > 0 {
		finish()
	}
	if len(fields) == 0 {
		fields[matchColumn] = ""
	}
	return fields
}

func matchPickerColumn(columns []string, prefix string) (int, bool) {
	best := -1
	for i, col := range columns {
		if !strings.HasPrefix(col, prefix) {
			continue
		}
		if best < 0 || len(col) < len(columns[best]) {
			best = i
		}
	}
	return best, best >= 0
}
