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
	text := args.Source.Text
	cursor := args.Cursor
	skip := args.Skip
	language := languageFor(args.Source.Lang)
	if language == nil {
		return Range{}, false
	}
	runes := []rune(text)
	if cursor < 0 || cursor >= len(runes) {
		return Range{}, false
	}
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

	c2b := buildCharToByte(text)
	b2c := buildByteToChar(text)
	b := uint(c2b[cursor])
	root := tree.RootNode()

	n := root.DescendantForByteRange(b, b+1)
	for n != nil {
		if isBracketNodeAny(n) {
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
	language := languageFor(args.Lang)
	if language == nil {
		return Range{}, false
	}
	skip := args.Skip
	runes := []rune(args.Text)
	if args.Cursor < 0 || args.Cursor >= len(runes) {
		return Range{}, false
	}
	src := []byte(args.Text)
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

	c2b := buildCharToByte(args.Text)
	b2c := buildByteToChar(args.Text)
	b := uint(c2b[args.Cursor])
	root := tree.RootNode()

	n := root.DescendantForByteRange(b, b+1)
	for n != nil {
		if isBracketNode(n, pair) {
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
