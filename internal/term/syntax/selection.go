package syntax

import sitter "github.com/tree-sitter/go-tree-sitter"

type (
	// SelectionArgs describes an editor range for syntax selection
	SelectionArgs struct {
		Text   string
		Lang   string
		Cursor int
		Range  Range
	}

	// Range is a character range selected by syntax
	Range struct {
		From int
		To   int
	}
)

// ExpandSelection returns the smallest Tree-sitter named node that strictly
// contains the current range, or the smallest named node at the cursor for an
// empty range
func ExpandSelection(args SelectionArgs) (Range, bool) {
	nodes, ok := nodePathAt(args.Text, args.Lang, args.Cursor)
	if !ok {
		return Range{}, false
	}
	bounds := args.Range.bounds()
	if bounds.From == bounds.To {
		return nodes[0], true
	}
	for _, n := range nodes {
		if n.From < bounds.From || n.To > bounds.To {
			return n, true
		}
	}
	return Range{}, false
}

// ShrinkSelection returns the largest Tree-sitter named node under the cursor
// that is strictly contained by the current range
func ShrinkSelection(args SelectionArgs) (Range, bool) {
	nodes, ok := nodePathAt(args.Text, args.Lang, args.Cursor)
	if !ok {
		return Range{}, false
	}
	bounds := args.Range.bounds()
	for i := len(nodes) - 1; i >= 0; i-- {
		n := nodes[i]
		if n.From > bounds.From && n.To < bounds.To {
			return n, true
		}
	}
	return Range{}, false
}

func (r Range) bounds() Range {
	if r.From > r.To {
		return Range{From: r.To, To: r.From}
	}
	return r
}

func nodePathAt(text, lang string, cursor int) ([]Range, bool) {
	language, ok := languageFor(lang)
	if !ok {
		return nil, false
	}
	src := []byte(text)
	p := sitter.NewParser()
	defer p.Close()
	if err := p.SetLanguage(language); err != nil {
		return nil, false
	}
	tree := p.Parse(src, nil)
	if tree == nil {
		return nil, false
	}
	defer tree.Close()
	c2b := buildCharToByte(text)
	cursor = min(max(cursor, 0), len(c2b)-1)
	b := c2b[cursor]
	end := b
	if end < len(src) {
		end++
	}
	root := tree.RootNode()
	n := root.NamedDescendantForByteRange(uint(b), uint(end))
	b2c := buildByteToChar(text)
	var nodes []Range
	for n != nil {
		if n.IsNamed() && !n.IsExtra() && n.EndByte() > n.StartByte() {
			r, ok := nodeCharRange(n, b2c)
			if ok {
				nodes = append(nodes, r)
			}
		}
		n = n.Parent()
	}
	if len(nodes) == 0 {
		return nil, false
	}
	return nodes, true
}

func nodeCharRange(n *sitter.Node, b2c []int) (Range, bool) {
	from, to := int(n.StartByte()), int(n.EndByte())
	if from < 0 || to > len(b2c)-1 || to <= from {
		return Range{}, false
	}
	return Range{From: b2c[from], To: b2c[to]}, true
}

func buildCharToByte(text string) []int {
	out := make([]int, 0, len(text)+1)
	for bi := range text {
		out = append(out, bi)
	}
	return append(out, len(text))
}
