package core

import (
	"fmt"
	"unicode/utf8"
)

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
			pos += utf8.RuneCountInString(op.text)
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
			out = out.delete(utf8.RuneCountInString(op.text))
		}
	}
	return out, nil
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
			newPos += utf8.RuneCountInString(op.text)
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
