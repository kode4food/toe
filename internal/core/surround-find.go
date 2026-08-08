package core

// surroundSearch is a scan start position and how many enclosing pairs to skip
type surroundSearch struct {
	pos int
	nth int
}

func (r Rope) surroundFindNthOpen(
	pair BracketPair, at surroundSearch,
) (int, bool) {
	pos := at.pos
	if pos >= r.LenChars() {
		return 0, false
	}
	if ch, err := r.CharAt(pos); err == nil && ch == pair.Open {
		return pos, true
	}
	if pos > 0 {
		if ch, err := r.CharAt(pos - 1); err == nil && ch == pair.Open {
			return pos - 1, true
		}
	}
	for range at.nth {
		stepOver := 0
		found := false
		for i := pos - 1; i >= 0; i-- {
			ch, err := r.CharAt(i)
			if err != nil {
				return 0, false
			}
			if ch == pair.Close {
				stepOver++
			} else if ch == pair.Open {
				if stepOver == 0 {
					pos = i
					found = true
					break
				}
				stepOver--
			}
		}
		if !found {
			return 0, false
		}
	}
	return pos, true
}

func (r Rope) surroundFindNthClose(
	pair BracketPair, at surroundSearch,
) (int, bool) {
	pos := at.pos
	if pos >= r.LenChars() {
		return 0, false
	}
	if ch, err := r.CharAt(pos); err == nil && ch == pair.Close {
		return pos, true
	}
	for range at.nth {
		stepOver := 0
		found := false
		for i := pos + 1; i < r.LenChars(); i++ {
			ch, err := r.CharAt(i)
			if err != nil {
				return 0, false
			}
			if ch == pair.Open {
				stepOver++
			} else if ch == pair.Close {
				if stepOver == 0 {
					pos = i
					found = true
					break
				}
				stepOver--
			}
		}
		if !found {
			return 0, false
		}
	}
	return pos, true
}
