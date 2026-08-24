package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func (ec *EditorComponent) handleKeyPress(
	cx *Context, msg tea.KeyPressMsg,
) (EventResult, tea.Cmd) {
	p := cx.Editor.Tree().Get(cx.Editor.Tree().Focus())
	k := FromTeaKey(msg)
	// a raw-input pane (terminal) forwards keystrokes to its child, so let the
	// keymap claim anything bound in the current mode before the pane sees it
	if pi, ok := p.(PaneInput); ok && !ec.keymapClaims(cx, k) {
		if result, handled := pi.HandleEvent(cx, msg); handled {
			return result, nil
		}
	}

	ec.language.completionGen++

	if len(ec.keys.path) == 0 &&
		k.Code.Char == 'z' && k.Mods == command.ModCtrl {
		return consumed(), tea.Suspend
	}

	if ec.macroSlot.recording {
		ec.macroSlot.keys = append(ec.macroSlot.keys, k)
	}

	if len(ec.keys.input) > 0 &&
		k.Code.Special == command.Escape && k.Mods == command.ModNone {
		ec.cancelPending(cx)
		return consumed(), nil
	}

	if ec.keys.continuation != nil {
		ec.handleContinuation(cx, cx.Editor.Mode(), k)
		ec.syncEditorMessages(cx)
		ec.handleReplay(cx)
		return consumed(), nil
	}

	mode := cx.Editor.Mode()

	if countable(mode, k) && cx.Keymaps.AcceptsCount(mode, ec.keys.path) {
		ch := k.Code.Char
		cur := ec.keys.count
		if ch >= '1' && ch <= '9' || (ch == '0' && cur > 0) {
			ec.keys.input = append(ec.keys.input, keyInput{countDigit: true})
			ec.setCount(cx, cur*10+int(ch-'0'))
			ec.setHints(cx, mode)
			return consumed(), nil
		}
	}

	if len(ec.keys.input) > 0 &&
		k.Code.Special == command.Backspace && k.Mods == command.ModNone {
		ec.popPending(cx, mode)
		return consumed(), nil
	}

	ec.keys.path = append(ec.keys.path, k)
	ec.keys.input = append(ec.keys.input, keyInput{})
	lookup, ok := cx.Keymaps.Lookup(mode, ec.keys.path)
	if ok && !lookup.Enabled(cx.Editor) {
		ok = false
	}
	switch {
	case ok:
		ec.clearCommandMessage()
		ec.clearHints()
		res := lookup.Action(cx.Editor)
		ec.keys.continuation = res.Continuation
		if res.Continuation == nil {
			ec.keys.path = nil
			ec.clearInput(cx)
		} else {
			ec.keys.frames = nil
			ec.setHints(cx, mode)
		}
		ec.setCommandResult(res)
		ec.syncEditorMessages(cx)
		ec.handleReplay(cx)

		return consumed(), signalToCmd(res.Signal)

	case lookup.Prefix:
		if mode == view.ModeInsert && len(ec.keys.path) == 1 {
			if ec.keys.path[0].IsTypable() {
				if layer := ec.insertTypable(
					cx, ec.keys.path[0],
				); layer != nil {
					return consumedWith(layer), nil
				}
				return consumed(), nil
			}
		}
		ec.setHints(cx, mode)
		return consumed(), nil

	default:
		if mode == view.ModeInsert && len(ec.keys.path) == 1 {
			if ec.keys.path[0].IsTypable() {
				if layer := ec.insertTypable(
					cx, ec.keys.path[0],
				); layer != nil {
					return consumedWith(layer), nil
				}
				return consumed(), nil
			}
		}
		ec.keys.path = ec.keys.path[:len(ec.keys.path)-1]
		ec.keys.input = ec.keys.input[:len(ec.keys.input)-1]
		return consumed(), nil
	}
}

func (ec *EditorComponent) keymapClaims(cx *Context, k command.KeyEvent) bool {
	if len(ec.keys.path) > 0 || ec.keys.continuation != nil {
		return true
	}
	if cx.Editor.Mode() == view.ModeTerminal && k.Mods == command.ModNone {
		return false
	}
	lookup, ok := cx.Keymaps.Lookup(
		cx.Editor.Mode(), []command.KeyEvent{k},
	)
	if ok && !lookup.Enabled(cx.Editor) {
		return false
	}
	return ok || lookup.Prefix
}

