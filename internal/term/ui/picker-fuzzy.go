package ui

import (
	"math"
	"strings"
	"unicode"
)

const (
	fuzzyGapLeading  = -0.005
	fuzzyGapTrailing = -0.005
	fuzzyGapInner    = -0.01
	fuzzyConsecutive = 1.0
	fuzzyBonusSlash  = 0.9
	fuzzyBonusWord   = 0.8
	fuzzyBonusCap    = 0.7
	fuzzyBonusDot    = 0.6

	fuzzyScoreMin = -math.MaxFloat64
	fuzzyScoreMax = math.MaxInt32
	fuzzyScale    = 1000
	fuzzyMaxLen   = 1024
)

func fuzzyMatchItem(
	query string, item PickerItem, columns []string, matchColumn int,
) (int, []int, bool) {
	fields := parsePickerQuery(columns, matchColumn, query)
	score := 0
	var indices []int
	for col, pat := range fields {
		key := item.columnText(col)
		s, idx, ok := fuzzyMatch(pat, key)
		if !ok {
			return 0, nil, false
		}
		score += s
		if col == matchColumn {
			indices = idx
		}
	}
	return score, indices, true
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

func fuzzyMatch(pat, text string) (int, []int, bool) {
	if pat == "" {
		return 0, nil, true
	}
	pr := []rune(strings.ToLower(pat))
	tr := []rune(text)
	if !fuzzyHasMatch(pr, tr) {
		return 0, nil, false
	}
	if len(pr) == len(tr) {
		return fuzzyScoreMax, fuzzySequence(len(pr)), true
	}
	if len(tr) > fuzzyMaxLen {
		return -fuzzyScoreMax, fuzzySequence(len(pr)), true
	}
	score, indices := fuzzyAlign(pr, tr)
	return int(math.Round(score * fuzzyScale)), indices, true
}

func fuzzyHasMatch(pat, text []rune) bool {
	j := 0
	for _, c := range text {
		if unicode.ToLower(c) != pat[j] {
			continue
		}
		if j++; j == len(pat) {
			return true
		}
	}
	return false
}

func fuzzySequence(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func fuzzyAlign(pat, text []rune) (float64, []int) {
	// end[i][j] scores the best alignment of pat[:i+1] ending with pat[i] on
	// text[j]; best[i][j] the best alignment of pat[:i+1] within text[:j+1]
	n, m := len(pat), len(text)
	bonus := fuzzyBonuses(text)
	best := make([]float64, n*m)
	end := make([]float64, n*m)
	for i := range pat {
		gap := fuzzyGapInner
		if i == n-1 {
			gap = fuzzyGapTrailing
		}
		prev := fuzzyScoreMin
		for j, c := range text {
			k := i*m + j
			switch {
			case pat[i] != unicode.ToLower(c):
				end[k] = fuzzyScoreMin
				prev += gap
			default:
				score := fuzzyScoreMin
				switch {
				case i == 0:
					score = float64(j)*fuzzyGapLeading + bonus[j]
				case j > 0:
					score = max(
						best[(i-1)*m+j-1]+bonus[j],
						end[(i-1)*m+j-1]+fuzzyConsecutive,
					)
				}
				end[k] = score
				prev = max(score, prev+gap)
			}
			best[k] = prev
		}
	}
	return best[n*m-1], fuzzyPositions(best, end, n, m)
}

func fuzzyPositions(best, end []float64, n, m int) []int {
	out := make([]int, n)
	consecutive := false
	j := m - 1
	for i := n - 1; i >= 0; i-- {
		for ; j >= 0; j-- {
			k := i*m + j
			if end[k] == fuzzyScoreMin || (!consecutive && end[k] != best[k]) {
				continue
			}
			consecutive = i > 0 && j > 0 &&
				best[k] == end[(i-1)*m+j-1]+fuzzyConsecutive
			out[i] = j
			j--
			break
		}
	}
	return out
}

func fuzzyBonuses(text []rune) []float64 {
	out := make([]float64, len(text))
	prev := '/'
	for i, c := range text {
		out[i] = fuzzyBonus(prev, c)
		prev = c
	}
	return out
}

func fuzzyBonus(prev, cur rune) float64 {
	switch {
	case unicode.IsUpper(cur):
		if unicode.IsLower(prev) {
			return fuzzyBonusCap
		}
	case !unicode.IsLower(cur) && !unicode.IsDigit(cur):
		return 0
	}
	switch prev {
	case '/', '\\':
		return fuzzyBonusSlash
	case '-', '_', ' ':
		return fuzzyBonusWord
	case '.':
		return fuzzyBonusDot
	}
	return 0
}
