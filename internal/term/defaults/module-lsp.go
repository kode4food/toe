package defaults

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kode4food/toe/internal/lsp"
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

func lspModule(model ui.Model) command.Module {
	spc := prefixed(char(' '))
	g := prefixed(char('g'))
	return command.Module{
		Commands: map[string]command.Command{
			actGotoDeclaration: {
				DocString: "Goto declaration",
				Run:       Continuation(model.GotoDeclarationAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('D'))),
			},
			actGotoDefinition: {
				DocString: "Goto definition",
				Run:       Continuation(model.GotoDefinitionAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('d'))),
			},
			actGotoTypeDefinition: {
				DocString: "Goto type definition",
				Run:       Continuation(model.GotoTypeDefinitionAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('y'))),
			},
			actGotoImplementation: {
				DocString: "Goto implementation",
				Run:       Continuation(model.GotoImplementationAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('i'))),
			},
			actGotoReference: {
				DocString: "Goto references",
				Run:       Continuation(model.GotoReferenceAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('r'))),
			},
			actSelectReferences: {
				DocString: "Select symbol references",
				Run:       Continuation(model.SelectReferencesAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('h'))),
			},
			actCodeAction: {
				DocString: "Perform code action",
				Run:       Continuation(model.CodeActionPickerAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('a'))),
			},
			actHover: {
				DocString: "Show docs for item under cursor",
				Run:       Continuation(model.HoverAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('k'))),
			},
			actRenameSymbol: {
				DocString: "Rename symbol",
				Run:       Continuation(model.RenameSymbolAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('r'))),
			},
			actSignatureHelp: {
				DocString: "Show signature help",
				Run:       Continuation(model.SignatureHelpAction()),
				Modes:     []string{"INS"},
			},
			actSymbolPicker: {
				DocString: "Open symbol picker",
				Run:       Continuation(model.SymbolPickerAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('s'))),
			},
			actWorkspaceSymbol: {
				DocString: "Open workspace symbol picker",
				Run:       Continuation(model.WorkspaceSymbolPickerAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('S'))),
			},
			actLSPRestart: {
				DocString: "Restart language servers for the current document",
				Run:       runLSPRestart,
				Signature: command.Signature{},
			},
			actLSPStop: {
				DocString: "Stop language servers for the current document",
				Run:       runLSPStop,
				Signature: command.Signature{},
			},
			actLSPWorkspaceCommand: {
				DocString: "Execute a language server workspace command",
				Run:       runLSPWorkspaceCommand(model),
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
		return command.Result{Message: "error: LSP not defined for document"}
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
		return command.Result{Message: "error: LSP not defined for document"}
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
			return command.Result{
				Message: "error: LSP not defined for document",
			}
		}
		if args == nil || args.Empty() {
			return Continuation(model.PickerAction(
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
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil, nil, false
	}
	ctl := e.LanguageServerController()
	return doc, ctl, ctl != nil
}

func lspCommandError(err error) command.Result {
	switch {
	case errors.Is(err, lsp.ErrNoLanguageServer):
		return command.Result{Message: "error: LSP not defined for document"}
	case errors.Is(err, lsp.ErrUnknownLanguageServer):
		return command.Result{Message: "error: " + err.Error()}
	case errors.Is(err, lsp.ErrWorkspaceCommand):
		return command.Result{Message: "error: " + err.Error()}
	default:
		return command.Result{Message: "error: " + err.Error()}
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
