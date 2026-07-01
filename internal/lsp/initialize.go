package lsp

import (
	"fmt"
	"os"
	"path/filepath"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// InitializeConfig holds parameters for LSP server initialization
type InitializeConfig struct {
	ClientName            string
	ClientVersion         string
	WorkspaceRoot         string
	InitializationOptions protocol.LSPAny
}

// DefaultClientName is the client identifier sent to language servers
const DefaultClientName = "toe"

// NewInitializeParams builds an InitializeParams message from the given config
func NewInitializeParams(cfg InitializeConfig) *protocol.InitializeParams {
	pid := int32(os.Getpid())
	params := &protocol.InitializeParams{}
	params.ProcessID = &pid
	params.ClientInfo = protocol.ClientInfo{
		Name:    clientName(cfg.ClientName),
		Version: protocol.NewOptional(cfg.ClientVersion),
	}
	params.Capabilities = DefaultClientCapabilities()
	params.InitializationOptions = cfg.InitializationOptions
	setInitializeWorkspace(params, cfg.WorkspaceRoot)
	return params
}

// DefaultClientCapabilities returns the editor's LSP client capabilities
func DefaultClientCapabilities() protocol.ClientCapabilities {
	yes := true
	no := false
	return protocol.ClientCapabilities{
		Workspace: &protocol.WorkspaceClientCapabilities{
			Configuration:          &yes,
			WorkspaceFolders:       &yes,
			ApplyEdit:              &yes,
			DidChangeConfiguration: dynamicRegistration(false),
			DidChangeWatchedFiles: &protocol.
				DidChangeWatchedFilesClientCapabilities{
				DynamicRegistration:    &yes,
				RelativePatternSupport: &yes,
			},
			Symbol: &protocol.WorkspaceSymbolClientCapabilities{
				DynamicRegistration: &no,
			},
			WorkspaceEdit: &protocol.WorkspaceEditClientCapabilities{
				DocumentChanges: &yes,
				ResourceOperations: []protocol.ResourceOperationKind{
					protocol.ResourceOperationKindCreate,
					protocol.ResourceOperationKindRename,
					protocol.ResourceOperationKindDelete,
				},
				FailureHandling:       protocol.FailureHandlingKindAbort,
				NormalizesLineEndings: &no,
			},
			InlayHint: &protocol.InlayHintWorkspaceClientCapabilities{
				RefreshSupport: &no,
			},
			FileOperations: &protocol.FileOperationClientCapabilities{
				WillCreate: &yes,
				DidCreate:  &yes,
				WillRename: &yes,
				DidRename:  &yes,
				WillDelete: &yes,
				DidDelete:  &yes,
			},
		},
		TextDocument: &protocol.TextDocumentClientCapabilities{
			Synchronization: &protocol.TextDocumentSyncClientCapabilities{
				DidSave: &yes,
			},
			Completion: &protocol.CompletionClientCapabilities{
				ContextSupport: &yes,
				CompletionItem: &protocol.ClientCompletionItemOptions{
					SnippetSupport:    &no,
					DeprecatedSupport: &yes,
					DocumentationFormat: []protocol.MarkupKind{
						protocol.MarkupKindMarkdown,
						protocol.MarkupKindPlainText,
					},
					TagSupport: protocol.CompletionItemTagOptions{
						ValueSet: []protocol.CompletionItemTag{
							protocol.CompletionItemTagDeprecated,
						},
					},
					LabelDetailsSupport: &yes,
					ResolveSupport: protocol.ClientCompletionItemResolveOptions{
						Properties: []string{
							"documentation",
							"detail",
							"additionalTextEdits",
						},
					},
				},
			},
			Hover: &protocol.HoverClientCapabilities{
				ContentFormat: []protocol.MarkupKind{
					protocol.MarkupKindMarkdown,
					protocol.MarkupKindPlainText,
				},
			},
			SignatureHelp: &protocol.SignatureHelpClientCapabilities{
				ContextSupport:       &yes,
				SignatureInformation: signatureInfoCapabilities(),
			},
			DocumentSymbol: &protocol.DocumentSymbolClientCapabilities{
				HierarchicalDocumentSymbolSupport: &no,
			},
			CodeAction: &protocol.CodeActionClientCapabilities{
				DynamicRegistration: &no,
				ResolveSupport: protocol.ClientCodeActionResolveOptions{
					Properties: []string{"edit", "command"},
				},
			},
			DocumentLink: &protocol.DocumentLinkClientCapabilities{
				DynamicRegistration: &no,
				TooltipSupport:      &yes,
			},
			ColorProvider: &protocol.DocumentColorClientCapabilities{
				DynamicRegistration: &no,
			},
			InlayHint: &protocol.InlayHintClientCapabilities{
				DynamicRegistration: &no,
			},
		},
		Window: &protocol.WindowClientCapabilities{
			WorkDoneProgress: &yes,
			ShowDocument: &protocol.ShowDocumentClientCapabilities{
				Support: true,
			},
		},
		General: &protocol.GeneralClientCapabilities{
			PositionEncodings: []protocol.PositionEncodingKind{
				protocol.PositionEncodingKindUTF8,
				protocol.PositionEncodingKindUTF32,
				protocol.PositionEncodingKindUTF16,
			},
		},
	}
}

// OffsetEncoding resolves the preferred position encoding
func OffsetEncoding(
	capabilities protocol.ServerCapabilities,
) protocol.PositionEncodingKind {
	switch capabilities.PositionEncoding {
	case protocol.PositionEncodingKindUTF8,
		protocol.PositionEncodingKindUTF16,
		protocol.PositionEncodingKindUTF32:
		return capabilities.PositionEncoding
	default:
		return protocol.PositionEncodingKindUTF16
	}
}

func signatureInfoCapabilities() *protocol.ClientSignatureInformationOptions {
	yes := true
	return &protocol.ClientSignatureInformationOptions{
		DocumentationFormat: []protocol.MarkupKind{
			protocol.MarkupKindMarkdown,
			protocol.MarkupKindPlainText,
		},
		ParameterInformation:   sigParamCaps(),
		ActiveParameterSupport: &yes,
	}
}

func sigParamCaps() *protocol.ClientSignatureParameterInformationOptions {
	yes := true
	return &protocol.ClientSignatureParameterInformationOptions{
		LabelOffsetSupport: &yes,
	}
}

func clientName(name string) string {
	if name == "" {
		return DefaultClientName
	}
	return name
}

func dynamicRegistration(
	enabled bool,
) *protocol.DidChangeConfigurationClientCapabilities {
	return &protocol.DidChangeConfigurationClientCapabilities{
		DynamicRegistration: &enabled,
	}
}

func setInitializeWorkspace(params *protocol.InitializeParams, root string) {
	if root == "" {
		return
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return
	}
	u := uri.File(abs)
	raw := fmt.Appendf(nil,
		`{"workspaceFolders":[{"uri":%q,"name":%q}]}`,
		string(u), filepath.Base(abs),
	)
	var folders protocol.WorkspaceFoldersInitializeParams
	if err := protocol.Unmarshal(raw, &folders); err != nil {
		return
	}
	params.WorkspaceFoldersInitializeParams = folders
}
