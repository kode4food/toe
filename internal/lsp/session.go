package lsp

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sync"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	// Session owns runtime language-server clients for an editor
	Session struct {
		ctx       context.Context
		cwd       string
		registry  *Registry
		editor    *view.Editor
		languages map[string]language.Language
		clients   map[string]*Client
		docs      map[view.DocumentId][]string
		diagIDs   map[view.DocumentId]map[string]string
		comps     map[string]completionCandidate
		actions   map[string]codeActionCandidate
		links     map[string]documentLinkCandidate
		progress  map[string]map[string]progressEntry
		watches   map[string]map[string][]fileWatch
		watcher   *fsWatcher
		roots     map[string]string
		mu        sync.RWMutex
	}

	clientHandler struct {
		protocol.UnimplementedClient

		session *Session
		name    string
	}

	documentNotifier func(
		*Client, context.Context, DocumentSnapshot,
	) (bool, error)
)

var (
	ErrNoLanguageServer        = errors.New("LSP not defined for document")
	ErrUnknownLanguageServer   = errors.New("unknown language server")
	ErrWorkspaceCommand        = errors.New("workspace command unavailable")
	ErrCompletionUnavailable   = errors.New("completion unavailable")
	ErrCodeActionUnavailable   = errors.New("code action unavailable")
	ErrDocumentLinkUnavailable = errors.New("document link unavailable")
	ErrFormatSelection         = errors.New("format selection unsupported")
	ErrLanguageServerExited    = errors.New("language server exited")
	ErrLanguageServerRequest   = errors.New("language server request failed")
)

var _ view.DocumentObserver = (*Session)(nil)
var _ view.FileOperationController = (*Session)(nil)
var _ view.LanguageServerController = (*Session)(nil)
var _ protocol.Client = (*clientHandler)(nil)

// NewSession creates an LSP session from language configuration
func NewSession(ctx context.Context, cwd string) *Session {
	langs := loadLanguages(cwd)
	return &Session{
		ctx:       ctx,
		cwd:       cwd,
		registry:  NewRegistry(langs.LanguageServers),
		languages: languagesByName(langs),
		clients:   map[string]*Client{},
		docs:      map[view.DocumentId][]string{},
		diagIDs:   map[view.DocumentId]map[string]string{},
		comps:     map[string]completionCandidate{},
		actions:   map[string]codeActionCandidate{},
		links:     map[string]documentLinkCandidate{},
		progress:  map[string]map[string]progressEntry{},
		watches:   map[string]map[string][]fileWatch{},
		roots:     map[string]string{},
	}
}

// ReloadConfig reloads language-server config and restarts open documents
func (s *Session) ReloadConfig() error {
	langs := loadLanguages(s.cwd)
	clients := s.resetConfig(langs)
	s.clearDocumentState()
	closeClients(clients)
	if s.editor == nil {
		return nil
	}
	for _, doc := range s.editor.AllDocuments() {
		s.DocumentOpened(doc)
	}
	return nil
}

func (s *Session) PublishDiagnostics(
	_ context.Context, params *protocol.PublishDiagnosticsParams,
) error {
	return s.publishDiagnostics("lsp", params)
}

func (h *clientHandler) Configuration(
	_ context.Context, params *protocol.ConfigurationParams,
) ([]protocol.LSPAny, error) {
	out := make([]protocol.LSPAny, len(params.Items))
	for i := range out {
		out[i] = protocol.LSPAny("null")
	}
	return out, nil
}

func (h *clientHandler) WorkspaceFolders(
	context.Context,
) ([]protocol.WorkspaceFolder, error) {
	return h.session.workspaceFolders(), nil
}

func (h *clientHandler) RegisterCapability(
	_ context.Context, params *protocol.RegistrationParams,
) error {
	return h.session.registerCapability(h.name, params)
}

func (h *clientHandler) UnregisterCapability(
	_ context.Context, params *protocol.UnregistrationParams,
) error {
	h.session.unregisterCapability(h.name, params)
	return nil
}

func (h *clientHandler) WorkDoneProgressCreate(
	_ context.Context, params *protocol.WorkDoneProgressCreateParams,
) error {
	h.session.createProgress(h.name, params.Token)
	return nil
}

func (h *clientHandler) Progress(
	_ context.Context, params *protocol.ProgressParams,
) error {
	h.session.updateProgress(h.name, params)
	return nil
}

