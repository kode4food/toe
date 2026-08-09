package files

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

const (
	actGotoDeclaration     = "goto_declaration"
	actGotoDefinition      = "goto_definition"
	actGotoTypeDefinition  = "goto_type_definition"
	actGotoImplementation  = "goto_implementation"
	actGotoReference       = "goto_reference"
	actSelectReferences    = "select_references_to_symbol_under_cursor"
	actCodeAction          = "code_action"
	actHover               = "hover"
	actRenameSymbol        = "rename_symbol"
	actSignatureHelp       = "signature-help"
	actSymbolPicker        = "symbol_picker"
	actWorkspaceSymbol     = "workspace_symbol_picker"
	actLSPRestart          = "lsp-restart"
	actLSPStop             = "lsp-stop"
	actLSPWorkspaceCommand = "lsp-workspace-command"
)

const errorLSPUndefinedKey i18n.Key = "error.lspUndefined"

var (
	//go:embed i18n/lsp.*.json
	lspFS embed.FS

	errLSPUndefined = i18n.NewError(errorLSPUndefinedKey)
)

// LspModule returns the language-server navigation and action commands
func LspModule(model ui.Model) command.Module {
	g := kit.Prefixed(kit.Char('g'))
	return command.Module{
		Translations: i18n.LoadTranslations(lspFS),
		Commands: []command.Command{
			{
				Name:      actGotoDeclaration,
				DocString: "Goto declaration",
				Run:       kit.Runner(model.GotoDeclarationAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(g(kit.Char('D'))),
			},
			{
				Name:      actGotoDefinition,
				DocString: "Goto definition",
				Run:       kit.Runner(model.GotoDefinitionAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(g(kit.Char('d'))),
			},
			{
				Name:      actGotoTypeDefinition,
				DocString: "Goto type definition",
				Run:       kit.Runner(model.GotoTypeDefinitionAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(g(kit.Char('y'))),
			},
			{
				Name:      actGotoImplementation,
				DocString: "Goto implementation",
				Run:       kit.Runner(model.GotoImplementationAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(g(kit.Char('i'))),
			},
			{
				Name:      actGotoReference,
				DocString: "Goto references",
				Run:       kit.Runner(model.GotoReferenceAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(g(kit.Char('r'))),
			},
			{
				Name:      actSelectReferences,
				DocString: "Select symbol references",
				Run:       kit.Runner(model.SelectReferencesAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('h'),
			},
			{
				Name:      actCodeAction,
				DocString: "Perform code action",
				Run:       kit.Runner(model.CodeActionPickerAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('a'),
			},
			{
				Name:      actHover,
				DocString: "Show docs for item under cursor",
				Run:       kit.Runner(model.HoverAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('k'),
			},
			{
				Name:      actRenameSymbol,
				DocString: "Rename symbol",
				Run:       kit.Runner(model.RenameSymbolAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('r'),
			},
			{
				Name:      actSignatureHelp,
				DocString: "Show signature help",
				Run:       kit.Runner(model.SignatureHelpAction),
				Modes:     view.ModeInsert,
			},
			{
				Name:      actSymbolPicker,
				DocString: "Open symbol picker",
				Run:       kit.Runner(model.SymbolPickerAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('s'),
			},
			{
				Name:      actWorkspaceSymbol,
				DocString: "Open workspace symbol picker",
				Run: kit.Runner(
					model.WorkspaceSymbolPickerAction,
				),
				Modes: command.PaneModes,
				Keys:  kit.Leader('S'),
			},
			{
				Name:      actLSPRestart,
				DocString: "Restart language servers for the current document",
				Run:       runLSPRestart,
				Modes:     command.DocModes,
				Signature: command.Signature{},
			},
			{
				Name:      actLSPStop,
				DocString: "Stop language servers for the current document",
				Run:       runLSPStop,
				Modes:     command.DocModes,
				Signature: command.Signature{},
			},
			{
				Name:      actLSPWorkspaceCommand,
				DocString: "Execute a language server workspace command",
				Run:       runLSPWorkspaceCommand(model),
				Modes:     command.DocModes,
				Signature: command.Signature{
					RawAfter: 1,
				},
			},
		},
	}
}

func runLSPRestart(e *view.Editor, args *command.Args) command.Result {
	doc, ctl, ok := lspCommandContext(e)
	if !ok {
		return command.Result{Error: errLSPUndefined}
	}
	names, err := ctl.RestartLanguageServers(doc, positionals(args))
	if err != nil {
		return lspCommandError(err)
	}
	return command.Result{Message: lspNamesMessage("restarted", names)}
}

func runLSPStop(e *view.Editor, args *command.Args) command.Result {
	doc, ctl, ok := lspCommandContext(e)
	if !ok {
		return command.Result{Error: errLSPUndefined}
	}
	names, err := ctl.StopLanguageServers(doc, positionals(args))
	if err != nil {
		return lspCommandError(err)
	}
	return command.Result{Message: lspNamesMessage("stopped", names)}
}

func runLSPWorkspaceCommand(model ui.Model) command.Run {
	return func(e *view.Editor, args *command.Args) command.Result {
		doc, ctl, ok := lspCommandContext(e)
		if !ok {
			return command.Result{Error: errLSPUndefined}
		}
		if args == nil || args.Empty() {
			return kit.Runner(model.PickerAction(
				ui.LSPWorkspaceCommandPicker,
			))(e, args)
		}
		name, _ := args.First()
		err := ctl.ExecuteWorkspaceCommand(doc, name, lspCommandArgs(args))
		if err != nil {
			return lspCommandError(err)
		}
		return command.Result{Message: "executed workspace command: " + name}
	}
}

func lspCommandContext(
	e *view.Editor,
) (*view.Document, view.LanguageServerController, bool) {
	doc := e.FocusedDocument()
	if doc == nil {
		return nil, nil, false
	}
	ctl := e.LanguageServerController()
	return doc, ctl, ctl != nil
}

func lspCommandError(err error) command.Result {
	switch {
	case errors.Is(err, view.ErrNoLanguageServer):
		return command.Result{Error: errLSPUndefined}
	case errors.Is(err, view.ErrUnknownLanguageServer):
		return command.Result{Error: err}
	case errors.Is(err, view.ErrWorkspaceCommand):
		return command.Result{Error: err}
	default:
		return command.Result{Error: err}
	}
}

func lspNamesMessage(action string, names []string) string {
	if len(names) == 0 {
		return "no language servers " + action
	}
	return fmt.Sprintf(
		"language servers %s: %s", action, strings.Join(names, ", "),
	)
}

func positionals(args *command.Args) []string {
	if args == nil {
		return nil
	}
	return args.Positionals()
}

func lspCommandArgs(args *command.Args) []string {
	pos := args.Positionals()
	if len(pos) < 2 {
		return nil
	}
	return pos[1:]
}
