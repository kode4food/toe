package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const countArrow = "\u2192" // '→' - rightwards arrow

func (e *EditorComponent) handleKeyPress(
	cx *Context, msg tea.KeyPressMsg,
) (EventResult, tea.Cmd) {
	p := cx.Editor.Tree().Get(cx.Editor.Tree().Focus())
	k := FromTeaKey(msg)
	// a raw-input pane (terminal) forwards keystrokes to its child, so let the
	// keymap claim anything bound in the current mode before the pane sees it
	if pi, ok := p.(PaneInput); ok && !e.keymapClaims(cx, k) {
		if result, handled := pi.HandleEvent(cx, msg); handled {
			return result, nil
		}
	}

	e.language.completionGen++

	if len(e.keys.pending) == 0 &&
		k.Code.Char == 'z' && k.Mods == command.ModCtrl {
		return consumed(), tea.Suspend
	}

	if e.macroSlot.recording {
		e.macroSlot.keys = append(e.macroSlot.keys, k)
	}

	if e.keys.continuation != nil {
		cont := e.keys.continuation(cx.Editor, k)
		e.keys.continuation = cont
		if cont == nil {
			e.keys.hint = ""
			cx.Editor.ResetCount()
		}
		e.syncEditorMessages(cx)
		e.handleReplay(cx)
		return consumed(), nil
	}

	mode := cx.Editor.Mode()
	modeStr := mode.String()

	if countable(mode, k) {
		ch := k.Code.Char
		cur := cx.Editor.Count()
		if ch >= '1' && ch <= '9' || (ch == '0' && cur > 0) {
			cx.Editor.SetCount(cur*10 + int(ch-'0'))
			e.keys.status = e.pendingStatus(cx)
			return consumed(), nil
		}
	}

	e.keys.pending = append(e.keys.pending, k)
	act, found, prefix := cx.Keymaps.Lookup(modeStr, e.keys.pending)
	switch {
	case found:
		e.keys.pending = nil
		e.keys.status = ""
		e.clearCommandMessage()
		e.keys.infoTitle = ""
		e.keys.infoItems = nil
		cont := act(cx.Editor)
		e.keys.continuation = cont
		if cont == nil {
			e.keys.hint = ""
			cx.Editor.ResetCount()
		}
		e.syncEditorMessages(cx)
		e.handleReplay(cx)

		if ov := e.keys.nextLayer; ov != nil {
			e.keys.nextLayer = nil
			return consumedWith(func(cx *Context, comp *Compositor) tea.Cmd {
				layer, cmd := ov(cx)
				if layer != nil {
					comp.Push(layer)
				}
				return cmd
			}), nil
		}
		return consumed(), nil

	case prefix:
		if mode == view.ModeInsert && len(e.keys.pending) == 1 {
			if e.keys.pending[0].IsTypable() {
				if layer := e.insertTypable(
					cx, e.keys.pending[0],
				); layer != nil {
					return consumedWith(layer), nil
				}
				return consumed(), nil
			}
		}
		e.keys.status = e.pendingStatus(cx)
		title, items := cx.Keymaps.PendingHints(modeStr, e.keys.pending)
		e.keys.infoTitle = title
		e.keys.infoItems = items
		return consumed(), nil

	default:
		if mode == view.ModeInsert && len(e.keys.pending) == 1 {
			if e.keys.pending[0].IsTypable() {
				if layer := e.insertTypable(
					cx, e.keys.pending[0],
				); layer != nil {
					return consumedWith(layer), nil
				}
				return consumed(), nil
			}
		}
		e.keys.pending = nil
		e.keys.status = ""
		e.keys.infoTitle = ""
		e.keys.infoItems = nil
		cx.Editor.ResetCount()
		return consumed(), nil
	}
}

func (e *EditorComponent) pendingStatus(cx *Context) string {
	var sb strings.Builder
	for _, pk := range e.keys.pending {
		sb.WriteString(pk.String())
	}
	if c := cx.Editor.Count(); c > 0 {
		if sb.Len() > 0 {
			_, _ = fmt.Fprintf(&sb, " %s ", countArrow)
		}
		_, _ = fmt.Fprintf(&sb, "%d", c)
	}
	return sb.String()
}

// keymapClaims reports whether k continues or completes a binding in the
// focused pane's mode, so a raw-input pane must not swallow it first
func (e *EditorComponent) keymapClaims(cx *Context, k command.KeyEvent) bool {
	if len(e.keys.pending) > 0 || e.keys.continuation != nil {
		return true
	}
	if cx.Editor.Mode() == view.ModeTerminal && k.Mods == command.ModNone {
		return false
	}
	_, found, prefix := cx.Keymaps.Lookup(
		cx.Editor.Mode().String(), []command.KeyEvent{k},
	)
	return found || prefix
}

