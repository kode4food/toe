package core

import "slices"

const MaxPlaintextScan = 10000

var (
	// bracketTable contains pairs where only one side is a bracket
	bracketTable = [][2]rune{
		{'(', ')'},
		{'{', '}'},
		{'[', ']'},
		{'<', '>'},
		{'‘', '’'},
		{'“', '”'},
		{'«', '»'},
		{'「', '」'},
		{'（', '）'},
	}

	bracketPairTable = append(
		slices.Clone(bracketTable),
		[2]rune{'"', '"'},
		[2]rune{'\'', '\''},
		[2]rune{'`', '`'},
		[2]rune{'|', '|'},
	)
)

// FindMatchingBracketPlaintext returns the position of the bracket matching the
// one at cursorPos, scanning at most MaxPlaintextScan characters. Returns (pos,
// true) on success, (0, false) if not found or not on a bracket
func FindMatchingBracketPlaintext(doc Rope, cursorPos int) (int, bool) {
	if cursorPos >= doc.LenChars() {
		return 0, false
	}
	bracket, err := doc.CharAt(cursorPos)
	if err != nil || !IsValidBracket(bracket) {
		return 0, false
	}
	openCh, closeCh := GetPair(bracket)
	matching := closeCh
	isOpen := openCh == bracket
	if !isOpen {
		matching = openCh
	}

	count := 1
	if isOpen {
		end := min(doc.LenChars(), cursorPos+MaxPlaintextScan+1)
		for i := cursorPos + 1; i < end; i++ {
			ch, err := doc.CharAt(i)
			if err != nil {
				break
			}
			switch ch {
			case bracket:
				count++
			case matching:
				count--
				if count == 0 {
					return i, true
				}
			}
		}
	} else {
		for i := cursorPos - 1; i >= 0 && cursorPos-i <= MaxPlaintextScan; i-- {
			ch, err := doc.CharAt(i)
			if err != nil {
				break
			}
			switch ch {
			case bracket:
				count++
			case matching:
				count--
				if count == 0 {
					return i, true
				}
			}
		}
	}
	return 0, false
}

// GetPair returns the open and close characters for a bracket pair. If ch is
// not in any pair, returns (ch, ch)
func GetPair(ch rune) (openCh, closeCh rune) {
	for _, p := range bracketPairTable {
		if p[0] == ch || p[1] == ch {
			return p[0], p[1]
		}
	}
	return ch, ch
}

// IsOpenBracket reports whether ch is an opening bracket
func IsOpenBracket(ch rune) bool {
	for _, p := range bracketTable {
		if p[0] == ch {
			return true
		}
	}
	return false
}

// IsCloseBracket reports whether ch is a closing bracket
func IsCloseBracket(ch rune) bool {
	for _, p := range bracketTable {
		if p[1] == ch {
			return true
		}
	}
	return false
}

// IsValidBracket reports whether ch is either side of a bracket pair
func IsValidBracket(ch rune) bool {
	return IsOpenBracket(ch) || IsCloseBracket(ch)
}

// IsOpenPair reports whether ch is an opening character of any pair
func IsOpenPair(ch rune) bool {
	for _, p := range bracketPairTable {
		if p[0] == ch {
			return true
		}
	}
	return false
}

// IsClosePair reports whether ch is a closing character of any pair
func IsClosePair(ch rune) bool {
	for _, p := range bracketPairTable {
		if p[1] == ch {
			return true
		}
	}
	return false
}

// IsValidPair reports whether ch appears in any pair (open or close)
func IsValidPair(ch rune) bool {
	for _, p := range bracketPairTable {
		if p[0] == ch || p[1] == ch {
			return true
		}
	}
	return false
}
