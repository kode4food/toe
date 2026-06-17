package core

import (
	"github.com/rivo/uniseg"
)

type (
	// GraphemeStr is an immutable string slice representing one grapheme
	// cluster. Go strings are already immutable so no special type is needed
	GraphemeStr = string

	// GraphemeKind identifies the grapheme variant
	GraphemeKind int

	// Grapheme classifies a single grapheme cluster for rendering
	Grapheme struct {
		kind  GraphemeKind
		text  GraphemeStr
		width int
	}
)

const (
	GraphemeKindNewline GraphemeKind = iota + 1
	GraphemeKindTab
	GraphemeKindOther
)

// TabWidthAt returns the visual width of a tab character at the given x
// position
func TabWidthAt(visualX, tabWidth int) int {
	return tabWidth - (visualX % tabWidth)
}

// NewGrapheme constructs a Grapheme from a cluster string, visual x position,
// and tab width setting
func NewGrapheme(g GraphemeStr, visualX, tabWidth int) Grapheme {
	if StrIsLineEnding(g) {
		return Grapheme{kind: GraphemeKindNewline, width: 1}
	}
	if g == "\t" {
		w := TabWidthAt(visualX, tabWidth)
		return Grapheme{kind: GraphemeKindTab, text: g, width: w}
	}
	return Grapheme{kind: GraphemeKindOther, text: g, width: graphemeWidth(g)}
}

// NewDecorationGrapheme constructs an Other grapheme from a static decoration
// string
func NewDecorationGrapheme(g GraphemeStr) Grapheme {
	return NewGrapheme(g, 0, 4)
}

func (g *Grapheme) Kind() GraphemeKind {
	return g.kind
}

func (g *Grapheme) Text() GraphemeStr {
	return g.text
}

// Width returns the visual display width of the grapheme in terminal columns
func (g *Grapheme) Width() int {
	return g.width
}

// IsWhitespace reports whether this grapheme is whitespace
func (g *Grapheme) IsWhitespace() bool {
	if g.kind != GraphemeKindOther {
		return true
	}
	for _, ch := range g.text {
		return CharIsWhitespace(ch)
	}
	return false
}

// IsWordBoundary reports whether this grapheme begins a word boundary
func (g *Grapheme) IsWordBoundary() bool {
	if g.kind != GraphemeKindOther {
		return true
	}
	for _, ch := range g.text {
		return !CharIsWord(ch)
	}
	return false
}

// ChangePosition updates the visual width of a Tab grapheme when its column
// changes
func (g *Grapheme) ChangePosition(visualX, tabWidth int) {
	if g.kind == GraphemeKindTab {
		g.width = TabWidthAt(visualX, tabWidth)
	}
}

// NthPrevGraphemeBoundary returns the char index n grapheme clusters before
// charIdx
func NthPrevGraphemeBoundary(doc Rope, charIdx, n int) int {
	if charIdx == 0 || n == 0 {
		return charIdx
	}
	// Collect all grapheme boundary char indices from 0..charIdx
	s, err := doc.Slice(0, charIdx)
	if err != nil {
		return 0
	}
	boundaries := graphemeBoundaries(s.String())
	// boundaries[0] == 0, boundaries[len] == charIdx
	k := len(boundaries) - 1 - n
	if k < 0 {
		return 0
	}
	return boundaries[k]
}

// NthNextGraphemeBoundary returns the char index n grapheme clusters after
// charIdx
func NthNextGraphemeBoundary(doc Rope, charIdx, n int) int {
	total := doc.LenChars()
	if charIdx >= total || n == 0 {
		return charIdx
	}
	s, err := doc.Slice(charIdx, total)
	if err != nil {
		return total
	}
	rest := s.String()
	pos := charIdx
	state := -1
	for range n {
		cl, rem, _, newState := uniseg.FirstGraphemeClusterInString(rest, state)
		if cl == "" {
			break
		}
		pos += runeLen(cl)
		rest = rem
		state = newState
	}
	return pos
}

// PrevGraphemeBoundary returns the char index one grapheme cluster before
// charIdx
func PrevGraphemeBoundary(doc Rope, charIdx int) int {
	return NthPrevGraphemeBoundary(doc, charIdx, 1)
}

// NextGraphemeBoundary returns the char index one grapheme cluster after
// charIdx
func NextGraphemeBoundary(doc Rope, charIdx int) int {
	return NthNextGraphemeBoundary(doc, charIdx, 1)
}

// EnsureGraphemeBoundaryNext snaps charIdx to the next grapheme boundary if it
// is not already on one
func EnsureGraphemeBoundaryNext(doc Rope, charIdx int) int {
	if charIdx == 0 {
		return charIdx
	}
	return NextGraphemeBoundary(doc, charIdx-1)
}

// EnsureGraphemeBoundaryPrev snaps charIdx to the previous grapheme boundary if
// it is not already on one
func EnsureGraphemeBoundaryPrev(doc Rope, charIdx int) int {
	total := doc.LenChars()
	if charIdx == total {
		return charIdx
	}
	return PrevGraphemeBoundary(doc, charIdx+1)
}

// graphemeBoundaries returns the list of char-indexed grapheme cluster start
// positions for s, including a trailing sentinel equal to runeLen(s)
func graphemeBoundaries(s string) []int {
	out := make([]int, 0, len(s)+1)
	pos := 0
	state := -1
	for s != "" {
		out = append(out, pos)
		cl, rest, _, newState := uniseg.FirstGraphemeClusterInString(s, state)
		pos += runeLen(cl)
		s = rest
		state = newState
	}
	out = append(out, pos)
	return out
}

// graphemeWidth returns the display width of a grapheme cluster string
func graphemeWidth(g string) int {
	if len(g) == 0 {
		return 0
	}
	if g[0] <= 127 {
		return 1
	}
	_, _, w, _ := uniseg.FirstGraphemeClusterInString(g, -1)
	if w < 1 {
		return 1
	}
	return w
}
