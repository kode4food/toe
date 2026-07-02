package core

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ReflowHardWrap reformats text to fit within width columns by breaking at word
// boundaries. Existing line breaks are first collapsed into spaces, then the
// result is wrapped at word boundaries without splitting words
func ReflowHardWrap(text string, width int) string {
	if width <= 0 || text == "" {
		return text
	}

	// Collapse existing hard wraps into spaces to form a single paragraph
	words := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r)
	})

	if len(words) == 0 {
		return ""
	}

	var b strings.Builder
	lineLen := 0

	for i, w := range words {
		wLen := utf8.RuneCountInString(w)
		if i == 0 {
			b.WriteString(w)
			lineLen = wLen
			continue
		}
		if lineLen+1+wLen > width {
			b.WriteByte('\n')
			b.WriteString(w)
			lineLen = wLen
		} else {
			b.WriteByte(' ')
			b.WriteString(w)
			lineLen += 1 + wLen
		}
	}

	return b.String()
}
