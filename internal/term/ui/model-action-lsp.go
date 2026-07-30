package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

func (m Model) RenameSymbolAction() command.KeyAction {
	ec := m.component
	return func(e *view.Editor) command.Continuation {
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
			e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoRename))
			return nil
		}
		prefill, err := ls.RenameSymbolPrefill(doc, v.ID())
		if err != nil {
			e.SetStatusMsg(i18n.ErrorText(err))
			return nil
		}
		ec.keys.nextLayer = func(cx *Context) (Component, tea.Cmd) {
			return newPromptComponent(cx, promptComponentArgs{
				ec:      ec,
				kind:    promptRegex,
				prompt:  i18n.Text(i18n.PromptRename),
				prefill: prefill,
				fn: func(e *view.Editor, name string) error {
					return renameSymbol(e, name)
				},
			}), nil
		}
		return nil
	}
}

func (m Model) CompletionAction() command.KeyAction {
	ec := m.component
	return func(e *view.Editor) command.Continuation {
		if e.FocusedDocument() == nil {
			return nil
		}
		if e.FocusedView() == nil {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			return nil
		}
		ec.keys.nextLayer = func(cx *Context) (Component, tea.Cmd) {
			return nil, ec.completionCmd(cx, false)
		}
		return nil
	}
}

func (m Model) HoverAction() command.KeyAction {
	ec := m.component
	return func(e *view.Editor) command.Continuation {
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
			e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoHover))
			return nil
		}
		text, err := ls.Hover(doc, v.ID())
		if err != nil {
			e.SetStatusMsg(i18n.ErrorText(err))
			return nil
		}
		if text == "" {
			e.SetStatusMsg(i18n.Text(i18n.StatusNoHoverResults))
			return nil
		}
		anchor := newHoverAnchor(doc, v)
		ec.keys.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newHoverComponent(ec, anchor, text), nil
		}
		return nil
	}
}

func (m Model) SignatureHelpAction() command.KeyAction {
	ec := m.component
	return func(e *view.Editor) command.Continuation {
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
			e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoSignatureHelp))
			return nil
		}
		call, ok := currentSignatureCall(m.context)
		if !ok {
			return nil
		}
		ec.language.signatureHidden = nil
		help, err := ls.SignatureHelp(doc, v.ID())
		if err != nil {
			e.SetStatusMsg(i18n.ErrorText(err))
			return nil
		}
		if len(help.Signatures) == 0 {
			return nil
		}
		ec.keys.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newSignatureHelpComponent(ec, call, help), nil
		}
		return nil
	}
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
