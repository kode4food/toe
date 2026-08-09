package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

const (
	promptRenameKey             i18n.Key = "prompt.rename"
	statusNoHoverResultsKey     i18n.Key = "status.noHoverResults"
	statusLSPNoRenameKey        i18n.Key = "status.lspNoRename"
	statusLSPNoHoverKey         i18n.Key = "status.lspNoHover"
	statusLSPNoSignatureHelpKey i18n.Key = "status.lspNoSignatureHelp"
)

// RenameSymbolAction prompts for a new name and applies the server's edits
func (m Model) RenameSymbolAction(e *view.Editor) {
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
		e.SetStatusMsg(i18n.Text(statusLSPNoRenameKey))
		return
	}
	prefill, err := ls.RenameSymbolPrefill(doc, v.ID())
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return
	}
	ec.queueNextLayer(func(cx *Context) (Component, tea.Cmd) {
		return newPromptComponent(cx, promptComponentArgs{
			editor:  ec,
			kind:    promptRegex,
			prompt:  i18n.Text(promptRenameKey),
			prefill: prefill,
			handler: func(e *view.Editor, name string) error {
				return renameSymbol(e, name)
			},
		}), nil
	})
}

// CompletionAction requests completions at the cursor
func (m Model) CompletionAction(e *view.Editor) {
	ec := m.component
	if e.FocusedDocument() == nil {
		return
	}
	if e.FocusedView() == nil {
		return
	}
	ls := e.LanguageServerController()
	if ls == nil {
		return
	}
	ec.queueNextLayer(func(cx *Context) (Component, tea.Cmd) {
		return nil, ec.completionCmd(cx, false)
	})
}

// HoverAction requests documentation for the symbol at the cursor
func (m Model) HoverAction(e *view.Editor) {
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
		e.SetStatusMsg(i18n.Text(statusLSPNoHoverKey))
		return
	}
	text, err := ls.Hover(doc, v.ID())
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return
	}
	if text == "" {
		e.SetStatusMsg(i18n.Text(statusNoHoverResultsKey))
		return
	}
	anchor := newHoverAnchor(doc, v)
	ec.queueNextLayer(func(*Context) (Component, tea.Cmd) {
		return newHoverComponent(ec, anchor, text), nil
	})
}

// SignatureHelpAction requests parameter hints for the call at the cursor
func (m Model) SignatureHelpAction(e *view.Editor) {
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
		e.SetStatusMsg(i18n.Text(statusLSPNoSignatureHelpKey))
		return
	}
	call, ok := currentSignatureCall(m.context)
	if !ok {
		return
	}
	ec.language.signatureHidden = nil
	help, err := ls.SignatureHelp(doc, v.ID())
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return
	}
	if len(help.Signatures) == 0 {
		return
	}
	ec.queueNextLayer(func(*Context) (Component, tea.Cmd) {
		return newSignatureHelpComponent(ec, call, help), nil
	})
}

func renameSymbol(e *view.Editor, name string) error {
	doc := e.FocusedDocument()
	if doc == nil {
		return nil
	}
	v := e.FocusedView()
	if v == nil {
		return nil
	}
	ls := e.LanguageServerController()
	if ls == nil {
		return nil
	}
	return ls.RenameSymbol(doc, v.ID(), name)
}
