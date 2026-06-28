package lsp_test

import (
	"context"
	"net"
	"testing"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type (
	syncServer struct {
		protocol.UnimplementedServer
		sync        protocol.TextDocumentSync
		opened      chan *protocol.DidOpenTextDocumentParams
		changed     chan *protocol.DidChangeTextDocumentParams
		saved       chan *protocol.DidSaveTextDocumentParams
		closed      chan *protocol.DidCloseTextDocumentParams
		diagnostic  chan *protocol.DocumentDiagnosticParams
		completion  chan *protocol.CompletionParams
		command     chan *protocol.ExecuteCommandParams
		initialized chan struct{}
	}

	wholeDocumentChange = protocol.TextDocumentContentChangeWholeDocument
	partialChange       = protocol.TextDocumentContentChangePartial
)

func TestTextSync(t *testing.T) {
	t.Run("sends full sync notifications", func(t *testing.T) {
		yes := true
		kind := protocol.TextDocumentSyncKindFull
		server := &syncServer{
			sync: &protocol.TextDocumentSyncOptions{
				OpenClose: &yes,
				Change:    &kind,
				Save: &protocol.SaveOptions{
					IncludeText: &yes,
				},
			},
			opened:      make(chan *protocol.DidOpenTextDocumentParams, 1),
			changed:     make(chan *protocol.DidChangeTextDocumentParams, 1),
			saved:       make(chan *protocol.DidSaveTextDocumentParams, 1),
			closed:      make(chan *protocol.DidCloseTextDocumentParams, 1),
			initialized: make(chan struct{}),
		}
		ctx, client, close := newSyncClient(t, server)
		defer close()

		doc := lsp.DocumentSnapshot{
			URI:        uri.File("/tmp/main.go"),
			LanguageID: "go",
			Version:    7,
			Text:       "package main\n",
		}
		ok, err := client.DidOpen(ctx, doc)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, doc.Text, (<-server.opened).TextDocument.Text)

		ok, err = client.DidChange(ctx, doc)
		assert.NoError(t, err)
		assert.True(t, ok)
		change := <-server.changed
		assert.Equal(t, doc.Version, change.TextDocument.Version)
		whole, ok := change.ContentChanges[0].(*wholeDocumentChange)
		assert.True(t, ok)
		assert.Equal(t, doc.Text, whole.Text)

		ok, err = client.DidSave(ctx, doc)
		assert.NoError(t, err)
		assert.True(t, ok)
		save := <-server.saved
		assert.Equal(t, doc.Text, *save.Text)

		ok, err = client.DidClose(ctx, doc)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, doc.URI, (<-server.closed).TextDocument.URI)
	})

	t.Run("sends incremental changes", func(t *testing.T) {
		kind := protocol.TextDocumentSyncKindIncremental
		server := &syncServer{
			sync: &protocol.TextDocumentSyncOptions{
				Change: &kind,
			},
			opened:      make(chan *protocol.DidOpenTextDocumentParams, 1),
			changed:     make(chan *protocol.DidChangeTextDocumentParams, 1),
			saved:       make(chan *protocol.DidSaveTextDocumentParams, 1),
			closed:      make(chan *protocol.DidCloseTextDocumentParams, 1),
			initialized: make(chan struct{}),
		}
		ctx, client, close := newSyncClient(t, server)
		defer close()

		before := core.NewRope("hello\n")
		cs, err := core.NewChangeSetFromChanges(before, []core.Change{
			core.TextChange(5, 5, "!"),
		})
		assert.NoError(t, err)
		ok, err := client.DidChangeDocument(ctx, lsp.DocumentSnapshot{
			URI:     uri.File("/tmp/main.go"),
			Version: 8,
			Text:    "hello!\n",
		}, view.DocumentChange{Before: before, Changes: cs})

		assert.NoError(t, err)
		assert.True(t, ok)
		change := <-server.changed
		partial, ok := change.ContentChanges[0].(*partialChange)
		assert.True(t, ok)
		assert.Equal(t, "!", partial.Text)
		assert.Equal(t, uint32(0), partial.Range.Start.Line)
		assert.Equal(t, uint32(5), partial.Range.Start.Character)
		assert.Equal(t, partial.Range.Start, partial.Range.End)
	})

	t.Run("requests pull diagnostics", func(t *testing.T) {
		resultID := "diag-1"
		server := &syncServer{
			diagnostic:  make(chan *protocol.DocumentDiagnosticParams, 1),
			initialized: make(chan struct{}),
		}
		ctx, client, close := newSyncClient(t, server)
		defer close()
		prev := "diag-0"
		doc := lsp.DocumentSnapshot{
			URI: uri.File("/tmp/main.go"),
		}

		report, ok, err := client.DocumentDiagnostics(ctx, doc, &prev)

		assert.NoError(t, err)
		assert.True(t, ok)
		params := <-server.diagnostic
		assert.Equal(t, doc.URI, params.TextDocument.URI)
		assert.Equal(t, prev, *params.PreviousResultID)
		full, ok := report.(*protocol.RelatedFullDocumentDiagnosticReport)
		assert.True(t, ok)
		assert.Equal(t, resultID, *full.ResultID)
		assert.Len(t, full.Items, 1)
	})

	t.Run("requests completions", func(t *testing.T) {
		server := &syncServer{
			completion:  make(chan *protocol.CompletionParams, 1),
			initialized: make(chan struct{}),
		}
		ctx, client, close := newSyncClient(t, server)
		defer close()
		doc := lsp.DocumentSnapshot{
			URI:  uri.File("/tmp/main.go"),
			Text: "fmt.Pr",
		}
		completionContext := protocol.CompletionContext{
			TriggerKind: protocol.CompletionTriggerKindInvoked,
		}

		list, ok, err := client.Completion(ctx, doc, 6, completionContext)

		assert.NoError(t, err)
		assert.True(t, ok)
		params := <-server.completion
		assert.Equal(t, doc.URI, params.TextDocument.URI)
		assert.Equal(t, uint32(0), params.Position.Line)
		assert.Equal(t, uint32(6), params.Position.Character)
		assert.Equal(t, completionContext, params.Context)
		assert.True(t, list.Incomplete)
		assert.Len(t, list.Items, 2)
		assert.Equal(t, "Println", list.Items[0].Label)
		assert.Equal(t, "fmt.Println", list.Items[0].Filter)
		assert.Equal(t, "a", list.Items[0].Sort)
		assert.Equal(t, "Println($1)", list.Items[0].Insert)
		assert.True(t, list.Items[0].Preselect)
	})

	t.Run("executes commands", func(t *testing.T) {
		server := &syncServer{
			command:     make(chan *protocol.ExecuteCommandParams, 1),
			initialized: make(chan struct{}),
		}
		ctx, client, close := newSyncClient(t, server)
		defer close()
		params := &protocol.ExecuteCommandParams{
			Command: "do.it",
			Arguments: []protocol.LSPAny{
				protocol.LSPAny(`"arg"`),
			},
		}

		err := client.ExecuteCommand(ctx, params)

		assert.NoError(t, err)
		got := <-server.command
		assert.Equal(t, params.Command, got.Command)
		assert.Equal(t, params.Arguments, got.Arguments)
	})
}

