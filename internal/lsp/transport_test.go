package lsp_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view/language"
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
const testServerWorkspaceSymbolsEnv = "TOE_LSP_WORKSPACE_SYMBOLS"
const testServerRenameEnv = "TOE_LSP_RENAME"
const testServerCodeActionEnv = "TOE_LSP_CODE_ACTION"
const testServerHoverContentEnv = "TOE_LSP_HOVER_CONTENT"
const testServerCallbacksEnv = "TOE_LSP_CALLBACKS"
const testServerNavLinksEnv = "TOE_LSP_NAV_LINKS"
const testServerMultiCodeActionEnv = "TOE_LSP_MULTI_CODE_ACTION"
const testServerPullDiagnosticsEnv = "TOE_LSP_PULL_DIAGNOSTICS"
const testServerSymbolSliceEnv = "TOE_LSP_SYMBOL_SLICE"
const testServerManySymbolsEnv = "TOE_LSP_MANY_SYMBOLS"
const testServerManyCompletionsEnv = "TOE_LSP_MANY_COMPLETIONS"
const testServerFormatEnv = "TOE_LSP_FORMAT"
const testServerHighlightEnv = "TOE_LSP_HIGHLIGHT"
const testServerHighlightMultiEnv = "TOE_LSP_HIGHLIGHT_MULTI"
const testServerDocumentLinkEnv = "TOE_LSP_DOCUMENT_LINK"
const testServerDocumentLinkResolveEnv = "TOE_LSP_DOCUMENT_LINK_RESOLVE"
const testServerRenameDefaultBehaviorEnv = "TOE_LSP_RENAME_DEFAULT_BEHAVIOR"
const testServerRenameRangeEnv = "TOE_LSP_RENAME_RANGE"
const testServerDiagCodeActionEnv = "TOE_LSP_DIAG_CODE_ACTION"
const testServerSignatureOffsetEnv = "TOE_LSP_SIGNATURE_OFFSET"
const testServerProgressEnv = "TOE_LSP_PROGRESS"
const testServerFileWatchEnv = "TOE_LSP_FILE_WATCH"
const testServerFileWatchNotifyEnv = "TOE_LSP_FILE_WATCH_NOTIFY"
const testServerInlayHintsEnv = "TOE_LSP_INLAY_HINTS"
const testServerDocumentColorEnv = "TOE_LSP_DOCUMENT_COLOR"
const testServerWsEditCodeActionEnv = "TOE_LSP_WS_EDIT_CODE_ACTION"
const testServerWsEditOldPathEnv = "TOE_LSP_WS_EDIT_OLD_PATH"
const testServerWsEditNewPathEnv = "TOE_LSP_WS_EDIT_NEW_PATH"
const testServerAltCompletionEnv = "TOE_LSP_ALT_COMPLETION"
const testServerCompletionListEnv = "TOE_LSP_COMPLETION_LIST"
const testServerNavLocationSliceEnv = "TOE_LSP_NAV_LOCATION_SLICE"
const testServerDiagnosticUnchangedEnv = "TOE_LSP_DIAGNOSTIC_UNCHANGED"
const testServerDiagRegOptionsEnv = "TOE_LSP_DIAG_REG_OPTIONS"
const testServerDiagnosticErrorEnv = "TOE_LSP_DIAGNOSTIC_ERROR"
const testServerAllErrorEnv = "TOE_LSP_ALL_ERROR"
const testServerFileOperationsEnv = "TOE_LSP_FILE_OPERATIONS"
const testServerFileOpFolderEnv = "TOE_LSP_FILE_OP_FOLDER"
const testServerWatchRegEdgeEnv = "TOE_LSP_WATCH_REG_EDGE"
const testServerNoResolveEnv = "TOE_LSP_NO_RESOLVE"
const testServerCAResolveEnv = "TOE_LSP_CA_RESOLVE"
const testServerFileOpWillEditEnv = "TOE_LSP_FILE_OP_WILL_EDIT"
const testServerSignatureHelpActiveEnv = "TOE_LSP_SIGNATURE_HELP_ACTIVE"
const testServerSignatureNoParamsEnv = "TOE_LSP_SIGNATURE_NO_PARAMS"
const testServerSignatureMissingLabelEnv = "TOE_LSP_SIGNATURE_MISSING_LABEL"
const testServerSignatureActiveOutEnv = "TOE_LSP_SIGNATURE_ACTIVE_OUT"

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
		_, client, err := lsp.Start(&lsp.TransportConfig{
			Ctx:    t.Context(),
			Name:   "test",
			Server: cfg,
		})
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
		_, _, err := lsp.Start(&lsp.TransportConfig{
			Ctx:  t.Context(),
			Name: "test",
		})

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
}

