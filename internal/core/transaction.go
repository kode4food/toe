package core

// Transaction is an undoable change unit with optional selection state
type Transaction struct {
	changes   ChangeSet
	selection *Selection
}

// NewTransaction returns an empty transaction sized for doc
func NewTransaction(doc Rope) Transaction {
	return Transaction{changes: NewChangeSet(doc)}
}

// Changes is the edit the transaction applies
func (t Transaction) Changes() ChangeSet {
	return t.changes
}

// Selection is the selection to adopt afterwards, nil to keep the current
func (t Transaction) Selection() *Selection {
	return t.selection
}

// Apply runs the transaction's changes against doc
func (t Transaction) Apply(doc Rope) (Rope, error) {
	if t.changes.IsEmpty() {
		return doc, nil
	}
	return t.changes.Apply(doc)
}

// Invert returns the transaction undoing this one, given the document it was
// built against. The result carries no selection
func (t Transaction) Invert(original Rope) (Transaction, error) {
	cs, err := t.changes.Invert(original)
	if err != nil {
		return Transaction{}, err
	}
	return Transaction{changes: cs}, nil
}

// Compose returns a transaction equivalent to applying this one then other,
// keeping other's selection
func (t Transaction) Compose(other Transaction) Transaction {
	return Transaction{
		changes:   t.changes.Compose(other.changes),
		selection: other.selection,
	}
}

// WithChanges returns a copy carrying cs
func (t Transaction) WithChanges(cs ChangeSet) Transaction {
	t.changes = cs
	return t
}

// WithSelection returns a copy that adopts s once applied
func (t Transaction) WithSelection(s Selection) Transaction {
	t.selection = &s
	return t
}
