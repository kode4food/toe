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
	locations     []view.Location
	symbols       []view.Symbol
	commands      []string
	actions       []view.CodeAction
	highlights    []view.DocumentHighlight
	signatureHelp view.SignatureHelp
	applied       string
	renamed       string
}

type gotoActionCase struct {
	desc   string
	action func(ui.Model) command.KeyAction
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

	t.Run("opens workspace symbol picker", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "source.go")
		target := filepath.Join(dir, "target.go")
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o600))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(source)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{
			symbols: []view.Symbol{
				{
					Name: "WorkspaceMain", Kind: "function",
					Container: "workspace",
					Location: view.Location{
						Path: target, From: 3, To: 6,
					},
				},
			},
		})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "workspace_symbol_picker",
			m.WorkspaceSymbolPickerAction(),
			[]command.KeyEvent{char('w')},
		)
		m = resize(m, 80, 24)

		m = sendKey(m, 'w')
		_ = m.View()
		next, cmd := m.Update(tea.KeyPressMsg{
			Code: 'm',
			Text: "m",
		})
		m = next.(ui.Model)
		if cmd == nil {
			t.Fatal("expected dynamic picker command")
		}
		msg := runTestCmd(t, cmd)
		next, _ = m.Update(msg)
		m = next.(ui.Model)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "function")
		assert.Contains(t, out, "WorkspaceMain")
		assert.Contains(t, out, "workspace")
		assert.Contains(t, out, "target.go")

		_ = sendSpecial(m, tea.KeyEnter)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, target, doc.Path())
		v, ok := e.FocusedView()
		assert.True(t, ok)
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, core.NewRange(6, 3), sel.Primary())
	})
}

func TestRenameSymbolAction(t *testing.T) {
	t.Run("opens prompt and renames", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		ctl := &locationController{}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "rename_symbol", m.RenameSymbolAction(),
			[]command.KeyEvent{char('r')},
		)
		m = resize(m, 80, 24)

		m = sendKey(m, 'r')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "rename-to: old")

		m = sendSpecial(m, tea.KeyBackspace)
		m = sendSpecial(m, tea.KeyBackspace)
		m = sendSpecial(m, tea.KeyBackspace)
		m = sendKey(m, 'n')
		m = sendKey(m, 'e')
		m = sendKey(m, 'w')
		_ = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, "new", ctl.renamed)
	})

	t.Run("no language server sets status", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		m := ui.New(e, command.NewKeymaps())

		cont := m.RenameSymbolAction()(e)

		assert.Nil(t, cont)
		doc, _ := e.FocusedDocument()
		v, _ := e.FocusedView()
		_ = doc.SelectionFor(v.ID())
	})
}

func TestCodeActionPickerAction(t *testing.T) {
	t.Run("opens and applies action", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		ctl := &locationController{
			actions: []view.CodeAction{
				{
					ID: "session:0", Title: "Fix old",
					Kind: "quickfix", Server: "session",
				},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "code_action", m.CodeActionPickerAction(),
			[]command.KeyEvent{char('a')},
		)
		m = resize(m, 80, 24)

		m = sendKey(m, 'a')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "quickfix")
		assert.Contains(t, out, "Fix old")

		_ = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, "session:0", ctl.applied)
	})

	t.Run("no language server sets status", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		m := ui.New(e, command.NewKeymaps())

		cont := m.CodeActionPickerAction()(e)

		assert.Nil(t, cont)
	})

	t.Run("no code actions sets status", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{})
		m := ui.New(e, command.NewKeymaps())

		cont := m.CodeActionPickerAction()(e)

		assert.Nil(t, cont)
		assert.Contains(t, e.TakeStatusMsg(), "No code actions")
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
	return c.commands
}

func (c *locationController) Completions(
	*view.Document, view.Id,
) (view.CompletionResult, error) {
	return view.CompletionResult{}, nil
}

