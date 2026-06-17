package core

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type (
	// Rope stores text as a balanced binary tree of string leaves
	Rope struct {
		root *ropeNode
	}

	ropeNode struct {
		left  *ropeNode
		right *ropeNode
		text  string
		chars int
		lines int
		depth int
	}
)

const (
	DefaultRopeLeafChars = 1024
	maxRopeDepthSkew     = 1
)

var (
	ErrRopeIndexOutOfRange = errors.New("rope index out of range")
	ErrRopeLineOutOfRange  = errors.New("rope line out of range")
)

func NewRope(text string) Rope {
	return Rope{root: buildRopeNode(text)}
}

func (r Rope) LenChars() int {
	if r.root == nil {
		return 0
	}
	return r.root.chars
}

func (r Rope) LenLines() int {
	if r.root == nil {
		return 1
	}
	return r.root.lines + 1
}

func (r Rope) String() string {
	var b strings.Builder
	writeRopeString(&b, r.root)
	return b.String()
}

func (r Rope) Slice(from, to int) (Rope, error) {
	if from < 0 || to < from || to > r.LenChars() {
		return Rope{}, fmt.Errorf("%w: %d..%d", ErrRopeIndexOutOfRange,
			from, to)
	}
	left, rest := splitRopeNode(r.root, from)
	_ = left
	mid, _ := splitRopeNode(rest, to-from)
	return Rope{root: mid}, nil
}

// SliceString returns the substring between char offsets [from, to) without
// constructing a new rope. It walks the tree and copies only the spanned leaf
// ranges, avoiding the node splitting, concatenation, and rune recounting that
// Slice followed by String performs — the hot path for per-line rendering
func (r Rope) SliceString(from, to int) (string, error) {
	if from < 0 || to < from || to > r.LenChars() {
		return "", fmt.Errorf("%w: %d..%d", ErrRopeIndexOutOfRange, from, to)
	}
	if from == to {
		return "", nil
	}
	var b strings.Builder
	b.Grow(to - from)
	r.ForEachSegment(from, to, func(seg string) {
		b.WriteString(seg)
	})
	return b.String(), nil
}

// ForEachSegment applies fn to each contiguous leaf substring spanning
// [from, to). The segments are slices into the rope's leaves, so nothing is
// copied; callers that fold over a range fold over whole chunks instead of
// paying a function call per rune
func (r Rope) ForEachSegment(from, to int, fn func(string)) {
	if from < 0 {
		from = 0
	}
	if n := r.LenChars(); to > n {
		to = n
	}
	if from >= to {
		return
	}
	forEachSegmentNode(r.root, from, to, fn)
}

func (r Rope) Insert(pos int, text string) (Rope, error) {
	if pos < 0 || pos > r.LenChars() {
		return Rope{}, fmt.Errorf("%w: %d", ErrRopeIndexOutOfRange, pos)
	}
	left, right := splitRopeNode(r.root, pos)
	ins := buildRopeNode(text)
	return Rope{root: concatRopeNode(concatRopeNode(left, ins), right)}, nil
}

func (r Rope) Delete(from, to int) (Rope, error) {
	if from < 0 || to < from || to > r.LenChars() {
		return Rope{}, fmt.Errorf("%w: %d..%d", ErrRopeIndexOutOfRange,
			from, to)
	}
	left, rest := splitRopeNode(r.root, from)
	_, right := splitRopeNode(rest, to-from)
	return Rope{root: concatRopeNode(left, right)}, nil
}

func (r Rope) CharAt(pos int) (rune, error) {
	if pos < 0 || pos >= r.LenChars() {
		return 0, fmt.Errorf("%w: %d", ErrRopeIndexOutOfRange, pos)
	}
	return charAtRopeNode(r.root, pos), nil
}

func (r Rope) Line(line int) (Rope, error) {
	if line < 0 || line >= r.LenLines() {
		return Rope{}, fmt.Errorf("%w: %d", ErrRopeLineOutOfRange, line)
	}
	from, err := r.LineToChar(line)
	if err != nil {
		return Rope{}, err
	}
	to := r.LenChars()
	if line+1 < r.LenLines() {
		to, err = r.LineToChar(line + 1)
		if err != nil {
			return Rope{}, err
		}
	}
	return r.Slice(from, to)
}

func (r Rope) LineToChar(line int) (int, error) {
	if line < 0 || line >= r.LenLines() {
		return 0, fmt.Errorf("%w: %d", ErrRopeLineOutOfRange, line)
	}
	if line == 0 {
		return 0, nil
	}
	return lineToCharRopeNode(r.root, line), nil
}

func (r Rope) CharToLine(pos int) (int, error) {
	if pos < 0 || pos > r.LenChars() {
		return 0, fmt.Errorf("%w: %d", ErrRopeIndexOutOfRange, pos)
	}
	return charToLineRopeNode(r.root, pos), nil
}

