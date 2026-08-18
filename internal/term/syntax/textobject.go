package syntax

import sitter "github.com/tree-sitter/go-tree-sitter"

var textObjectNames = map[rune]string{
	'f': "function",
	't': "class",
	'a': "parameter",
	'c': "call",
	'e': "entry",
}

// FindTextObjectArgs identifies the textobject to find: the kind selected by
// Char at Cursor, taken inner (Inside) or whole
type FindTextObjectArgs struct {
	Text   string
	Lang   string
	Cursor int
	Char   rune
	Inside bool
}

// FindTextObject finds the innermost textobject at Cursor for Lang. Inside
// returns the node's inner content (delimiters stripped) vs its full range
func FindTextObject(args FindTextObjectArgs) (Range, bool) {
	name, ok := textObjectNames[args.Char]
	if !ok {
		return Range{}, false
	}
	language := languageFor(args.Lang)
	if language == nil {
		return Range{}, false
	}
	qb, ok := embeddedTextobjectQuery(args.Lang)
	if !ok {
		return Range{}, false
	}
	q, qErr := sitter.NewQuery(language, string(qb))
	if qErr != nil {
		return Range{}, false
	}
	defer q.Close()

	src := []byte(args.Text)
	tree, ok := parseTree(src, language)
	if !ok {
		return Range{}, false
	}
	defer tree.Close()

	runes := []rune(args.Text)
	if args.Cursor < 0 || args.Cursor >= len(runes) {
		return Range{}, false
	}
	c2b := buildCharToByte(args.Text)
	b2c := buildByteToChar(args.Text)
	cursorByte := uint(c2b[args.Cursor])

	suffix := name + ".around"
	if args.Inside {
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
			if args.Inside && isBracketNodeAny(&n) {
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
