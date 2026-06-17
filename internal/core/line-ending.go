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

func LineEndingFromChar(ch rune) (LineEnding, bool) {
	if ch == '\n' {
		return LineEndingLF, true
	}
	return "", false
}

func ParseLineEnding(s string) (LineEnding, bool) {
	switch s {
	case "\r\n":
		return LineEndingCRLF, true
	case "\n":
		return LineEndingLF, true
	default:
		return "", false
	}
}

func StrIsLineEnding(s string) bool {
	_, ok := ParseLineEnding(s)
	return ok
}

func AutoDetectLineEndingString(s string) (LineEnding, bool) {
	var prev rune
	for _, ch := range s {
		if ch != '\n' {
			prev = ch
			continue
		}
		if prev == '\r' {
			return LineEndingCRLF, true
		}
		return LineEndingLF, true
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
	return "", false
}

func StringWithoutLineEnding(s string) string {
	if e, ok := GetLineEndingOfString(s); ok {
		return s[:len(s)-len(string(e))]
	}
	return s
}
