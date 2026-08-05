package core

import "unicode"

// CharCategory classifies a character for reference word and motion behavior
type CharCategory int

const (
	CharCategoryWhitespace CharCategory = iota + 1
	CharCategoryEOL
	CharCategoryWord
	CharCategoryPunctuation
	CharCategoryUnknown
)

// CategorizeChar classifies ch for word-wise motion
func CategorizeChar(ch rune) CharCategory {
	if CharIsLineEnding(ch) {
		return CharCategoryEOL
	}
	if unicode.IsSpace(ch) {
		return CharCategoryWhitespace
	}
	if CharIsWord(ch) {
		return CharCategoryWord
	}
	if CharIsPunctuation(ch) {
		return CharCategoryPunctuation
	}
	return CharCategoryUnknown
}

// CharIsLineEnding reports whether ch terminates a line
func CharIsLineEnding(ch rune) bool {
	_, ok := LineEndingFromChar(ch)
	return ok
}

// CharIsWhitespace reports whether ch is a space or tab
func CharIsWhitespace(ch rune) bool {
	switch ch {
	case '\u0009', '\u0020', '\u00A0', '\u180E', '\u202F',
		'\u205F', '\u3000', '\uFEFF':
		return true
	}
	return ch >= '\u2000' && ch <= '\u200B'
}

// CharIsPunctuation reports whether ch is punctuation or a symbol
func CharIsPunctuation(ch rune) bool {
	return unicode.IsPunct(ch) || unicode.IsSymbol(ch)
}

// CharIsWord reports whether ch may appear inside a word
func CharIsWord(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch)
}
