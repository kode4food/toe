package defaults_test

import (
	"errors"
	"testing"

	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
)

type fakeLanguageServerController struct {
	restarted []string
	stopped   []string
	executed  string
	args      []string
	location  string
	symbols   []view.Symbol
	err       error
}

func TestLSPCommands(t *testing.T) {
	t.Run("restart reports server names", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := runCmdArgs(t, km, e, "lsp-restart", "gopls")

		assert.Equal(t, []string{"gopls"}, ctl.restarted)
		assert.Equal(t, "language servers restarted: gopls", res.Message)
	})

	t.Run("stop reports server names", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := runCmdArgs(t, km, e, "lsp-stop", "gopls")

		assert.Equal(t, []string{"gopls"}, ctl.stopped)
		assert.Equal(t, "language servers stopped: gopls", res.Message)
	})

	t.Run("missing controller reports no lsp", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")

		res := runCmdArgs(t, km, e, "lsp-restart", "")

		assert.Equal(t, "error: LSP not defined for document", res.Message)
	})

	t.Run("workspace command reports error", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")
		ctl := &fakeLanguageServerController{
			err: errors.New("workspace command unavailable: test"),
		}
		e.SetLanguageServerController(ctl)

		res := runCmdArgs(t, km, e, "lsp-workspace-command", "test {}")

		assert.Equal(t,
			"error: workspace command unavailable: test", res.Message,
		)
	})

	t.Run("workspace command executes", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := runCmdArgs(t, km, e, "lsp-workspace-command", "test {}")

		assert.Equal(t, "test", ctl.executed)
		assert.Equal(t, []string{"{}"}, ctl.args)
		assert.Equal(t, "executed workspace command: test", res.Message)
	})

	t.Run("workspace command opens picker", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := runCmdArgs(t, km, e, "lsp-workspace-command", "")

		assert.Empty(t, res.Message)
	})

	t.Run("goto definition is registered", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := runCmdArgs(t, km, e, "goto_definition", "")

		assert.Empty(t, res.Message)
		assert.Equal(t, "definition", ctl.location)
	})

	t.Run("goto reference is registered", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")
		ctl := &fakeLanguageServerController{}
		e.SetLanguageServerController(ctl)

		res := runCmdArgs(t, km, e, "goto_reference", "")

		assert.Empty(t, res.Message)
		assert.Equal(t, "reference", ctl.location)
	})

	t.Run("symbol picker is registered", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")
		ctl := &fakeLanguageServerController{
			symbols: []view.Symbol{
				{Name: "main", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)

		res := runCmdArgs(t, km, e, "symbol_picker", "")

		assert.Empty(t, res.Message)
	})
}

func (c *fakeLanguageServerController) RestartLanguageServers(
	_ *view.Document, names []string,
) ([]string, error) {
	c.restarted = append([]string(nil), names...)
	return names, c.err
}

func (c *fakeLanguageServerController) StopLanguageServers(
	_ *view.Document, names []string,
) ([]string, error) {
	c.stopped = append([]string(nil), names...)
	return names, c.err
}

func (c *fakeLanguageServerController) ExecuteWorkspaceCommand(
	_ *view.Document, name string, args []string,
) error {
	c.executed = name
	c.args = append([]string(nil), args...)
	return c.err
}

func (c *fakeLanguageServerController) WorkspaceCommands(
	*view.Document,
) []string {
	return []string{"test"}
}

func (c *fakeLanguageServerController) Completions(
	*view.Document, view.Id,
) ([]view.CompletionItem, error) {
	return nil, c.err
}

func (c *fakeLanguageServerController) TriggerCompletions(
	*view.Document, view.Id,
) ([]view.CompletionItem, error) {
	return nil, c.err
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

func (c *fakeLanguageServerController) DocumentSymbols(
	*view.Document,
) ([]view.Symbol, error) {
	return c.symbols, c.err
}
