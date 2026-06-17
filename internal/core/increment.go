package core

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// dateFormat maps a Go time layout to whether it has date/time components
type dateFormat struct {
	layout  string
	hasDate bool
	hasTime bool
}

const incrementSeparator = '_'

// IncrementInteger increments or decrements the integer literal in text by
// amount. Supported bases: decimal (no prefix), hex (0x), octal (0o), binary
// (0b). Underscore separators are preserved. Returns the new string and true
// on success, or ("", false) if text is not a valid integer literal
func IncrementInteger(text string, amount int64) (string, bool) {
	if text == "" ||
		text[0] == incrementSeparator ||
		text[len(text)-1] == incrementSeparator {
		return "", false
	}

	runes := []rune(text)
	n := len(runes)
	var sepRTL []int
	for i := range n {
		if runes[n-1-i] == incrementSeparator {
			sepRTL = append(sepRTL, i)
		}
	}

	stripped := strings.Map(func(r rune) rune {
		if r == incrementSeparator {
			return -1
		}
		return r
	}, text)

	var result string
	var ok bool

	switch {
	case strings.HasPrefix(stripped, "0x"):
		result, ok = incrementHex(stripped, amount, len(stripped)-2-len(sepRTL))
	case strings.HasPrefix(stripped, "0o"):
		result, ok = incrementOctal(stripped, amount, len(stripped)-2-len(sepRTL))
	case strings.HasPrefix(stripped, "0b"):
		result, ok = incrementBinary(stripped, amount, len(stripped)-2-len(sepRTL))
	default:
		result, ok = incrementDecimal(stripped, amount, len(sepRTL))
	}

	if !ok {
		return "", false
	}

	result = reinjectSeparators(result, sepRTL, len(text), stripped)
	return result, true
}

// IncrementDateTime increments or decrements the date/time literal in text by
// amount. Date-only formats advance by days; time-only formats advance by
// minutes; date+time formats advance by minutes. Returns the new string and
// true on success
func IncrementDateTime(text string, amount int64) (string, bool) {
	if text == "" {
		return "", false
	}
	for _, f := range dateFormats() {
		t, err := time.Parse(f.layout, text)
		if err != nil {
			continue
		}
		var out time.Time
		switch {
		case f.hasDate && f.hasTime:
			out = t.Add(time.Duration(amount) * time.Minute)
		case f.hasDate:
			out = t.AddDate(0, 0, int(amount))
		case f.hasTime:
			out = t.Add(time.Duration(amount) * time.Minute)
		default:
			continue
		}
		return out.Format(f.layout), true
	}
	return "", false
}

func dateFormats() []dateFormat {
	return []dateFormat{
		{"2006-01-02 15:04:05", true, true},
		{"2006/01/02 15:04:05", true, true},
		{"2006-01-02 15:04", true, true},
		{"2006/01/02 15:04", true, true},
		{"2006-01-02", true, false},
		{"2006/01/02", true, false},
		{"Mon Jan 02 2006", true, false},
		{"02-Jan-2006", true, false},
		{"2006 Jan 02", true, false},
		{"Jan 02, 2006", true, false},
		{"3:04:05 pm", false, true},
		{"3:04 pm", false, true},
		{"3:04:05 PM", false, true},
		{"3:04 PM", false, true},
		{"15:04:05", false, true},
		{"15:04", false, true},
	}
}

func incrementDecimal(
	number string, amount int64, sepCount int,
) (string, bool) {
	var value int64
	_, err := fmt.Sscanf(number, "%d", &value)
	if err != nil {
		return "", false
	}
	newVal := saturatingAddI64(value, amount)

	fmtLen := len(number) - sepCount
	switch {
	case value < 0 && newVal >= 0:
		fmtLen--
	case value >= 0 && newVal < 0:
		fmtLen++
	}

	if strings.HasPrefix(number, "0") || strings.HasPrefix(number, "-0") {
		return fmt.Sprintf("%0*d", fmtLen, newVal), true
	}
	return fmt.Sprintf("%d", newVal), true
}

func incrementHex(number string, amount int64, fmtLen int) (string, bool) {
	digits := number[2:]
	var value uint64
	_, err := fmt.Sscanf(digits, "%x", &value)
	if err != nil {
		return "", false
	}
	newVal := saturatingAddU64(value, amount)

	lower, upper := 0, 0
	for _, ch := range digits {
		if unicode.IsLower(ch) {
			lower++
		} else if unicode.IsUpper(ch) {
			upper++
		}
	}
	if upper > lower {
		return fmt.Sprintf("0x%0*X", fmtLen, newVal), true
	}
	return fmt.Sprintf("0x%0*x", fmtLen, newVal), true
}

func incrementOctal(number string, amount int64, fmtLen int) (string, bool) {
	digits := number[2:]
	var value uint64
	_, err := fmt.Sscanf(digits, "%o", &value)
	if err != nil {
		return "", false
	}
	newVal := saturatingAddU64(value, amount)
	return fmt.Sprintf("0o%0*o", fmtLen, newVal), true
}

func incrementBinary(number string, amount int64, fmtLen int) (string, bool) {
	digits := number[2:]
	var value uint64
	for _, ch := range digits {
		if ch != '0' && ch != '1' {
			return "", false
		}
		value = value<<1 | uint64(ch-'0')
	}
	newVal := saturatingAddU64(value, amount)
	return fmt.Sprintf("0b%0*b", fmtLen, newVal), true
}

func saturatingAddI64(a, b int64) int64 {
	r, overflow := addInt64Overflow(a, b)
	if overflow {
		if b > 0 {
			return 1<<63 - 1
		}
		return -1 << 63
	}
	return r
}

func addInt64Overflow(a, b int64) (int64, bool) {
	r := a + b
	if (b > 0 && r < a) || (b < 0 && r > a) {
		return r, true
	}
	return r, false
}

func saturatingAddU64(a uint64, amount int64) uint64 {
	if amount >= 0 {
		r := a + uint64(amount)
		if r < a {
			return ^uint64(0)
		}
		return r
	}
	sub := uint64(-amount)
	if sub > a {
		return 0
	}
	return a - sub
}

func reinjectSeparators(
	result string, sepRTL []int, origLen int, stripped string,
) string {
	if len(sepRTL) == 0 {
		return result
	}
	runes := []rune(result)
	n := len(runes)
	for _, rtl := range sepRTL {
		if rtl < n {
			pos := n - rtl
			if pos > 0 {
				tail := append([]rune{incrementSeparator}, runes[pos:]...)
				runes = append(runes[:pos], tail...)
				n++
			}
		}
	}
	if len(runes) > origLen && len(sepRTL) > 0 {
		spacing := sepRTL[0]
		if len(sepRTL) >= 2 {
			spacing = sepRTL[0] - sepRTL[1] - 1
		}
		prefix := 0
		if !strings.HasPrefix(stripped, "0") || strings.HasPrefix(stripped, "0x") ||
			strings.HasPrefix(stripped, "0o") || strings.HasPrefix(stripped, "0b") {
			if stripped[0] == '0' {
				prefix = 2
			}
		}
		idx := -1
		for i, ch := range runes {
			if ch == incrementSeparator {
				idx = i
				break
			}
		}
		if idx >= 0 && spacing > 0 {
			for idx-prefix > spacing {
				idx -= spacing
				tail := append([]rune{incrementSeparator}, runes[idx:]...)
				runes = append(runes[:idx], tail...)
			}
		}
	}
	return string(runes)
}
