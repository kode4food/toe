package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

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
			e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoDocSymbols))
			return nil
		}
		symbols, err := ls.DocumentSymbols(doc)
		if err != nil {
			e.SetStatusMsg(err.Error())
			return nil
		}
		if len(symbols) == 0 {
			e.SetStatusMsg(i18n.Text(i18n.StatusNoDocumentSymbols))
			return nil
		}
		opener := symbolPickerLayer(symbols)
		cx.lastLayer = opener
		ec.nextLayer = opener(e)
		return nil
	}
}

func (m Model) WorkspaceSymbolPickerAction() command.KeyAction {
	ec := m.component
	cx := m.context
	return func(e *view.Editor) command.Continuation {
		_, ok := e.FocusedDocument()
		if !ok {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoWorkSymbols))
			return nil
		}
		opener := workspaceSymbolPickerLayer()
		cx.lastLayer = opener
		ec.nextLayer = opener(e)
		return nil
	}
}

func (m Model) CodeActionPickerAction() command.KeyAction {
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
			e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoCodeActions))
			return nil
		}
		actions, err := ls.CodeActions(doc, v.ID())
		if err != nil {
			e.SetStatusMsg(err.Error())
			return nil
		}
		if len(actions) == 0 {
			e.SetStatusMsg(i18n.Text(i18n.StatusNoCodeActions))
			return nil
		}
		docID := doc.ID()
		viewID := v.ID()
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newCodeActionMenu(ec, docID, viewID, actions), nil
		}
		return nil
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

func workspaceSymbolPickerLayer() func(*view.Editor) layerFunc {
	return func(e *view.Editor) layerFunc {
		p := newLSPWorkspaceSymbolPicker(e)
		cmd := p.feedCmd
		p.feedCmd = nil
		return func(_ *Context) (Component, tea.Cmd) {
			return newPickerComponent(p), cmd
		}
	}
}