func (s *processServer) Initialize(
	ctx context.Context, _ *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	var completionProvider *protocol.CompletionOptions
	if os.Getenv(testServerCompletionEnv) == "1" {
		completionProvider = &protocol.CompletionOptions{
			TriggerCharacters: []string{"."},
		}
		if os.Getenv(testServerNoResolveEnv) != "1" {
			completionProvider.ResolveProvider = new(true)
		}
	}
	var signatureProvider *protocol.SignatureHelpOptions
	if os.Getenv(testServerSignatureEnv) == "1" {
		signatureProvider = &protocol.SignatureHelpOptions{
			TriggerCharacters:   []string{"("},
			RetriggerCharacters: []string{","},
		}
	}
	var codeActionProvider protocol.CodeActionProvider
	if os.Getenv(testServerCodeActionEnv) == "1" {
		codeActionProvider = &protocol.CodeActionOptions{
			ResolveProvider: new(true),
		}
		if os.Getenv(testServerNoResolveEnv) == "1" {
			codeActionProvider = protocol.Boolean(true)
		}
	}
	if os.Getenv(testServerWorkspaceFoldersEnv) != "" {
		folders, err := s.client.WorkspaceFolders(ctx)
		if err != nil {
			return nil, err
		}
		s.folders = folders
	}
	if os.Getenv(testServerCallbacksEnv) == "1" {
		_ = s.client.Progress(ctx, &protocol.ProgressParams{
			Token: protocol.String("tok"),
		})
		_ = s.client.LogTrace(ctx, &protocol.LogTraceParams{
			Message: "trace",
		})
		_ = s.client.ShowMessage(ctx, &protocol.ShowMessageParams{
			Message: "hello",
		})
		_ = s.client.LogMessage(ctx, &protocol.LogMessageParams{
			Message: "log",
		})
		_ = s.client.Telemetry(ctx, protocol.LSPAny(`"event"`))
		_, _ = s.client.Configuration(ctx, &protocol.ConfigurationParams{
			Items: []protocol.ConfigurationItem{{}},
		})
		_ = s.client.WorkDoneProgressCreate(ctx,
			&protocol.WorkDoneProgressCreateParams{
				Token: protocol.String("tok"),
			},
		)
		_, _ = s.client.ShowMessageRequest(ctx,
			&protocol.ShowMessageRequestParams{Message: "question"},
		)
		_, _ = s.client.ShowDocument(ctx,
			&protocol.ShowDocumentParams{URI: "file:///dev/null"},
		)
		_, _ = s.client.ApplyEdit(ctx,
			&protocol.ApplyWorkspaceEditParams{
				Edit: protocol.WorkspaceEdit{
					Changes: map[uri.URI][]protocol.TextEdit{
						"untitled:Untitled-1": {{NewText: "x"}},
					},
				},
			},
		)
		_ = s.client.PublishDiagnostics(ctx,
			&protocol.PublishDiagnosticsParams{
				URI: "file:///dev/null",
			},
		)
		_ = s.client.DiagnosticRefresh(ctx)
	}
	navigation := protocol.Boolean(os.Getenv(testServerNavigationEnv) == "1")
	symbols := protocol.Boolean(os.Getenv(testServerSymbolsEnv) == "1")
	workspaceSymbols := protocol.Boolean(
		os.Getenv(testServerWorkspaceSymbolsEnv) == "1",
	)
	var renameProvider protocol.RenameProvider
	if os.Getenv(testServerRenameEnv) == "1" {
		renameProvider = &protocol.RenameOptions{
			PrepareProvider: new(true),
		}
	}
	var diagnosticProvider protocol.DiagnosticProvider
	if os.Getenv(testServerPullDiagnosticsEnv) == "1" {
		id := "test"
		if os.Getenv(testServerDiagRegOptionsEnv) == "1" {
			diagnosticProvider = &protocol.DiagnosticRegistrationOptions{
				DiagnosticOptions: protocol.DiagnosticOptions{Identifier: &id},
			}
		} else {
			diagnosticProvider = &protocol.DiagnosticOptions{Identifier: &id}
		}
	}
	format := protocol.Boolean(os.Getenv(testServerFormatEnv) == "1")
	var docLinkProvider *protocol.DocumentLinkOptions
	if os.Getenv(testServerDocumentLinkEnv) == "1" {
		docLinkProvider = &protocol.DocumentLinkOptions{
			ResolveProvider: new(
				os.Getenv(testServerDocumentLinkResolveEnv) == "1",
			),
		}
	}
	inlayHints := protocol.Boolean(os.Getenv(testServerInlayHintsEnv) == "1")
	docColors := protocol.Boolean(os.Getenv(testServerDocumentColorEnv) == "1")
	var workspace *protocol.WorkspaceOptions
	if os.Getenv(testServerFileOperationsEnv) == "1" {
		fileKind := protocol.FileOperationPatternKindFile
		fileFilter := []protocol.FileOperationFilter{
			{
				Scheme: new("https"),
				Pattern: protocol.FileOperationPattern{
					Glob:    "**",
					Matches: fileKind,
				},
			},
			{
				Pattern: protocol.FileOperationPattern{
					Glob:    "*.SESSION",
					Matches: fileKind,
					Options: &protocol.FileOperationPatternOptions{
						IgnoreCase: new(true),
					},
				},
			},
			{
				Pattern: protocol.FileOperationPattern{
					Glob:    "**",
					Matches: fileKind,
				},
			},
		}
		fileOps := protocol.FileOperationRegistrationOptions{
			Filters: fileFilter,
		}
		workspace = &protocol.WorkspaceOptions{
			FileOperations: &protocol.FileOperationOptions{
				WillCreate: fileOps,
				DidCreate:  fileOps,
				WillRename: fileOps,
				DidRename:  fileOps,
				WillDelete: fileOps,
				DidDelete:  fileOps,
			},
		}
	}
	if os.Getenv(testServerFileOpFolderEnv) == "1" {
		folderKind := protocol.FileOperationPatternKindFolder
		folderFilter := []protocol.FileOperationFilter{{
			Pattern: protocol.FileOperationPattern{
				Glob:    "**",
				Matches: folderKind,
			},
		}}
		folderOps := protocol.FileOperationRegistrationOptions{
			Filters: folderFilter,
		}
		workspace = &protocol.WorkspaceOptions{
			FileOperations: &protocol.FileOperationOptions{
				WillCreate: folderOps,
				DidCreate:  folderOps,
				WillRename: folderOps,
				DidRename:  folderOps,
				WillDelete: folderOps,
				DidDelete:  folderOps,
			},
		}
	}
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			PositionEncoding: protocol.PositionEncodingKindUTF16,
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: new(true),
				Change:    new(protocol.TextDocumentSyncKindFull),
			},
			CompletionProvider:    completionProvider,
			SignatureHelpProvider: signatureProvider,
			HoverProvider:         protocol.Boolean(true),
			ExecuteCommandProvider: protocol.ExecuteCommandOptions{
				Commands: []string{"session.afterCompletion"},
			},
			DeclarationProvider:             navigation,
			DefinitionProvider:              navigation,
			TypeDefinitionProvider:          navigation,
			ImplementationProvider:          navigation,
			ReferencesProvider:              navigation,
			CodeActionProvider:              codeActionProvider,
			DocumentSymbolProvider:          symbols,
			WorkspaceSymbolProvider:         workspaceSymbols,
			RenameProvider:                  renameProvider,
			DiagnosticProvider:              diagnosticProvider,
			DocumentFormattingProvider:      format,
			DocumentRangeFormattingProvider: format,
			DocumentHighlightProvider: protocol.Boolean(
				os.Getenv(testServerHighlightEnv) == "1",
			),
			DocumentLinkProvider: docLinkProvider,
			InlayHintProvider:    inlayHints,
			ColorProvider:        docColors,
			Workspace:            workspace,
		},
	}, nil
}

