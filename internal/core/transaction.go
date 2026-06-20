package core

import (
	"errors"
	"fmt"
	"slices"
)

type (
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

	// Operation is one piece of a change set
	Operation struct {
		kind OperationKind
		n    int
		text string
	}

	// OperationKind identifies the operation variant
	OperationKind int

	// Assoc controls which side of an edit a mapped position sticks to
	Assoc int

	// ChangeSet is an ordered set of edits over a document snapshot
	ChangeSet struct {
		ops      []Operation
		len      int
		lenAfter int
	}

	// Transaction is an undoable change unit with optional selection state
	Transaction struct {
		changes   ChangeSet
		selection *Selection
	}
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

func NewTransaction(doc Rope) Transaction {
	return Transaction{changes: NewChangeSet(doc)}
}

// Text returns replacement text, or empty string for a deletion
func (c Change) Text() string { return c.text }

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
		return runeLen(o.text)
	}
	return o.n
}

func (o Operation) Text() string {
	return o.text
}

func (a Assoc) Sticky() bool {
	return a == AssocBeforeSticky || a == AssocAfterSticky
}

func (a Assoc) insertOffset(s string) int {
	chars := runeLen(s)
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

func (c ChangeSet) Apply(doc Rope) (Rope, error) {
	if doc.LenChars() != c.len {
		return Rope{}, fmt.Errorf("%w: %d != %d", ErrChangeSetLengthMismatch,
			doc.LenChars(), c.len)
	}
	out := doc
	pos := 0
	for _, op := range c.ops {
		switch op.kind {
		case OperationRetain:
			pos += op.n
		case OperationDelete:
			next, err := out.Delete(pos, pos+op.n)
			if err != nil {
				return Rope{}, err
			}
			out = next
		case OperationInsert:
			next, err := out.Insert(pos, op.text)
			if err != nil {
				return Rope{}, err
			}
			out = next
			pos += runeLen(op.text)
		}
	}
	return out, nil
}

func (c ChangeSet) Invert(original Rope) (ChangeSet, error) {
	if original.LenChars() != c.len {
		return ChangeSet{}, fmt.Errorf("%w: %d != %d",
			ErrChangeSetLengthMismatch, original.LenChars(), c.len)
	}
	out := ChangeSet{}
	pos := 0
	for _, op := range c.ops {
		switch op.kind {
		case OperationRetain:
			out = out.retain(op.n)
			pos += op.n
		case OperationDelete:
			text, err := original.Slice(pos, pos+op.n)
			if err != nil {
				return ChangeSet{}, err
			}
			out = out.insert(text.String())
			pos += op.n
		case OperationInsert:
			out = out.delete(runeLen(op.text))
		}
	}
	return out, nil
}

// Compose combines two changesets: if c transforms docA→docB and other
// transforms docB→docC, the result transforms docA→docC
func (c ChangeSet) Compose(other ChangeSet) ChangeSet {
	if len(c.ops) == 0 {
		return other
	}
	if len(other.ops) == 0 {
		return c
	}

	out := ChangeSet{}

	// Iterator state: index into ops slice and remaining amount/text within
	// the current op (for Retain/Delete: remaining count; for Insert:
	// remaining string suffix)
	ai, bi := 0, 0
	var aRem int    // remaining chars in current Retain/Delete op from c
	var bRem int    // remaining chars in current Retain/Delete op from other
	var aStr string // remaining text in current Insert op from c
	var bStr string // remaining text in current Insert op from other
	aKind := OperationKind(0)
	bKind := OperationKind(0)

	loadA := func() {
		if ai < len(c.ops) {
			op := c.ops[ai]
			ai++
			aKind = op.kind
			if op.kind == OperationInsert {
				aStr = op.text
			} else {
				aRem = op.n
			}
		} else {
			aKind = 0
		}
	}
	loadB := func() {
		if bi < len(other.ops) {
			op := other.ops[bi]
			bi++
			bKind = op.kind
			if op.kind == OperationInsert {
				bStr = op.text
			} else {
				bRem = op.n
			}
		} else {
			bKind = 0
		}
	}

	loadA()
	loadB()

	for {
		// Rule 1: Delete in A — emit delete, advance A only (b unchanged)
		if aKind == OperationDelete {
			out = out.delete(aRem)
			loadA()
			continue
		}
		// Rule 2: Insert in B — emit insert, advance B only (a unchanged)
		if bKind == OperationInsert {
			out = out.insert(bStr)
			loadB()
			continue
		}
		// Both exhausted
		if aKind == 0 && bKind == 0 {
			break
		}

		switch {
		// (Retain, Retain)
		case aKind == OperationRetain && bKind == OperationRetain:
			if aRem < bRem {
				out = out.retain(aRem)
				bRem -= aRem
				loadA()
			} else if aRem == bRem {
				out = out.retain(aRem)
				loadA()
				loadB()
			} else {
				out = out.retain(bRem)
				aRem -= bRem
				loadB()
			}

		// (Insert in A, Delete in B) — cancel out
		case aKind == OperationInsert && bKind == OperationDelete:
			aLen := runeLen(aStr)
			if aLen < bRem {
				bRem -= aLen
				loadA()
			} else if aLen == bRem {
				loadA()
				loadB()
			} else {
				// bRem < aLen: trim the front of aStr by bRem runes
				aStr = runeDropPrefix(aStr, bRem)
				loadB()
			}

		// (Insert in A, Retain in B) — emit prefix of insert
		case aKind == OperationInsert && bKind == OperationRetain:
			aLen := runeLen(aStr)
			if aLen < bRem {
				out = out.insert(aStr)
				bRem -= aLen
				loadA()
			} else if aLen == bRem {
				out = out.insert(aStr)
				loadA()
				loadB()
			} else {
				before, after := runeSplitAt(aStr, bRem)
				out = out.insert(before)
				aStr = after
				loadB()
			}

		// (Retain, Delete)
		case aKind == OperationRetain && bKind == OperationDelete:
			if aRem < bRem {
				out = out.delete(aRem)
				bRem -= aRem
				loadA()
			} else if aRem == bRem {
				out = out.delete(aRem)
				loadA()
				loadB()
			} else {
				out = out.delete(bRem)
				aRem -= bRem
				loadB()
			}
		}
	}

	return out
}

func (c ChangeSet) MapPos(pos int, assoc Assoc) (int, error) {
	if pos < 0 || pos > c.len {
		return 0, fmt.Errorf("%w: %d", ErrRopeIndexOutOfRange, pos)
	}
	oldPos := 0
	newPos := 0
	for i, op := range c.ops {
		switch op.kind {
		case OperationRetain:
			oldEnd := oldPos + op.n
			if pos < oldEnd || (pos == oldEnd && i == len(c.ops)-1) {
				return newPos + (pos - oldPos), nil
			}
			oldPos = oldEnd
			newPos += op.n
		case OperationDelete:
			oldEnd := oldPos + op.n
			if pos < oldEnd {
				return newPos, nil
			}
			oldPos = oldEnd
		case OperationInsert:
			if pos == oldPos {
				return newPos + assoc.insertOffset(op.text), nil
			}
			newPos += runeLen(op.text)
		}
	}
	return newPos, nil
}

func (c ChangeSet) MapRange(r Range) (Range, error) {
	var a Assoc
	var h Assoc
	if r.Anchor == r.Head {
		a = AssocAfterSticky
		h = AssocAfterSticky
	} else if r.Anchor < r.Head {
		a = AssocAfterSticky
		h = AssocBeforeSticky
	} else {
		a = AssocBeforeSticky
		h = AssocAfterSticky
	}
	anchor, err := c.MapPos(r.Anchor, a)
	if err != nil {
		return Range{}, err
	}
	head, err := c.MapPos(r.Head, h)
	if err != nil {
		return Range{}, err
	}
	return Range{Anchor: anchor, Head: head}, nil
}

func (t Transaction) Changes() ChangeSet {
	return t.changes
}

func (t Transaction) Selection() *Selection {
	return t.selection
}

func (t Transaction) Apply(doc Rope) (Rope, error) {
	if t.changes.Empty() {
		return doc, nil
	}
	return t.changes.Apply(doc)
}

func (t Transaction) Invert(original Rope) (Transaction, error) {
	cs, err := t.changes.Invert(original)
	if err != nil {
		return Transaction{}, err
	}
	return Transaction{changes: cs}, nil
}

func (t Transaction) Compose(other Transaction) Transaction {
	return Transaction{
		changes:   t.changes.Compose(other.changes),
		selection: other.selection,
	}
}

func (t Transaction) WithChanges(cs ChangeSet) Transaction {
	t.changes = cs
	return t
}

func (t Transaction) WithSelection(s Selection) Transaction {
	t.selection = &s
	return t
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
	c.lenAfter += runeLen(text)
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
	for i := range s {
		if n == 0 {
			return s[i:]
		}
		n--
	}
	return ""
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