func (ec *EditorComponent) insertTypable(
	cx *Context, k command.KeyEvent,
) Callback {
	action.InsertChar(cx.Editor, k.Code.Char)
	ec.keys.path = nil
	ec.clearHints()
	ec.clearInput(cx)
	tick := ec.autoCompletionTick(cx)
	if layer := ec.triggerSignatureHelpLayer(cx); layer != nil {
		return layerWithCmd(layer, tick)
	}
	if layer := ec.triggerCompletionLayer(cx); layer != nil {
		return layerWithCmd(layer, tick)
	}
	if tick == nil {
		return nil
	}
	return func(*Context, *Compositor) tea.Cmd {
		return tick
	}
}

func (ec *EditorComponent) autoCompletionTick(cx *Context) tea.Cmd {
	ec.language.autoGen++
	opts := ec.completion
	if !opts.Auto || !wordPrefixReady(cx, opts.TriggerLen) {
		return nil
	}
	gen := ec.language.autoGen
	d := time.Duration(opts.Delay) * time.Millisecond
	return tea.Tick(d, func(time.Time) tea.Msg {
		return autoCompletionMsg{gen: gen}
	})
}

func (ec *EditorComponent) triggerCompletionLayer(cx *Context) Callback {
	cmd := ec.completionCmd(cx, true)
	if cmd == nil {
		return nil
	}
	return func(*Context, *Compositor) tea.Cmd {
		return cmd
	}
}

