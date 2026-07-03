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
		e.documentChanged(doc, DocumentChange{
			Before:  before,
			Changes: changes,
		})
	}
	return nil
}

// ApplyToDocument applies a transaction without changing the focused view
func (e *Editor) ApplyToDocument(doc *Document, tx core.Transaction) error {
	if doc == nil {
		return ErrNoDocument
	}
	v, ok := e.viewForDocument(doc.ID())
	if !ok {
		v, ok = e.FocusedView()
		if !ok {
			return ErrNoView
		}
	}
	if v.DocID() == doc.ID() && v.Mode() == ModeInsert {
		doc.BeginInsertGroup(v.ID())
	}
	rev := doc.Revision()
	before := doc.Text()
	changes := tx.Changes()
	if err := doc.Apply(tx, v.ID()); err != nil {
		return err
	}
	if doc.Revision() != rev {
		e.documentChanged(doc, DocumentChange{
			Before:  before,
			Changes: changes,
		})
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
	doc.buf.Lock()
	before := doc.buf.text
	txns := doc.buf.history.Earlier(kind)
	for _, tx := range txns {
		inv, err := tx.Invert(doc.buf.text)
		if err != nil {
			doc.buf.Unlock()
			return false
		}
		newText, err := inv.Apply(doc.buf.text)
		if err != nil {
			doc.buf.Unlock()
			return false
		}
		doc.buf.text = newText
		if txSel := tx.Selection(); txSel != nil {
			doc.SetSelectionFor(v.ID(), *txSel)
		}
	}
	doc.buf.unsaved = len(txns) > 0 || doc.buf.unsaved
	if len(txns) == 0 {
		doc.buf.Unlock()
		return false
	}
	doc.buf.version++
	afterStr := doc.buf.text.String()
	doc.buf.Unlock()
	e.documentChanged(doc, wholeDocumentChange(before, afterStr))
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
	doc.buf.Lock()
	before := doc.buf.text
	txns := doc.buf.history.Later(kind)
	for _, tx := range txns {
		newText, err := tx.Apply(doc.buf.text)
		if err != nil {
			doc.buf.Unlock()
			return false
		}
		doc.buf.text = newText
		if txSel := tx.Selection(); txSel != nil {
			doc.SetSelectionFor(v.ID(), *txSel)
		}
	}
	doc.buf.unsaved = len(txns) > 0 || doc.buf.unsaved
	if len(txns) == 0 {
		doc.buf.Unlock()
		return false
	}
	doc.buf.version++
	afterStr := doc.buf.text.String()
	doc.buf.Unlock()
	e.documentChanged(doc, wholeDocumentChange(before, afterStr))
	return true
}

func (e *Editor) viewForDocument(id DocumentId) (*View, bool) {
	for _, v := range e.AllViews() {
		if v.DocID() == id {
			return v, true
		}
	}
	return nil, false
}
