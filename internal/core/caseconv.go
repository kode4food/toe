package core

import (
	"strings"
	"unicode"
)

// ToPascalCase converts text to PascalCase: non-alphanumeric chars are stripped
// and treated as word separators, and each word's first character is uppercased
// while the remaining characters are kept as-is
func ToPascalCase(text string) string {
	var buf strings.Builder
	atWordStart := true
	for _, c := range text {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			atWordStart = true
			continue
		}
		if atWordStart {
			atWordStart = false
			buf.WriteRune(unicode.ToUpper(c))
		} else {
			buf.WriteRune(c)
		}
	}
	return buf.String()
}

// ToCamelCase strips non-alphanumeric chars and lowercases all remaining
// characters. The first-word pass exhausts the iterator before the subsequent
// pascal-case pass can run, so all characters end up lowercased
func ToCamelCase(text string) string {
	var buf strings.Builder
	for _, c := range text {
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			buf.WriteRune(unicode.ToLower(c))
		}
	}
	return buf.String()
}

// ToUpperCase converts all characters in text to their Unicode uppercase form
func ToUpperCase(text string) string {
	return strings.ToUpper(text)
}

// ToLowerCase converts all characters in text to their Unicode lowercase form
func ToLowerCase(text string) string {
	return strings.ToLower(text)
}
