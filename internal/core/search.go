package core

type (
	// CharMatcher tests whether a character satisfies a condition
	CharMatcher interface {
		MatchChar(ch rune) bool
	}

	// RuneMatcher is a CharMatcher that matches a single rune
	RuneMatcher rune
)

func (r RuneMatcher) MatchChar(ch rune) bool {
	return rune(r) == ch
}

// FindNthChar finds the position of the nth character matching m, starting from
// pos in the given direction
func (r Rope) FindNthChar(
	n int, m CharMatcher, pos int, dir Direction,
) (int, bool) {
	if n == 0 {
		return 0, false
	}
	switch dir {
	case DirectionForward:
		for i := pos; i < r.LenChars(); i++ {
			ch, err := r.CharAt(i)
			if err != nil {
				return 0, false
			}
			if m.MatchChar(ch) {
				n--
				if n == 0 {
					return i, true
				}
			}
		}
	case DirectionBackward:
		for i := pos - 1; i >= 0; i-- {
			ch, err := r.CharAt(i)
			if err != nil {
				return 0, false
			}
			if m.MatchChar(ch) {
				n--
				if n == 0 {
					return i, true
				}
			}
		}
	}
	return 0, false
}