func (h *clientHandler) LogTrace(
	context.Context, *protocol.LogTraceParams,
) error {
	return nil
}

func (h *clientHandler) ShowMessage(
	context.Context, *protocol.ShowMessageParams,
) error {
	return nil
}

func (h *clientHandler) ShowMessageRequest(
	context.Context, *protocol.ShowMessageRequestParams,
) (*protocol.MessageActionItem, error) {
	return nil, nil
}

func (h *clientHandler) LogMessage(
	context.Context, *protocol.LogMessageParams,
) error {
	return nil
}

func (h *clientHandler) ShowDocument(
	context.Context, *protocol.ShowDocumentParams,
) (*protocol.ShowDocumentResult, error) {
	return &protocol.ShowDocumentResult{Success: false}, nil
}

func (h *clientHandler) Telemetry(context.Context, protocol.LSPAny) error {
	return nil
}

func (h *clientHandler) ApplyEdit(
	_ context.Context, params *protocol.ApplyWorkspaceEditParams,
) (*protocol.ApplyWorkspaceEditResult, error) {
	encoding := protocol.PositionEncodingKindUTF16
	if client, ok := h.session.client(h.name); ok {
		encoding = client.OffsetEncoding()
	}
	if err := h.session.applyWorkspaceEdit(params.Edit, encoding); err != nil {
		return &protocol.ApplyWorkspaceEditResult{
			Applied:       false,
			FailureReason: new(err.Error()),
		}, nil
	}
	return &protocol.ApplyWorkspaceEditResult{Applied: true}, nil
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
	s.mu.RLock()
	if lang, ok := s.languages[doc.Lang()]; ok && lang.LanguageID != "" {
		snap.LanguageID = lang.LanguageID
	}
	s.mu.RUnlock()
	var err error
	for _, client := range clients {
		if !client.SupportsFeature(FeaturePullDiagnostics) {
			continue
		}
		provider := client.Name()
		report, sent, e := client.DocumentDiagnostics(
			s.ctx, snap, s.previousDiagnosticID(doc.ID(), provider),
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

// RestartLanguageServers stops and restarts the named servers for the document
func (s *Session) RestartLanguageServers(
	doc *view.Document, names []string,
) ([]string, error) {
	lang, ok := s.languageForDocument(doc)
	if !ok {
		return nil, ErrNoLanguageServer
	}
	selected, err := selectLanguageServers(lang, names)
	if err != nil {
		return nil, err
	}
	s.clearDocumentHighlightsForServers(selected)
	s.stopClients(selected)
	for _, name := range selected {
		_, _ = s.startClient(name, doc, lang)
	}
	s.notify(doc, (*Client).DidOpen)
	return selected, nil
}

// StopLanguageServers shuts down the named language servers for the document
func (s *Session) StopLanguageServers(
	doc *view.Document, names []string,
) ([]string, error) {
	lang, ok := s.languageForDocument(doc)
	if !ok {
		return nil, ErrNoLanguageServer
	}
	selected, err := selectLanguageServers(lang, names)
	if err != nil {
		return nil, err
	}
	s.clearDocumentHighlightsForServers(selected)
	s.stopClients(selected)
	return selected, nil
}

// ExecuteWorkspaceCommand runs a named workspace command on the matching server
func (s *Session) ExecuteWorkspaceCommand(
	doc *view.Document, name string, args []string,
) error {
	clients := s.clientsForDocument(doc)
	var matches []*Client
	for _, client := range clients {
		if clientSupportsCommand(client, name) {
			matches = append(matches, client)
		}
	}
	switch len(matches) {
	case 0:
		return fmt.Errorf("%w: %s", ErrWorkspaceCommand, name)
	case 1:
		params := &protocol.ExecuteCommandParams{
			Command:   name,
			Arguments: stringsToLSPAny(args),
		}
		return matches[0].ExecuteCommand(s.ctx, params)
	default:
		return fmt.Errorf("%w: %s", ErrWorkspaceCommand, name)
	}
}

// WorkspaceCommands returns commands advertised by attached servers
func (s *Session) WorkspaceCommands(doc *view.Document) []string {
	clients := s.clientsForDocument(doc)
	var out []string
	for _, client := range clients {
		capabilities, ok := client.Capabilities()
		if !ok || len(capabilities.ExecuteCommandProvider.Commands) == 0 {
			continue
		}
		out = append(out, capabilities.ExecuteCommandProvider.Commands...)
	}
	return out
}

// Close terminates all clients owned by the session
func (s *Session) Close() error {
	s.clearAllDocumentHighlights()
	s.closeFileWatcher()
	s.mu.RLock()
	clients := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.mu.RUnlock()

	var err error
	for _, client := range clients {
		if cerr := client.Close(); err == nil {
			err = cerr
		}
	}
	return err
}

// DocumentOpened starts matching servers and sends didOpen notifications
func (s *Session) DocumentOpened(doc *view.Document) {
	s.notify(doc, (*Client).DidOpen)
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
	s.notify(doc, (*Client).DidClose)
	doc.ClearAllDocumentHighlights()
	doc.ClearDocumentLinks()
	doc.ClearDocumentColors()
	s.mu.Lock()
	delete(s.docs, doc.ID())
	delete(s.diagIDs, doc.ID())
	s.clearDocumentLinksLocked(doc.ID())
	s.mu.Unlock()
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
		s.setPreviousDiagnosticID(doc.ID(), provider, r.ResultID)
	case *protocol.RelatedUnchangedDocumentDiagnosticReport:
		s.setPreviousDiagnosticID(doc.ID(), provider, &r.ResultID)
	}
}

func (s *Session) previousDiagnosticID(
	id view.DocumentId, provider string,
) *string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids, ok := s.diagIDs[id]
	if !ok {
		return nil
	}
	prev, ok := ids[provider]
	if !ok {
		return nil
	}
	return &prev
}

func (s *Session) setPreviousDiagnosticID(
	id view.DocumentId, provider string, resultID *string,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if resultID == nil {
		if ids, ok := s.diagIDs[id]; ok {
			delete(ids, provider)
		}
		return
	}
	ids, ok := s.diagIDs[id]
	if !ok {
		ids = map[string]string{}
		s.diagIDs[id] = ids
	}
	ids[provider] = *resultID
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
	s.mu.RLock()
	if lang, ok := s.languages[doc.Lang()]; ok && lang.LanguageID != "" {
		snap.LanguageID = lang.LanguageID
	}
	s.mu.RUnlock()
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
	s.mu.RLock()
	if lang, ok := s.languages[doc.Lang()]; ok && lang.LanguageID != "" {
		snap.LanguageID = lang.LanguageID
	}
	s.mu.RUnlock()
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
		client, ok := s.client(name)
		if !ok {
			var started bool
			client, started = s.startClient(name, doc, lang)
			if !started {
				continue
			}
		}
		out = append(out, client)
	}
	s.setDocumentServers(doc.ID(), names)
	return out
}

