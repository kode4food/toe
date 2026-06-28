package lsp_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view/language"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type (
	processServer struct {
		protocol.UnimplementedServer
		client  protocol.Client
		exit    chan struct{}
		folders []protocol.WorkspaceFolder
	}

	stdioConn struct{}
)

const testServerEnv = "TOE_LSP_TEST_SERVER"
const testServerDidOpenFileEnv = "TOE_LSP_DID_OPEN_FILE"
const testServerCompletionEnv = "TOE_LSP_COMPLETION"
const testServerSignatureEnv = "TOE_LSP_SIGNATURE"
const testServerExitOnCompletionEnv = "TOE_LSP_EXIT_ON_COMPLETION"
const testServerSilentExitOnCompletionEnv = "TOE_LSP_SILENT_EXIT"
const testServerStaleCompletionEnv = "TOE_LSP_STALE_COMPLETION"
const testServerWorkspaceFoldersEnv = "TOE_LSP_WORKSPACE_FOLDERS"
const testServerNavigationEnv = "TOE_LSP_NAVIGATION"
const testServerNavigationTargetEnv = "TOE_LSP_NAVIGATION_TARGET"
const testServerSymbolsEnv = "TOE_LSP_SYMBOLS"

var _ io.ReadWriteCloser = stdioConn{}

func TestTransport(t *testing.T) {
	t.Run("starts stdio server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		cfg := language.Server{
			Command: exe,
			Args:    []string{"-test.run=TestLSPServerProcess"},
			Environment: map[string]string{
				testServerEnv: "1",
			},
		}
		_, client, err := lsp.Start(t.Context(), "test", cfg, "", nil)
		assert.NoError(t, err)

		result, err := client.Initialize(
			t.Context(), lsp.NewInitializeParams(lsp.InitializeConfig{}),
		)
		assert.NoError(t, err)
		assert.Equal(t,
			protocol.PositionEncodingKindUTF16,
			result.Capabilities.PositionEncoding,
		)
		assert.Equal(t, "test", client.Name())
		assert.NoError(t, client.Close())
	})

	t.Run("requires command", func(t *testing.T) {
		_, _, err := lsp.Start(
			t.Context(), "test", language.Server{}, "", nil,
		)

		assert.True(t, errors.Is(err, lsp.ErrCommandRequired))
	})
}

func TestLSPServerProcess(t *testing.T) {
	if os.Getenv(testServerEnv) != "1" {
		return
	}
	ctx := context.Background()
	server := &processServer{exit: make(chan struct{})}
	_, conn, client := protocol.NewServer(
		ctx, server, jsonrpc2.NewHeaderStream(stdioConn{}),
	)
	server.client = client
	<-server.exit
	_ = conn.Close()
	os.Exit(0)
}

func (s *processServer) Initialize(
	ctx context.Context, _ *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	yes := true
	kind := protocol.TextDocumentSyncKindFull
	var completionProvider *protocol.CompletionOptions
	if os.Getenv(testServerCompletionEnv) == "1" {
		resolve := true
		completionProvider = &protocol.CompletionOptions{
			TriggerCharacters: []string{"."},
			ResolveProvider:   &resolve,
		}
	}
	var signatureProvider *protocol.SignatureHelpOptions
	if os.Getenv(testServerSignatureEnv) == "1" {
		signatureProvider = &protocol.SignatureHelpOptions{
			TriggerCharacters:   []string{"("},
			RetriggerCharacters: []string{","},
		}
	}
	if os.Getenv(testServerWorkspaceFoldersEnv) != "" {
		folders, err := s.client.WorkspaceFolders(ctx)
		if err != nil {
			return nil, err
		}
		s.folders = folders
	}
	navigation := protocol.Boolean(os.Getenv(testServerNavigationEnv) == "1")
	symbols := protocol.Boolean(os.Getenv(testServerSymbolsEnv) == "1")
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			PositionEncoding: protocol.PositionEncodingKindUTF16,
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: &yes,
				Change:    &kind,
			},
			CompletionProvider:    completionProvider,
			SignatureHelpProvider: signatureProvider,
			HoverProvider:         protocol.Boolean(true),
			ExecuteCommandProvider: protocol.ExecuteCommandOptions{
				Commands: []string{"session.afterCompletion"},
			},
			DeclarationProvider:    navigation,
			DefinitionProvider:     navigation,
			TypeDefinitionProvider: navigation,
			ImplementationProvider: navigation,
			ReferencesProvider:     navigation,
			DocumentSymbolProvider: symbols,
		},
	}, nil
}

func (s *processServer) Initialized(
	context.Context, *protocol.InitializedParams,
) error {
	return nil
}

