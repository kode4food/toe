package core

import "unicode/utf8"

// composeCtx holds iterator state for ChangeSet.Compose
type composeCtx struct {
	a, b         []Operation
	ai, bi       int
	aKind, bKind OperationKind
	aRem, bRem   int
	aStr, bStr   string
	out          ChangeSet
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
	ctx := &composeCtx{a: c.ops, b: other.ops}
	ctx.loadA()
	ctx.loadB()
	return ctx.run()
}

func (c *composeCtx) loadA() {
	if c.ai < len(c.a) {
		op := c.a[c.ai]
		c.ai++
		c.aKind = op.kind
		if op.kind == OperationInsert {
			c.aStr = op.text
		} else {
			c.aRem = op.n
		}
	} else {
		c.aKind = 0
	}
}

func (c *composeCtx) loadB() {
	if c.bi < len(c.b) {
		op := c.b[c.bi]
		c.bi++
		c.bKind = op.kind
		if op.kind == OperationInsert {
			c.bStr = op.text
		} else {
			c.bRem = op.n
		}
	} else {
		c.bKind = 0
	}
}

// advancePair consumes min(aRem,bRem) chars, emitting a retain or delete op,
// and advances whichever iterator(s) are exhausted
func (c *composeCtx) advancePair(kind OperationKind) {
	if c.aRem < c.bRem {
		c.emitN(c.aRem, kind)
		c.bRem -= c.aRem
		c.loadA()
	} else if c.aRem == c.bRem {
		c.emitN(c.aRem, kind)
		c.loadA()
		c.loadB()
	} else {
		c.emitN(c.bRem, kind)
		c.aRem -= c.bRem
		c.loadB()
	}
}

func (c *composeCtx) emitN(n int, kind OperationKind) {
	if kind == OperationDelete {
		c.out = c.out.delete(n)
	} else {
		c.out = c.out.retain(n)
	}
}

// stepInsertDelete handles (Insert-A, Delete-B): the inserted text is consumed
// by the deletion; no output is emitted
func (c *composeCtx) stepInsertDelete() {
	aLen := utf8.RuneCountInString(c.aStr)
	if aLen < c.bRem {
		c.bRem -= aLen
		c.loadA()
	} else if aLen == c.bRem {
		c.loadA()
		c.loadB()
	} else {
		c.aStr = runeDropPrefix(c.aStr, c.bRem)
		c.loadB()
	}
}

// stepInsertRetain handles (Insert-A, Retain-B): emit the prefix of the insert
// that fits within the retain window
func (c *composeCtx) stepInsertRetain() {
	aLen := utf8.RuneCountInString(c.aStr)
	if aLen < c.bRem {
		c.out = c.out.insert(c.aStr)
		c.bRem -= aLen
		c.loadA()
	} else if aLen == c.bRem {
		c.out = c.out.insert(c.aStr)
		c.loadA()
		c.loadB()
	} else {
		before, after := runeSplitAt(c.aStr, c.bRem)
		c.out = c.out.insert(before)
		c.aStr = after
		c.loadB()
	}
}

func (c *composeCtx) run() ChangeSet {
	for {
		switch {
		case c.aKind == OperationDelete:
			c.out = c.out.delete(c.aRem)
			c.loadA()
		case c.bKind == OperationInsert:
			c.out = c.out.insert(c.bStr)
			c.loadB()
		case c.aKind == 0 && c.bKind == 0:
			return c.out
		case c.aKind == OperationRetain && c.bKind == OperationRetain:
			c.advancePair(OperationRetain)
		case c.aKind == OperationInsert && c.bKind == OperationDelete:
			c.stepInsertDelete()
		case c.aKind == OperationInsert && c.bKind == OperationRetain:
			c.stepInsertRetain()
		case c.aKind == OperationRetain && c.bKind == OperationDelete:
			c.advancePair(OperationDelete)
		}
	}
}
