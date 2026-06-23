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
		ops      []Operation
		len      int
		lenAfter int
	}

	// Operation is one piece of a change set
	Operation struct {
		kind OperationKind
		n    int
		text string
	}

	// Change describes a replacement over a character range
	Change struct {
		From int
		To   int
		text string
	}

	// Deletion describes a removed character range
	Deletion struct {
		From int
		To   int
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

func NewChangeSet(doc Rope) ChangeSet {
	n := doc.LenChars()
	return ChangeSet{len: n, lenAfter: n}
}

// Text returns replacement text, or empty string for a deletion
func (c Change) Text() string {
	return c.text
}

func TextChange(from, to int, text string) Change {
	return Change{From: from, To: to, text: text}
}

func DeleteChange(from, to int) Change {
	return Change{From: from, To: to}
}

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

func (o Operation) Kind() OperationKind {
	return o.kind
}

func (o Operation) LenChars() int {
	if o.kind == OperationInsert {
		return utf8.RuneCountInString(o.text)
	}
	return o.n
}

func (o Operation) Text() string {
	return o.text
}

func (a Assoc) Sticky() bool {
	return a == AssocBeforeSticky || a == AssocAfterSticky
}

func (c ChangeSet) Operations() []Operation {
	return slices.Clone(c.ops)
}

func (c ChangeSet) Changes() []Change {
	var out []Change
	pos := 0
	for i := 0; i < len(c.ops); i++ {
		op := c.ops[i]
		switch op.kind {
		case OperationRetain:
			pos += op.n
		case OperationDelete:
			out = append(out, DeleteChange(pos, pos+op.n))
			pos += op.n
		case OperationInsert:
			from := pos
			to := from
			if i+1 < len(c.ops) && c.ops[i+1].kind == OperationDelete {
				to = from + c.ops[i+1].n
				i++
				pos = to
			}
			out = append(out, TextChange(from, to, op.text))
		}
	}
	return out
}

func (c ChangeSet) Len() int {
	return c.len
}

func (c ChangeSet) LenAfter() int {
	return c.lenAfter
}

func (c ChangeSet) Empty() bool {
	return len(c.ops) == 0 ||
		(len(c.ops) == 1 && c.ops[0].kind == OperationRetain &&
			c.ops[0].n == c.len)
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
	c.len += n
	if updateLenAfter {
		c.lenAfter += n
	}
	if len(c.ops) > 0 && c.ops[len(c.ops)-1].kind == kind {
		c.ops[len(c.ops)-1].n += n
		return c
	}
	c.ops = append(c.ops, Operation{kind: kind, n: n})
	return c
}

func (c ChangeSet) insert(text string) ChangeSet {
	if text == "" {
		return c
	}
	c.lenAfter += utf8.RuneCountInString(text)
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
	_, after := runeSplitAt(s, n)
	return after
}

func runeSplitAt(s string, n int) (string, string) {
	for i := range s {
		if n == 0 {
			return s[:i], s[i:]
		}
		n--
	}
	return s, ""
}
