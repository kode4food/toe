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

// DefaultClientCapabilities returns the editor's declared LSP client capabilities
func DefaultClientCapabilities() protocol.ClientCapabilities {
	yes := true
	no := false
	return protocol.ClientCapabilities{
		Workspace: &protocol.WorkspaceClientCapabilities{
			Configuration:          &yes,
			WorkspaceFolders:       &yes,
			ApplyEdit:              &yes,
			DidChangeConfiguration: dynamicRegistration(false),
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
		},
		TextDocument: &protocol.TextDocumentClientCapabilities{
			Synchronization: &protocol.TextDocumentSyncClientCapabilities{
				DidSave: &yes,
			},
			Completion: &protocol.CompletionClientCapabilities{
				ContextSupport: &yes,
				CompletionItem: &protocol.ClientCompletionItemOptions{
					SnippetSupport: &no,
					DocumentationFormat: []protocol.MarkupKind{
						protocol.MarkupKindMarkdown,
						protocol.MarkupKindPlainText,
					},
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

// OffsetEncoding resolves the preferred position encoding from server capabilities
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
