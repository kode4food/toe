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

func hardWrapPrefixChar(ch rune) bool {
	switch ch {
	case ' ', '-', '+', '*', '>', '#', '/':
		return true
	default:
		return false
	}
}