func (s *processServer) Initialized(
	ctx context.Context, _ *protocol.InitializedParams,
) error {
	if os.Getenv(testServerProgressEnv) == "1" {
		tok := protocol.String("test-progress")
		_ = s.client.WorkDoneProgressCreate(ctx,
			&protocol.WorkDoneProgressCreateParams{
				Token: tok,
			},
		)
		begin := `{"kind":"begin","title":"Loading","message":"start"}`
		_ = s.client.Progress(ctx, &protocol.ProgressParams{
			Token: tok,
			Value: protocol.LSPAny(begin),
		})
		report := `{"kind":"report","message":"halfway","percentage":50}`
		_ = s.client.Progress(ctx, &protocol.ProgressParams{
			Token: tok,
			Value: protocol.LSPAny(report),
		})
		_ = s.client.Progress(ctx, &protocol.ProgressParams{
			Token: tok,
			Value: protocol.LSPAny(`{"kind":"end","message":"done"}`),
		})
		iTok := protocol.Integer(42)
		_ = s.client.WorkDoneProgressCreate(ctx,
			&protocol.WorkDoneProgressCreateParams{
				Token: iTok,
			},
		)
		_ = s.client.Progress(ctx, &protocol.ProgressParams{
			Token: iTok,
			Value: protocol.LSPAny(`{"kind":"begin","title":"Indexing"}`),
		})
		_ = s.client.Progress(ctx, &protocol.ProgressParams{
			Token: iTok,
			Value: protocol.LSPAny(`{"kind":"end"}`),
		})
		_ = s.client.Progress(ctx, &protocol.ProgressParams{
			Token: protocol.String("unknown"),
			Value: protocol.LSPAny(`{"kind":"report","message":"ghost"}`),
		})
	}
	if os.Getenv(testServerFileWatchEnv) == "1" {
		watchers := `{"watchers":[` +
			`{"globPattern":"*.session"},` +
			`{"globPattern":{"pattern":"*.session","baseUri":"file:///tmp"}},` +
			`{"globPattern":{"pattern":"**/*.session",` +
			`"baseUri":{"uri":"file:///tmp","name":"myFolder"}}},` +
			`{"globPattern":""}` +
			`]}`
		_ = s.client.RegisterCapability(ctx, &protocol.RegistrationParams{
			Registrations: []protocol.Registration{{
				ID:              "watch-session",
				Method:          "workspace/didChangeWatchedFiles",
				RegisterOptions: protocol.LSPAny(watchers),
			}},
		})
		_ = s.client.RegisterCapability(ctx, &protocol.RegistrationParams{
			Registrations: []protocol.Registration{{
				ID:     "watch-tmp",
				Method: "workspace/didChangeWatchedFiles",
				RegisterOptions: protocol.LSPAny(
					`{"watchers":[{"globPattern":"*.tmp"}]}`,
				),
			}},
		})
		_ = s.client.UnregisterCapability(ctx, &protocol.UnregistrationParams{
			Unregisterations: []protocol.Unregistration{{
				ID:     "watch-tmp",
				Method: "workspace/didChangeWatchedFiles",
			}},
		})
	}
	if os.Getenv(testServerWatchRegEdgeEnv) == "1" {
		_ = s.client.RegisterCapability(ctx, nil)
		_ = s.client.RegisterCapability(ctx, &protocol.RegistrationParams{
			Registrations: []protocol.Registration{{
				ID: "other", Method: "textDocument/other",
			}},
		})
		_ = s.client.RegisterCapability(ctx, &protocol.RegistrationParams{
			Registrations: []protocol.Registration{{
				ID:              "bad-json",
				Method:          "workspace/didChangeWatchedFiles",
				RegisterOptions: protocol.LSPAny("not-json"),
			}},
		})
		_ = s.client.RegisterCapability(ctx, &protocol.RegistrationParams{
			Registrations: []protocol.Registration{{
				ID:     "solo",
				Method: "workspace/didChangeWatchedFiles",
				RegisterOptions: protocol.LSPAny(
					`{"watchers":[{"globPattern":"*.watched"}]}`,
				),
			}},
		})
		_ = s.client.UnregisterCapability(ctx, nil)
		_ = s.client.UnregisterCapability(ctx, &protocol.UnregistrationParams{
			Unregisterations: []protocol.Unregistration{{
				ID: "solo", Method: "textDocument/other",
			}},
		})
		_ = s.client.UnregisterCapability(ctx, &protocol.UnregistrationParams{
			Unregisterations: []protocol.Unregistration{{
				ID: "solo", Method: "workspace/didChangeWatchedFiles",
			}},
		})
	}
	return nil
}

