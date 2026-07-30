package view

import "github.com/kode4food/toe/internal/core"

type historyOp func(*Document, Id) bool

// Apply applies a transaction to the focused document for the focused view
func (e *Editor) Apply(tx core.Transaction) error {
	v := e.FocusedView()
	if v == nil {
		return ErrNoView
	}
	doc := e.Document(v.DocID())
	if doc == nil {
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
	v := e.viewForDocument(doc.ID())
	if v == nil {
		v = e.FocusedView()
		if v == nil {
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
	v := e.FocusedView()
	if v == nil {
		return
	}
	if doc := e.Document(v.DocID()); doc != nil {
		doc.CommitInsertHistory(v.ID())
	}
}

// Undo reverts one history step in the focused document
func (e *Editor) Undo() bool { return e.applyUndoRedo((*Document).Undo) }

// Redo reapplies one reverted step in the focused document
func (e *Editor) Redo() bool { return e.applyUndoRedo((*Document).Redo) }

// Earlier navigates history backward by the given UndoKind
func (e *Editor) Earlier(kind core.UndoKind) bool {
	v := e.FocusedView()
	if v == nil {
		return false
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return false
	}
	doc.content.Lock()
	before := doc.content.text
	txns := doc.edits.history.Earlier(kind)
	for _, tx := range txns {
		newText, err := tx.Apply(doc.content.text)
		if err != nil {
			doc.content.Unlock()
			return false
		}
		doc.content.text = newText
		if txSel := tx.Selection(); txSel != nil {
			doc.SetSelectionFor(v.ID(), *txSel)
		}
	}
	if len(txns) == 0 {
		doc.content.Unlock()
		return false
	}
	doc.content.version++
	afterStr := doc.content.text.String()
	doc.content.Unlock()
	doc.MarkDirty()
	e.documentChanged(doc, wholeDocumentChange(before, afterStr))
	return true
}

// Later navigates history forward by the given UndoKind
func (e *Editor) Later(kind core.UndoKind) bool {
	v := e.FocusedView()
	if v == nil {
		return false
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return false
	}
	doc.content.Lock()
	before := doc.content.text
	txns := doc.edits.history.Later(kind)
	for _, tx := range txns {
		newText, err := tx.Apply(doc.content.text)
		if err != nil {
			doc.content.Unlock()
			return false
		}
		doc.content.text = newText
		if txSel := tx.Selection(); txSel != nil {
			doc.SetSelectionFor(v.ID(), *txSel)
		}
	}
	if len(txns) == 0 {
		doc.content.Unlock()
		return false
	}
	doc.content.version++
	afterStr := doc.content.text.String()
	doc.content.Unlock()
	doc.MarkDirty()
	e.documentChanged(doc, wholeDocumentChange(before, afterStr))
	return true
}

func (e *Editor) applyUndoRedo(fn historyOp) bool {
	v := e.FocusedView()
	if v == nil {
		return false
	}
	doc := e.Document(v.DocID())
	if doc == nil {
		return false
	}
	before := doc.Text()
	if !fn(doc, v.ID()) {
		return false
	}
	e.documentChanged(doc, wholeDocumentChange(before, doc.Text().String()))
	return true
}

func (e *Editor) viewForDocument(id DocumentId) *View {
	for _, v := range e.AllViews() {
		if v.DocID() == id {
			return v
		}
	}
	return nil
}
