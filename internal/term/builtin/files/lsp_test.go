package files_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/view"
)

type fakeLanguageServerController struct {
	restarted []string
	stopped   []string
	executed  string
	args      []string
	location  string
	applied   string
	renamed   string
	symbols   []view.Symbol
	err       error
}

func TestLSPCommands(t *testing.T) {
	t.Run("restart reports server names", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "lsp-restart", "gopls")

		assert.Equal(t, []string{"gopls"}, ctl.restarted)
		assert.Equal(t, "language servers restarted: gopls", res.Message)
	})

	t.Run("stop reports server names", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "lsp-stop", "gopls")

		assert.Equal(t, []string{"gopls"}, ctl.stopped)
		assert.Equal(t, "language servers stopped: gopls", res.Message)
	})

	t.Run("missing controller reports no lsp", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")

		res := test.RunCmdArgs(t, km, e, "lsp-restart", "")

		assert.Equal(t, "error: LSP not defined for document", res.Message)
	})

	t.Run("workspace command reports error", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{
			err: errors.New("workspace command unavailable: test"),
		}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "lsp-workspace-command", "test {}")

		assert.Equal(t,
			"error: workspace command unavailable: test", res.Message,
		)
	})

	t.Run("workspace command executes", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "lsp-workspace-command", "test {}")

		assert.Equal(t, "test", ctl.executed)
		assert.Equal(t, []string{"{}"}, ctl.args)
		assert.Equal(t, "executed workspace command: test", res.Message)
	})

	t.Run("workspace command opens picker", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "lsp-workspace-command", "")

		assert.Empty(t, res.Message)
	})

	t.Run("goto definition is registered", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "goto_definition", "")

		assert.Empty(t, res.Message)
		assert.Equal(t, "definition", ctl.location)
	})

	t.Run("goto reference is registered", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "goto_reference", "")

		assert.Empty(t, res.Message)
		assert.Equal(t, "reference", ctl.location)
	})

	t.Run("rename symbol is registered", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "rename_symbol", "")

		assert.Empty(t, res.Message)
	})

	t.Run("code action is registered", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{
			symbols: nil,
		}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "code_action", "")

		assert.Empty(t, res.Message)
	})

	t.Run("symbol picker is registered", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{
			symbols: []view.Symbol{
				{Name: "main", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "symbol_picker", "")

		assert.Empty(t, res.Message)
	})

	t.Run("workspace symbol picker is registered", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := test.RunCmdArgs(t, km, e, "workspace_symbol_picker", "")

		assert.Empty(t, res.Message)
	})
}

func TestLSPCommandErrors(t *testing.T) {
	t.Run("generic error", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{
			err: errors.New("restart failed"),
		}
		e.SetLanguageServerController(ctl)
		res := test.RunCmdArgs(t, km, e, "lsp-restart", "gopls")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("ErrNoLanguageServer", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{err: lsp.ErrNoLanguageServer}
		e.SetLanguageServerController(ctl)
		res := test.RunCmdArgs(t, km, e, "lsp-restart", "gopls")
		assert.Contains(t, res.Message, "LSP not defined")
	})

	t.Run("ErrUnknownLanguageServer", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{err: lsp.ErrUnknownLanguageServer}
		e.SetLanguageServerController(ctl)
		res := test.RunCmdArgs(t, km, e, "lsp-restart", "gopls")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("ErrWorkspaceCommand", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{err: lsp.ErrWorkspaceCommand}
		e.SetLanguageServerController(ctl)
		res := test.RunCmdArgs(t, km, e, "lsp-restart", "gopls")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("stop error", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{
			err: errors.New("stop failed"),
		}
		e.SetLanguageServerController(ctl)
		res := test.RunCmdArgs(t, km, e, "lsp-stop", "gopls")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("stop no document", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		e.CloseView(v.ID())
		res := test.RunCmdArgs(t, km, e, "lsp-stop", "gopls")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("workspace command no document", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		e.CloseView(v.ID())
		res := test.RunCmdArgs(t, km, e, "lsp-workspace-command", "test {}")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("restart empty names", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)
		res := test.RunCmd(t, km, e, "lsp-restart")
		assert.Contains(t, res.Message, "no language servers")
	})

	t.Run("stop empty names", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)
		res := test.RunCmd(t, km, e, "lsp-stop")
		assert.Contains(t, res.Message, "no language servers")
	})

	t.Run("one arg passes nil args", func(t *testing.T) {
		e, km, _ := test.EnvWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)
		res := test.RunCmdArgs(t, km, e, "lsp-workspace-command", "test")
		assert.Contains(t, res.Message, "executed workspace command: test")
		assert.Nil(t, ctl.args)
	})
}