func (s *processServer) DidChangeWatchedFiles(
	_ context.Context, _ *protocol.DidChangeWatchedFilesParams,
) error {
	path := os.Getenv(testServerFileWatchNotifyEnv)
	if path != "" {
		_ = os.WriteFile(path, []byte("watched"), 0o644)
	}
	return nil
}

func (s *processServer) InlayHint(
	_ context.Context, _ *protocol.InlayHintParams,
) ([]protocol.InlayHint, error) {
	if os.Getenv(testServerInlayHintsEnv) != "1" {
		return nil, nil
	}
	return []protocol.InlayHint{
		{
			Position:     protocol.Position{Line: 0, Character: 0},
			Label:        protocol.String("string-label"),
			Kind:         protocol.InlayHintKindType,
			PaddingLeft:  new(true),
			PaddingRight: new(true),
		},
		{
			Position: protocol.Position{Line: 0, Character: 0},
			Label: protocol.InlayHintLabelPartSlice{
				{Value: "part1"},
				{Value: "-part2"},
			},
			Kind: protocol.InlayHintKindParameter,
		},
	}, nil
}

func (s *processServer) DocumentColor(
	_ context.Context, _ *protocol.DocumentColorParams,
) ([]protocol.ColorInformation, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("document color error")
	}
	if os.Getenv(testServerDocumentColorEnv) != "1" {
		return nil, nil
	}
	return []protocol.ColorInformation{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 3},
			},
			Color: protocol.Color{Red: 1.0, Green: 0.5, Blue: 0.0},
		},
	}, nil
}

func (s *processServer) DidChange(
	_ context.Context, _ *protocol.DidChangeTextDocumentParams,
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
	if os.Getenv(testServerManyCompletionsEnv) == "1" {
		return manyCompletionItems(), nil
	}
	if os.Getenv(testServerAltCompletionEnv) == "1" {
		return altCompletionItems(), nil
	}
	if os.Getenv(testServerCompletionListEnv) == "1" {
		return &protocol.CompletionList{
			IsIncomplete: true,
			Items: []protocol.CompletionItem{
				{Label: "Println", Kind: protocol.CompletionItemKindFunction},
			},
		}, nil
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
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("completion resolve error")
	}
	item.Detail = protocol.NewOptional("func Println(a ...any)")
	item.Documentation = protocol.String("Println formats its operands.")
	return item, nil
}

func (s *processServer) Hover(
	context.Context, *protocol.HoverParams,
) (*protocol.Hover, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("hover error")
	}
	switch os.Getenv(testServerHoverContentEnv) {
	case "string":
		return &protocol.Hover{
			Contents: protocol.String("hover string"),
		}, nil
	case "marked":
		return &protocol.Hover{
			Contents: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "```go\nfunc Foo()\n```",
			},
		}, nil
	case "markdown":
		return &protocol.Hover{
			Contents: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "**bold**",
			},
		}, nil
	default:
		return &protocol.Hover{
			Contents: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "hover docs",
			},
		}, nil
	}
}

func (s *processServer) SignatureHelp(
	context.Context, *protocol.SignatureHelpParams,
) (*protocol.SignatureHelp, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("signature error")
	}
	if os.Getenv(testServerSignatureEnv) != "1" {
		return nil, nil
	}
	if os.Getenv(testServerSignatureHelpActiveEnv) == "1" {
		return signatureHelpFromJSON(`{
			"activeParameter": 1,
			"signatures": [{
				"label": "Println(a, b)",
				"parameters": [
					{"label": "a"},
					{"label": "b", "documentation": "second docs"}
				]
			}]
		}`)
	}
	if os.Getenv(testServerSignatureNoParamsEnv) == "1" {
		return signatureHelpFromJSON(`{
			"signatures": [{"label": "Now()"}]
		}`)
	}
	if os.Getenv(testServerSignatureMissingLabelEnv) == "1" {
		return signatureHelpFromJSON(`{
			"signatures": [{
				"label": "Println(a)",
				"parameters": [{"label": "missing"}]
			}]
		}`)
	}
	if os.Getenv(testServerSignatureActiveOutEnv) == "1" {
		return signatureHelpFromJSON(`{
			"activeSignature": 3,
			"signatures": [{"label": "Now()"}]
		}`)
	}
	return &protocol.SignatureHelp{
		ActiveSignature: new(uint32(0)),
		Signatures: []protocol.SignatureInformation{
			{
				Label:         "Println(a ...any)",
				Documentation: protocol.String("signature docs"),
				Parameters: []protocol.ParameterInformation{
					{
						Label: func() protocol.ParameterInformationLabel {
							if os.Getenv(testServerSignatureOffsetEnv) == "1" {
								return protocol.ParameterInformationLabelTuple{
									8, 16,
								}
							}
							return protocol.String("a ...any")
						}(),
						Documentation: protocol.String("parameter docs"),
					},
				},
			},
		},
	}, nil
}