func (r Rope) LineEndCharIndex(line int) (int, error) {
	start, err := r.LineToChar(line)
	if err != nil {
		return 0, err
	}
	end := r.LenChars()
	if line+1 < r.LenLines() {
		end, err = r.LineToChar(line + 1)
		if err != nil {
			return 0, err
		}
	}
	if end <= start {
		return end, nil
	}
	// The line ending is at most the final two chars; read only those rather
	// than the whole line to locate where the content ends
	tail, err := r.SliceString(max(end-2, start), end)
	if err != nil {
		return end, nil
	}
	if e, ok := GetLineEndingOfString(tail); ok {
		end -= len(e)
	}
	return end, nil
}

func buildRopeNode(text string) *ropeNode {
	if text == "" {
		return nil
	}
	if runeLen(text) <= DefaultRopeLeafChars {
		return newLeafRopeNode(text)
	}
	left, right := splitStringAtChar(text, runeLen(text)/2)
	return concatRopeNode(buildRopeNode(left), buildRopeNode(right))
}

func newLeafRopeNode(text string) *ropeNode {
	return &ropeNode{
		text:  text,
		chars: runeLen(text),
		lines: strings.Count(text, "\n"),
		depth: 1,
	}
}

func concatRopeNode(left, right *ropeNode) *ropeNode {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	n := &ropeNode{left: left, right: right}
	refreshRopeNode(n)
	return balanceRopeNode(n)
}

func splitRopeNode(n *ropeNode, pos int) (*ropeNode, *ropeNode) {
	if n == nil {
		return nil, nil
	}
	if n.left == nil && n.right == nil {
		left, right := splitStringAtChar(n.text, pos)
		return buildRopeNode(left), buildRopeNode(right)
	}

	leftChars := ropeChars(n.left)
	if pos < leftChars {
		a, b := splitRopeNode(n.left, pos)
		return a, concatRopeNode(b, n.right)
	}
	a, b := splitRopeNode(n.right, pos-leftChars)
	return concatRopeNode(n.left, a), b
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

func charAtRopeNode(n *ropeNode, pos int) rune {
	if n.left == nil && n.right == nil {
		for i, ch := range n.text {
			if pos == 0 {
				return ch
			}
			pos--
			_ = i
		}
		return 0
	}
	leftChars := ropeChars(n.left)
	if pos < leftChars {
		return charAtRopeNode(n.left, pos)
	}
	return charAtRopeNode(n.right, pos-leftChars)
}

func lineToCharRopeNode(n *ropeNode, line int) int {
	if n == nil || line == 0 {
		return 0
	}
	if n.left == nil && n.right == nil {
		pos := 0
		seen := 0
		for _, ch := range n.text {
			pos++
			if ch != '\n' {
				continue
			}
			seen++
			if seen == line {
				return pos
			}
		}
		return pos
	}
	leftLines := ropeLines(n.left)
	if line <= leftLines {
		return lineToCharRopeNode(n.left, line)
	}
	return ropeChars(n.left) + lineToCharRopeNode(n.right, line-leftLines)
}

func charToLineRopeNode(n *ropeNode, pos int) int {
	if n == nil || pos == 0 {
		return 0
	}
	if n.left == nil && n.right == nil {
		line := 0
		count := 0
		for _, ch := range n.text {
			if count >= pos {
				break
			}
			count++
			if ch == '\n' {
				line++
			}
		}
		return line
	}
	leftChars := ropeChars(n.left)
	if pos <= leftChars {
		return charToLineRopeNode(n.left, pos)
	}
	return ropeLines(n.left) + charToLineRopeNode(n.right, pos-leftChars)
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

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

func splitStringAtChar(s string, pos int) (string, string) {
	if pos <= 0 {
		return "", s
	}
	if pos >= runeLen(s) {
		return s, ""
	}
	i := 0
	for b := range s {
		if i == pos {
			return s[:b], s[b:]
		}
		i++
	}
	return s, ""
}

// forEachSegmentNode walks leaves in [from, to) handing each spanned leaf
// substring to fn
func forEachSegmentNode(n *ropeNode, from, to int, fn func(string)) {
	if n == nil || from >= to {
		return
	}
	if n.left == nil && n.right == nil {
		fn(charSubstring(n.text, from, to))
		return
	}
	lc := ropeChars(n.left)
	if from < lc {
		forEachSegmentNode(n.left, from, min(to, lc), fn)
	}
	if to > lc {
		forEachSegmentNode(n.right, max(from-lc, 0), to-lc, fn)
	}
}

// charSubstring returns the substring of s between rune offsets [from, to)
func charSubstring(s string, from, to int) string {
	startByte, endByte := 0, len(s)
	i := 0
	for b := range s {
		if i == from {
			startByte = b
		}
		if i == to {
			endByte = b
			break
		}
		i++
	}
	return s[startByte:endByte]
}
