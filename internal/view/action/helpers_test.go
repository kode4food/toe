package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

type stubLSP struct {
	resolveLink func(
		*view.Document, view.DocumentLink,
	) (view.DocumentLink, error)
}

var _ view.LanguageServerController = (*stubLSP)(nil)

func (s *stubLSP) RestartLanguageServers(
	*view.Document, []string,
) ([]string, error) {
	return nil, nil
}

func (s *stubLSP) StopLanguageServers(
	*view.Document, []string,
) ([]string, error) {
	return nil, nil
}

func (s *stubLSP) ExecuteWorkspaceCommand(
	*view.Document, string, []string,
) error {
	return nil
}

func (s *stubLSP) WorkspaceCommands(*view.Document) []string {
	return nil
}

func (s *stubLSP) Completions(
	*view.Document, view.Id,
) (view.CompletionResult, error) {
	return view.CompletionResult{}, nil
}

func (s *stubLSP) TriggerCompletions(
	*view.Document, view.Id,
) (view.CompletionResult, error) {
	return view.CompletionResult{}, nil
}

func (s *stubLSP) ResolveCompletion(
	_ *view.Document, _ view.Id, item view.CompletionItem,
) (view.CompletionItem, error) {
	return item, nil
}

func (s *stubLSP) ApplyCompletion(
	*view.Document, view.Id, view.CompletionItem,
) error {
	return nil
}

func (s *stubLSP) Hover(*view.Document, view.Id) (string, error) {
	return "", nil
}

func (s *stubLSP) SignatureHelp(
	*view.Document, view.Id,
) (view.SignatureHelp, error) {
	return view.SignatureHelp{}, nil
}

func (s *stubLSP) TriggerSignatureHelp(
	*view.Document, view.Id,
) (view.SignatureHelp, error) {
	return view.SignatureHelp{}, nil
}

func (s *stubLSP) GotoDeclaration(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubLSP) GotoDefinition(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubLSP) GotoTypeDefinition(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubLSP) GotoImplementation(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubLSP) GotoReference(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubLSP) RenameSymbolPrefill(
	*view.Document, view.Id,
) (string, error) {
	return "", nil
}

func (s *stubLSP) RenameSymbol(*view.Document, view.Id, string) error {
	return nil
}

func (s *stubLSP) CodeActions(
	*view.Document, view.Id,
) ([]view.CodeAction, error) {
	return nil, nil
}

func (s *stubLSP) ApplyCodeAction(
	*view.Document, view.Id, view.CodeAction,
) error {
	return nil
}

func (s *stubLSP) DocumentHighlights(
	*view.Document, view.Id,
) ([]view.DocumentHighlight, error) {
	return nil, nil
}

func (s *stubLSP) DocumentLinks(
	*view.Document,
) ([]view.DocumentLink, error) {
	return nil, nil
}

func (s *stubLSP) ResolveDocumentLink(
	doc *view.Document, link view.DocumentLink,
) (view.DocumentLink, error) {
	if s.resolveLink != nil {
		return s.resolveLink(doc, link)
	}
	return link, nil
}

func (s *stubLSP) FormatDocument(*view.Document, view.Id) error {
	return nil
}

func (s *stubLSP) FormatSelection(*view.Document, view.Id) error {
	return nil
}

func (s *stubLSP) DocumentSymbols(*view.Document) ([]view.Symbol, error) {
	return nil, nil
}

func (s *stubLSP) WorkspaceSymbols(
	*view.Document, string,
) ([]view.Symbol, error) {
	return nil, nil
}

func editorWithNoView(t *testing.T) *view.Editor {
	t.Helper()
	e := view.NewEditor("/tmp")
	v, ok := e.FocusedView()
	assert.True(t, ok)
	e.CloseView(v.ID())
	return e
}

func viewCount(t *testing.T, e *view.Editor) int {
	t.Helper()
	return len(e.AllViews())
}