func (c *fakeLanguageServerController) RestartLanguageServers(
	_ *view.Document, names []string,
) ([]string, error) {
	c.restarted = slices.Clone(names)
	return names, c.err
}

func (c *fakeLanguageServerController) StopLanguageServers(
	_ *view.Document, names []string,
) ([]string, error) {
	c.stopped = slices.Clone(names)
	return names, c.err
}

func (c *fakeLanguageServerController) ExecuteWorkspaceCommand(
	_ *view.Document, name string, args []string,
) error {
	c.executed = name
	c.args = slices.Clone(args)
	return c.err
}

func (c *fakeLanguageServerController) WorkspaceCommands(
	*view.Document,
) []string {
	return []string{"test"}
}

func (c *fakeLanguageServerController) Completions(
	*view.Document, view.Id,
) (view.CompletionResult, error) {
	return view.CompletionResult{}, c.err
}

func (c *fakeLanguageServerController) TriggerCompletions(
	*view.Document, view.Id,
) (view.CompletionResult, error) {
	return view.CompletionResult{}, c.err
}

func (c *fakeLanguageServerController) ResolveCompletion(
	_ *view.Document, _ view.Id, item view.CompletionItem,
) (view.CompletionItem, error) {
	return item, c.err
}

func (c *fakeLanguageServerController) ApplyCompletion(
	*view.Document, view.Id, view.CompletionItem,
) error {
	return c.err
}

func (c *fakeLanguageServerController) Hover(
	*view.Document, view.Id,
) (string, error) {
	return "", c.err
}

func (c *fakeLanguageServerController) SignatureHelp(
	*view.Document, view.Id,
) (view.SignatureHelp, error) {
	return view.SignatureHelp{}, c.err
}

func (c *fakeLanguageServerController) TriggerSignatureHelp(
	*view.Document, view.Id,
) (view.SignatureHelp, error) {
	return view.SignatureHelp{}, c.err
}

func (c *fakeLanguageServerController) GotoDeclaration(
	*view.Document, view.Id,
) ([]view.Location, error) {
	c.location = "declaration"
	return nil, c.err
}

func (c *fakeLanguageServerController) GotoDefinition(
	*view.Document, view.Id,
) ([]view.Location, error) {
	c.location = "definition"
	return nil, c.err
}

func (c *fakeLanguageServerController) GotoTypeDefinition(
	*view.Document, view.Id,
) ([]view.Location, error) {
	c.location = "type-definition"
	return nil, c.err
}

func (c *fakeLanguageServerController) GotoImplementation(
	*view.Document, view.Id,
) ([]view.Location, error) {
	c.location = "implementation"
	return nil, c.err
}

func (c *fakeLanguageServerController) GotoReference(
	*view.Document, view.Id,
) ([]view.Location, error) {
	c.location = "reference"
	return nil, c.err
}

func (c *fakeLanguageServerController) RenameSymbolPrefill(
	*view.Document, view.Id,
) (string, error) {
	return "", c.err
}

func (c *fakeLanguageServerController) RenameSymbol(
	_ *view.Document, _ view.Id, name string,
) error {
	c.renamed = name
	return c.err
}

func (c *fakeLanguageServerController) CodeActions(
	*view.Document, view.Id,
) ([]view.CodeAction, error) {
	return []view.CodeAction{{ID: "test:0", Title: "fix"}}, c.err
}

func (c *fakeLanguageServerController) ApplyCodeAction(
	_ *view.Document, _ view.Id, action view.CodeAction,
) error {
	c.applied = action.ID
	return c.err
}

func (c *fakeLanguageServerController) DocumentHighlights(
	*view.Document, view.Id,
) ([]view.DocumentHighlight, error) {
	return nil, c.err
}

func (c *fakeLanguageServerController) DocumentLinks(
	*view.Document,
) ([]view.DocumentLink, error) {
	return nil, c.err
}

func (c *fakeLanguageServerController) ResolveDocumentLink(
	_ *view.Document, link view.DocumentLink,
) (view.DocumentLink, error) {
	return link, c.err
}

func (c *fakeLanguageServerController) FormatDocument(
	*view.Document, view.Id,
) error {
	return c.err
}

func (c *fakeLanguageServerController) FormatSelection(
	*view.Document, view.Id,
) error {
	return c.err
}

func (c *fakeLanguageServerController) DocumentSymbols(
	*view.Document,
) ([]view.Symbol, error) {
	return c.symbols, c.err
}

func (c *fakeLanguageServerController) WorkspaceSymbols(
	*view.Document, string,
) ([]view.Symbol, error) {
	return c.symbols, c.err
}

func (c *fakeLanguageServerController) Busy() bool {
	return false
}