func (s *Session) client(name string) (*Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clients[name]
	return client, ok
}

func (s *Session) setDocumentServers(id view.DocumentId, names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[id] = names
}

func (s *Session) resetConfig(langs language.Languages) []*Client {
	s.closeFileWatcher()
	s.mu.Lock()
	clients := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.registry = NewRegistry(langs.LanguageServers)
	s.languages = languagesByName(langs)
	s.clients = map[string]*Client{}
	s.docs = map[view.DocumentId][]string{}
	s.diagIDs = map[view.DocumentId]map[string]string{}
	s.comps = map[string]completionCandidate{}
	s.actions = map[string]codeActionCandidate{}
	s.links = map[string]documentLinkCandidate{}
	s.progress = map[string]map[string]progressEntry{}
	s.watches = map[string]map[string][]fileWatch{}
	s.watcher = nil
	s.roots = map[string]string{}
	s.mu.Unlock()
	return clients
}

func (s *Session) languageForDocument(
	doc *view.Document,
) (language.Language, bool) {
	if doc == nil {
		return language.Language{}, false
	}
	s.mu.RLock()
	lang, ok := s.languages[doc.Lang()]
	s.mu.RUnlock()
	if !ok || len(lang.LanguageServers) == 0 {
		return language.Language{}, false
	}
	return lang, true
}

func (s *Session) startClient(
	name string, doc *view.Document, lang language.Language,
) (*Client, bool) {
	root := s.workspaceRoot(doc, lang)
	handler := &clientHandler{session: s, name: name}
	s.mu.RLock()
	client, err := s.registry.Start(s.ctx, name, root, handler)
	s.mu.RUnlock()
	if err != nil {
		return nil, false
	}
	s.setWorkspaceRoot(name, root)
	params := NewInitializeParams(InitializeConfig{WorkspaceRoot: root})
	if _, err := client.Initialize(s.ctx, params); err != nil {
		_ = client.Close()
		return nil, false
	}
	s.setClient(name, client)
	return client, true
}