func signatureHelpFromJSON(text string) (*protocol.SignatureHelp, error) {
	var help protocol.SignatureHelp
	if err := protocol.Unmarshal([]byte(text), &help); err != nil {
		return nil, err
	}
	return &help, nil
}

func (s *processServer) Declaration(
	context.Context, *protocol.DeclarationParams,
) (protocol.DeclarationResult, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("declaration error")
	}
	if os.Getenv(testServerNavigationEnv) != "1" {
		return nil, nil
	}
	if os.Getenv(testServerNavLinksEnv) == "1" {
		return s.navigationLinkSlice(), nil
	}
	if os.Getenv(testServerNavLocationSliceEnv) == "1" {
		return s.navigationLocationSlice(), nil
	}
	return s.navigationLocation(), nil
}

func (s *processServer) Definition(
	context.Context, *protocol.DefinitionParams,
) (protocol.DefinitionResult, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("definition error")
	}
	if os.Getenv(testServerNavigationEnv) != "1" {
		return nil, nil
	}
	if os.Getenv(testServerNavLinksEnv) == "1" {
		return s.navigationDefLinkSlice(), nil
	}
	if os.Getenv(testServerNavLocationSliceEnv) == "1" {
		return s.navigationLocationSlice(), nil
	}
	return s.navigationLocation(), nil
}

func (s *processServer) TypeDefinition(
	context.Context, *protocol.TypeDefinitionParams,
) (protocol.DefinitionResult, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("type definition error")
	}
	if os.Getenv(testServerNavigationEnv) != "1" {
		return nil, nil
	}
	return s.navigationLocation(), nil
}

func (s *processServer) Implementation(
	context.Context, *protocol.ImplementationParams,
) (protocol.DefinitionResult, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("implementation error")
	}
	if os.Getenv(testServerNavigationEnv) != "1" {
		return nil, nil
	}
	return s.navigationLocation(), nil
}

func (s *processServer) References(
	_ context.Context, params *protocol.ReferenceParams,
) ([]protocol.Location, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("references error")
	}
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
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("document symbol error")
	}
	if os.Getenv(testServerSymbolsEnv) != "1" {
		return nil, nil
	}
	if os.Getenv(testServerManySymbolsEnv) == "1" {
		return manyDocumentSymbols(), nil
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

func altCompletionItems() protocol.CompletionItemSlice {
	insertReplaceEdit := &protocol.InsertReplaceEdit{
		NewText: "Println",
		Insert: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 2},
		},
		Replace: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 2},
		},
	}
	sortA := "aaa"
	sortB := "bbb"
	sortZ := "zzz"
	return protocol.CompletionItemSlice{
		// Preselect=true, Sort="aaa" (should be first); has a Command
		{
			Label:     "Println",
			Kind:      protocol.CompletionItemKindFunction,
			TextEdit:  insertReplaceEdit,
			Preselect: protocol.NewOptional(true),
			SortText:  protocol.NewOptional(sortA),
			LabelDetails: &protocol.CompletionItemLabelDetails{
				Detail:      new("(n int, format string)"),
				Description: new("fmt"),
			},
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "Prints a line",
			},
			Command: protocol.Command{
				Command: "session.afterCompletion",
				Title:   "after",
			},
		},
		// Preselect=false, no TextEdit (label fallback)
		{
			Label:     "Printf",
			Kind:      protocol.CompletionItemKindFunction,
			Preselect: protocol.NewOptional(false),
			SortText:  protocol.NewOptional(sortZ),
		},
		// Preselect=true, Sort="bbb" (exercises Sort > branch vs "aaa")
		{
			Label:     "Putchar",
			Kind:      protocol.CompletionItemKindFunction,
			Preselect: protocol.NewOptional(true),
			SortText:  protocol.NewOptional(sortB),
		},
		// Preselect=false, Sort="zzz" (exercises Label < branch vs Printf)
		{
			Label:     "Puts",
			Kind:      protocol.CompletionItemKindFunction,
			Preselect: protocol.NewOptional(false),
			SortText:  protocol.NewOptional(sortZ),
		},
	}
}

func manyCompletionItems() protocol.CompletionItemSlice {
	kinds := []protocol.CompletionItemKind{
		protocol.CompletionItemKindText,
		protocol.CompletionItemKindMethod,
		protocol.CompletionItemKindFunction,
		protocol.CompletionItemKindConstructor,
		protocol.CompletionItemKindField,
		protocol.CompletionItemKindVariable,
		protocol.CompletionItemKindClass,
		protocol.CompletionItemKindInterface,
		protocol.CompletionItemKindModule,
		protocol.CompletionItemKindProperty,
		protocol.CompletionItemKindUnit,
		protocol.CompletionItemKindValue,
		protocol.CompletionItemKindEnum,
		protocol.CompletionItemKindKeyword,
		protocol.CompletionItemKindSnippet,
		protocol.CompletionItemKindColor,
		protocol.CompletionItemKindFile,
		protocol.CompletionItemKindReference,
		protocol.CompletionItemKindFolder,
		protocol.CompletionItemKindEnumMember,
		protocol.CompletionItemKindConstant,
		protocol.CompletionItemKindStruct,
		protocol.CompletionItemKindEvent,
		protocol.CompletionItemKindOperator,
		protocol.CompletionItemKindTypeParameter,
		255,
	}
	out := make(protocol.CompletionItemSlice, 0, len(kinds))
	for i, k := range kinds {
		label := fmt.Sprintf("item%d", i)
		out = append(out, protocol.CompletionItem{Label: label, Kind: k})
	}
	return out
}