func (e *EditorComponent) insertTypable(
	cx *Context, k command.KeyEvent,
) Callback {
	action.InsertChar(cx.Editor, k.Code.Char)
	e.keys.pending = nil
	e.keys.status = ""
	e.keys.infoTitle = ""
	e.keys.infoItems = nil
	cx.Editor.ResetCount()
	tick := e.autoCompletionTick(cx)
	if layer := e.triggerSignatureHelpLayer(cx); layer != nil {
		return layerWithCmd(layer, tick)
	}
	if layer := e.triggerCompletionLayer(cx); layer != nil {
		return layerWithCmd(layer, tick)
	}
	if tick == nil {
		return nil
	}
	return func(*Context, *Compositor) tea.Cmd {
		return tick
	}
}

func (e *EditorComponent) autoCompletionTick(cx *Context) tea.Cmd {
	e.language.autoGen++
	opts := e.completion
	if !opts.Auto || !wordPrefixReady(cx, opts.TriggerLen) {
		return nil
	}
	gen := e.language.autoGen
	d := time.Duration(opts.Delay) * time.Millisecond
	return tea.Tick(d, func(time.Time) tea.Msg {
		return autoCompletionMsg{gen: gen}
	})
}

func (e *EditorComponent) triggerCompletionLayer(cx *Context) Callback {
	cmd := e.completionCmd(cx, true)
	if cmd == nil {
		return nil
	}
	return func(*Context, *Compositor) tea.Cmd {
		return cmd
	}
}

func (e *EditorComponent) completionCmd(cx *Context, trigger bool) tea.Cmd {
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
	anchor := newCompletionAnchor(doc, v.ID())
	e.language.completionGen++
	gen := e.language.completionGen
	return func() tea.Msg {
		var res view.CompletionResult
		var err error
		if trigger {
			res, err = ls.TriggerCompletions(doc, v.ID())
		} else {
			res, err = ls.Completions(doc, v.ID())
		}
		return completionMsg{
			gen:        gen,
			anchor:     anchor,
			items:      res.Items,
			incomplete: res.Incomplete,
			err:        err,
		}
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
		e.language.signatureHidden = nil
		return nil
	}
	if e.language.signatureHidden != nil &&
		*e.language.signatureHidden == call {
		return nil
	}
	e.language.signatureHidden = nil
	help, err := ls.TriggerSignatureHelp(doc, v.ID())
	if err != nil {
		cx.Editor.SetStatusMsg(i18n.ErrorText(err))
		return nil
	}
	if len(help.Signatures) == 0 {
		return nil
	}
	return func(_ *Context, comp *Compositor) tea.Cmd {
		pushSignatureHelpLayer(comp, newSignatureHelpComponent(e, call, help))
		return nil
	}
}

func countable(mode view.Mode, k command.KeyEvent) bool {
	return k.Mods == command.ModNone &&
		k.Code.Special == command.SpecialNone &&
		(mode == view.ModeNormal || mode == view.ModeSelect)
}

func newCompletionAnchor(doc *view.Document, viewID view.Id) completionAnchor {
	sel := doc.SelectionFor(viewID)
	return completionAnchor{
		docID:  doc.ID(),
		viewID: viewID,
		rev:    doc.Revision(),
		pos:    sel.Primary().Cursor(doc.Text()),
	}
}

func completionRequestValid(cx *Context, anchor completionAnchor) bool {
	if cx.Editor.Mode() != view.ModeInsert {
		return false
	}
	doc, ok := cx.Editor.FocusedDocument()
	if !ok || doc.ID() != anchor.docID || doc.Revision() != anchor.rev {
		return false
	}
	v, ok := cx.Editor.FocusedView()
	if !ok || v.ID() != anchor.viewID {
		return false
	}
	pos := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
	return pos == anchor.pos
}

// wordPrefixReady reports whether the limit characters before the cursor are
// all word characters, reading only those so a long line costs no more
func wordPrefixReady(cx *Context, limit int) bool {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return false
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return false
	}
	text := doc.Text()
	pos := doc.SelectionFor(v.ID()).Primary().Cursor(text)
	if pos < limit {
		return false
	}
	left, err := text.SliceString(pos-limit, pos)
	if err != nil {
		return false
	}
	return !strings.ContainsFunc(left, notWordChar)
}

func notWordChar(r rune) bool {
	return !core.CharIsWord(r)
}

func layerWithCmd(layer Callback, cmd tea.Cmd) Callback {
	if cmd == nil {
		return layer
	}
	return func(cx *Context, comp *Compositor) tea.Cmd {
		return tea.Batch(layer(cx, comp), cmd)
	}
}