func (s *Session) setClient(name string, client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[name] = client
}

func (s *Session) setWorkspaceRoot(name, root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roots[name] = root
}

func (s *Session) workspaceFolders() []protocol.WorkspaceFolder {
	s.mu.RLock()
	roots := maps.Clone(s.roots)
	s.mu.RUnlock()
	out := make([]protocol.WorkspaceFolder, 0, len(roots))
	for _, root := range roots {
		out = append(out, protocol.WorkspaceFolder{
			URI:  uri.File(root),
			Name: filepath.Base(root),
		})
	}
	slices.SortFunc(out, func(a, b protocol.WorkspaceFolder) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return out
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

func (s *Session) offsetForProvider(
	provider string,
) protocol.PositionEncodingKind {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clients[provider]
	if !ok {
		return protocol.PositionEncodingKindUTF16
	}
	return client.OffsetEncoding()
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

func (s *Session) workspaceRoot(
	doc *view.Document, lang language.Language,
) string {
	workspace, ok := ResolveWorkspace(WorkspaceRequest{
		FilePath:       doc.Path(),
		Workspace:      s.cwd,
		WorkspaceIsCWD: true,
		RootMarkers:    lang.Roots,
	})
	if ok {
		return workspace
	}
	if dir := filepath.Dir(doc.Path()); dir != "." {
		return dir
	}
	return s.cwd
}

func loadLanguages(cwd string) language.Languages {
	global, ok := loader.LanguagesFile()
	if !ok {
		global = ""
	}
	workspace := loader.WorkspaceLanguagesFile(cwd)
	langs, ok := language.LoadLanguagesForWorkspace(global, workspace, cwd)
	if !ok {
		return language.Languages{}
	}
	return langs
}

func languagesByName(langs language.Languages) map[string]language.Language {
	out := make(map[string]language.Language, len(langs.Languages))
	for _, lang := range langs.Languages {
		out[lang.Name] = lang
	}
	return out
}

func serverNames(features []language.ServerFeatures) []string {
	out := make([]string, 0, len(features))
	seen := map[string]bool{}
	for _, feature := range features {
		if feature.Name == "" || seen[feature.Name] {
			continue
		}
		seen[feature.Name] = true
		out = append(out, feature.Name)
	}
	return out
}

func selectLanguageServers(
	lang language.Language, requested []string,
) ([]string, error) {
	names := serverNames(lang.LanguageServers)
	if len(requested) == 0 {
		return names, nil
	}
	valid := make(map[string]bool, len(names))
	for _, name := range names {
		valid[name] = true
	}
	out := make([]string, 0, len(requested))
	for _, name := range requested {
		if !valid[name] {
			return nil, fmt.Errorf("%w: %s", ErrUnknownLanguageServer, name)
		}
		out = append(out, name)
	}
	return out, nil
}

func (s *Session) stopClients(names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range names {
		client := s.clients[name]
		if client == nil {
			continue
		}
		_ = client.Close()
		delete(s.clients, name)
	}
}

func (s *Session) dropClient(name string, client *Client) {
	s.clearDocumentHighlightsForServers([]string{name})
	s.mu.Lock()
	if s.clients[name] == client {
		delete(s.clients, name)
	}
	s.mu.Unlock()
	_ = client.Close()
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

// Attach starts an LSP session for the editor and observes document changes
func Attach(ctx context.Context, e *view.Editor) *Session {
	s := NewSession(ctx, e.Cwd())
	s.editor = e
	e.SetLanguageServerController(s)
	e.AddDocumentObserver(s)
	for _, doc := range e.AllDocuments() {
		s.DocumentOpened(doc)
	}
	return s
}

func closeClients(clients []*Client) {
	for _, client := range clients {
		_ = client.Close()
	}
}

func clientSupportsCommand(client *Client, name string) bool {
	capabilities, ok := client.Capabilities()
	if !ok {
		return false
	}
	return slices.Contains(capabilities.ExecuteCommandProvider.Commands, name)
}

func stringsToLSPAny(args []string) []protocol.LSPAny {
	if len(args) == 0 {
		return nil
	}
	out := make([]protocol.LSPAny, len(args))
	for i, arg := range args {
		b, _ := json.Marshal(arg)
		out[i] = b
	}
	return out
}