func manyDocumentSymbols() protocol.DocumentSymbolSlice {
	zeroRange := protocol.Range{}
	kinds := []protocol.SymbolKind{
		protocol.SymbolKindFile, protocol.SymbolKindModule,
		protocol.SymbolKindNamespace, protocol.SymbolKindPackage,
		protocol.SymbolKindClass, protocol.SymbolKindMethod,
		protocol.SymbolKindProperty, protocol.SymbolKindField,
		protocol.SymbolKindConstructor, protocol.SymbolKindEnum,
		protocol.SymbolKindInterface, protocol.SymbolKindFunction,
		protocol.SymbolKindVariable, protocol.SymbolKindConstant,
		protocol.SymbolKindString, protocol.SymbolKindNumber,
		protocol.SymbolKindBoolean, protocol.SymbolKindArray,
		protocol.SymbolKindObject, protocol.SymbolKindKey,
		protocol.SymbolKindNull, protocol.SymbolKindEnumMember,
		protocol.SymbolKindStruct, protocol.SymbolKindEvent,
		protocol.SymbolKindOperator, protocol.SymbolKindTypeParameter,
		255,
	}
	out := make(protocol.DocumentSymbolSlice, 0, len(kinds))
	for i, k := range kinds {
		name := fmt.Sprintf("sym%d", i)
		out = append(out, protocol.DocumentSymbol{
			Name: name, Kind: k, SelectionRange: zeroRange,
		})
	}
	return out
}

func (s *processServer) Symbols(
	_ context.Context, params *protocol.WorkspaceSymbolParams,
) (protocol.WorkspaceSymbolResult, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("symbols error")
	}
	if os.Getenv(testServerWorkspaceSymbolsEnv) != "1" {
		return nil, nil
	}
	loc := s.navigationLocation()
	if loc == nil || params.Query == "" {
		return protocol.SymbolInformationSlice{}, nil
	}
	container := "workspace"
	if os.Getenv(testServerSymbolSliceEnv) == "1" {
		return protocol.WorkspaceSymbolSlice{
			{
				BaseSymbolInformation: protocol.BaseSymbolInformation{
					Name: "WorkspaceMain", Kind: protocol.SymbolKindFunction,
					ContainerName: &container,
				},
				Location: loc,
				Data:     protocol.LSPAny("null"),
			},
		}, nil
	}
	return protocol.SymbolInformationSlice{
		{
			BaseSymbolInformation: protocol.BaseSymbolInformation{
				Name: "WorkspaceMain", Kind: protocol.SymbolKindFunction,
				ContainerName: &container,
			},
			Location: *loc,
		},
	}, nil
}

func (s *processServer) PrepareRename(
	context.Context, *protocol.PrepareRenameParams,
) (protocol.PrepareRenameResult, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("prepare rename error")
	}
	if os.Getenv(testServerRenameEnv) != "1" {
		return nil, nil
	}
	if os.Getenv(testServerRenameDefaultBehaviorEnv) == "1" {
		return &protocol.PrepareRenameDefaultBehavior{
			DefaultBehavior: true,
		}, nil
	}
	if os.Getenv(testServerRenameRangeEnv) == "1" {
		r := protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 3},
		}
		return &r, nil
	}
	return &protocol.PrepareRenamePlaceholder{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 3},
		},
		Placeholder: "old",
	}, nil
}

func (s *processServer) Rename(
	_ context.Context, params *protocol.RenameParams,
) (*protocol.WorkspaceEdit, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("rename error")
	}
	if os.Getenv(testServerRenameEnv) != "1" {
		return nil, nil
	}
	return &protocol.WorkspaceEdit{
		Changes: map[uri.URI][]protocol.TextEdit{
			params.TextDocument.URI: {
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 3},
					},
					NewText: params.NewName,
				},
			},
		},
	}, nil
}

func (s *processServer) DocumentHighlight(
	_ context.Context, params *protocol.DocumentHighlightParams,
) ([]protocol.DocumentHighlight, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("highlight error")
	}
	if os.Getenv(testServerHighlightEnv) != "1" {
		return nil, nil
	}
	pos := params.Position
	if os.Getenv(testServerHighlightMultiEnv) == "1" {
		return []protocol.DocumentHighlight{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: pos.Line, Character: 0},
					End:   protocol.Position{Line: pos.Line, Character: 3},
				},
				Kind: protocol.DocumentHighlightKindText,
			},
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: pos.Line, Character: 0},
					End:   protocol.Position{Line: pos.Line, Character: 5},
				},
				Kind: protocol.DocumentHighlightKindText,
			},
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: pos.Line, Character: 6},
					End:   protocol.Position{Line: pos.Line, Character: 7},
				},
				Kind: protocol.DocumentHighlightKindText,
			},
		}, nil
	}
	return []protocol.DocumentHighlight{
		{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      pos.Line,
					Character: pos.Character,
				},
				End: protocol.Position{
					Line:      pos.Line,
					Character: pos.Character + 3,
				},
			},
			Kind: protocol.DocumentHighlightKindText,
		},
	}, nil
}

