package syntax

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/kode4food/toe/internal/core"
)

var brackets = [][2]rune{
	{'(', ')'},
	{'{', '}'},
	{'[', ']'},
}

// FindMatchingBracket returns the char position of the bracket matching the
// one at cursorPos, using the parse tree so string/comment brackets don't
// falsely match
func FindMatchingBracket(
	source core.Source, cursorPos int,
) (int, bool) {
	text := source.Text
	runes := []rune(text)
	if cursorPos < 0 || cursorPos >= len(runes) {
		return 0, false
	}
	ch := runes[cursorPos]
	pair, ok := bracketPairFor(ch)
	if !ok {
		return 0, false
	}
	isOpen := ch == pair.Open

	language := languageFor(source.Lang)
	if language == nil {
		return 0, false
	}
	src := []byte(text)
	tree, ok := parseTree(src, language)
	if !ok {
		return 0, false
	}
	defer tree.Close()

	c2b := buildCharToByte(text)
	b2c := buildByteToChar(text)
	b := uint(c2b[cursorPos])
	root := tree.RootNode()

	if isOpen {
		n := root.DescendantForByteRange(b, b+1)
		for n != nil {
			if n.StartByte() == b && isBracketNode(n, pair) {
				lastChar := b2c[n.EndByte()] - 1
				if lastChar >= 0 && lastChar < len(runes) {
					return lastChar, true
				}
			}
			n = n.Parent()
		}
	} else {
		bEnd := uint(c2b[cursorPos+1])
		n := root.DescendantForByteRange(b, bEnd)
		for n != nil {
			if n.EndByte() == bEnd && isBracketNode(n, pair) {
				firstChar := b2c[n.StartByte()]
				if firstChar >= 0 && firstChar < len(runes) {
					return firstChar, true
				}
			}
			n = n.Parent()
		}
	}
	return 0, false
}

func bracketPairFor(ch rune) (core.BracketPair, bool) {
	for _, p := range brackets {
		if p[0] == ch || p[1] == ch {
			return core.BracketPair{Open: p[0], Close: p[1]}, true
		}
	}
	return core.BracketPair{}, false
}

func isBracketNode(n *sitter.Node, pair core.BracketPair) bool {
	count := n.ChildCount()
	if count == 0 {
		return false
	}
	first := n.Child(0)
	last := n.Child(count - 1)
	return !first.IsNamed() && first.Kind() == string(pair.Open) &&
		!last.IsNamed() && last.Kind() == string(pair.Close)
}
