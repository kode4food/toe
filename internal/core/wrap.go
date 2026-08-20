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

	// hardWrapPrefixes is the prefix carried by a paragraph's first line and
	// the one carried by every line after it
	hardWrapPrefixes struct {
		initial    string
		subsequent string
	}

	stringPair struct {
		left  string
		right string
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
	refilled := fillHardWrap(&uw, width)
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

	prefixes := detectHardWrapPrefixes(lines)

	var b strings.Builder
	for i, line := range lines {
		prefix := prefixes.initial
		if i > 0 {
			b.WriteByte(' ')
			prefix = prefixes.subsequent
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
		initial:    prefixes.initial,
		subsequent: prefixes.subsequent,
		ending:     ending,
		trailing:   trailing,
	}
}

func detectHardWrapPrefixes(lines []string) hardWrapPrefixes {
	initial := hardWrapPrefix(lines[0])
	if len(lines) == 1 {
		return hardWrapPrefixes{initial: initial}
	}

	subsequent := hardWrapPrefix(lines[1])
	for _, line := range lines[2:] {
		subsequent = commonHardWrapPrefix(stringPair{
			left:  subsequent,
			right: hardWrapPrefix(line),
		})
	}
	return hardWrapPrefixes{initial: initial, subsequent: subsequent}
}

func hardWrapPrefix(line string) string {
	for i, ch := range line {
		if !isHardWrapPrefixChar(ch) {
			return line[:i]
		}
	}
	return line
}

func commonHardWrapPrefix(pair stringPair) string {
	i := 0
	for i < len(pair.left) && i < len(pair.right) {
		ra, aw := utf8.DecodeRuneInString(pair.left[i:])
		rb, bw := utf8.DecodeRuneInString(pair.right[i:])
		if ra != rb {
			return pair.left[:i]
		}
		i += min(aw, bw)
	}
	return pair.left[:i]
}

func isHardWrapPrefixChar(ch rune) bool {
	switch ch {
	case ' ', '-', '+', '*', '>', '#', '/':
		return true
	default:
		return false
	}
}
