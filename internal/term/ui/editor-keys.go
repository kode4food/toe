package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	act "github.com/kode4food/toe/internal/view/action"
)

func (e *EditorComponent) handleKeyPress(
	msg tea.KeyPressMsg, cx *Context,
) (EventResult, tea.Cmd) {
	k := FromTeaKey(msg)

	if len(e.pending) == 0 && k.Code.Char == 'z' && k.Mods == command.ModCtrl {
		return consumed(), tea.Suspend
	}

	if e.macroSlot.recording {
		e.macroSlot.keys = append(e.macroSlot.keys, k)
	}

	if e.continuation != nil {
		cont := e.continuation(cx.Editor, k)
		e.continuation = cont
		if cont == nil {
			e.hint = ""
			cx.Editor.ResetCount()
		}
		e.syncEditorMessages(cx)
		e.handleReplay(cx)
		return consumed(), nil
	}

	mode := cx.Editor.Mode()
	modeStr := mode.String()

	noPending := len(e.pending) == 0 && k.Mods == command.ModNone &&
		k.Code.Special == ""
	if noPending && mode == view.ModeSelect {
		ch := k.Code.Char
		cur := cx.Editor.Count()
		if ch >= '1' && ch <= '9' || (ch == '0' && cur > 0) {
			cx.Editor.SetCount(cur*10 + int(ch-'0'))
			e.status = fmt.Sprintf("%d", cx.Editor.Count())
			return consumed(), nil
		}
	}

	e.pending = append(e.pending, k)
	action, found, prefix := cx.Keymaps.Lookup(modeStr, e.pending)
	switch {
	case found:
		e.pending = nil
		e.status = ""
		e.cmdMsg = ""
		e.infoTitle = ""
		e.infoItems = nil
		cont := action(cx.Editor)
		e.continuation = cont
		if cont == nil {
			e.hint = ""
			cx.Editor.ResetCount()
		}
		e.syncEditorMessages(cx)
		e.handleReplay(cx)

		if ov := e.nextLayer; ov != nil {
			e.nextLayer = nil
			return consumedWith(func(comp *Compositor, cx *Context) tea.Cmd {
				layer, cmd := ov(cx)
				if layer != nil {
					comp.Push(layer)
				}
				return cmd
			}), nil
		}
		return consumed(), nil

	case prefix:
		if mode == view.ModeInsert && len(e.pending) == 1 {
			if e.pending[0].IsTypable() {
				if layer := e.insertTypable(cx, e.pending[0]); layer != nil {
					return consumedWith(layer), nil
				}
				return consumed(), nil
			}
		}
		var sb strings.Builder
		if c := cx.Editor.Count(); c > 0 {
			_, _ = fmt.Fprintf(&sb, "%d", c)
		}
		for _, pk := range e.pending {
			sb.WriteString(pk.String())
		}
		e.status = sb.String()
		title, items := cx.Keymaps.PendingHints(modeStr, e.pending)
		e.infoTitle = title
		e.infoItems = items
		return consumed(), nil

	default:
		if mode == view.ModeInsert && len(e.pending) == 1 {
			if e.pending[0].IsTypable() {
				if layer := e.insertTypable(cx, e.pending[0]); layer != nil {
					return consumedWith(layer), nil
				}
				return consumed(), nil
			}
		}
		e.pending = nil
		e.status = ""
		e.infoTitle = ""
		e.infoItems = nil
		cx.Editor.ResetCount()
		return consumed(), nil
	}
}

func (e *EditorComponent) insertTypable(
	cx *Context, k command.KeyEvent,
) Callback {
	act.InsertChar(cx.Editor, k.Code.Char)
	e.pending = nil
	e.status = ""
	e.infoTitle = ""
	e.infoItems = nil
	cx.Editor.ResetCount()
	if layer := e.triggerCompletionLayer(cx); layer != nil {
		return layer
	}
	return e.triggerSignatureHelpLayer(cx)
}

func (e *EditorComponent) triggerCompletionLayer(cx *Context) Callback {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return nil
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return nil
	}
	ls := cx.Editor.LanguageServerController()
	if ls == nil {
		return nil
	}
	items, err := ls.TriggerCompletions(doc, v.ID())
	if err != nil {
		cx.Editor.SetStatusMsg(err.Error())
		return nil
	}
	if len(items) == 0 {
		return nil
	}
	anchor := newCompletionAnchor(doc, v.ID())
	return func(comp *Compositor, _ *Context) tea.Cmd {
		comp.Push(newCompletionComponent(e, items, anchor))
		return nil
	}
}

func (e *EditorComponent) triggerSignatureHelpLayer(cx *Context) Callback {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return nil
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return nil
	}
	ls := cx.Editor.LanguageServerController()
	if ls == nil {
		return nil
	}
	call, ok := currentSignatureCall(cx)
	if !ok {
		e.signatureHidden = nil
		return nil
	}
	if e.signatureHidden != nil && *e.signatureHidden == call {
		return nil
	}
	e.signatureHidden = nil
	help, err := ls.TriggerSignatureHelp(doc, v.ID())
	if err != nil {
		cx.Editor.SetStatusMsg(err.Error())
		return nil
	}
	if len(help.Signatures) == 0 {
		return nil
	}
	return func(comp *Compositor, _ *Context) tea.Cmd {
		pushSignatureHelpLayer(comp, newSignatureHelpComponent(e, call, help))
		return nil
	}
}

func newCompletionAnchor(doc *view.Document, viewID view.Id) completionAnchor {
	sel := doc.SelectionFor(viewID)
	return completionAnchor{
		docID:  doc.ID(),
		viewID: viewID,
		pos:    sel.Primary().Cursor(doc.Text()),
	}
}
