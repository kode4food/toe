package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type locationGetter func(
	view.LanguageServerController, *view.Document, view.Id,
) ([]view.Location, error)

func (m Model) WithStartupCmd(cmd tea.Cmd) Model {
	m.initCmd = tea.Batch(m.initCmd, cmd)
	return m
}

func (m Model) WithInitialPicker(fn PickerFunc) Model {
	m.compositor.startup = func(cx *Context) (Component, tea.Cmd) {
		p := fn(cx.Editor)
		if p == nil {
			return nil, nil
		}
		cmd := p.feedCmd
		p.feedCmd = nil
		return newPickerComponent(p), cmd
	}
	return m
}

func (m Model) PickerAction(fn PickerFunc) command.KeyAction {
	ec := m.component
	cx := m.context
	opener := func(e *view.Editor) layerFunc {
		p := fn(e)
		if p == nil {
			return nil
		}
		cmd := p.feedCmd
		p.feedCmd = nil
		return func(_ *Context) (Component, tea.Cmd) {
			return newPickerComponent(p), cmd
		}
	}
	return func(e *view.Editor) command.Continuation {
		layer := opener(e)
		if layer == nil {
			return nil
		}
		cx.lastLayer = opener
		ec.nextLayer = layer
		return nil
	}
}

func (m Model) CmdModeAction() command.KeyAction {
	ec := m.component
	return func(_ *view.Editor) command.Continuation {
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec: ec, kind: promptCmd,
			}), nil
		}
		return nil
	}
}

func (m Model) SearchAction(forward bool) command.KeyAction {
	ec := m.component
	return func(_ *view.Editor) command.Continuation {
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec: ec, kind: promptSearch, forward: forward,
			}), nil
		}
		return nil
	}
}

func (m Model) RegexAction(
	prompt string, fn func(*view.Editor, string) error,
) command.KeyAction {
	ec := m.component
	return func(_ *view.Editor) command.Continuation {
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec: ec, kind: promptRegex, prompt: prompt, fn: fn,
			}), nil
		}
		return nil
	}
}

func (m Model) ShellAction(
	prompt string, fn func(*view.Editor, string) error,
) command.KeyAction {
	ec := m.component
	return func(_ *view.Editor) command.Continuation {
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec: ec, kind: promptShell, prompt: prompt, fn: fn,
			}), nil
		}
		return nil
	}
}

func (m Model) GlobalSearchAction() command.KeyAction {
	ec := m.component
	cx := m.context
	opener := func(e *view.Editor) layerFunc {
		p := NewPicker(e, &globalSearchSource{
			pickerMeta: pickerMeta{
				title:   "Global search",
				columns: []string{"path"},
			},
		})
		cmd := p.feedCmd
		p.feedCmd = nil
		return func(_ *Context) (Component, tea.Cmd) {
			return newPickerComponent(p), cmd
		}
	}
	return func(e *view.Editor) command.Continuation {
		cx.lastLayer = opener
		ec.nextLayer = opener(e)
		return nil
	}
}

func (m Model) CommandPaletteAction() command.KeyAction {
	ec := m.component
	cx := m.context
	opener := func(e *view.Editor) layerFunc {
		p := CommandPalettePicker(e, cx.Keymaps)
		cmd := p.feedCmd
		p.feedCmd = nil
		return func(_ *Context) (Component, tea.Cmd) {
			return newPickerComponent(p), cmd
		}
	}
	return func(e *view.Editor) command.Continuation {
		cx.lastLayer = opener
		ec.nextLayer = opener(e)
		return nil
	}
}

func (m Model) LastPickerAction() command.KeyAction {
	ec := m.component
	cx := m.context
	return func(e *view.Editor) command.Continuation {
		if cx.lastLayer == nil {
			return nil
		}
		ec.nextLayer = cx.lastLayer(e)
		return nil
	}
}

func (m Model) CompletionAction() command.KeyAction {
	ec := m.component
	return func(e *view.Editor) command.Continuation {
		doc, ok := e.FocusedDocument()
		if !ok {
			return nil
		}
		v, ok := e.FocusedView()
		if !ok {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			return nil
		}
		items, err := ls.Completions(doc, v.ID())
		if err != nil {
			e.SetStatusMsg(err.Error())
			return nil
		}
		if len(items) == 0 {
			return nil
		}
		anchor := newCompletionAnchor(doc, v.ID())
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newCompletionComponent(ec, items, anchor), nil
		}
		return nil
	}
}

func (m Model) HoverAction() command.KeyAction {
	ec := m.component
	return func(e *view.Editor) command.Continuation {
		doc, ok := e.FocusedDocument()
		if !ok {
			return nil
		}
		v, ok := e.FocusedView()
		if !ok {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			e.SetStatusMsg("No configured language server supports hover")
			return nil
		}
		text, err := ls.Hover(doc, v.ID())
		if err != nil {
			e.SetStatusMsg(err.Error())
			return nil
		}
		if text == "" {
			e.SetStatusMsg("No hover results available.")
			return nil
		}
		anchor := newHoverAnchor(doc, v)
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newHoverComponent(ec, anchor, text), nil
		}
		return nil
	}
}

