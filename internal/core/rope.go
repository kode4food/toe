package core

import (
	"errors"
	"fmt"
	"strings"
)

// Rope stores text as a balanced binary tree of string leaves
type Rope struct {
	root *ropeNode
}

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