func (c *locationController) TriggerCompletions(
	*view.Document, view.Id,
) (view.CompletionResult, error) {
	return view.CompletionResult{}, nil
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
	return c.signatureHelp, nil
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

func (c *locationController) RenameSymbolPrefill(
	doc *view.Document, viewID view.Id,
) (string, error) {
	sel := doc.SelectionFor(viewID)
	r := core.TextObjectWord(
		doc.Text(), sel.Primary(), core.TextObjectInside, false,
	)
	return r.Fragment(doc.Text())
}

func (c *locationController) RenameSymbol(
	_ *view.Document, _ view.Id, name string,
) error {
	c.renamed = name
	return nil
}

func (c *locationController) CodeActions(
	*view.Document, view.Id,
) ([]view.CodeAction, error) {
	return c.actions, nil
}

func (c *locationController) ApplyCodeAction(
	_ *view.Document, _ view.Id, action view.CodeAction,
) error {
	c.applied = action.ID
	return nil
}

func (c *locationController) DocumentHighlights(
	*view.Document, view.Id,
) ([]view.DocumentHighlight, error) {
	return c.highlights, nil
}

func (c *locationController) DocumentLinks(
	*view.Document,
) ([]view.DocumentLink, error) {
	return nil, nil
}

func (c *locationController) ResolveDocumentLink(
	_ *view.Document, link view.DocumentLink,
) (view.DocumentLink, error) {
	return link, nil
}

func (c *locationController) FormatDocument(
	*view.Document, view.Id,
) error {
	return nil
}

func (c *locationController) FormatSelection(
	*view.Document, view.Id,
) error {
	return nil
}

func (c *locationController) DocumentSymbols(
	*view.Document,
) ([]view.Symbol, error) {
	return c.symbols, nil
}

func (c *locationController) WorkspaceSymbols(
	*view.Document, string,
) ([]view.Symbol, error) {
	return c.symbols, nil
}

var gotoActionCases = []gotoActionCase{
	{"GotoDeclarationAction", ui.Model.GotoDeclarationAction},
	{"GotoTypeDefinitionAction", ui.Model.GotoTypeDefinitionAction},
	{"GotoImplementationAction", ui.Model.GotoImplementationAction},
	{"GotoReferenceAction", ui.Model.GotoReferenceAction},
}

func TestGotoActions(t *testing.T) {
	t.Run("no language server sets status", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "source.go")
		assert.NoError(t, os.WriteFile(path, []byte("source\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		m := ui.New(e, command.NewKeymaps())

		cont := m.GotoDeclarationAction()(e)

		assert.Nil(t, cont)
		assert.Contains(t, e.TakeStatusMsg(), "No configured language server")
	})

	t.Run("not found sets status", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "source.go")
		assert.NoError(t, os.WriteFile(path, []byte("source\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{})
		m := ui.New(e, command.NewKeymaps())

		cont := m.GotoDeclarationAction()(e)

		assert.Nil(t, cont)
	})

	for _, tc := range gotoActionCases {
		t.Run(tc.desc+" single target", func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source.go")
			target := filepath.Join(dir, "target.go")
			assert.NoError(t, os.WriteFile(
				source, []byte("source\n"), 0o600,
			))
			assert.NoError(t, os.WriteFile(
				target, []byte("target\n"), 0o600,
			))
			e := view.NewEditor(dir)
			_, err := e.OpenFile(source)
			assert.NoError(t, err)
			e.SetLanguageServerController(&locationController{
				locations: []view.Location{
					{Path: target, From: 3, To: 3},
				},
			})
			m := ui.New(e, command.NewKeymaps())

			cont := tc.action(m)(e)

			assert.Nil(t, cont)
			doc, ok := e.FocusedDocument()
			assert.True(t, ok)
			assert.Equal(t, target, doc.Path())
		})
	}
}

func TestHoverAction(t *testing.T) {
	t.Run("no focused document returns nil", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps())

		cont := m.HoverAction()(e)

		assert.Nil(t, cont)
	})

	t.Run("no server sets status", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "source.go")
		assert.NoError(t, os.WriteFile(path, []byte("source\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		m := ui.New(e, command.NewKeymaps())

		cont := m.HoverAction()(e)

		assert.Nil(t, cont)
	})

	t.Run("empty hover sets status", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "source.go")
		assert.NoError(t, os.WriteFile(path, []byte("source\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{})
		m := ui.New(e, command.NewKeymaps())

		cont := m.HoverAction()(e)

		assert.Nil(t, cont)
	})
}

func TestSignatureHelpAction(t *testing.T) {
	t.Run("no focused document returns nil", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps())

		assert.Nil(t, m.SignatureHelpAction()(e))
	})

	t.Run("no language server sets status", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("foo(\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		m := ui.New(e, command.NewKeymaps())

		assert.Nil(t, m.SignatureHelpAction()(e))
		assert.Contains(t, e.TakeStatusMsg(), "signature-help")
	})

	t.Run("cursor not in call returns nil", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{})
		m := ui.New(e, command.NewKeymaps())

		assert.Nil(t, m.SignatureHelpAction()(e))
	})

	t.Run("empty signatures returns nil", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("foo(\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		doc.SetSelectionFor(v.ID(), core.PointSelection(4))
		e.SetLanguageServerController(&locationController{})
		m := ui.New(e, command.NewKeymaps())

		assert.Nil(t, m.SignatureHelpAction()(e))
	})

	t.Run("opens signature help layer", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("foo(\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		doc.SetSelectionFor(v.ID(), core.PointSelection(4))
		ctl := &locationController{
			signatureHelp: view.SignatureHelp{
				Signatures: []view.SignatureInformation{{Label: "foo(x int)"}},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "sig_help", m.SignatureHelpAction(),
			[]command.KeyEvent{char('s')},
		)
		m = resize(m, 80, 24)
		m = sendKey(m, 's')
		assert.NotEmpty(t, stripANSI(m.View().Content))
	})
}