func (s *processServer) Formatting(
	_ context.Context, _ *protocol.DocumentFormattingParams,
) ([]protocol.TextEdit, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("format error")
	}
	if os.Getenv(testServerFormatEnv) != "1" {
		return nil, nil
	}
	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 3},
			},
			NewText: "fmt",
		},
	}, nil
}

func (s *processServer) RangeFormatting(
	_ context.Context, _ *protocol.DocumentRangeFormattingParams,
) ([]protocol.TextEdit, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("range format error")
	}
	if os.Getenv(testServerFormatEnv) != "1" {
		return nil, nil
	}
	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 3},
			},
			NewText: "rng",
		},
	}, nil
}

func (s *processServer) Diagnostic(
	_ context.Context, _ *protocol.DocumentDiagnosticParams,
) (protocol.DocumentDiagnosticReport, error) {
	if os.Getenv(testServerPullDiagnosticsEnv) != "1" {
		return nil, nil
	}
	if os.Getenv(testServerDiagnosticErrorEnv) == "1" {
		return nil, errors.New("diagnostic server error")
	}
	if os.Getenv(testServerDiagnosticUnchangedEnv) == "1" {
		resultID := "unchanged-result-1"
		report := protocol.UnchangedDocumentDiagnosticReport{
			Kind:     string(protocol.DocumentDiagnosticReportKindUnchanged),
			ResultID: resultID,
		}
		return &protocol.RelatedUnchangedDocumentDiagnosticReport{
			UnchangedDocumentDiagnosticReport: report,
		}, nil
	}
	return &protocol.RelatedFullDocumentDiagnosticReport{
		FullDocumentDiagnosticReport: protocol.FullDocumentDiagnosticReport{
			Kind:     string(protocol.DocumentDiagnosticReportKindFull),
			ResultID: new("full-result-1"),
			Items: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 3},
					},
					Message:  protocol.String("test diagnostic"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 3},
					},
					Message: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: "**warning**: test",
					},
					Severity: protocol.DiagnosticSeverityWarning,
				},
			},
		},
	}, nil
}

func (s *processServer) ExecuteCommand(
	_ context.Context, _ *protocol.ExecuteCommandParams,
) (protocol.LSPAny, error) {
	return nil, nil
}

func (s *processServer) DocumentLink(
	_ context.Context, _ *protocol.DocumentLinkParams,
) ([]protocol.DocumentLink, error) {
	if os.Getenv(testServerDocumentLinkEnv) != "1" {
		return nil, nil
	}
	resolvable := os.Getenv(testServerDocumentLinkResolveEnv) == "1"
	link := protocol.DocumentLink{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 3},
		},
	}
	if resolvable {
		link.Data = protocol.LSPAny(`"resolve-me"`)
	} else {
		link.Target = new(uri.File(os.Getenv(testServerNavigationTargetEnv)))
	}
	empty := protocol.DocumentLink{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 1},
			End:   protocol.Position{Line: 0, Character: 1},
		},
	}
	return []protocol.DocumentLink{link, empty}, nil
}

func (s *processServer) DocumentLinkResolve(
	_ context.Context, link *protocol.DocumentLink,
) (*protocol.DocumentLink, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("document link resolve error")
	}
	link.Target = new(uri.File(os.Getenv(testServerNavigationTargetEnv)))
	return link, nil
}

