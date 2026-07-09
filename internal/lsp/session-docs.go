package lsp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

// docState tracks, per document, which servers are attached, whether didOpen
// has been sent (or is still pending an in-flight open), and the last
// pull-diagnostics result ID per provider
type docState struct {
	sync.RWMutex
	serverNames map[view.DocumentId][]string
	opened      map[view.DocumentId]bool
	pendingOpen map[view.DocumentId]bool
	diagIDs     map[view.DocumentId]map[string]string
}

// PullDiagnostics requests fresh diagnostics from document servers
func (s *Session) PullDiagnostics(doc *view.Document) error {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return nil
	}
	clients := s.clientsForDocument(doc)
	if len(clients) == 0 {
		return ErrNoLanguageServer
	}
	if id := s.servers.languageID(doc.Lang()); id != "" {
		snap.LanguageID = id
	}
	var err error
	for _, client := range clients {
		if !client.SupportsFeature(FeaturePullDiagnostics) {
			continue
		}
		provider := client.Name()
		report, sent, e := client.DocumentDiagnostics(
			s.ctx, snap, s.docs.previousDiagnosticID(doc.ID(), provider),
		)
		if e != nil {
			err = errors.Join(err, fmt.Errorf("%w: %s", e, provider))
			continue
		}
		if sent {
			s.applyDiagnosticReport(doc, provider, client, report)
		}
	}
	return err
}

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

func (h *clientHandler) PublishDiagnostics(
	_ context.Context, params *protocol.PublishDiagnosticsParams,
) error {
	return h.session.publishDiagnostics(h.name, params)
}

func (h *clientHandler) DiagnosticRefresh(context.Context) error {
	h.session.pullAllDiagnosticsAsync()
	return nil
}

func (s *Session) publishDiagnostics(
	provider string, params *protocol.PublishDiagnosticsParams,
) error {
	if s.editor == nil {
		return nil
	}
	path := params.URI.FsPath()
	var target *view.Document
	for _, doc := range s.editor.AllDocuments() {
		if doc.Path() == path {
			target = doc
			break
		}
	}
	if target == nil {
		return nil
	}
	if version, ok := params.Version.Get(); ok {
		if int(version) != target.Revision() {
			return nil
		}
	}
	target.ReplaceDiagnostics(
		provider,
		s.convertDiagnostics(
			provider, target, params.Diagnostics, s.offsetForProvider(provider),
		),
	)
	return nil
}

func (s *Session) applyDiagnosticReport(
	doc *view.Document, provider string, client *Client,
	report protocol.DocumentDiagnosticReport,
) {
	switch r := report.(type) {
	case *protocol.RelatedFullDocumentDiagnosticReport:
		doc.ReplaceDiagnostics(
			provider,
			s.convertDiagnostics(
				provider, doc, r.Items, client.OffsetEncoding(),
			),
		)
		s.docs.setPreviousDiagnosticID(doc.ID(), provider, r.ResultID)
	case *protocol.RelatedUnchangedDocumentDiagnosticReport:
		s.docs.setPreviousDiagnosticID(doc.ID(), provider, &r.ResultID)
	}
}

func (s *Session) pullDiagnosticsAsync(doc *view.Document) {
	go func() {
		_ = s.PullDiagnostics(doc)
	}()
}

func (s *Session) pullAllDiagnosticsAsync() {
	if s.editor == nil {
		return
	}
	for _, doc := range s.editor.AllDocuments() {
		if !doc.Loaded() {
			continue
		}
		s.pullDiagnosticsAsync(doc)
	}
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

func (s *Session) convertDiagnostics(
	provider string, doc *view.Document, diags []protocol.Diagnostic,
	encoding protocol.PositionEncodingKind,
) []view.Diagnostic {
	out := make([]view.Diagnostic, 0, len(diags))
	for _, diag := range diags {
		dr, ok := lspRangeToChars(doc, diag.Range, encoding)
		if !ok {
			continue
		}
		source, _ := diag.Source.Get()
		out = append(out, view.Diagnostic{
			Range: view.DiagnosticRange{
				From: dr.From(),
				To:   dr.To(),
			},
			Severity: diagnosticSeverity(diag.Severity),
			Message:  markupText(diag.Message),
			Source:   source,
			Provider: provider,
		})
	}
	return out
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

func (d *docState) previousDiagnosticID(
	id view.DocumentId, provider string,
) *string {
	d.RLock()
	defer d.RUnlock()
	ids, ok := d.diagIDs[id]
	if !ok {
		return nil
	}
	prev, ok := ids[provider]
	if !ok {
		return nil
	}
	return &prev
}

func (d *docState) setPreviousDiagnosticID(
	id view.DocumentId, provider string, resultID *string,
) {
	d.Lock()
	defer d.Unlock()
	if resultID == nil {
		if ids, ok := d.diagIDs[id]; ok {
			delete(ids, provider)
		}
		return
	}
	ids, ok := d.diagIDs[id]
	if !ok {
		ids = map[string]string{}
		d.diagIDs[id] = ids
	}
	ids[provider] = *resultID
}

func (d *docState) setServerNames(id view.DocumentId, names []string) {
	d.Lock()
	defer d.Unlock()
	d.serverNames[id] = names
}

// claimOpen reports whether the caller is the first to open id, marking it
// opened as a side effect
func (d *docState) claimOpen(id view.DocumentId) bool {
	d.Lock()
	defer d.Unlock()
	if d.opened[id] {
		return false
	}
	d.opened[id] = true
	return true
}

func (d *docState) markPendingOpen(docs []*view.Document) {
	d.Lock()
	defer d.Unlock()
	for _, doc := range docs {
		d.pendingOpen[doc.ID()] = true
	}
}

// consumePendingOpen reports whether id still needs its startup open; false
// means the document was closed before this call ran
func (d *docState) consumePendingOpen(id view.DocumentId) bool {
	d.Lock()
	defer d.Unlock()
	want := d.pendingOpen[id]
	delete(d.pendingOpen, id)
	return want
}

// cancelPendingOpen cancels any in-flight startup open for id and reports
// whether didOpen was already sent for it
func (d *docState) cancelPendingOpen(id view.DocumentId) bool {
	d.Lock()
	defer d.Unlock()
	opened := d.opened[id]
	delete(d.pendingOpen, id)
	return opened
}

// forget drops all bookkeeping for a closed document
func (d *docState) forget(id view.DocumentId) {
	d.Lock()
	defer d.Unlock()
	delete(d.serverNames, id)
	delete(d.opened, id)
	delete(d.diagIDs, id)
}

func (d *docState) reset() {
	d.Lock()
	defer d.Unlock()
	d.serverNames = map[view.DocumentId][]string{}
	d.opened = map[view.DocumentId]bool{}
	d.pendingOpen = map[view.DocumentId]bool{}
	d.diagIDs = map[view.DocumentId]map[string]string{}
}

func diagnosticSeverity(
	severity protocol.DiagnosticSeverity,
) view.DiagnosticSeverity {
	switch severity {
	case protocol.DiagnosticSeverityError:
		return view.DiagnosticSeverityError
	case protocol.DiagnosticSeverityWarning:
		return view.DiagnosticSeverityWarning
	case protocol.DiagnosticSeverityInformation:
		return view.DiagnosticSeverityInfo
	default:
		return view.DiagnosticSeverityHint
	}
}
