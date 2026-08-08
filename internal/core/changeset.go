package core

import (
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"
)

type (
	// ChangeSet is an ordered set of edits over a document snapshot
	ChangeSet struct {
		ops            []Operation
		charCount      int
		charCountAfter int
	}

	// Operation is one piece of a change set
	Operation struct {
		kind      OperationKind
		charCount int
		text      string
	}

	// Change describes a replacement over a character span
	Change struct {
		Span
		text string
	}

	// OperationKind identifies the operation variant
	OperationKind int

	// Assoc controls which side of an edit a mapped position sticks to
	Assoc int
)

const (
	OperationRetain OperationKind = iota + 1
	OperationDelete
	OperationInsert
)

const (
	AssocBefore Assoc = iota + 1
	AssocAfter
	AssocAfterWord
	AssocBeforeWord
	AssocBeforeSticky
	AssocAfterSticky
)

var (
	ErrChangeSetLengthMismatch = errors.New("change set length mismatch")
	ErrChangeOutOfRange        = errors.New("change out of range")
	ErrChangeOrder             = errors.New("change order invalid")
)

// NewChangeSet returns an empty change set sized for doc
func NewChangeSet(doc Rope) ChangeSet {
	n := doc.LenChars()
	return ChangeSet{charCount: n, charCountAfter: n}
}

// Text returns replacement text, or empty string for a deletion
func (c Change) Text() string {
	return c.text
}

// TextChange replaces the characters the span covers with text
func TextChange(s Span, text string) Change {
	return Change{Span: s, text: text}
}

// DeleteChange removes the characters the span covers
func DeleteChange(s Span) Change {
	return Change{Span: s}
}

// NewChangeSetFromChanges builds a change set from non-overlapping changes,
// which must be ordered by position
func NewChangeSetFromChanges(doc Rope, changes []Change) (ChangeSet, error) {
	cs := ChangeSet{}
	last := 0
	n := doc.LenChars()
	for _, c := range changes {
		if c.From < last || c.To < c.From {
			return ChangeSet{}, fmt.Errorf("%w: %d..%d", ErrChangeOrder,
				c.From, c.To)
		}
		if c.To > n {
			return ChangeSet{}, fmt.Errorf("%w: %d..%d", ErrChangeOutOfRange,
				c.From, c.To)
		}
		cs = cs.retain(c.From - last)
		span := c.To - c.From
		if c.text != "" {
			cs = cs.insert(c.text)
		}
		cs = cs.delete(span)
		last = c.To
	}
	return cs.retain(n - last), nil
}

// Kind reports whether the operation retains, deletes, or inserts
func (o Operation) Kind() OperationKind {
	return o.kind
}

// LenChars is the character count the operation inserts or spans
func (o Operation) LenChars() int {
	if o.kind == OperationInsert {
		return utf8.RuneCountInString(o.text)
	}
	return o.charCount
}

// Text is the inserted text, empty for retain and delete
func (o Operation) Text() string {
	return o.text
}

// Sticky reports whether a position stays put across an insertion at it
func (a Assoc) Sticky() bool {
	return a == AssocBeforeSticky || a == AssocAfterSticky
}

// Operations returns a copy of the operation sequence
func (c ChangeSet) Operations() []Operation {
	return slices.Clone(c.ops)
}

// Changes returns the operations rebuilt as position-based edits
func (c ChangeSet) Changes() []Change {
	var out []Change
	pos := 0
	for i := 0; i < len(c.ops); i++ {
		op := c.ops[i]
		switch op.kind {
		case OperationRetain:
			pos += op.charCount
		case OperationDelete:
			out = append(out, DeleteChange(Span{
				From: pos,
				To:   pos + op.charCount,
			}))
			pos += op.charCount
		case OperationInsert:
			from := pos
			to := from
			if i+1 < len(c.ops) && c.ops[i+1].kind == OperationDelete {
				to = from + c.ops[i+1].charCount
				i++
				pos = to
			}
			out = append(out, TextChange(Span{From: from, To: to}, op.text))
		}
	}
	return out
}

// Len is the document length the change set expects as input
func (c ChangeSet) Len() int {
	return c.charCount
}

// LenAfter is the document length the change set produces
func (c ChangeSet) LenAfter() int {
	return c.charCountAfter
}

// Empty reports whether the change set would leave a document unaltered
func (c ChangeSet) Empty() bool {
	return len(c.ops) == 0 ||
		(len(c.ops) == 1 && c.ops[0].kind == OperationRetain &&
			c.ops[0].charCount == c.charCount)
}

func (a Assoc) insertOffset(s string) int {
	chars := utf8.RuneCountInString(s)
	switch a {
	case AssocAfter, AssocAfterSticky:
		return chars
	case AssocAfterWord:
		return countWordPrefix(s)
	case AssocBeforeWord:
		return chars - countWordSuffix(s)
	default:
		return 0
	}
}

func (c ChangeSet) retain(n int) ChangeSet {
	return c.appendCountOp(n, OperationRetain, true)
}

func (c ChangeSet) delete(n int) ChangeSet {
	return c.appendCountOp(n, OperationDelete, false)
}

func (c ChangeSet) appendCountOp(
	n int, kind OperationKind, updateLenAfter bool,
) ChangeSet {
	if n == 0 {
		return c
	}
	c.charCount += n
	if updateLenAfter {
		c.charCountAfter += n
	}
	if len(c.ops) > 0 && c.ops[len(c.ops)-1].kind == kind {
		c.ops[len(c.ops)-1].charCount += n
		return c
	}
	c.ops = append(c.ops, Operation{kind: kind, charCount: n})
	return c
}

func (c ChangeSet) insert(text string) ChangeSet {
	if text == "" {
		return c
	}
	c.charCountAfter += utf8.RuneCountInString(text)
	if out, ok := c.mergeInsert(text); ok {
		return out
	}
	c.ops = append(c.ops, Operation{kind: OperationInsert, text: text})
	return c
}

func (c ChangeSet) mergeInsert(text string) (ChangeSet, bool) {
	if len(c.ops) == 0 {
		return c, false
	}
	last := c.ops[len(c.ops)-1]
	if last.kind == OperationInsert {
		c.ops[len(c.ops)-1].text += text
		return c, true
	}
	if last.kind != OperationDelete {
		return c, false
	}
	if len(c.ops) > 1 && c.ops[len(c.ops)-2].kind == OperationInsert {
		c.ops[len(c.ops)-2].text += text
		return c, true
	}
	c.ops[len(c.ops)-1] = Operation{kind: OperationInsert, text: text}
	c.ops = append(c.ops, last)
	return c, true
}

func countWordPrefix(s string) int {
	n := 0
	for _, ch := range s {
		if !CharIsWord(ch) {
			return n
		}
		n++
	}
	return n
}

func countWordSuffix(s string) int {
	r := []rune(s)
	n := 0
	for i := len(r) - 1; i >= 0; i-- {
		if !CharIsWord(r[i]) {
			return n
		}
		n++
	}
	return n
}

func runeDropPrefix(s string, n int) string {
	return runeSplitAt(s, n).after
}

func runeSplitAt(s string, n int) stringSplit {
	for i := range s {
		if n == 0 {
			return stringSplit{before: s[:i], after: s[i:]}
		}
		n--
	}
	return stringSplit{before: s}
}
