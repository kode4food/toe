package core

// FindNthChar finds the position of the nth matching character, starting from
// pos in the given direction
func (r Rope) FindNthChar(
	n int, match rune, pos int, dir Direction,
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
			if ch == match {
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
			if ch == match {
				n--
				if n == 0 {
					return i, true
				}
			}
		}
	}
	return 0, false
}
