package view

import "github.com/kode4food/toe/internal/core"

// Apply applies a transaction to the focused document for the focused view
func (e *Editor) Apply(tx core.Transaction) error {
	v, ok := e.FocusedView()
	if !ok {
		return ErrNoView
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return ErrNoDocument
	}
	if v.Mode() == ModeInsert {
		doc.BeginInsertGroup(v.ID())
	}
	return doc.Apply(tx, v.ID())
}

// CommitInsertHistory flushes any pending insert-mode history accumulation on
// the focused document into a single history revision
func (e *Editor) CommitInsertHistory() {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return
	}
	doc.CommitInsertHistory(v.ID())
}

// Undo reverts one history step in the focused document
func (e *Editor) Undo() bool {
	v, ok := e.FocusedView()
	if !ok {
		return false
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return false
	}
	return doc.Undo(v.ID())
}

// Redo reapplies one reverted step in the focused document
func (e *Editor) Redo() bool {
	v, ok := e.FocusedView()
	if !ok {
		return false
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return false
	}
	return doc.Redo(v.ID())
}

// Earlier navigates history backward by the given UndoKind
func (e *Editor) Earlier(kind core.UndoKind) bool {
	v, ok := e.FocusedView()
	if !ok {
		return false
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return false
	}
	txns := doc.history.Earlier(kind)
	for _, tx := range txns {
		inv, err := tx.Invert(doc.text)
		if err != nil {
			return false
		}
		newText, err := inv.Apply(doc.text)
		if err != nil {
			return false
		}
		doc.text = newText
		if txSel := tx.Selection(); txSel != nil {
			doc.SetSelectionFor(v.ID(), *txSel)
		}
	}
	doc.modified = len(txns) > 0 || doc.modified
	return len(txns) > 0
}

// Later navigates history forward by the given UndoKind
func (e *Editor) Later(kind core.UndoKind) bool {
	v, ok := e.FocusedView()
	if !ok {
		return false
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return false
	}
	txns := doc.history.Later(kind)
	for _, tx := range txns {
		newText, err := tx.Apply(doc.text)
		if err != nil {
			return false
		}
		doc.text = newText
		if txSel := tx.Selection(); txSel != nil {
			doc.SetSelectionFor(v.ID(), *txSel)
		}
	}
	doc.modified = len(txns) > 0 || doc.modified
	return len(txns) > 0
}
