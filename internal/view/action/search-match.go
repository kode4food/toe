package action

import (
	"regexp"
	"unicode"
	"unicode/utf8"
)

type searchMatch struct {
	pos     int
	wrapped bool
}

func compileSearchRegexp(
	pattern string, smartCase bool,
) (*regexp.Regexp, error) {
	if smartCase && !hasUppercase(pattern) {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

func hasUppercase(pattern string) bool {
	for _, ch := range pattern {
		if unicode.IsUpper(ch) {
			return true
		}
	}
	return false
}

func findNextMatch(
	re *regexp.Regexp, text string, from int, wrap bool,
) searchMatch {
	runes := []rune(text)
	wrapped := false
	if from >= len(runes) {
		if !wrap {
			return searchMatch{pos: -1}
		}
		from = 0
		wrapped = true
	}
	byteFrom := runeOffsetToByteOffset(text, from)
	for _, idx := range re.FindAllStringIndex(text[byteFrom:], -1) {
		if idx[0] == idx[1] {
			continue
		}
		pos := from + byteOffsetToRuneOffset(text[byteFrom:], idx[0])
		return searchMatch{pos: pos, wrapped: wrapped}
	}
	if wrap {
		for _, idx := range re.FindAllStringIndex(text[:byteFrom], -1) {
			if idx[0] == idx[1] {
				continue
			}
			pos := byteOffsetToRuneOffset(text, idx[0])
			return searchMatch{pos: pos, wrapped: true}
		}
	}
	return searchMatch{pos: -1}
}

func findPrevMatch(
	re *regexp.Regexp, text string, before int, wrap bool,
) searchMatch {
	runes := []rune(text)
	wrapped := false
	if before <= 0 {
		if !wrap {
			return searchMatch{pos: -1}
		}
		before = len(runes)
		wrapped = true
	}
	byteEnd := runeOffsetToByteOffset(text, before)
	all := re.FindAllStringIndex(text[:byteEnd], -1)
	if last, ok := lastNonEmptyMatch(all); ok {
		pos := byteOffsetToRuneOffset(text, last[0])
		return searchMatch{pos: pos, wrapped: wrapped}
	}
	if wrap {
		all2 := re.FindAllStringIndex(text[byteEnd:], -1)
		if last, ok := lastNonEmptyMatch(all2); ok {
			pos := before + byteOffsetToRuneOffset(text[byteEnd:], last[0])
			return searchMatch{pos: pos, wrapped: true}
		}
	}
	return searchMatch{pos: -1}
}

func lastNonEmptyMatch(matches [][]int) ([]int, bool) {
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		if m[0] != m[1] {
			return m, true
		}
	}
	return nil, false
}

func runeOffsetToByteOffset(s string, runeOff int) int {
	for i := range s {
		if runeOff == 0 {
			return i
		}
		runeOff--
	}
	return len(s)
}

func byteOffsetToRuneOffset(s string, byteOff int) int {
	return utf8.RuneCountInString(s[:byteOff])
}
