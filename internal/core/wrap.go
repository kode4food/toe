package core

import (
	"strings"
	"unicode/utf8"
)

type (
	hardWrapUnfilled struct {
		text       string
		initial    string
		subsequent string
		ending     string
		trailing   bool
	}

	hardWrapWord struct {
		text  string
		width int
	}
)

const (
	hardWrapLF   = "\n"
	hardWrapCRLF = "\r\n"

	wrapLinePenalty   = 1000
	wrapShortTailCost = 25
	wrapShortTailDiv  = 4
)

// ReflowHardWrap reformats text to fit within width columns by breaking at word
// boundaries. Existing line breaks are first collapsed into spaces and common
// quote, comment, and list prefixes are retained on wrapped rows
func ReflowHardWrap(text string, width int) string {
	if width <= 0 || text == "" {
		return text
	}

	uw := unfillHardWrap(text)
	body := strings.TrimSuffix(uw.text, uw.ending)
	refilled := fillHardWrap(
		body, width, uw.initial, uw.subsequent, uw.ending,
	)
	if uw.trailing && refilled != "" {
		refilled += uw.ending
	}
	return refilled
}

func unfillHardWrap(text string) hardWrapUnfilled {
	ending := hardWrapLF
	if strings.Contains(text, hardWrapCRLF) {
		ending = hardWrapCRLF
	}
	normalized := strings.ReplaceAll(text, hardWrapCRLF, hardWrapLF)
	trailing := strings.HasSuffix(normalized, hardWrapLF)
	lines := strings.Split(normalized, hardWrapLF)
	if trailing {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return hardWrapUnfilled{ending: ending, trailing: trailing}
	}

	initial, subsequent := detectHardWrapPrefixes(lines)

	var b strings.Builder
	for i, line := range lines {
		prefix := initial
		if i > 0 {
			b.WriteByte(' ')
			prefix = subsequent
		}
		if len(line) <= len(prefix) {
			continue
		}
		b.WriteString(line[len(prefix):])
	}
	if trailing {
		b.WriteString(ending)
	}
	return hardWrapUnfilled{
		text:       b.String(),
		initial:    initial,
		subsequent: subsequent,
		ending:     ending,
		trailing:   trailing,
	}
}

func detectHardWrapPrefixes(lines []string) (string, string) {
	initial := hardWrapPrefix(lines[0])
	if len(lines) == 1 {
		return initial, ""
	}

	subsequent := hardWrapPrefix(lines[1])
	for _, line := range lines[2:] {
		subsequent = commonHardWrapPrefix(
			subsequent, hardWrapPrefix(line),
		)
	}
	return initial, subsequent
}

func hardWrapPrefix(line string) string {
	for i, ch := range line {
		if !hardWrapPrefixChar(ch) {
			return line[:i]
		}
	}
	return line
}

func commonHardWrapPrefix(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) {
		ra, aw := utf8.DecodeRuneInString(a[i:])
		rb, bw := utf8.DecodeRuneInString(b[i:])
		if ra != rb {
			return a[:i]
		}
		i += min(aw, bw)
	}
	return a[:i]
}

func fillHardWrap(
	text string, width int, initial, subsequent, ending string,
) string {
	words := hardWrapWordItems(
		hardWrapWords(text), hardWrapCapacity(width, initial, subsequent),
	)
	if len(words) == 0 {
		return ""
	}

	breaks := hardWrapBreaks(words, width, initial, subsequent)
	var b strings.Builder
	start := 0
	for i, end := range breaks {
		if i > 0 {
			b.WriteString(ending)
		}
		if i == 0 {
			b.WriteString(initial)
		} else {
			b.WriteString(subsequent)
		}
		writeHardWrapWords(&b, words[start:end])
		start = end
	}
	return b.String()
}

func hardWrapCapacity(width int, initial, subsequent string) int {
	capacity := width - textWidth(initial)
	if subsequent != "" {
		capacity = min(capacity, width-textWidth(subsequent))
	}
	return max(capacity, 1)
}

func hardWrapWordItems(words []string, capacity int) []hardWrapWord {
	res := make([]hardWrapWord, 0, len(words))
	for _, word := range words {
		for word != "" {
			part, rest := hardWrapTake(word, capacity)
			if part == "" {
				part, rest = hardWrapTake(word, 1)
			}
			res = append(res, hardWrapWord{text: part, width: textWidth(part)})
			word = rest
		}
	}
	return res
}

func hardWrapBreaks(
	words []hardWrapWord, width int, initial, subsequent string,
) []int {
	n := len(words)
	cost := make([]int, n+1)
	next := make([]int, n)
	for i := n - 1; i >= 0; i-- {
		bestCost := int(^uint(0) >> 1)
		bestNext := i + 1
		limit := width - textWidth(subsequent)
		if i == 0 {
			limit = width - textWidth(initial)
		}
		limit = max(limit, 1)
		lineW := 0
		for j := i; j < n; j++ {
			if j > i {
				lineW++
			}
			lineW += words[j].width
			if lineW > limit {
				break
			}
			c := hardWrapLineCost(words, i, j+1, lineW, limit, n)
			if j+1 < n {
				c += wrapLinePenalty + cost[j+1]
			}
			if c < bestCost {
				bestCost = c
				bestNext = j + 1
			}
		}
		cost[i] = bestCost
		next[i] = bestNext
	}

	var breaks []int
	for i := 0; i < n; i = next[i] {
		breaks = append(breaks, next[i])
	}
	return breaks
}

func hardWrapLineCost(
	words []hardWrapWord, start, end, lineW, limit, n int,
) int {
	if end == n {
		if end-start == 1 && words[start].width*wrapShortTailDiv < limit {
			return wrapShortTailCost
		}
		return 0
	}
	gap := limit - lineW
	return gap * gap
}

func writeHardWrapWords(b *strings.Builder, words []hardWrapWord) {
	for i, word := range words {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(word.text)
	}
}

func hardWrapWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == ' '
	})
}

func hardWrapTake(s string, width int) (string, string) {
	w := 0
	for i, ch := range s {
		next := w + graphemeWidth(string(ch))
		if next > width {
			return s[:i], s[i:]
		}
		w = next
	}
	return s, ""
}

func hardWrapPrefixChar(ch rune) bool {
	switch ch {
	case ' ', '-', '+', '*', '>', '#', '/':
		return true
	default:
		return false
	}
}

func textWidth(s string) int {
	w := 0
	for _, ch := range s {
		w += graphemeWidth(string(ch))
	}
	return w
}