func (ec *EditorComponent) completionCmd(cx *Context, trigger bool) tea.Cmd {
	doc := cx.Editor.FocusedDocument()
	if doc == nil {
		return nil
	}
	v := cx.Editor.FocusedView()
	if v == nil {
		return nil
	}
	ls := cx.Editor.LanguageServerController()
	if ls == nil {
		return nil
	}
	anchor := newCompletionAnchor(doc, v.ID())
	ec.language.completionGen++
	gen := ec.language.completionGen
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

func (ec *EditorComponent) triggerSignatureHelpLayer(cx *Context) Callback {
	doc := cx.Editor.FocusedDocument()
	if doc == nil {
		return nil
	}
	v := cx.Editor.FocusedView()
	if v == nil {
		return nil
	}
	ls := cx.Editor.LanguageServerController()
	if ls == nil {
		return nil
	}
	call, ok := currentSignatureCall(cx)
	if !ok {
		ec.language.signatureHidden = nil
		return nil
	}
	if ec.language.signatureHidden != nil &&
		*ec.language.signatureHidden == call {
		return nil
	}
	ec.language.signatureHidden = nil
	help, err := ls.TriggerSignatureHelp(doc, v.ID())
	if err != nil {
		cx.Editor.SetStatusMsg(i18n.ErrorText(err))
		return nil
	}
	if len(help.Signatures) == 0 {
		return nil
	}
	return func(_ *Context, comp *Compositor) tea.Cmd {
		pushSignatureHelpLayer(comp, newSignatureHelpComponent(ec, call, help))
		return nil
	}
}

func (ec *EditorComponent) popPending(cx *Context, mode view.Mode) {
	last := ec.keys.input[len(ec.keys.input)-1]
	ec.keys.input = ec.keys.input[:len(ec.keys.input)-1]
	if last.countDigit {
		ec.setCount(cx, ec.keys.count/10)
	} else {
		ec.keys.path = ec.keys.path[:len(ec.keys.path)-1]
	}
	if len(ec.keys.path) == 0 && ec.keys.count == 0 {
		ec.clearHints()
		return
	}
	ec.setHints(cx, mode)
}

func (ec *EditorComponent) setCount(cx *Context, n int) {
	ec.keys.count = n
	cx.Editor.SetCount(n)
}

func (ec *EditorComponent) clearInput(cx *Context) {
	ec.keys.input = nil
	ec.setCount(cx, 0)
}

func (ec *EditorComponent) handleContinuation(
	cx *Context, mode view.Mode, k command.KeyEvent,
) {
	next, step := ec.keys.continuation(cx.Editor, k)
	switch step {
	case command.ContinuationStay:
		if next != nil {
			ec.keys.continuation = next
		}
	case command.ContinuationPush:
		ec.pushContinuation(cx, mode, k, next)
	case command.ContinuationPop:
		ec.popContinuation(cx, mode)
	default:
		ec.keys.continuation = nil
		ec.keys.frames = nil
		ec.keys.path = nil
		ec.clearHints()
		ec.clearInput(cx)
	}
}

func (ec *EditorComponent) pushContinuation(
	cx *Context, mode view.Mode, k command.KeyEvent,
	next command.Continuation,
) {
	ec.keys.frames = append(ec.keys.frames, ec.keys.continuation)
	ec.keys.input = append(ec.keys.input, keyInput{})
	ec.keys.continuation = next
	ec.keys.path = append(ec.keys.path, k)
	ec.setHints(cx, mode)
}

func (ec *EditorComponent) popContinuation(cx *Context, mode view.Mode) {
	if n := len(ec.keys.frames); n > 0 {
		frame := ec.keys.frames[n-1]
		ec.keys.frames = ec.keys.frames[:n-1]
		ec.keys.input = ec.keys.input[:len(ec.keys.input)-1]
		ec.keys.path = ec.keys.path[:len(ec.keys.path)-1]
		ec.keys.continuation = frame
		ec.setHints(cx, mode)
		return
	}
	ec.keys.continuation = nil
	ec.clearHints()
	ec.popPending(cx, mode)
}

func (ec *EditorComponent) clearHints() {
	ec.keys.infoTitle = ""
	ec.keys.infoItems = nil
}

func (ec *EditorComponent) setHints(cx *Context, mode view.Mode) {
	counting := ec.keys.count > 0 && ec.keys.continuation == nil
	title := ec.keys.infoTitle
	ec.keys.infoTitle, ec.keys.infoItems = cx.Keymaps.PendingHints(
		cx.Editor, mode, ec.keys.path, counting,
	)
	switch {
	case counting && len(ec.keys.path) == 0:
		ec.keys.infoTitle = i18n.Text(i18n.StatusCounted)
	case ec.keys.infoTitle == "" && ec.keys.continuation != nil:
		ec.keys.infoTitle = title
	}
}

func newCompletionAnchor(doc *view.Document, viewID view.Id) completionAnchor {
	sel := doc.SelectionFor(viewID)
	text := doc.Text()
	return completionAnchor{
		docID:  doc.ID(),
		viewID: viewID,
		rev:    doc.Revision(),
		pos:    wordStart(text, sel.Primary().Cursor(text)),
	}
}

func countable(mode view.Mode, k command.KeyEvent) bool {
	return k.Mods == command.ModNone &&
		k.Code.Special == command.SpecialNone &&
		(mode == view.ModeNormal || mode == view.ModeSelect)
}

func wordStart(text core.Rope, pos int) int {
	for pos > 0 {
		ch, err := text.CharAt(pos - 1)
		if err != nil || !core.CharIsWord(ch) {
			break
		}
		pos--
	}
	return pos
}

func completionRequestValid(cx *Context, anchor completionAnchor) bool {
	if cx.Editor.Mode() != view.ModeInsert {
		return false
	}
	doc := cx.Editor.FocusedDocument()
	if doc == nil || doc.ID() != anchor.docID ||
		doc.Revision() != anchor.rev {
		return false
	}
	v := cx.Editor.FocusedView()
	if v == nil || v.ID() != anchor.viewID {
		return false
	}
	pos := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
	return pos >= anchor.pos
}

func wordPrefixReady(cx *Context, limit int) bool {
	doc := cx.Editor.FocusedDocument()
	if doc == nil {
		return false
	}
	v := cx.Editor.FocusedView()
	if v == nil {
		return false
	}
	text := doc.Text()
	pos := doc.SelectionFor(v.ID()).Primary().Cursor(text)
	if pos < limit {
		return false
	}
	left, err := text.SliceString(core.Span{From: pos - limit, To: pos})
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
