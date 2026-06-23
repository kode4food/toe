package core

import (
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// TabWidthAt returns the visual width of a tab character at the given x
// position
func TabWidthAt(visualX, tabWidth int) int {
	return tabWidth - (visualX % tabWidth)
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
		pos += utf8.RuneCountInString(cl)
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
// positions for s, including a trailing sentinel equal to utf8.RuneCountInString(s)
func graphemeBoundaries(s string) []int {
	out := make([]int, 0, len(s)+1)
	pos := 0
	state := -1
	for s != "" {
		out = append(out, pos)
		cl, rest, _, newState := uniseg.FirstGraphemeClusterInString(s, state)
		pos += utf8.RuneCountInString(cl)
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
