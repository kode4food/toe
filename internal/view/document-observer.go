package view

import "github.com/kode4food/toe/internal/core"

type (
	// DocumentObserver receives editor document lifecycle notifications
	DocumentObserver interface {
		DocumentOpened(*Document)
		DocumentChanged(*Document, DocumentChange)
		DocumentSaved(*Document)
		DocumentClosed(*Document)
	}

	// DocumentChange describes an editor text change for document observers
	DocumentChange struct {
		Before  core.Rope
		Changes core.ChangeSet
	}
)

// AddDocumentObserver installs a document lifecycle observer. Observers are
// notified in registration order
func (e *Editor) AddDocumentObserver(o DocumentObserver) {
	e.docObservers = append(e.docObservers, o)
}

func (e *Editor) documentOpened(doc *Document) {
	for _, o := range e.docObservers {
		o.DocumentOpened(doc)
	}
}

func (e *Editor) documentChanged(doc *Document, change DocumentChange) {
	for _, o := range e.docObservers {
		o.DocumentChanged(doc, change)
	}
}

func (e *Editor) documentSaved(doc *Document) {
	for _, o := range e.docObservers {
		o.DocumentSaved(doc)
	}
}

func (e *Editor) documentClosed(doc *Document) {
	for _, o := range e.docObservers {
		o.DocumentClosed(doc)
	}
}

func wholeDocumentChange(before core.Rope, text string) DocumentChange {
	cs, err := core.NewChangeSetFromChanges(before, []core.Change{
		core.TextChange(0, before.LenChars(), text),
	})
	if err != nil {
		return DocumentChange{Before: before}
	}
	return DocumentChange{Before: before, Changes: cs}
}
