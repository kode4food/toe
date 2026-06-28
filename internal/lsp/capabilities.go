package lsp

import (
	"reflect"

	"go.lsp.dev/protocol"
)

// Feature names an LSP capability group the editor can query before issuing
// requests
type Feature int

// Feature constants name the LSP capability groups the editor supports
const (
	FeatureFormat Feature = iota
	FeatureRangeFormat
	FeatureGotoDeclaration
	FeatureGotoDefinition
	FeatureGotoTypeDefinition
	FeatureGotoReference
	FeatureGotoImplementation
	FeatureSignatureHelp
	FeatureHover
	FeatureDocumentHighlight
	FeatureCompletion
	FeatureCodeAction
	FeatureDocumentLinks
	FeatureWorkspaceCommand
	FeatureDocumentSymbols
	FeatureWorkspaceSymbols
	FeatureDiagnostics
	FeaturePullDiagnostics
	FeatureRename
	FeatureInlayHints
	FeatureDocumentColors
	FeatureCallHierarchy
)

// SupportsFeature reports whether the capability set supports the named feature
func SupportsFeature(
	capabilities protocol.ServerCapabilities, feature Feature,
) bool {
	switch feature {
	case FeatureFormat:
		return capabilityEnabled(capabilities.DocumentFormattingProvider)
	case FeatureRangeFormat:
		return capabilityEnabled(capabilities.DocumentRangeFormattingProvider)
	case FeatureGotoDeclaration:
		return capabilityEnabled(capabilities.DeclarationProvider)
	case FeatureGotoDefinition:
		return capabilityEnabled(capabilities.DefinitionProvider)
	case FeatureGotoTypeDefinition:
		return capabilityEnabled(capabilities.TypeDefinitionProvider)
	case FeatureGotoReference:
		return capabilityEnabled(capabilities.ReferencesProvider)
	case FeatureGotoImplementation:
		return capabilityEnabled(capabilities.ImplementationProvider)
	case FeatureSignatureHelp:
		return capabilities.SignatureHelpProvider != nil
	case FeatureHover:
		return capabilityEnabled(capabilities.HoverProvider)
	case FeatureDocumentHighlight:
		return capabilityEnabled(capabilities.DocumentHighlightProvider)
	case FeatureCompletion:
		return capabilities.CompletionProvider != nil
	case FeatureCodeAction:
		return capabilityEnabled(capabilities.CodeActionProvider)
	case FeatureDocumentLinks:
		return capabilities.DocumentLinkProvider != nil
	case FeatureWorkspaceCommand:
		return len(capabilities.ExecuteCommandProvider.Commands) > 0
	case FeatureDocumentSymbols:
		return capabilityEnabled(capabilities.DocumentSymbolProvider)
	case FeatureWorkspaceSymbols:
		return capabilityEnabled(capabilities.WorkspaceSymbolProvider)
	case FeatureDiagnostics:
		return true
	case FeaturePullDiagnostics:
		return capabilities.DiagnosticProvider != nil
	case FeatureRename:
		return capabilityEnabled(capabilities.RenameProvider)
	case FeatureInlayHints:
		return capabilityEnabled(capabilities.InlayHintProvider)
	case FeatureDocumentColors:
		return capabilityEnabled(capabilities.ColorProvider)
	case FeatureCallHierarchy:
		return capabilityEnabled(capabilities.CallHierarchyProvider)
	default:
		return false
	}
}

func capabilityEnabled(value any) bool {
	if value == nil {
		return false
	}
	if b, ok := value.(protocol.Boolean); ok {
		return bool(b)
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return false
	}
	return true
}
