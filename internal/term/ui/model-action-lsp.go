package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

func (m Model) RenameSymbolAction() command.KeyAction {
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
				"No configured language server supports symbol renaming",
			)
			return nil
		}
		prefill, err := ls.RenameSymbolPrefill(doc, v.ID())
		if err != nil {
			e.SetStatusMsg(err.Error())
			return nil
		}
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec: ec, kind: promptRegex, prompt: "rename-to:",
				buf: prefill,
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
		if _, ok := e.FocusedDocument(); !ok {
			return nil
		}
		if _, ok := e.FocusedView(); !ok {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			return nil
		}
		ec.nextLayer = func(cx *Context) (Component, tea.Cmd) {
			return nil, ec.completionCmd(cx, false)
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

func renameSymbol(e *view.Editor, name string) error {
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
	return ls.RenameSymbol(doc, v.ID(), name)
}
