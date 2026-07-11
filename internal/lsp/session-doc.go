package lsp

import (
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

// DocumentOpened starts matching servers and sends didOpen notifications
func (s *Session) DocumentOpened(doc *view.Document) {
	s.clientsForDocument(doc)
	s.pullDiagnosticsAsync(doc)
	s.documentLinksAsync(doc)
	s.documentColorsAsync(doc)
	s.inlayHintsAsync(doc)
}

// DocumentChanged sends didChange notifications to attached servers
func (s *Session) DocumentChanged(
	doc *view.Document, change view.DocumentChange,
) {
	s.notifyChange(doc, change)
	s.pullDiagnosticsAsync(doc)
	s.documentLinksAsync(doc)
	s.documentColorsAsync(doc)
	s.inlayHintsAsync(doc)
}

// DocumentSaved sends didSave notifications to attached servers
func (s *Session) DocumentSaved(doc *view.Document) {
	s.notify(doc, (*Client).DidSave)
	s.pullDiagnosticsAsync(doc)
	s.documentLinksAsync(doc)
	s.documentColorsAsync(doc)
	s.inlayHintsAsync(doc)
	s.didChangeWatchedFile(doc.Path())
}

// DocumentClosed sends didClose notifications and forgets the document
func (s *Session) DocumentClosed(doc *view.Document) {
	if s.docs.cancelPendingOpen(doc.ID()) {
		s.notify(doc, (*Client).DidClose)
	}
	doc.ClearAllDocumentHighlights()
	doc.ClearDocumentLinks()
	doc.ClearDocumentColors()
	s.docs.forget(doc.ID())
	s.candidates.clearLinksForDoc(doc.ID())
}

func (s *Session) notify(doc *view.Document, send documentNotifier) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return
	}
	clients := s.clientsForDocument(doc)
	if len(clients) == 0 {
		return
	}
	if id := s.servers.languageID(doc.Lang()); id != "" {
		snap.LanguageID = id
	}
	for _, client := range clients {
		_, _ = send(client, s.ctx, snap)
	}
}

func (s *Session) notifyChange(doc *view.Document, change view.DocumentChange) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return
	}
	clients := s.clientsForDocument(doc)
	if len(clients) == 0 {
		return
	}
	if id := s.servers.languageID(doc.Lang()); id != "" {
		snap.LanguageID = id
	}
	for _, client := range clients {
		_, _ = client.DidChangeDocument(s.ctx, snap, change)
	}
}

func (s *Session) clientsForDocument(doc *view.Document) []*Client {
	lang, ok := s.languageForDocument(doc)
	if !ok {
		return nil
	}
	names := serverNames(lang.LanguageServers)
	if len(names) == 0 {
		return nil
	}

	out := make([]*Client, 0, len(names))
	for _, name := range names {
		client, ok := s.servers.client(name)
		if !ok {
			var started bool
			client, started = s.ensureClient(name, doc, lang)
			if !started {
				continue
			}
		}
		out = append(out, client)
	}
	s.docs.setServerNames(doc.ID(), names)
	s.ensureDidOpen(doc, out, lang)
	return out
}

func (s *Session) ensureDidOpen(
	doc *view.Document, clients []*Client, lang language.Language,
) {
	if len(clients) == 0 {
		return
	}
	id := doc.ID()
	if !s.docs.claimOpen(id) {
		return
	}

	snap, ok := SnapshotDocument(doc)
	if !ok {
		s.docs.forget(id)
		return
	}
	if lang.LanguageID != "" {
		snap.LanguageID = lang.LanguageID
	}
	for _, client := range clients {
		_, _ = client.DidOpen(s.ctx, snap)
	}
}

func (s *Session) clearAllDocumentHighlights() {
	if s.editor == nil {
		return
	}
	for _, doc := range s.editor.AllDocuments() {
		doc.ClearAllDocumentHighlights()
	}
}

func (s *Session) clearDocumentState() {
	if s.editor == nil {
		return
	}
	for _, doc := range s.editor.AllDocuments() {
		doc.ClearDiagnostics()
		doc.ClearAllDocumentHighlights()
		doc.ClearDocumentLinks()
		doc.ClearDocumentColors()
		doc.ClearAllInlayHints()
	}
}

func (s *Session) clearDocumentHighlightsForServers(names []string) {
	if s.editor == nil || len(names) == 0 {
		return
	}
	selected := make(map[string]bool, len(names))
	for _, name := range names {
		selected[name] = true
	}
	for _, doc := range s.editor.AllDocuments() {
		lang, ok := s.languageForDocument(doc)
		if !ok {
			continue
		}
		for _, name := range serverNames(lang.LanguageServers) {
			if selected[name] {
				doc.ClearAllDocumentHighlights()
				break
			}
		}
	}
}
