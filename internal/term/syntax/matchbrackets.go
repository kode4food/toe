package syntax

import sitter "github.com/tree-sitter/go-tree-sitter"

var syntaxBrackets = [][2]rune{
	{'(', ')'},
	{'{', '}'},
	{'[', ']'},
}

// FindMatchingBracket returns the char position of the bracket matching the
// one at cursorPos, using the parse tree so string/comment brackets don't
// falsely match
func FindMatchingBracket(text, lang string, cursorPos int) (int, bool) {
	runes := []rune(text)
	if cursorPos < 0 || cursorPos >= len(runes) {
		return 0, false
	}
	ch := runes[cursorPos]
	openCh, closeCh, ok := bracketPairFor(ch)
	if !ok {
		return 0, false
	}
	isOpen := ch == openCh

	language, ok := languageFor(lang)
	if !ok {
		return 0, false
	}
	src := []byte(text)
	p := sitter.NewParser()
	defer p.Close()
	if err := p.SetLanguage(language); err != nil {
		return 0, false
	}
	tree := p.Parse(src, nil)
	if tree == nil {
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
			if n.StartByte() == b && isBracketNode(n, openCh, closeCh) {
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
			if n.EndByte() == bEnd && isBracketNode(n, openCh, closeCh) {
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

func bracketPairFor(ch rune) (open, close rune, ok bool) {
	for _, p := range syntaxBrackets {
		if p[0] == ch || p[1] == ch {
			return p[0], p[1], true
		}
	}
	return 0, 0, false
}

// anonymous first/last children confirm real bracket delimiters, not nodes
// whose content merely starts/ends with a bracket character
func isBracketNode(n *sitter.Node, openCh, closeCh rune) bool {
	count := n.ChildCount()
	if count == 0 {
		return false
	}
	first := n.Child(0)
	last := n.Child(count - 1)
	return !first.IsNamed() && first.Kind() == string(openCh) &&
		!last.IsNamed() && last.Kind() == string(closeCh)
}
