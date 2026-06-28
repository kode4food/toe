package view

import "github.com/kode4food/toe/internal/core"

// DocumentChange describes an editor text change for document observers
type DocumentChange struct {
	Before  core.Rope
	Changes core.ChangeSet
}

// DocumentObserver receives editor document lifecycle notifications
type DocumentObserver interface {
	DocumentOpened(*Document)
	DocumentChanged(*Document, DocumentChange)
	DocumentSaved(*Document)
	DocumentClosed(*Document)
}

// SetDocumentObserver installs a document lifecycle observer
func (e *Editor) SetDocumentObserver(o DocumentObserver) {
	e.docObserver = o
}

func (e *Editor) documentOpened(doc *Document) {
	if e.docObserver != nil {
		e.docObserver.DocumentOpened(doc)
	}
}

func (e *Editor) documentChanged(doc *Document, change DocumentChange) {
	if e.docObserver != nil {
		e.docObserver.DocumentChanged(doc, change)
	}
}

func (e *Editor) documentSaved(doc *Document) {
	if e.docObserver != nil {
		e.docObserver.DocumentSaved(doc)
	}
}

func (e *Editor) documentClosed(doc *Document) {
	if e.docObserver != nil {
		e.docObserver.DocumentClosed(doc)
	}
}

func newDocumentChange(before core.Rope, cs core.ChangeSet) DocumentChange {
	return DocumentChange{
		Before:  before,
		Changes: cs,
	}
}

func wholeDocumentChange(before core.Rope, text string) DocumentChange {
	cs, err := core.NewChangeSetFromChanges(before, []core.Change{
		core.TextChange(0, before.LenChars(), text),
	})
	if err != nil {
		return DocumentChange{Before: before}
	}
	return newDocumentChange(before, cs)
}
