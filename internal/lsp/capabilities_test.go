package lsp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/lsp"
)

type capabilityCase struct {
	name    string
	feature lsp.Feature
	caps    protocol.ServerCapabilities
	want    bool
}

func TestCapabilities(t *testing.T) {
	for _, tc := range capabilityCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := lsp.SupportsFeature(tc.caps, tc.feature)

			assert.Equal(t, tc.want, got)
		})
	}
}

func capabilityCases() []capabilityCase {
	return []capabilityCase{
		{
			name:    "format true",
			feature: lsp.FeatureFormat,
			caps: protocol.ServerCapabilities{
				DocumentFormattingProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "format false",
			feature: lsp.FeatureFormat,
			caps: protocol.ServerCapabilities{
				DocumentFormattingProvider: protocol.Boolean(false),
			},
		},
		{
			name:    "declaration options",
			feature: lsp.FeatureGotoDeclaration,
			caps: protocol.ServerCapabilities{
				DeclarationProvider: &protocol.DeclarationOptions{},
			},
			want: true,
		},
		{
			name:    "range format",
			feature: lsp.FeatureRangeFormat,
			caps: protocol.ServerCapabilities{
				DocumentRangeFormattingProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "definition",
			feature: lsp.FeatureGotoDefinition,
			caps: protocol.ServerCapabilities{
				DefinitionProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "type definition",
			feature: lsp.FeatureGotoTypeDefinition,
			caps: protocol.ServerCapabilities{
				TypeDefinitionProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "references",
			feature: lsp.FeatureGotoReference,
			caps: protocol.ServerCapabilities{
				ReferencesProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "implementation",
			feature: lsp.FeatureGotoImplementation,
			caps: protocol.ServerCapabilities{
				ImplementationProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "signature help",
			feature: lsp.FeatureSignatureHelp,
			caps: protocol.ServerCapabilities{
				SignatureHelpProvider: &protocol.SignatureHelpOptions{},
			},
			want: true,
		},
		{
			name:    "completion provider",
			feature: lsp.FeatureCompletion,
			caps: protocol.ServerCapabilities{
				CompletionProvider: &protocol.CompletionOptions{},
			},
			want: true,
		},
		{
			name:    "hover provider",
			feature: lsp.FeatureHover,
			caps: protocol.ServerCapabilities{
				HoverProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "document highlight",
			feature: lsp.FeatureDocumentHighlight,
			caps: protocol.ServerCapabilities{
				DocumentHighlightProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "code action",
			feature: lsp.FeatureCodeAction,
			caps: protocol.ServerCapabilities{
				CodeActionProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "document links",
			feature: lsp.FeatureDocumentLinks,
			caps: protocol.ServerCapabilities{
				DocumentLinkProvider: &protocol.DocumentLinkOptions{},
			},
			want: true,
		},
		{
			name:    "workspace command",
			feature: lsp.FeatureWorkspaceCommand,
			caps: protocol.ServerCapabilities{
				ExecuteCommandProvider: protocol.ExecuteCommandOptions{
					Commands: []string{"organize"},
				},
			},
			want: true,
		},
		{
			name:    "empty workspace command",
			feature: lsp.FeatureWorkspaceCommand,
			caps:    protocol.ServerCapabilities{},
		},
		{
			name:    "document symbols",
			feature: lsp.FeatureDocumentSymbols,
			caps: protocol.ServerCapabilities{
				DocumentSymbolProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "workspace symbols",
			feature: lsp.FeatureWorkspaceSymbols,
			caps: protocol.ServerCapabilities{
				WorkspaceSymbolProvider: protocol.Boolean(true),
			},
			want: true,
		},
		{
			name:    "push diagnostics",
			feature: lsp.FeatureDiagnostics,
			caps:    protocol.ServerCapabilities{},
			want:    true,
		},
		{
			name:    "pull diagnostics",
			feature: lsp.FeaturePullDiagnostics,
			caps: protocol.ServerCapabilities{
				DiagnosticProvider: &protocol.DiagnosticOptions{},
			},
			want: true,
		},
		{
			name:    "rename options",
			feature: lsp.FeatureRename,
			caps: protocol.ServerCapabilities{
				RenameProvider: &protocol.RenameOptions{},
			},
			want: true,
		},
		{
			name:    "inlay hints",
			feature: lsp.FeatureInlayHints,
			caps: protocol.ServerCapabilities{
				InlayHintProvider: &protocol.InlayHintOptions{},
			},
			want: true,
		},
		{
			name:    "document colors",
			feature: lsp.FeatureDocumentColors,
			caps: protocol.ServerCapabilities{
				ColorProvider: &protocol.DocumentColorOptions{},
			},
			want: true,
		},
		{
			name:    "call hierarchy",
			feature: lsp.FeatureCallHierarchy,
			caps: protocol.ServerCapabilities{
				CallHierarchyProvider: &protocol.CallHierarchyOptions{},
			},
			want: true,
		},
		{
			name:    "nil pointer disabled",
			feature: lsp.FeatureCodeAction,
			caps: protocol.ServerCapabilities{
				CodeActionProvider: (*protocol.CodeActionOptions)(nil),
			},
		},
		{
			name:    "unknown feature",
			feature: lsp.Feature(999),
			caps:    protocol.ServerCapabilities{},
		},
	}
}