func (s *processServer) DidOpen(
	_ context.Context, p *protocol.DidOpenTextDocumentParams,
) error {
	path := os.Getenv(testServerDidOpenFileEnv)
	if path == "" {
		return nil
	}
	text := string(p.TextDocument.URI)
	if len(s.folders) > 0 {
		text += "\n" + string(s.folders[0].URI)
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func (s *processServer) Completion(
	context.Context, *protocol.CompletionParams,
) (protocol.CompletionResult, error) {
	if os.Getenv(testServerCompletionEnv) != "1" {
		return nil, nil
	}
	if os.Getenv(testServerExitOnCompletionEnv) == "1" {
		_, _ = fmt.Fprintln(os.Stderr, "completion process exited")
		os.Exit(7)
	}
	if os.Getenv(testServerSilentExitOnCompletionEnv) == "1" {
		os.Exit(7)
	}
	end := protocol.Position{Line: 0, Character: 2}
	if os.Getenv(testServerStaleCompletionEnv) == "1" {
		end = protocol.Position{Line: 0, Character: 0}
	}
	return protocol.CompletionItemSlice{
		{
			Label: "Println",
			Kind:  protocol.CompletionItemKindFunction,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   end,
				},
				NewText: "Println",
			},
			AdditionalTextEdits: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 0},
					},
					NewText: "// add\n",
				},
			},
		},
	}, nil
}

func (s *processServer) CompletionResolve(
	_ context.Context, item *protocol.CompletionItem,
) (*protocol.CompletionItem, error) {
	item.Detail = protocol.NewOptional("func Println(a ...any)")
	item.Documentation = protocol.String("Println formats its operands.")
	return item, nil
}

func (s *processServer) Hover(
	context.Context, *protocol.HoverParams,
) (*protocol.Hover, error) {
	return &protocol.Hover{
		Contents: &protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "hover docs",
		},
	}, nil
}

func (s *processServer) SignatureHelp(
	context.Context, *protocol.SignatureHelpParams,
) (*protocol.SignatureHelp, error) {
	if os.Getenv(testServerSignatureEnv) != "1" {
		return nil, nil
	}
	active := uint32(0)
	return &protocol.SignatureHelp{
		ActiveSignature: &active,
		Signatures: []protocol.SignatureInformation{
			{
				Label:         "Println(a ...any)",
				Documentation: protocol.String("signature docs"),
				Parameters: []protocol.ParameterInformation{
					{
						Label:         protocol.String("a ...any"),
						Documentation: protocol.String("parameter docs"),
					},
				},
			},
		},
	}, nil
}

func (s *processServer) Declaration(
	context.Context, *protocol.DeclarationParams,
) (protocol.DeclarationResult, error) {
	if os.Getenv(testServerNavigationEnv) != "1" {
		return nil, nil
	}
	return s.navigationLocation(), nil
}

func (s *processServer) Definition(
	context.Context, *protocol.DefinitionParams,
) (protocol.DefinitionResult, error) {
	if os.Getenv(testServerNavigationEnv) != "1" {
		return nil, nil
	}
	return s.navigationLocation(), nil
}

func (s *processServer) TypeDefinition(
	context.Context, *protocol.TypeDefinitionParams,
) (protocol.DefinitionResult, error) {
	if os.Getenv(testServerNavigationEnv) != "1" {
		return nil, nil
	}
	return s.navigationLocation(), nil
}

func (s *processServer) Implementation(
	context.Context, *protocol.ImplementationParams,
) (protocol.DefinitionResult, error) {
	if os.Getenv(testServerNavigationEnv) != "1" {
		return nil, nil
	}
	return s.navigationLocation(), nil
}

func (s *processServer) References(
	_ context.Context, params *protocol.ReferenceParams,
) ([]protocol.Location, error) {
	if os.Getenv(testServerNavigationEnv) != "1" {
		return nil, nil
	}
	loc := s.navigationLocation()
	if loc == nil {
		return nil, nil
	}
	if !params.Context.IncludeDeclaration {
		return nil, nil
	}
	return []protocol.Location{*loc}, nil
}

func (s *processServer) DocumentSymbol(
	context.Context, *protocol.DocumentSymbolParams,
) (protocol.DocumentSymbolResult, error) {
	if os.Getenv(testServerSymbolsEnv) != "1" {
		return nil, nil
	}
	return protocol.DocumentSymbolSlice{
		{
			Name: "outer",
			Kind: protocol.SymbolKindFunction,
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			Children: []protocol.DocumentSymbol{
				{
					Name: "inner",
					Kind: protocol.SymbolKindVariable,
					SelectionRange: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 4},
						End:   protocol.Position{Line: 1, Character: 9},
					},
				},
			},
		},
	}, nil
}

func (s *processServer) navigationLocation() *protocol.Location {
	path := os.Getenv(testServerNavigationTargetEnv)
	if path == "" {
		return nil
	}
	return &protocol.Location{
		URI: uri.File(path),
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 3},
			End:   protocol.Position{Line: 0, Character: 6},
		},
	}
}

func (s *processServer) Shutdown(context.Context) error {
	return nil
}

func (s *processServer) Exit(context.Context) error {
	close(s.exit)
	return nil
}

func (stdioConn) Read(b []byte) (int, error) {
	return os.Stdin.Read(b)
}

func (stdioConn) Write(b []byte) (int, error) {
	return os.Stdout.Write(b)
}

func (stdioConn) Close() error {
	return nil
}
