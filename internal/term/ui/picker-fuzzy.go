package ui

import (
	"math"
	"strings"
	"unicode"
)

type matcher struct {
	fields      map[int][]rune
	matchColumn int
	text        []rune
	bonus       []float64
	best        []float64
	end         []float64
}

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

func newMatcher(query string, columns []string, matchColumn int) *matcher {
	fields := parsePickerQuery(columns, matchColumn, query)
	patterns := make(map[int][]rune, len(fields))
	for col, pat := range fields {
		patterns[col] = []rune(strings.ToLower(pat))
	}
	return &matcher{fields: patterns, matchColumn: matchColumn}
}

func (m *matcher) match(item *PickerItem) (MatchResult, bool) {
	var out MatchResult
	for col, pat := range m.fields {
		key := item.columnText(col)
		res, ok := m.matchText(pat, key)
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

func (m *matcher) matchText(pat []rune, text string) (MatchResult, bool) {
	if len(pat) == 0 {
		return MatchResult{}, true
	}
	m.text = m.text[:0]
	for _, ch := range text {
		m.text = append(m.text, ch)
	}
	if !fuzzyHasMatch(fuzzyHasMatchArgs{pattern: pat, text: m.text}) {
		return MatchResult{}, false
	}
	if len(pat) == len(m.text) {
		return MatchResult{
			Score:   fuzzyScoreMax,
			Indices: fuzzySequence(len(pat)),
		}, true
	}
	if len(m.text) > fuzzyMaxLen {
		return MatchResult{
			Score:   -fuzzyScoreMax,
			Indices: fuzzySequence(len(pat)),
		}, true
	}
	score, indices := m.align(pat)
	return MatchResult{
		Score:   int(math.Round(score * fuzzyScale)),
		Indices: indices,
	}, true
}

func (m *matcher) align(pat []rune) (float64, []int) {
	// end[i][j] scores the best alignment of pat[:i+1] ending with pat[i] on
	// text[j]; best[i][j] the best alignment of pat[:i+1] within text[:j+1]
	n, width := len(pat), len(m.text)
	m.fillBonuses()
	m.best = growFuzzyScores(m.best, n*width)
	m.end = growFuzzyScores(m.end, n*width)
	for i := range pat {
		gap := fuzzyGapInner
		if i == n-1 {
			gap = fuzzyGapTrailing
		}
		prev := fuzzyScoreMin
		for j, c := range m.text {
			k := i*width + j
			switch {
			case pat[i] != unicode.ToLower(c):
				m.end[k] = fuzzyScoreMin
				prev += gap
			default:
				score := fuzzyScoreMin
				switch {
				case i == 0:
					score = float64(j)*fuzzyGapLeading + m.bonus[j]
				case j > 0:
					score = max(
						m.best[(i-1)*width+j-1]+m.bonus[j],
						m.end[(i-1)*width+j-1]+fuzzyConsecutive,
					)
				}
				m.end[k] = score
				prev = max(score, prev+gap)
			}
			m.best[k] = prev
		}
	}
	return m.best[n*width-1], fuzzyPositions(fuzzyPositionsArgs{
		best:          m.best,
		end:           m.end,
		patternLength: n,
		textWidth:     width,
	})
}

func (m *matcher) fillBonuses() {
	m.bonus = growFuzzyScores(m.bonus, len(m.text))
	prev := '/'
	for i, c := range m.text {
		m.bonus[i] = fuzzyBonus(fuzzyBonusArgs{previous: prev, current: c})
		prev = c
	}
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

type fuzzyMatchArgs struct {
	pattern string
	text    string
}

func fuzzyMatch(args fuzzyMatchArgs) (MatchResult, bool) {
	var m matcher
	return m.matchText([]rune(strings.ToLower(args.pattern)), args.text)
}

type fuzzyHasMatchArgs struct {
	pattern []rune
	text    []rune
}

func fuzzyHasMatch(args fuzzyHasMatchArgs) bool {
	j := 0
	for _, c := range args.text {
		if unicode.ToLower(c) != args.pattern[j] {
			continue
		}
		if j++; j == len(args.pattern) {
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

type fuzzyPositionsArgs struct {
	best          []float64
	end           []float64
	patternLength int
	textWidth     int
}

func fuzzyPositions(args fuzzyPositionsArgs) []int {
	out := make([]int, args.patternLength)
	consecutive := false
	j := args.textWidth - 1
	for i := args.patternLength - 1; i >= 0; i-- {
		for ; j >= 0; j-- {
			k := i*args.textWidth + j
			if args.end[k] == fuzzyScoreMin ||
				(!consecutive && args.end[k] != args.best[k]) {
				continue
			}
			consecutive = i > 0 && j > 0 &&
				args.best[k] ==
					args.end[(i-1)*args.textWidth+j-1]+fuzzyConsecutive
			out[i] = j
			j--
			break
		}
	}
	return out
}

func growFuzzyScores(scores []float64, n int) []float64 {
	if cap(scores) < n {
		return make([]float64, n)
	}
	return scores[:n]
}

type fuzzyBonusArgs struct {
	previous rune
	current  rune
}

func fuzzyBonus(args fuzzyBonusArgs) float64 {
	switch {
	case unicode.IsUpper(args.current):
		if unicode.IsLower(args.previous) {
			return fuzzyBonusCap
		}
	case !unicode.IsLower(args.current) && !unicode.IsDigit(args.current):
		return 0
	}
	switch args.previous {
	case '/', '\\':
		return fuzzyBonusSlash
	case '-', '_', ' ':
		return fuzzyBonusWord
	case '.':
		return fuzzyBonusDot
	}
	return 0
}