func TestCompletionAction(t *testing.T) {
	t.Run("no focused document returns nil", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps())

		assert.Nil(t, m.CompletionAction()(e))
	})

	t.Run("no language server returns nil", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("foo\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		m := ui.New(e, command.NewKeymaps())

		assert.Nil(t, m.CompletionAction()(e))
	})

	t.Run("no completions returns nil", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("foo\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{})
		m := ui.New(e, command.NewKeymaps())

		assert.Nil(t, m.CompletionAction()(e))
	})
}

func TestLSPWorkspaceCommandPicker(t *testing.T) {
	t.Run("creates picker", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{
			commands: []string{"fmt.run", "lint.run"},
		})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "lsp_cmds",
			m.PickerAction(ui.LSPWorkspaceCommandPicker),
			[]command.KeyEvent{char('w')},
		)
		m = resize(m, 80, 24)

		m = openPickerAndFeed(m, 'w')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "fmt.run")
		assert.Contains(t, out, "lint.run")
	})

	t.Run("accepts and executes command", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		ctl := &locationController{
			commands: []string{"fmt.run"},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "lsp_cmds",
			m.PickerAction(ui.LSPWorkspaceCommandPicker),
			[]command.KeyEvent{char('w')},
		)
		m = resize(m, 80, 24)
		m = openPickerAndFeed(m, 'w')
		sendSpecial(m, tea.KeyEnter)
	})
}

func TestSelectReferencesAction(t *testing.T) {
	t.Run("no results sets status", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "source.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello world\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{})
		m := ui.New(e, command.NewKeymaps())

		cont := m.SelectReferencesAction()(e)

		assert.Nil(t, cont)
	})

	t.Run("with highlights selects ranges", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "source.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello world hello\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetLanguageServerController(&locationController{
			highlights: []view.DocumentHighlight{
				{From: 0, To: 5},
				{From: 12, To: 17},
			},
		})
		m := ui.New(e, command.NewKeymaps())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))

		cont := m.SelectReferencesAction()(e)

		assert.Nil(t, cont)
		sel := doc.SelectionFor(v.ID())
		assert.Len(t, sel.Ranges(), 2)
	})
}
