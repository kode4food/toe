package core

import "strings"

// IndentStyle describes whether indentation uses tabs or spaces
type IndentStyle struct {
	tabs  bool
	width uint8
}

// MaxIndent is the maximum spaces per indent level
const MaxIndent = 16

var indents = strings.Repeat(" ", MaxIndent)

// Tabs returns an IndentStyle that uses tab characters
func Tabs() IndentStyle {
	return IndentStyle{tabs: true}
}

// Spaces returns an IndentStyle that uses n space characters per level
func Spaces(n uint8) IndentStyle {
	if n == 0 || n > MaxIndent {
		n = 1
	}
	return IndentStyle{width: n}
}

// ParseIndentStyle creates an IndentStyle from an indent string such as
// "    " (four spaces) or "\t"
func ParseIndentStyle(s string) IndentStyle {
	if len(s) == 0 || s[0] != ' ' {
		return Tabs()
	}
	return Spaces(uint8(min(len(s), MaxIndent)))
}

// IsTabs reports whether this style uses tab characters
func (i IndentStyle) IsTabs() bool {
	return i.tabs
}

// Width returns the number of spaces per indent level (0 for tabs)
func (i IndentStyle) Width() uint8 {
	return i.width
}

// AsStr returns the string for one indent level
func (i IndentStyle) AsStr() string {
	if i.tabs {
		return "\t"
	}
	n := i.width
	if n == 0 || n > MaxIndent {
		n = 1
	}
	return indents[:n]
}

// IndentWidth returns the number of columns one indent level occupies
func (i IndentStyle) IndentWidth(tabWidth int) int {
	if i.tabs {
		return tabWidth
	}
	return int(i.width)
}

// AutoDetect attempts to detect the indentation style used in doc. Returns the
// detected style and true, or false if confidence is too low
func AutoDetect(doc Rope) (IndentStyle, bool) {
	var histogram [MaxIndent + 1]int
	prevIsTabs := false
	prevLeading := 0

	for i := range min(doc.LenLines(), 1000) {
		line, err := doc.Line(i)
		if err != nil {
			continue
		}
		runes := []rune(line.String())
		if len(runes) == 0 {
			continue
		}

		first := runes[0]
		var isTabs bool
		switch {
		case first == '\t':
			isTabs = true
		case first == ' ':
		case CharIsLineEnding(first):
			continue // blank line
		default:
			prevIsTabs = false
			prevLeading = 0
			continue
		}

		leading, skip := countLeading(runes, isTabs)
		if skip {
			continue
		}

		if (prevIsTabs == isTabs || prevLeading == 0) &&
			prevLeading < leading {
			if isTabs {
				histogram[0]++
			} else {
				amount := leading - prevLeading
				if amount <= MaxIndent {
					histogram[amount]++
				}
			}
		}
		prevIsTabs = isTabs
		prevLeading = leading
	}

	histogram[0] *= 2
	if histogram[1] > 1 {
		histogram[1] /= 2
	}

	best, bestFreq := 0, 0
	for i, freq := range histogram {
		if freq > bestFreq {
			best, bestFreq = i, freq
		}
	}
	runnerUp := 0
	for i, freq := range histogram {
		if i != best && freq > runnerUp {
			runnerUp = freq
		}
	}

	if bestFreq < 1 {
		return IndentStyle{}, false
	}
	if float64(runnerUp)/float64(bestFreq) >= 0.66 {
		return IndentStyle{}, false
	}
	if best == 0 {
		return Tabs(), true
	}
	return Spaces(uint8(best)), true
}

func countLeading(runes []rune, isTabs bool) (int, bool) {
	leading := 1
	for _, ch := range runes[1:] {
		switch {
		case ch == '\t' && isTabs:
			leading++
		case ch == ' ' && !isTabs:
			leading++
		case CharIsLineEnding(ch):
			return 0, true
		case CharIsWhitespace(ch):
			return leading, false
		default:
			return leading, false
		}
		if leading > 256 {
			return 0, true
		}
	}
	return leading, false
}
