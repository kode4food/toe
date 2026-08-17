package view

import "github.com/kode4food/toe/internal/core"

// MessagesDocument returns the editor's message log, creating it on first use.
// It is the only way to reach the log: an editor has exactly one, and it is
// registered like any other buffer so the buffer picker lists it
func (e *Editor) MessagesDocument() *Document {
	if doc, ok := e.documents.byID[e.documents.messagesID]; ok {
		return doc
	}
	doc := e.newMessagesDocument()
	e.documents.byID[doc.ID()] = doc
	e.documentOpened(doc)
	return doc
}

// AppendMessage records a message in the log document. The append bypasses the
// read-only flag that keeps the user from editing the log
func (e *Editor) AppendMessage(msg string) {
	if msg == "" {
		return
	}
	doc := e.MessagesDocument()
	text := doc.Text()
	at := text.LenChars()
	cs, err := core.NewChangeSetFromChanges(text, []core.Change{
		core.TextChange(
			core.Span{From: at, To: at}, msg+string(doc.LineEnding()),
		),
	})
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithChanges(cs)
	if err := doc.Apply(tx, InvalidViewId); err != nil {
		return
	}
	doc.markSaved()
}

func (e *Editor) newMessagesDocument() *Document {
	doc := e.newDocument()
	doc.identity.docType = DocTypeLog
	doc.SetDisplayName(MessagesBufferName)
	doc.SetLang(MessagesLanguage)
	doc.SetReadOnly(true)
	e.documents.messagesID = doc.ID()
	return doc
}
