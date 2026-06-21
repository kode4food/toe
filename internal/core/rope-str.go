package core

import "unicode/utf8"

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

func splitStringAtChar(s string, pos int) (string, string) {
	if pos <= 0 {
		return "", s
	}
	if pos >= runeLen(s) {
		return s, ""
	}
	i := 0
	for b := range s {
		if i == pos {
			return s[:b], s[b:]
		}
		i++
	}
	return s, ""
}

// charSubstring returns the substring of s between rune offsets [from, to)
func charSubstring(s string, from, to int) string {
	startByte, endByte := 0, len(s)
	i := 0
	for b := range s {
		if i == from {
			startByte = b
		}
		if i == to {
			endByte = b
			break
		}
		i++
	}
	return s[startByte:endByte]
}
