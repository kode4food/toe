package lsp_test

import (
	"testing"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
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
	}
}
