package lsp

import (
	"context"
	"errors"
	"sync"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	// Session owns runtime language-server clients for an editor
	Session struct {
		ctx    context.Context
		cwd    string
		editor *view.Editor

		servers    serverState
		docs       docState
		candidates candidateState
		progress   progressState
		watch      watchState
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
		ctx: ctx,
		cwd: cwd,
		servers: serverState{
			registry:  NewRegistry(langs.LanguageServers),
			languages: languagesByName(langs),
			clients:   map[string]*Client{},
			roots:     map[string]string{},
			starting:  map[string]*sync.Mutex{},
		},
		docs: docState{
			serverNames: map[view.DocumentId][]string{},
			opened:      map[view.DocumentId]bool{},
			pendingOpen: map[view.DocumentId]bool{},
			diagIDs:     map[view.DocumentId]map[string]string{},
		},
		candidates: candidateState{
			completions: map[string]completionCandidate{},
			codeActions: map[string]codeActionCandidate{},
			links:       map[string]documentLinkCandidate{},
		},
		progress: progressState{
			byServer: map[string]map[string]progressEntry{},
		},
		watch: watchState{
			registrations: map[string]map[string][]fileWatch{},
		},
	}
}

// Attach starts an LSP session for the editor and observes document changes
func Attach(ctx context.Context, e *view.Editor) *Session {
	s := NewSession(ctx, e.Cwd())
	s.editor = e
	e.SetLanguageServerController(s)
	e.AddDocumentObserver(s)
	docs := e.VisibleDocuments()
	s.docs.markPendingOpen(docs)
	go func() {
		for _, doc := range docs {
			if !s.docs.consumePendingOpen(doc.ID()) {
				continue // closed before its startup open ran
			}
			s.DocumentOpened(doc)
		}
	}()
	return s
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
	for _, doc := range s.editor.VisibleDocuments() {
		s.DocumentOpened(doc)
	}
	return nil
}

// Close terminates all clients owned by the session
func (s *Session) Close() error {
	s.clearAllDocumentHighlights()
	s.closeFileWatcher()
	clients := s.servers.allClients()

	var err error
	for _, client := range clients {
		if cerr := client.Close(); err == nil {
			err = cerr
		}
	}
	return err
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
	if client, ok := h.session.servers.client(h.name); ok {
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

func (s *Session) resetConfig(langs language.Languages) []*Client {
	s.closeFileWatcher()
	clients := s.servers.reset(langs)
	s.docs.reset()
	s.candidates.reset()
	s.progress.reset()
	s.watch.reset()
	return clients
}

func closeClients(clients []*Client) {
	for _, client := range clients {
		_ = client.Close()
	}
}
