package core

import (
	"strings"
	"unicode/utf8"
)

type (
	ropeNode struct {
		left  *ropeNode
		right *ropeNode
		text  string
		chars int
		lines int
		depth int
	}

	ropePair struct {
		left  *ropeNode
		right *ropeNode
	}
)

func newLeafRopeNode(text string) *ropeNode {
	return &ropeNode{
		text:  text,
		chars: utf8.RuneCountInString(text),
		lines: countLineBreaks(text),
		depth: 1,
	}
}

func buildRopeNode(text string) *ropeNode {
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) <= DefaultRopeLeafChars {
		return newLeafRopeNode(text)
	}
	split := splitStringAtChar(text, utf8.RuneCountInString(text)/2)
	return concatRopeNode(ropePair{
		left:  buildRopeNode(split.before),
		right: buildRopeNode(split.after),
	})
}

func concatRopeNode(pair ropePair) *ropeNode {
	if pair.left == nil {
		return pair.right
	}
	if pair.right == nil {
		return pair.left
	}
	n := &ropeNode{left: pair.left, right: pair.right}
	refreshRopeNode(n)
	return balanceRopeNode(n)
}

func splitRopeNode(n *ropeNode, pos int) ropePair {
	if n == nil {
		return ropePair{}
	}
	if n.left == nil && n.right == nil {
		split := splitStringAtChar(n.text, pos)
		return ropePair{
			left:  buildRopeNode(split.before),
			right: buildRopeNode(split.after),
		}
	}

	leftChars := ropeChars(n.left)
	if pos < leftChars {
		pair := splitRopeNode(n.left, pos)
		return ropePair{
			left: pair.left,
			right: concatRopeNode(ropePair{
				left:  pair.right,
				right: n.right,
			}),
		}
	}
	pair := splitRopeNode(n.right, pos-leftChars)
	return ropePair{
		left: concatRopeNode(ropePair{
			left:  n.left,
			right: pair.left,
		}),
		right: pair.right,
	}
}

func balanceRopeNode(n *ropeNode) *ropeNode {
	if n == nil || n.left == nil || n.right == nil {
		return n
	}
	if ropeDepth(n.left)-ropeDepth(n.right) > maxRopeDepthSkew {
		return rotateRopeRight(n)
	}
	if ropeDepth(n.right)-ropeDepth(n.left) > maxRopeDepthSkew {
		return rotateRopeLeft(n)
	}
	return n
}

func rotateRopeRight(n *ropeNode) *ropeNode {
	p := n.left
	newN := &ropeNode{left: p.right, right: n.right}
	refreshRopeNode(newN)
	newP := &ropeNode{left: p.left, right: newN}
	refreshRopeNode(newP)
	return newP
}

func rotateRopeLeft(n *ropeNode) *ropeNode {
	p := n.right
	newN := &ropeNode{left: n.left, right: p.left}
	refreshRopeNode(newN)
	newP := &ropeNode{left: newN, right: p.right}
	refreshRopeNode(newP)
	return newP
}

func refreshRopeNode(n *ropeNode) {
	n.text = ""
	n.chars = ropeChars(n.left) + ropeChars(n.right)
	n.lines = ropeLines(n.left) + ropeLines(n.right)
	n.depth = max(ropeDepth(n.left), ropeDepth(n.right)) + 1
}

func writeRopeString(b *strings.Builder, n *ropeNode) {
	if n == nil {
		return
	}
	if n.left == nil && n.right == nil {
		b.WriteString(n.text)
		return
	}
	writeRopeString(b, n.left)
	writeRopeString(b, n.right)
}

func ropeChars(n *ropeNode) int {
	if n == nil {
		return 0
	}
	return n.chars
}

func ropeLines(n *ropeNode) int {
	if n == nil {
		return 0
	}
	return n.lines
}

func ropeDepth(n *ropeNode) int {
	if n == nil {
		return 0
	}
	return n.depth
}
