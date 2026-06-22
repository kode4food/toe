package ui

import "strings"

func fuzzyMatchItem(
	query string, item PickerItem, columns []string, primary int,
) (int, []int, bool) {
	fields := parsePickerQuery(columns, primary, query)
	score := 0
	var indices []int
	for col, pat := range fields {
		key := item.columnText(col)
		s, idx := fuzzyMatch(strings.ToLower(pat), key)
		if s < 0 {
			return 0, nil, false
		}
		score += s
		if col == primary {
			indices = idx
		}
	}
	return score, indices, true
}

func parsePickerQuery(
	columns []string, primary int, input string,
) map[int]string {
	fields := map[int]string{}
	if input == "" {
		fields[primary] = ""
		return fields
	}
	field := primary
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
			field = primary
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
		fields[primary] = ""
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

func fuzzyMatch(pat, text string) (int, []int) {
	if len(pat) == 0 {
		return 0, nil
	}
	pr := []rune(pat)
	tr := []rune(text)
	tl := []rune(strings.ToLower(text))
	if len(pr) > len(tr) {
		return -1, nil
	}

	indices := make([]int, 0, len(pr))
	j := 0
	for i, c := range tl {
		if j < len(pr) && c == pr[j] {
			indices = append(indices, i)
			j++
		}
	}
	if j < len(pr) {
		return -1, nil
	}

	score := 0
	prev := -2
	for _, idx := range indices {
		if prev >= 0 && idx == prev+1 {
			score += 5
		}
		switch idx {
		case 0:
			score += 10
		default:
			switch tr[idx-1] {
			case '/', '\\', '.', '-', '_', ' ':
				score += 8
			}
		}
		prev = idx
	}
	score -= len(tr) / 4
	return score, indices
}
