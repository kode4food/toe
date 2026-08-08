package core

import (
	"fmt"
	"unicode/utf8"
)

// Apply runs the change set against doc, returning the resulting rope
func (c ChangeSet) Apply(doc Rope) (Rope, error) {
	if doc.LenChars() != c.charCount {
		return Rope{}, fmt.Errorf("%w: %d != %d", ErrChangeSetLengthMismatch,
			doc.LenChars(), c.charCount)
	}
	out := doc
	pos := 0
	for _, op := range c.ops {
		switch op.kind {
		case OperationRetain:
			pos += op.charCount
		case OperationDelete:
			next, err := out.Delete(Span{
				From: pos,
				To:   pos + op.charCount,
			})
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
			pos += utf8.RuneCountInString(op.text)
		}
	}
	return out, nil
}

// Invert returns the change set undoing this one, given the document it was
// built against
func (c ChangeSet) Invert(original Rope) (ChangeSet, error) {
	if original.LenChars() != c.charCount {
		return ChangeSet{}, fmt.Errorf("%w: %d != %d",
			ErrChangeSetLengthMismatch, original.LenChars(), c.charCount)
	}
	out := ChangeSet{}
	pos := 0
	for _, op := range c.ops {
		switch op.kind {
		case OperationRetain:
			out = out.retain(op.charCount)
			pos += op.charCount
		case OperationDelete:
			text, err := original.Slice(Span{From: pos, To: pos + op.charCount})
			if err != nil {
				return ChangeSet{}, err
			}
			out = out.insert(text.String())
			pos += op.charCount
		case OperationInsert:
			out = out.delete(utf8.RuneCountInString(op.text))
		}
	}
	return out, nil
}

// MapPos rebases a position onto the resulting document; assoc decides which
// side of an insertion at pos it lands on
func (c ChangeSet) MapPos(pos int, assoc Assoc) (int, error) {
	if pos < 0 || pos > c.charCount {
		return 0, fmt.Errorf("%w: %d", ErrRopeIndexOutOfRange, pos)
	}
	oldPos := 0
	newPos := 0
	for i, op := range c.ops {
		switch op.kind {
		case OperationRetain:
			oldEnd := oldPos + op.charCount
			if pos < oldEnd || (pos == oldEnd && i == len(c.ops)-1) {
				return newPos + (pos - oldPos), nil
			}
			oldPos = oldEnd
			newPos += op.charCount
		case OperationDelete:
			oldEnd := oldPos + op.charCount
			if pos < oldEnd {
				return newPos, nil
			}
			oldPos = oldEnd
		case OperationInsert:
			if pos == oldPos {
				return newPos + assoc.insertOffset(op.text), nil
			}
			newPos += utf8.RuneCountInString(op.text)
		}
	}
	return newPos, nil
}

// MapRange rebases both ends of a range onto the resulting document
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
