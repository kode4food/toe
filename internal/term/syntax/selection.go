package syntax

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/kode4food/toe/internal/core"
)

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
	nodes, ok := nodePathAt(core.Source{Text: args.Text, Lang: args.Lang},

		args.Cursor)
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
	nodes, ok := nodePathAt(core.Source{Text: args.Text, Lang: args.Lang},

		args.Cursor)
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

// ParentNodeEnd returns the end of the innermost named node at the cursor,
// which is where a syntax-aware tab jumps to; false means no node was found,
// which is not the same as a node whose end the cursor already sits on
func ParentNodeEnd(args SelectionArgs) (int, bool) {
	nodes, ok := nodePathFor(core.Source{Text: args.Text, Lang: args.Lang},

		args.Cursor, false)
	if !ok {
		return 0, false
	}
	return nodes[0].To, true
}

func (r Range) bounds() Range {
	if r.From > r.To {
		return Range{From: r.To, To: r.From}
	}
	return r
}

func nodePathAt(source core.Source, cursor int) ([]Range, bool) {
	return nodePathFor(source, cursor, true)
}

func nodePathFor(
	source core.Source, cursor int, widen bool,
) ([]Range, bool) {
	text := source.Text
	language := languageFor(source.Lang)
	if language == nil {
		return nil, false
	}
	src := []byte(text)
	tree, ok := parseTree(src, language)
	if !ok {
		return nil, false
	}
	defer tree.Close()
	c2b := buildCharToByte(text)
	cursor = min(max(cursor, 0), len(c2b)-1)
	b := c2b[cursor]
	end := b
	if widen && end < len(src) {
		end++
	}
	root := tree.RootNode()
	n := root.NamedDescendantForByteRange(uint(b), uint(end))
	b2c := buildByteToChar(text)
	var nodes []Range
	for n != nil {
		if n.IsNamed() && !n.IsExtra() && n.EndByte() > n.StartByte() {
			if r, ok := nodeCharRange(n, b2c); ok {
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
