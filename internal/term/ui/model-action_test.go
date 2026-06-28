package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
)

type locationController struct {
	locations []view.Location
	symbols   []view.Symbol
}

func TestLocationAction(t *testing.T) {
	t.Run("jumps to single target", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "source.go")
		target := filepath.Join(dir, "target.go")
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o600))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(source)
		assert.NoError(t, err)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))
		e.SetLanguageServerController(&locationController{
			locations: []view.Location{
				{Path: target, From: 3, To: 3},
			},
		})
		m := ui.New(e, command.NewKeymaps())

		cont := m.GotoDefinitionAction()(e)

		assert.Nil(t, cont)
		doc, ok = e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, target, doc.Path())
		v, ok = e.FocusedView()
		assert.True(t, ok)
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 3, sel.Primary().Cursor(doc.Text()))
		assert.Equal(t, core.NewRange(3, 3), sel.Primary())
	})

	t.Run("opens picker for multiple targets", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "source.go")
		first := filepath.Join(dir, "first.go")
		second := filepath.Join(dir, "second.go")
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o600))
		assert.NoError(t, os.WriteFile(first, []byte("first\n"), 0o600))
		assert.NoError(t, os.WriteFile(second, []byte("second\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(source)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{
			locations: []view.Location{
				{Path: first, From: 2, To: 5},
				{Path: second, From: 3, To: 6},
			},
		})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "goto_definition", m.GotoDefinitionAction(),
			[]command.KeyEvent{char('d')},
		)
		m = resize(m, 80, 24)

		m = sendKey(m, 'd')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "first.go:1")
		assert.Contains(t, out, "second.go:1")

		_ = sendSpecial(m, tea.KeyEnter)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, first, doc.Path())
		v, ok := e.FocusedView()
		assert.True(t, ok)
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, sel.Primary().Cursor(doc.Text()))
		assert.Equal(t, core.NewRange(5, 2), sel.Primary())
	})
}

func TestSymbolPickerAction(t *testing.T) {
	t.Run("opens and accepts symbol", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(
			path, []byte("package main\nfunc main() {}\n"), 0o600,
		))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{
			symbols: []view.Symbol{
				{
					Name: "main", Kind: "function", Container: "package",
					Location: view.Location{Path: path, From: 18, To: 22},
				},
			},
		})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "symbol_picker", m.SymbolPickerAction(),
			[]command.KeyEvent{char('s')},
		)
		m = resize(m, 80, 24)

		m = sendKey(m, 's')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "function")
		assert.Contains(t, out, "main")
		assert.Contains(t, out, "package")

		_ = sendSpecial(m, tea.KeyEnter)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, core.NewRange(22, 18), sel.Primary())
	})
}

func (c *locationController) RestartLanguageServers(
	*view.Document, []string,
) ([]string, error) {
	return nil, nil
}

func (c *locationController) StopLanguageServers(
	*view.Document, []string,
) ([]string, error) {
	return nil, nil
}

func (c *locationController) ExecuteWorkspaceCommand(
	*view.Document, string, []string,
) error {
	return nil
}

func (c *locationController) WorkspaceCommands(*view.Document) []string {
	return nil
}

func (c *locationController) Completions(
	*view.Document, view.Id,
) ([]view.CompletionItem, error) {
	return nil, nil
}

func (c *locationController) TriggerCompletions(
	*view.Document, view.Id,
) ([]view.CompletionItem, error) {
	return nil, nil
}

func (c *locationController) ResolveCompletion(
	_ *view.Document, _ view.Id, item view.CompletionItem,
) (view.CompletionItem, error) {
	return item, nil
}

func (c *locationController) ApplyCompletion(
	*view.Document, view.Id, view.CompletionItem,
) error {
	return nil
}

func (c *locationController) Hover(*view.Document, view.Id) (string, error) {
	return "", nil
}

func (c *locationController) SignatureHelp(
	*view.Document, view.Id,
) (view.SignatureHelp, error) {
	return view.SignatureHelp{}, nil
}

func (c *locationController) TriggerSignatureHelp(
	*view.Document, view.Id,
) (view.SignatureHelp, error) {
	return view.SignatureHelp{}, nil
}

func (c *locationController) GotoDeclaration(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return c.locations, nil
}

func (c *locationController) GotoDefinition(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return c.locations, nil
}

func (c *locationController) GotoTypeDefinition(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return c.locations, nil
}

func (c *locationController) GotoImplementation(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return c.locations, nil
}

func (c *locationController) GotoReference(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return c.locations, nil
}

func (c *locationController) DocumentSymbols(
	*view.Document,
) ([]view.Symbol, error) {
	return c.symbols, nil
}
