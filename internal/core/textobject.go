package core

// TextObjectKind controls whether a selection covers the delimiter or just
// the content inside it
type TextObjectKind int

const (
	TextObjectAround TextObjectKind = iota + 1
	TextObjectInside
)

// TextObjectWord selects the word under or adjacent to the cursor long=true
// uses long-word (WORD) semantics: only whitespace is a boundary
// around=TextObjectAround includes trailing (or leading) whitespace
func TextObjectWord(doc Rope, r Range, kind TextObjectKind, long bool) Range {
	pos := r.Cursor(doc)
	start := wordBoundaryBackward(doc, pos, long)
	var end int
	if ch, err := doc.CharAt(pos); err != nil ||
		CategorizeChar(ch) == CharCategoryWhitespace ||
		CategorizeChar(ch) == CharCategoryEOL {
		end = pos
	} else {
		end = wordBoundaryForward(doc, pos+1, long)
	}
	if start == end {
		return NewRange(start, end)
	}
	if kind == TextObjectInside {
		return NewRange(start, end)
	}
	// Around: include trailing whitespace, or leading if none on right
	wsRight := 0
	for i := end; i < doc.LenChars(); i++ {
		ch, err := doc.CharAt(i)
		if err != nil {
			break
		}
		if !CharIsWhitespace(ch) {
			break
		}
		wsRight++
	}
	if wsRight > 0 {
		return NewRange(start, end+wsRight)
	}
	wsLeft := 0
	for i := start - 1; i >= 0; i-- {
		ch, err := doc.CharAt(i)
		if err != nil {
			break
		}
		if !CharIsWhitespace(ch) {
			break
		}
		wsLeft++
	}
	return NewRange(start-wsLeft, end)
}

// wordBoundaryBackward returns the start of the word ending at pos
func wordBoundaryBackward(doc Rope, pos int, long bool) int {
	if pos == 0 {
		return 0
	}
	var prev CharCategory
	if pos >= doc.LenChars() {
		prev = CharCategoryWhitespace
	} else {
		ch, err := doc.CharAt(pos)
		if err != nil {
			return pos
		}
		prev = CategorizeChar(ch)
	}
	for i := pos - 1; i >= 0; i-- {
		ch, err := doc.CharAt(i)
		if err != nil {
			return pos
		}
		cat := CategorizeChar(ch)
		if cat == CharCategoryEOL || cat == CharCategoryWhitespace {
			return i + 1
		}
		if !long && cat != prev && i+1 != 0 && i+1 != doc.LenChars() {
			return i + 1
		}
		prev = cat
		pos = i
	}
	return 0
}

// wordBoundaryForward returns the end (exclusive) of the word starting at pos
func wordBoundaryForward(doc Rope, pos int, long bool) int {
	n := doc.LenChars()
	if pos == 0 {
		return 0
	}
	var prev CharCategory
	if pos-1 >= 0 {
		ch, err := doc.CharAt(pos - 1)
		if err == nil {
			prev = CategorizeChar(ch)
		} else {
			prev = CharCategoryWhitespace
		}
	}
	for i := pos; i < n; i++ {
		ch, err := doc.CharAt(i)
		if err != nil {
			return i
		}
		cat := CategorizeChar(ch)
		if cat == CharCategoryEOL || cat == CharCategoryWhitespace {
			return i
		}
		if !long && cat != prev && i != 0 && i != n {
			return i
		}
		prev = cat
	}
	return n
}
