package core

func charAtRopeNode(n *ropeNode, pos int) rune {
	if n.left == nil && n.right == nil {
		for i, ch := range n.text {
			if pos == 0 {
				return ch
			}
			pos--
			_ = i
		}
		return 0
	}
	leftChars := ropeChars(n.left)
	if pos < leftChars {
		return charAtRopeNode(n.left, pos)
	}
	return charAtRopeNode(n.right, pos-leftChars)
}

func lineToCharRopeNode(n *ropeNode, line int) int {
	if n == nil || line == 0 {
		return 0
	}
	if n.left == nil && n.right == nil {
		pos := 0
		seen := 0
		runes := []rune(n.text)
		for i := 0; i < len(runes); i++ {
			ch := runes[i]
			pos++
			if ch == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
				pos++
				i++
				seen++
				if seen == line {
					return pos
				}
				continue
			}
			if _, ok := LineEndingFromChar(ch); ok {
				seen++
				if seen == line {
					return pos
				}
			}
		}
		return pos
	}
	leftLines := ropeLines(n.left)
	if line <= leftLines {
		return lineToCharRopeNode(n.left, line)
	}
	return ropeChars(n.left) + lineToCharRopeNode(n.right, line-leftLines)
}

func charToLineRopeNode(n *ropeNode, pos int) int {
	if n == nil || pos == 0 {
		return 0
	}
	if n.left == nil && n.right == nil {
		line := 0
		count := 0
		runes := []rune(n.text)
		for i := 0; i < len(runes); i++ {
			ch := runes[i]
			if count >= pos {
				break
			}
			count++
			if ch == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
				if count >= pos {
					break
				}
				count++
				i++
				line++
				continue
			}
			if _, ok := LineEndingFromChar(ch); ok {
				line++
			}
		}
		return line
	}
	leftChars := ropeChars(n.left)
	if pos <= leftChars {
		return charToLineRopeNode(n.left, pos)
	}
	return ropeLines(n.left) + charToLineRopeNode(n.right, pos-leftChars)
}

func forEachSegmentNode(n *ropeNode, from, to int, fn func(string)) {
	if n == nil || from >= to {
		return
	}
	if n.left == nil && n.right == nil {
		if from <= 0 && to >= n.chars {
			fn(n.text)
			return
		}
		fn(charSubstring(n.text, from, to))
		return
	}
	lc := ropeChars(n.left)
	if from < lc {
		forEachSegmentNode(n.left, from, min(to, lc), fn)
	}
	if to > lc {
		forEachSegmentNode(n.right, max(from-lc, 0), to-lc, fn)
	}
}
