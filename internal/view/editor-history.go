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
	rev := doc.Revision()
	before := doc.Text()
	changes := tx.Changes()
	if err := doc.Apply(tx, v.ID()); err != nil {
		return err
	}
	if doc.Revision() != rev {
		e.documentChanged(doc, newDocumentChange(before, changes))
	}
	return nil
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
	before := doc.Text()
	if !doc.Undo(v.ID()) {
		return false
	}
	e.documentChanged(doc, wholeDocumentChange(before, doc.Text().String()))
	return true
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
	before := doc.Text()
	if !doc.Redo(v.ID()) {
		return false
	}
	e.documentChanged(doc, wholeDocumentChange(before, doc.Text().String()))
	return true
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
	before := doc.text
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
	if len(txns) == 0 {
		return false
	}
	doc.version++
	e.documentChanged(doc, wholeDocumentChange(before, doc.text.String()))
	return true
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
	before := doc.text
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
	if len(txns) == 0 {
		return false
	}
	doc.version++
	e.documentChanged(doc, wholeDocumentChange(before, doc.text.String()))
	return true
}
