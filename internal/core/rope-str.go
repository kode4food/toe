package core

import "unicode/utf8"

func splitStringAtChar(s string, pos int) (string, string) {
	if pos <= 0 {
		return "", s
	}
	if pos >= utf8.RuneCountInString(s) {
		return s, ""
	}
	i := 0
	lastByte := 0
	var lastCh rune
	for b := range s {
		if i == pos {
			if lastCh == '\r' {
				if ch, _ := utf8.DecodeRuneInString(s[b:]); ch == '\n' {
					return s[:lastByte], s[lastByte:]
				}
			}
			return s[:b], s[b:]
		}
		lastByte = b
		lastCh, _ = utf8.DecodeRuneInString(s[b:])
		i++
	}
	return s, ""
}

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
