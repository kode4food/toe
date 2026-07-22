package core

import (
	"errors"
	"fmt"
)

// LineEnding is the actual line-ending byte sequence for a document
type LineEnding string

const (
	LineEndingLF   LineEnding = "\n"
	LineEndingCRLF LineEnding = "\r\n"
)

var ErrInvalidLineEnding = errors.New("invalid line ending")

// NativeLineEnding is defined in platform-specific files

func (l *LineEnding) UnmarshalText(text []byte) error {
	switch string(text) {
	case "lf":
		*l = LineEndingLF
	case "crlf":
		*l = LineEndingCRLF
	case "native":
		*l = NativeLineEnding()
	default:
		return fmt.Errorf("%w: %s", ErrInvalidLineEnding, text)
	}
	return nil
}

// ParseLineEnding parses a "lf", "crlf", or "native" option value
func ParseLineEnding(value string) (LineEnding, error) {
	var l LineEnding
	if err := l.UnmarshalText([]byte(value)); err != nil {
		return "", err
	}
	return l, nil
}

// LineEndingNames returns the recognized line-ending option values, in
// display order
func LineEndingNames() []string {
	return []string{"lf", "crlf", "native"}
}

func LineEndingFromChar(ch rune) (LineEnding, bool) {
	switch ch {
	case '\n', '\r', '\v', '\f', '\u0085', '\u2028', '\u2029':
		return LineEndingLF, true
	}
	return "", false
}

func AutoDetectLineEndingString(s string) (LineEnding, bool) {
	runes := []rune(s)
	lines := 0
	for i, ch := range runes {
		if ch == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
			return LineEndingCRLF, true
		}
		switch ch {
		case '\n', '\r', '\u0085', '\u2028':
			return LineEndingLF, true
		case '\v', '\f', '\u2029':
			lines++
			if lines >= 100 {
				return "", false
			}
		}
	}
	return "", false
}

func GetLineEndingOfString(s string) (LineEnding, bool) {
	if len(s) >= 2 && s[len(s)-2:] == string(LineEndingCRLF) {
		return LineEndingCRLF, true
	}
	if len(s) >= 1 && s[len(s)-1:] == string(LineEndingLF) {
		return LineEndingLF, true
	}
	runes := []rune(s)
	if len(runes) > 0 {
		if _, ok := LineEndingFromChar(runes[len(runes)-1]); ok {
			return LineEndingLF, true
		}
	}
	return "", false
}

func countLineBreaks(s string) int {
	runes := []rune(s)
	n := 0
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
			n++
			i++
			continue
		}
		if _, ok := LineEndingFromChar(runes[i]); ok {
			n++
		}
	}
	return n
}