func (m Model) SignatureHelpAction() command.KeyAction {
	ec := m.component
	return func(e *view.Editor) command.Continuation {
		doc, ok := e.FocusedDocument()
		if !ok {
			return nil
		}
		v, ok := e.FocusedView()
		if !ok {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			e.SetStatusMsg(
				"No configured language server supports signature-help",
			)
			return nil
		}
		call, ok := currentSignatureCall(m.context)
		if !ok {
			return nil
		}
		ec.signatureHidden = nil
		help, err := ls.SignatureHelp(doc, v.ID())
		if err != nil {
			e.SetStatusMsg(err.Error())
			return nil
		}
		if len(help.Signatures) == 0 {
			return nil
		}
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newSignatureHelpComponent(ec, call, help), nil
		}
		return nil
	}
}

func (m Model) GotoDeclarationAction() command.KeyAction {
	return m.gotoLocationAction(
		"No declaration found.",
		func(
			ls view.LanguageServerController,
			doc *view.Document,
			viewID view.Id,
		) ([]view.Location, error) {
			return ls.GotoDeclaration(doc, viewID)
		},
	)
}

func (m Model) GotoDefinitionAction() command.KeyAction {
	return m.gotoLocationAction(
		"No definition found.",
		func(
			ls view.LanguageServerController,
			doc *view.Document,
			viewID view.Id,
		) ([]view.Location, error) {
			return ls.GotoDefinition(doc, viewID)
		},
	)
}

func (m Model) GotoTypeDefinitionAction() command.KeyAction {
	return m.gotoLocationAction(
		"No type definition found.",
		func(
			ls view.LanguageServerController,
			doc *view.Document,
			viewID view.Id,
		) ([]view.Location, error) {
			return ls.GotoTypeDefinition(doc, viewID)
		},
	)
}

func (m Model) GotoImplementationAction() command.KeyAction {
	return m.gotoLocationAction(
		"No implementation found.",
		func(
			ls view.LanguageServerController,
			doc *view.Document,
			viewID view.Id,
		) ([]view.Location, error) {
			return ls.GotoImplementation(doc, viewID)
		},
	)
}

func (m Model) GotoReferenceAction() command.KeyAction {
	return m.gotoLocationAction(
		"No references found.",
		func(
			ls view.LanguageServerController,
			doc *view.Document,
			viewID view.Id,
		) ([]view.Location, error) {
			return ls.GotoReference(doc, viewID)
		},
	)
}

func (m Model) SymbolPickerAction() command.KeyAction {
	ec := m.component
	cx := m.context
	return func(e *view.Editor) command.Continuation {
		doc, ok := e.FocusedDocument()
		if !ok {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			e.SetStatusMsg(
				"No configured language server supports document symbols",
			)
			return nil
		}
		symbols, err := ls.DocumentSymbols(doc)
		if err != nil {
			e.SetStatusMsg(err.Error())
			return nil
		}
		if len(symbols) == 0 {
			e.SetStatusMsg("No document symbols found.")
			return nil
		}
		opener := symbolPickerLayer(symbols)
		cx.lastLayer = opener
		ec.nextLayer = opener(e)
		return nil
	}
}

func (m Model) MacroRecordAction(e *view.Editor) command.Continuation {
	return m.component.MacroRecordAction(e)
}

func (m Model) MacroReplayAction(e *view.Editor) command.Continuation {
	return m.component.MacroReplayAction(e)
}

func (m Model) gotoLocationAction(
	notFound string, get locationGetter,
) command.KeyAction {
	ec := m.component
	cx := m.context
	return func(e *view.Editor) command.Continuation {
		doc, ok := e.FocusedDocument()
		if !ok {
			return nil
		}
		v, ok := e.FocusedView()
		if !ok {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			e.SetStatusMsg("No configured language server supports navigation")
			return nil
		}
		locations, err := get(ls, doc, v.ID())
		if err != nil {
			e.SetStatusMsg(err.Error())
			return nil
		}
		switch len(locations) {
		case 0:
			e.SetStatusMsg(notFound)
		case 1:
			jumpToLocation(e, locations[0])
		default:
			opener := locationPickerLayer(locations)
			cx.lastLayer = opener
			ec.nextLayer = opener(e)
		}
		return nil
	}
}

func locationPickerLayer(
	locations []view.Location,
) func(*view.Editor) layerFunc {
	return func(e *view.Editor) layerFunc {
		p := newLSPLocationPicker(e, locations)
		cmd := p.feedCmd
		p.feedCmd = nil
		return func(_ *Context) (Component, tea.Cmd) {
			return newPickerComponent(p), cmd
		}
	}
}

func symbolPickerLayer(symbols []view.Symbol) func(*view.Editor) layerFunc {
	return func(e *view.Editor) layerFunc {
		p := newLSPSymbolPicker(e, symbols)
		cmd := p.feedCmd
		p.feedCmd = nil
		return func(_ *Context) (Component, tea.Cmd) {
			return newPickerComponent(p), cmd
		}
	}
}

func jumpToLocation(e *view.Editor, loc view.Location) {
	action.SaveSelection(e)
	v, err := e.SwitchFile(loc.Path)
	if err != nil {
		e.SetStatusMsg(err.Error())
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel, err := locationSelection(loc)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
}
