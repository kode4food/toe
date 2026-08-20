package core

import "unicode/utf8"

type stringSplit struct {
	before string
	after  string
}

func splitStringAtChar(s string, pos int) stringSplit {
	if pos <= 0 {
		return stringSplit{after: s}
	}
	if pos >= utf8.RuneCountInString(s) {
		return stringSplit{before: s}
	}
	i := 0
	lastByte := 0
	var lastCh rune
	for b := range s {
		if i == pos {
			if lastCh == '\r' {
				if ch, _ := utf8.DecodeRuneInString(s[b:]); ch == '\n' {
					return stringSplit{
						before: s[:lastByte],
						after:  s[lastByte:],
					}
				}
			}
			return stringSplit{before: s[:b], after: s[b:]}
		}
		lastByte = b
		lastCh, _ = utf8.DecodeRuneInString(s[b:])
		i++
	}
	return stringSplit{before: s}
}

func charSubstring(str string, s Span) string {
	startByte, endByte := 0, len(str)
	i := 0
	for b := range str {
		if i == s.From {
			startByte = b
		}
		if i == s.To {
			endByte = b
			break
		}
		i++
	}
	return str[startByte:endByte]
}
