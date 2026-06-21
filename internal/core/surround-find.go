package core

func (r Rope) surroundFindNthOpen(
	openCh, closeCh rune, pos, n int,
) (int, bool) {
	if pos >= r.LenChars() {
		return 0, false
	}
	// Check if pos is already on the open char
	if ch, err := r.CharAt(pos); err == nil && ch == openCh {
		return pos, true
	}
	for range n {
		stepOver := 0
		found := false
		for i := pos - 1; i >= 0; i-- {
			ch, err := r.CharAt(i)
			if err != nil {
				return 0, false
			}
			if ch == closeCh {
				stepOver++
			} else if ch == openCh {
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
	openCh, closeCh rune, pos, n int,
) (int, bool) {
	if pos >= r.LenChars() {
		return 0, false
	}
	// Check if pos is already on the close char
	if ch, err := r.CharAt(pos); err == nil && ch == closeCh {
		return pos, true
	}
	for range n {
		stepOver := 0
		found := false
		for i := pos + 1; i < r.LenChars(); i++ {
			ch, err := r.CharAt(i)
			if err != nil {
				return 0, false
			}
			if ch == openCh {
				stepOver++
			} else if ch == closeCh {
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
