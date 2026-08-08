package core

import (
	"errors"
	"fmt"
	"strings"
)

type (
	// Rope stores text as a balanced binary tree of string leaves
	Rope struct {
		root *ropeNode
	}

	// Source is a document's text together with the language it is written
	// in. Highlighting, parsing, and language detection operate on this pair
	Source struct {
		Text string
		Lang string
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

// NewRope returns a rope holding text
func NewRope(text string) Rope {
	return Rope{root: buildRopeNode(text)}
}

// LenChars is the character count of the whole rope
func (r Rope) LenChars() int {
	if r.root == nil {
		return 0
	}
	return r.root.chars
}

// LenLines is the line count, counting a trailing ending as a new line
func (r Rope) LenLines() int {
	if r.root == nil {
		return 1
	}
	return r.root.lines + 1
}

// String returns the whole rope as text
func (r Rope) String() string {
	var b strings.Builder
	writeRopeString(&b, r.root)
	return b.String()
}

// Slice returns the characters the span covers as a new rope
func (r Rope) Slice(s Span) (Rope, error) {
	if err := r.checkSpan(s); err != nil {
		return Rope{}, err
	}
	rest := splitRopeNode(r.root, s.From).right
	return Rope{root: splitRopeNode(rest, s.Len()).left}, nil
}

// SliceString returns the substring the span covers without constructing a
// new rope; faster than Slice(s).String()
func (r Rope) SliceString(s Span) (string, error) {
	if err := r.checkSpan(s); err != nil {
		return "", err
	}
	if s.Empty() {
		return "", nil
	}
	var b strings.Builder
	b.Grow(s.Len())
	r.ForEachSegment(s, func(seg string) {
		b.WriteString(seg)
	})
	return b.String(), nil
}

// ForEachSegment applies fn to each contiguous leaf substring the span covers,
// without copying
func (r Rope) ForEachSegment(s Span, fn func(string)) {
	s.From = max(s.From, 0)
	s.To = min(s.To, r.LenChars())
	if s.Empty() || s.From > s.To {
		return
	}
	forEachSegmentNode(r.root, s, fn)
}

// Insert returns a rope with text added at pos
func (r Rope) Insert(pos int, text string) (Rope, error) {
	if pos < 0 || pos > r.LenChars() {
		return Rope{}, fmt.Errorf("%w: %d", ErrRopeIndexOutOfRange, pos)
	}
	pair := splitRopeNode(r.root, pos)
	withText := concatRopeNode(ropePair{
		left:  pair.left,
		right: buildRopeNode(text),
	})
	return Rope{root: concatRopeNode(ropePair{
		left:  withText,
		right: pair.right,
	})}, nil
}

// Delete returns a rope without the characters the span covers
func (r Rope) Delete(s Span) (Rope, error) {
	if err := r.checkSpan(s); err != nil {
		return Rope{}, err
	}
	pair := splitRopeNode(r.root, s.From)
	return Rope{root: concatRopeNode(ropePair{
		left:  pair.left,
		right: splitRopeNode(pair.right, s.Len()).right,
	})}, nil
}

// CharAt returns the character at pos
func (r Rope) CharAt(pos int) (rune, error) {
	if pos < 0 || pos >= r.LenChars() {
		return 0, fmt.Errorf("%w: %d", ErrRopeIndexOutOfRange, pos)
	}
	return charAtRopeNode(r.root, pos), nil
}

// Line returns the line's text, including its ending
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
	return r.Slice(Span{From: from, To: to})
}

// LineToChar returns the position where the line starts
func (r Rope) LineToChar(line int) (int, error) {
	if line < 0 || line >= r.LenLines() {
		return 0, fmt.Errorf("%w: %d", ErrRopeLineOutOfRange, line)
	}
	if line == 0 {
		return 0, nil
	}
	return lineToCharRopeNode(r.root, line), nil
}

// CharToLine returns the line containing pos
func (r Rope) CharToLine(pos int) (int, error) {
	if pos < 0 || pos > r.LenChars() {
		return 0, fmt.Errorf("%w: %d", ErrRopeIndexOutOfRange, pos)
	}
	return charToLineRopeNode(r.root, pos), nil
}

// LineEndCharIndex returns the position after the line's last character,
// excluding its ending
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
	tail, err := r.SliceString(Span{From: max(end-2, start), To: end})
	if err != nil {
		return end, nil
	}
	if e, ok := GetLineEndingOfString(tail); ok {
		end -= len(e)
	}
	return end, nil
}

func (r Rope) checkSpan(s Span) error {
	if s.From < 0 || s.To < s.From || s.To > r.LenChars() {
		return fmt.Errorf("%w: %d..%d", ErrRopeIndexOutOfRange, s.From, s.To)
	}
	return nil
}
