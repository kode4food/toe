package syntax

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/kode4food/toe/internal/core"
)

// FindSurroundPairArgs identifies the enclosing pair to find: the Skip-th pair
// around Cursor, in any bracket style
type FindSurroundPairArgs struct {
	Source core.Source
	Cursor int
	Skip   int
}

// FindSurroundPair returns the Range of the Skip-th bracket pair enclosing
// Cursor, or (Range{}, false) if none exists at that depth
func FindSurroundPair(args FindSurroundPairArgs) (Range, bool) {
	return findSurroundPair(args, isBracketNodeAny)
}

// FindSurroundPairForArgs identifies the enclosing bracket pair to find: the
// Skip-th pair matching Char around Cursor
type FindSurroundPairForArgs struct {
	Text   string
	Lang   string
	Cursor int
	Char   rune
	Skip   int
}

// FindSurroundPairFor returns the Range of the Skip-th enclosing pair matching
// Char (either bracket of the pair), or (Range{}, false) if none exists
func FindSurroundPairFor(args FindSurroundPairForArgs) (Range, bool) {
	pair, ok := bracketPairFor(args.Char)
	if !ok {
		return Range{}, false
	}
	return findSurroundPair(FindSurroundPairArgs{
		Source: core.Source{Text: args.Text, Lang: args.Lang},
		Cursor: args.Cursor,
		Skip:   args.Skip,
	}, func(n *sitter.Node) bool {
		return isBracketNode(n, pair)
	})
}

// findSurroundPair: isEnclose decides which nodes count as an enclosing pair
func findSurroundPair(
	args FindSurroundPairArgs, isEnclose func(*sitter.Node) bool,
) (Range, bool) {
	text := args.Source.Text
	language := languageFor(args.Source.Lang)
	if language == nil {
		return Range{}, false
	}
	runes := []rune(text)
	if args.Cursor < 0 || args.Cursor >= len(runes) {
		return Range{}, false
	}
	tree, ok := parseTree([]byte(text), language)
	if !ok {
		return Range{}, false
	}
	defer tree.Close()

	c2b := buildCharToByte(text)
	b2c := buildByteToChar(text)
	b := uint(c2b[args.Cursor])
	skip := args.Skip

	n := tree.RootNode().DescendantForByteRange(b, b+1)
	for n != nil {
		if isEnclose(n) {
			skip--
			if skip == 0 {
				f := b2c[n.StartByte()]
				t := b2c[n.EndByte()] - 1
				if f >= 0 && t < len(runes) {
					return Range{From: f, To: t}, true
				}
			}
		}
		n = n.Parent()
	}
	return Range{}, false
}

func isBracketNodeAny(n *sitter.Node) bool {
	count := n.ChildCount()
	if count == 0 {
		return false
	}
	first := n.Child(0)
	last := n.Child(count - 1)
	if first.IsNamed() || last.IsNamed() {
		return false
	}
	fk, lk := first.Kind(), last.Kind()
	for _, pair := range brackets {
		if fk == string(pair[0]) && lk == string(pair[1]) {
			return true
		}
	}
	return false
}