func (s *processServer) CodeAction(
	_ context.Context, params *protocol.CodeActionParams,
) ([]protocol.CommandOrCodeAction, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("code action error")
	}
	if os.Getenv(testServerCodeActionEnv) != "1" {
		return nil, nil
	}
	kind := protocol.CodeActionKindQuickFix
	actions := []protocol.CommandOrCodeAction{
		&protocol.CodeAction{
			Title:       "Fix old",
			Kind:        &kind,
			Diagnostics: params.Context.Diagnostics,
			IsPreferred: new(true),
			Edit: &protocol.WorkspaceEdit{
				Changes: map[uri.URI][]protocol.TextEdit{
					params.TextDocument.URI: {
						{
							Range: protocol.Range{
								Start: protocol.Position{
									Line: 0, Character: 0,
								},
								End: protocol.Position{
									Line: 0, Character: 3,
								},
							},
							NewText: "new",
						},
					},
				},
			},
		},
		&protocol.CodeAction{
			Title: "Disabled",
			Disabled: protocol.CodeActionDisabled{
				Reason: "not now",
			},
		},
	}
	if os.Getenv(testServerMultiCodeActionEnv) == "1" {
		actions = append(actions,
			&protocol.CodeAction{
				Title: "Extract method",
				Kind:  new(protocol.CodeActionKindRefactorExtract),
			},
			&protocol.CodeAction{
				Title: "Inline method",
				Kind:  new(protocol.CodeActionKindRefactorInline),
			},
			&protocol.CodeAction{
				Title: "Rewrite method",
				Kind:  new(protocol.CodeActionKindRefactorRewrite),
			},
			&protocol.CodeAction{
				Title: "Move method",
				Kind:  new(protocol.CodeActionKindRefactorMove),
			},
			&protocol.CodeAction{
				Title: "Surround method",
				Kind:  new(protocol.CodeActionKind("refactor.surround")),
			},
			&protocol.CodeAction{
				Title: "Refactor",
				Kind:  new(protocol.CodeActionKindRefactor),
			},
			&protocol.CodeAction{
				Title: "Organize imports",
				Kind:  new(protocol.CodeActionKindSource),
			},
			&protocol.CodeAction{
				Title: "Unknown action",
				Kind:  new(protocol.CodeActionKind("unknown.kind")),
			},
			&protocol.Command{
				Title:   "Run formatter",
				Command: "session.afterCompletion",
			},
			&protocol.Command{Title: ""},
			&protocol.CodeAction{Title: ""},
			&protocol.CodeAction{
				Title: "Edit and command",
				Kind:  new(protocol.CodeActionKindQuickFix),
				Edit: &protocol.WorkspaceEdit{
					Changes: map[uri.URI][]protocol.TextEdit{
						params.TextDocument.URI: {
							{
								Range: protocol.Range{
									Start: protocol.Position{
										Line: 0, Character: 0,
									},
									End: protocol.Position{
										Line: 0, Character: 0,
									},
								},
								NewText: "",
							},
						},
					},
				},
				Command: protocol.Command{
					Title:   "after",
					Command: "session.afterCompletion",
				},
			},
		)
	}
	if os.Getenv(testServerDiagCodeActionEnv) == "1" {
		actions = append(actions, &protocol.CodeAction{
			Title: "Fix other",
			Kind:  &kind,
		})
	}
	if os.Getenv(testServerWsEditCodeActionEnv) == "1" {
		oldPath := os.Getenv(testServerWsEditOldPathEnv)
		newPath := os.Getenv(testServerWsEditNewPathEnv)
		actions = append(actions, &protocol.CodeAction{
			Title: "Create file",
			Kind:  new(protocol.CodeActionKindQuickFix),
			Edit: &protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.CreateFile{
						URI:  uri.File(newPath + ".created"),
						Kind: "create",
					},
				},
			},
		})
		if oldPath != "" && newPath != "" {
			renameKind := protocol.CodeActionKindRefactorRewrite
			actions = append(actions,
				&protocol.CodeAction{
					Title: "Rename file",
					Kind:  &renameKind,
					Edit: &protocol.WorkspaceEdit{
						DocumentChanges: []protocol.DocumentChange{
							&protocol.RenameFile{
								OldURI: uri.File(oldPath),
								NewURI: uri.File(newPath + ".renamed"),
								Kind:   "rename",
							},
						},
					},
				},
				&protocol.CodeAction{
					Title: "Delete file",
					Kind:  &renameKind,
					Edit: &protocol.WorkspaceEdit{
						DocumentChanges: []protocol.DocumentChange{
							&protocol.DeleteFile{
								URI:  uri.File(newPath + ".created"),
								Kind: "delete",
							},
						},
					},
				},
			)
		}
	}
	return actions, nil
}

func (s *processServer) CodeActionResolve(
	_ context.Context, action *protocol.CodeAction,
) (*protocol.CodeAction, error) {
	switch os.Getenv(testServerCAResolveEnv) {
	case "error":
		return nil, errors.New("code action resolve error")
	case "nil":
		return nil, nil
	}
	return action, nil
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

func (s *processServer) navigationLocationSlice() protocol.LocationSlice {
	loc := s.navigationLocation()
	if loc == nil {
		return nil
	}
	return protocol.LocationSlice{*loc}
}

func (s *processServer) navigationLinkSlice() protocol.DeclarationLinkSlice {
	path := os.Getenv(testServerNavigationTargetEnv)
	if path == "" {
		return nil
	}
	rng := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 3},
		End:   protocol.Position{Line: 0, Character: 6},
	}
	link := protocol.LocationLink{
		TargetURI:            uri.File(path),
		TargetRange:          rng,
		TargetSelectionRange: rng,
	}
	return protocol.DeclarationLinkSlice{protocol.DeclarationLink(link)}
}

func (s *processServer) navigationDefLinkSlice() protocol.DefinitionLinkSlice {
	path := os.Getenv(testServerNavigationTargetEnv)
	if path == "" {
		return nil
	}
	rng := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 3},
		End:   protocol.Position{Line: 0, Character: 6},
	}
	link := protocol.LocationLink{
		TargetURI:            uri.File(path),
		TargetRange:          rng,
		TargetSelectionRange: rng,
	}
	return protocol.DefinitionLinkSlice{protocol.DefinitionLink(link)}
}

func (s *processServer) WillCreateFiles(
	_ context.Context, _ *protocol.CreateFilesParams,
) (*protocol.WorkspaceEdit, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("will create error")
	}
	if os.Getenv(testServerFileOpWillEditEnv) == "1" {
		return &protocol.WorkspaceEdit{}, nil
	}
	return nil, nil
}

func (s *processServer) DidCreateFiles(
	_ context.Context, _ *protocol.CreateFilesParams,
) error {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return errors.New("did create error")
	}
	return nil
}

func (s *processServer) WillRenameFiles(
	_ context.Context, _ *protocol.RenameFilesParams,
) (*protocol.WorkspaceEdit, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("will rename error")
	}
	return nil, nil
}

func (s *processServer) DidRenameFiles(
	_ context.Context, _ *protocol.RenameFilesParams,
) error {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return errors.New("did rename error")
	}
	return nil
}

func (s *processServer) WillDeleteFiles(
	_ context.Context, _ *protocol.DeleteFilesParams,
) (*protocol.WorkspaceEdit, error) {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return nil, errors.New("will delete error")
	}
	return nil, nil
}

func (s *processServer) DidDeleteFiles(
	_ context.Context, _ *protocol.DeleteFilesParams,
) error {
	if os.Getenv(testServerAllErrorEnv) == "1" {
		return errors.New("did delete error")
	}
	return nil
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
