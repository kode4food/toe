package syntax

import sitter "github.com/tree-sitter/go-tree-sitter"

// FindSurroundPair returns the Range of the skip-th bracket pair enclosing
// cursor, or (Range{}, false) if none exists at that depth
func FindSurroundPair(text, lang string, cursor, skip int) (Range, bool) {
	language, langOK := languageFor(lang)
	if !langOK {
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

// FindSurroundPairFor returns the Range of the skip-th enclosing pair matching
// ch (either bracket of the pair), or (Range{}, false) if none exists
func FindSurroundPairFor(
	text, lang string, cursor int, ch rune, skip int,
) (Range, bool) {
	openCh, closeCh, pairOK := bracketPairFor(ch)
	if !pairOK {
		return Range{}, false
	}
	language, langOK := languageFor(lang)
	if !langOK {
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
		if isBracketNode(n, openCh, closeCh) {
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
	for _, pair := range syntaxBrackets {
		if fk == string(pair[0]) && lk == string(pair[1]) {
			return true
		}
	}
	return false
}
