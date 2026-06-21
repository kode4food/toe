package core

// Transaction is an undoable change unit with optional selection state
type Transaction struct {
	changes   ChangeSet
	selection *Selection
}

func NewTransaction(doc Rope) Transaction {
	return Transaction{changes: NewChangeSet(doc)}
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
