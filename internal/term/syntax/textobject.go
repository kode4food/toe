package syntax

import sitter "github.com/tree-sitter/go-tree-sitter"

var textObjectNames = map[rune]string{
	'f': "function",
	't': "class",
	'a': "parameter",
	'c': "call",
	'e': "entry",
}

// IsTextObjectChar reports whether ch maps to a tree-sitter textobject
func IsTextObjectChar(ch rune) bool {
	_, ok := textObjectNames[ch]
	return ok
}

// FindTextObject finds the innermost textobject at cursor for lang. inside=true
// returns the node's inner content, stripping bracket delimiters when present;
// false returns the full node range. Returns (Range{}, false) when the language
// or char is unknown, or no matching textobject contains cursor
func FindTextObject(
	text, lang string, cursor int, ch rune, inside bool,
) (Range, bool) {
	name, ok := textObjectNames[ch]
	if !ok {
		return Range{}, false
	}
	language, ok := languageFor(lang)
	if !ok {
		return Range{}, false
	}
	qb, ok := embeddedTextobjectQuery(lang)
	if !ok {
		return Range{}, false
	}
	q, qErr := sitter.NewQuery(language, string(qb))
	if qErr != nil {
		return Range{}, false
	}
	defer q.Close()

	src := []byte(text)
	p := sitter.NewParser()
	defer p.Close()
	if err := p.SetLanguage(language); err != nil {
		return Range{}, false
	}
	tree := p.Parse(src, nil)
	if tree == nil {
		return Range{}, false
	}
	defer tree.Close()

	runes := []rune(text)
	if cursor < 0 || cursor >= len(runes) {
		return Range{}, false
	}
	c2b := buildCharToByte(text)
	b2c := buildByteToChar(text)
	cursorByte := uint(c2b[cursor])

	suffix := name + ".around"
	if inside {
		suffix = name + ".inside"
	}
	capNames := q.CaptureNames()
	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(q, tree.RootNode(), src)

	best := Range{From: -1}
	for {
		m := matches.Next()
		if m == nil {
			break
		}
		for _, c := range m.Captures {
			if capNames[c.Index] != suffix {
				continue
			}
			n := c.Node
			if n.StartByte() > cursorByte || n.EndByte() <= cursorByte {
				continue
			}
			from := b2c[n.StartByte()]
			to := b2c[n.EndByte()]
			if inside && isBracketNodeAny(&n) {
				from++
				to--
			}
			if from > to {
				continue
			}
			if best.From < 0 || (to-from) < (best.To-best.From) {
				best = Range{From: from, To: to}
			}
		}
	}
	if best.From < 0 {
		return Range{}, false
	}
	return best, true
}
