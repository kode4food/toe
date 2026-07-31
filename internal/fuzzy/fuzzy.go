// Package fuzzy provides fzf-style subsequence matching
package fuzzy

import (
	"math"
	"strings"
	"unicode"
)

type (
	// Result is a match's rank against a pattern, and the rune offsets in
	// the matched text to highlight
	Result struct {
		Score   int
		Indices []int
	}

	// Matcher scores repeated texts against one pattern, reusing its
	// scratch buffers across calls
	Matcher struct {
		pattern []rune
		text    []rune
		bonus   []float64
		best    []float64
		end     []float64
	}
)

const (
	gapLeading  = -0.005
	gapTrailing = -0.005
	gapInner    = -0.01
	consecutive = 1.0
	bonusSlash  = 0.9
	bonusWord   = 0.8
	bonusCap    = 0.7
	bonusDot    = 0.6

	scoreMin = -math.MaxFloat64
	scoreMax = math.MaxInt32
	scale    = 1000
	maxLen   = 1024
)

// NewMatcher prepares a case-insensitive matcher for pattern
func NewMatcher(pattern string) *Matcher {
	return &Matcher{pattern: []rune(strings.ToLower(pattern))}
}

// MatchArgs are the pattern and text for a single-call Match
type MatchArgs struct {
	Pattern string
	Text    string
}

// Match scores text against pattern in a single call
func Match(args MatchArgs) (Result, bool) {
	return NewMatcher(args.Pattern).Match(args.Text)
}

// Match scores text against the matcher's pattern
func (m *Matcher) Match(text string) (Result, bool) {
	if len(m.pattern) == 0 {
		return Result{}, true
	}
	m.text = m.text[:0]
	for _, ch := range text {
		m.text = append(m.text, ch)
	}
	if !hasMatch(hasMatchArgs{pattern: m.pattern, text: m.text}) {
		return Result{}, false
	}
	if len(m.pattern) == len(m.text) {
		return Result{
			Score: scoreMax, Indices: sequence(len(m.pattern)),
		}, true
	}
	if len(m.text) > maxLen {
		return Result{
			Score: -scoreMax, Indices: sequence(len(m.pattern)),
		}, true
	}
	score, indices := m.align()
	return Result{
		Score: int(math.Round(score * scale)), Indices: indices,
	}, true
}

func (m *Matcher) align() (float64, []int) {
	// end[i][j] scores the best alignment of pattern[:i+1] ending with
	// pattern[i] on text[j]; best[i][j] the best alignment of pattern[:i+1]
	// within text[:j+1]
	n, width := len(m.pattern), len(m.text)
	m.fillBonuses()
	m.best = growScores(m.best, n*width)
	m.end = growScores(m.end, n*width)
	for i, p := range m.pattern {
		gap := gapInner
		if i == n-1 {
			gap = gapTrailing
		}
		prev := scoreMin
		for j, c := range m.text {
			k := i*width + j
			switch {
			case p != unicode.ToLower(c):
				m.end[k] = scoreMin
				prev += gap
			default:
				score := scoreMin
				switch {
				case i == 0:
					score = float64(j)*gapLeading + m.bonus[j]
				case j > 0:
					score = max(
						m.best[(i-1)*width+j-1]+m.bonus[j],
						m.end[(i-1)*width+j-1]+consecutive,
					)
				}
				m.end[k] = score
				prev = max(score, prev+gap)
			}
			m.best[k] = prev
		}
	}
	return m.best[n*width-1], positions(positionsArgs{
		best: m.best, end: m.end, patternLength: n, textWidth: width,
	})
}

func (m *Matcher) fillBonuses() {
	m.bonus = growScores(m.bonus, len(m.text))
	prev := '/'
	for i, c := range m.text {
		m.bonus[i] = bonusFor(bonusArgs{previous: prev, current: c})
		prev = c
	}
}

type hasMatchArgs struct {
	pattern []rune
	text    []rune
}

func hasMatch(args hasMatchArgs) bool {
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

func sequence(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

type positionsArgs struct {
	best          []float64
	end           []float64
	patternLength int
	textWidth     int
}

func positions(args positionsArgs) []int {
	out := make([]int, args.patternLength)
	chained := false
	j := args.textWidth - 1
	for i := args.patternLength - 1; i >= 0; i-- {
		for ; j >= 0; j-- {
			k := i*args.textWidth + j
			if args.end[k] == scoreMin ||
				(!chained && args.end[k] != args.best[k]) {
				continue
			}
			chained = i > 0 && j > 0 &&
				args.best[k] ==
					args.end[(i-1)*args.textWidth+j-1]+consecutive
			out[i] = j
			j--
			break
		}
	}
	return out
}

func growScores(scores []float64, n int) []float64 {
	if cap(scores) < n {
		return make([]float64, n)
	}
	return scores[:n]
}

type bonusArgs struct {
	previous rune
	current  rune
}

func bonusFor(args bonusArgs) float64 {
	switch {
	case unicode.IsUpper(args.current):
		if unicode.IsLower(args.previous) {
			return bonusCap
		}
	case !unicode.IsLower(args.current) && !unicode.IsDigit(args.current):
		return 0
	}
	switch args.previous {
	case '/', '\\':
		return bonusSlash
	case '-', '_', ' ':
		return bonusWord
	case '.':
		return bonusDot
	}
	return 0
}
