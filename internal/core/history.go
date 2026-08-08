package core

type (
	// State is the document and selection at a point before a transaction
	State struct {
		Doc       Rope
		Selection Selection
	}

	// History stores committed document revisions
	History struct {
		revisions   []revision
		current     int
		lastEditPos int
	}

	// UndoKind selects how many steps history navigation travels
	UndoKind struct {
		steps int
	}

	revision struct {
		parent      int
		lastChild   int
		transaction Transaction
		inversion   Transaction
	}
)

// NewHistory returns an empty history positioned at the root revision
func NewHistory() History {
	root := revision{
		parent:      0,
		lastChild:   -1,
		transaction: NewTransaction(NewRope("")),
		inversion:   NewTransaction(NewRope("")),
	}
	return History{revisions: []revision{root}}
}

// CurrentRevision is the index of the revision in effect
func (h *History) CurrentRevision() int {
	return h.current
}

// AtRoot reports whether there is nothing left to undo
func (h *History) AtRoot() bool {
	return h.current == 0
}

// CommitRevision records tx as a new revision branching from the current one,
// storing its inverse for undo
func (h *History) CommitRevision(tx Transaction, st State) error {
	inv, err := tx.Invert(st.Doc)
	if err != nil {
		return err
	}
	inv = inv.WithSelection(st.Selection)
	// Record position of the first change for goto_last_modification
	if ch := tx.Changes(); len(ch.Changes()) > 0 {
		h.lastEditPos = ch.Changes()[0].From
	}
	next := len(h.revisions)
	h.revisions[h.current].lastChild = next
	h.revisions = append(h.revisions, revision{
		parent:      h.current,
		lastChild:   -1,
		transaction: tx,
		inversion:   inv,
	})
	h.current = next
	return nil
}

// LastEditPos returns the char offset of the most recently committed change
func (h *History) LastEditPos() int {
	return h.lastEditPos
}

// Undo steps back one revision, returning the transaction to apply
func (h *History) Undo() (Transaction, bool) {
	if h.AtRoot() {
		return Transaction{}, false
	}
	rev := h.revisions[h.current]
	h.current = rev.parent
	return rev.inversion, true
}

// Redo steps forward one revision, returning the transaction to apply
func (h *History) Redo() (Transaction, bool) {
	rev := h.revisions[h.current]
	if rev.lastChild < 0 {
		return Transaction{}, false
	}
	h.current = rev.lastChild
	return h.revisions[h.current].transaction, true
}

// Earlier returns the transactions that walk back by kind, without applying
// them
func (h *History) Earlier(kind UndoKind) []Transaction {
	return h.jumpBackward(kind.steps)
}

// Later returns the transactions that walk forward by kind, without applying
// them
func (h *History) Later(kind UndoKind) []Transaction {
	return h.jumpForward(kind.steps)
}

func (h *History) lowestCommonAncestor(pair Span) int {
	a := pair.From
	b := pair.To
	seen := map[int]bool{}
	for {
		seen[a] = true
		if seen[b] {
			return b
		}
		b = h.revisions[b].parent
		if seen[a] && seen[b] {
			return b
		}
		a = h.revisions[a].parent
	}
}

func (h *History) pathUp(pair Span) []int {
	n := pair.From
	var path []int
	for n != pair.To {
		path = append(path, n)
		n = h.revisions[n].parent
	}
	return path
}

func (h *History) jumpTo(to int) []Transaction {
	lca := h.lowestCommonAncestor(Span{From: h.current, To: to})
	up := h.pathUp(Span{From: h.current, To: lca})
	down := h.pathUp(Span{From: to, To: lca})
	h.current = to

	txns := make([]Transaction, 0, len(up)+len(down))
	for _, n := range up {
		txns = append(txns, h.revisions[n].inversion)
	}
	for i := len(down) - 1; i >= 0; i-- {
		txns = append(txns, h.revisions[down[i]].transaction)
	}
	return txns
}

func (h *History) jumpBackward(delta int) []Transaction {
	if delta < 0 {
		delta = 0
	}
	return h.jumpTo(max(h.current-delta, 0))
}

func (h *History) jumpForward(delta int) []Transaction {
	if delta < 0 {
		delta = 0
	}
	to := h.current + delta
	if to >= len(h.revisions) {
		to = len(h.revisions) - 1
	}
	return h.jumpTo(to)
}

// UndoSteps counts revisions rather than elapsed time
func UndoSteps(n int) UndoKind {
	return UndoKind{steps: n}
}
