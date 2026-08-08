package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

// SymbolPickerAction opens a picker over the focused document's symbols
func (m Model) SymbolPickerAction(e *view.Editor) {
	ec := m.component
	cx := m.context

	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	ls := e.LanguageServerController()
	if ls == nil {
		e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoDocSymbols))
		return
	}
	symbols, err := ls.DocumentSymbols(doc)
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return
	}
	if len(symbols) == 0 {
		e.SetStatusMsg(i18n.Text(i18n.StatusNoDocumentSymbols))
		return
	}
	opener := symbolPickerLayer(symbols)
	cx.lastLayer = opener
	ec.keys.nextLayer = opener(e)
}

// WorkspaceSymbolPickerAction opens a picker over workspace symbols
func (m Model) WorkspaceSymbolPickerAction(e *view.Editor) {
	ec := m.component
	cx := m.context

	ls := e.LanguageServerController()
	if ls == nil {
		e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoWorkSymbols))
		return
	}
	opener := workspaceSymbolPickerLayer()
	cx.lastLayer = opener
	ec.keys.nextLayer = opener(e)
}

// CodeActionPickerAction opens a menu of code actions at the cursor
func (m Model) CodeActionPickerAction(e *view.Editor) {
	ec := m.component
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	v := e.FocusedView()
	if v == nil {
		return
	}
	ls := e.LanguageServerController()
	if ls == nil {
		e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoCodeActions))
		return
	}
	actions, err := ls.CodeActions(doc, v.ID())
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return
	}
	if len(actions) == 0 {
		e.SetStatusMsg(i18n.Text(i18n.StatusNoCodeActions))
		return
	}
	docID := doc.ID()
	viewID := v.ID()
	ec.keys.nextLayer = func(*Context) (Component, tea.Cmd) {
		return newCodeActionMenu(ec, docID, viewID, actions), nil
	}
}

func symbolPickerLayer(symbols []view.Symbol) func(*view.Editor) layerFunc {
	return func(e *view.Editor) layerFunc {
		p := newLSPSymbolPicker(e, symbols)
		cmd := p.load.feedCmd
		p.load.feedCmd = nil
		return func(cx *Context) (Component, tea.Cmd) {
			return newPickerComponent(cx, p), cmd
		}
	}
}

func workspaceSymbolPickerLayer() func(*view.Editor) layerFunc {
	return func(e *view.Editor) layerFunc {
		p := newLSPWorkspaceSymbolPicker(e)
		cmd := p.load.feedCmd
		p.load.feedCmd = nil
		return func(cx *Context) (Component, tea.Cmd) {
			return newPickerComponent(cx, p), cmd
		}
	}
}