func (s *syncServer) Initialize(
	context.Context, *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync:   s.sync,
			DiagnosticProvider: &protocol.DiagnosticOptions{},
			CompletionProvider: &protocol.CompletionOptions{},
		},
	}, nil
}

func (s *syncServer) Initialized(
	context.Context, *protocol.InitializedParams,
) error {
	close(s.initialized)
	return nil
}

func (s *syncServer) DidOpen(
	_ context.Context, p *protocol.DidOpenTextDocumentParams,
) error {
	s.opened <- p
	return nil
}

func (s *syncServer) DidChange(
	_ context.Context, p *protocol.DidChangeTextDocumentParams,
) error {
	s.changed <- p
	return nil
}

func (s *syncServer) DidSave(
	_ context.Context, p *protocol.DidSaveTextDocumentParams,
) error {
	s.saved <- p
	return nil
}

func (s *syncServer) DidClose(
	_ context.Context, p *protocol.DidCloseTextDocumentParams,
) error {
	s.closed <- p
	return nil
}

func (s *syncServer) Diagnostic(
	_ context.Context, p *protocol.DocumentDiagnosticParams,
) (protocol.DocumentDiagnosticReport, error) {
	s.diagnostic <- p
	resultID := "diag-1"
	return &protocol.RelatedFullDocumentDiagnosticReport{
		FullDocumentDiagnosticReport: protocol.FullDocumentDiagnosticReport{
			Kind:     string(protocol.DocumentDiagnosticReportKindFull),
			ResultID: &resultID,
			Items: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 4},
					},
					Severity: protocol.DiagnosticSeverityWarning,
					Message:  protocol.String("pulled diagnostic"),
				},
			},
		},
	}, nil
}

func (s *syncServer) Completion(
	_ context.Context, p *protocol.CompletionParams,
) (protocol.CompletionResult, error) {
	s.completion <- p
	return &protocol.CompletionList{
		IsIncomplete: true,
		Items: []protocol.CompletionItem{
			{
				Label:      "Println",
				Detail:     protocol.NewOptional("func"),
				FilterText: protocol.NewOptional("fmt.Println"),
				SortText:   protocol.NewOptional("a"),
				InsertText: protocol.NewOptional("Println($1)"),
				Preselect:  protocol.NewOptional(true),
			},
			{Label: "Printf"},
		},
	}, nil
}

func (s *syncServer) ExecuteCommand(
	_ context.Context, p *protocol.ExecuteCommandParams,
) (protocol.LSPAny, error) {
	s.command <- p
	return protocol.LSPAny("null"), nil
}

func newSyncClient(
	t *testing.T, server *syncServer,
) (context.Context, *lsp.Client, func()) {
	t.Helper()
	ctx := t.Context()
	clientConn, serverConn := net.Pipe()
	_, serverRPC, _ := protocol.NewServer(
		ctx, server, jsonrpc2.NewHeaderStream(serverConn),
	)
	clientCtx, client := lsp.NewClient(ctx, clientConn, nil)
	_, err := client.Initialize(
		clientCtx, lsp.NewInitializeParams(lsp.InitializeConfig{}),
	)
	assert.NoError(t, err)
	assert.True(t, waitFor(server.initialized))
	return clientCtx, client, func() {
		client.Close()
		serverRPC.Close()
		clientConn.Close()
		serverConn.Close()
	}
}
