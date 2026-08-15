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

	if len(e.keys.path) == 0 &&
		k.Code.Char == 'z' && k.Mods == command.ModCtrl {
		return consumed(), tea.Suspend
	}

	if e.macroSlot.recording {
		e.macroSlot.keys = append(e.macroSlot.keys, k)
	}

	if len(e.keys.input) > 0 &&
		k.Code.Special == command.Escape && k.Mods == command.ModNone {
		e.cancelPending(cx)
		return consumed(), nil
	}

	if e.keys.continuation != nil {
		e.handleContinuation(cx, cx.Editor.Mode(), k)
		e.syncEditorMessages(cx)
		e.handleReplay(cx)
		return consumed(), nil
	}

	mode := cx.Editor.Mode()

	if countable(mode, k) && cx.Keymaps.AcceptsCount(mode, e.keys.path) {
		ch := k.Code.Char
		cur := e.keys.count
		if ch >= '1' && ch <= '9' || (ch == '0' && cur > 0) {
			e.keys.input = append(e.keys.input, keyInput{countDigit: true})
			e.setCount(cx, cur*10+int(ch-'0'))
			e.setHints(cx, mode)
			return consumed(), nil
		}
	}

	if len(e.keys.input) > 0 &&
		k.Code.Special == command.Backspace && k.Mods == command.ModNone {
		e.popPending(cx, mode)
		return consumed(), nil
	}

	e.keys.path = append(e.keys.path, k)
	e.keys.input = append(e.keys.input, keyInput{})
	lookup, ok := cx.Keymaps.Lookup(mode, e.keys.path)
	if ok && !lookup.Enabled(cx.Editor) {
		ok = false
	}
	switch {
	case ok:
		e.clearCommandMessage()
		e.clearHints()
		res := lookup.Action(cx.Editor)
		e.keys.continuation = res.Continuation
		if res.Continuation == nil {
			e.keys.path = nil
			e.clearInput(cx)
		} else {
			e.keys.frames = nil
			e.setHints(cx, mode)
		}
		e.setCommandResult(res)
		e.syncEditorMessages(cx)
		e.handleReplay(cx)

		return consumed(), signalToCmd(res.Signal)

	case lookup.Prefix:
		if mode == view.ModeInsert && len(e.keys.path) == 1 {
			if e.keys.path[0].IsTypable() {
				if layer := e.insertTypable(
					cx, e.keys.path[0],
				); layer != nil {
					return consumedWith(layer), nil
				}
				return consumed(), nil
			}
		}
		e.setHints(cx, mode)
		return consumed(), nil

	default:
		if mode == view.ModeInsert && len(e.keys.path) == 1 {
			if e.keys.path[0].IsTypable() {
				if layer := e.insertTypable(
					cx, e.keys.path[0],
				); layer != nil {
					return consumedWith(layer), nil
				}
				return consumed(), nil
			}
		}
		e.keys.path = e.keys.path[:len(e.keys.path)-1]
		e.keys.input = e.keys.input[:len(e.keys.input)-1]
		return consumed(), nil
	}
}

// keymapClaims reports whether k continues or completes a binding in the
// focused pane's mode, so a raw-input pane must not swallow it first
func (e *EditorComponent) keymapClaims(cx *Context, k command.KeyEvent) bool {
	if len(e.keys.path) > 0 || e.keys.continuation != nil {
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

func (e *EditorComponent) insertTypable(
	cx *Context, k command.KeyEvent,
) Callback {
	action.InsertChar(cx.Editor, k.Code.Char)
	e.keys.path = nil
	e.clearHints()
	e.clearInput(cx)
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

// popPending drops the last thing typed toward a command
func (e *EditorComponent) popPending(cx *Context, mode view.Mode) {
	last := e.keys.input[len(e.keys.input)-1]
	e.keys.input = e.keys.input[:len(e.keys.input)-1]
	if last.countDigit {
		e.setCount(cx, e.keys.count/10)
	} else {
		e.keys.path = e.keys.path[:len(e.keys.path)-1]
	}
	if len(e.keys.path) == 0 && e.keys.count == 0 {
		e.clearHints()
		return
	}
	e.setHints(cx, mode)
}

// setCount mirrors the count for the renderer, which cannot take it
func (e *EditorComponent) setCount(cx *Context, n int) {
	e.keys.count = n
	cx.Editor.SetCount(n)
}

func (e *EditorComponent) clearInput(cx *Context) {
	e.keys.input = nil
	e.setCount(cx, 0)
}

func (e *EditorComponent) handleContinuation(
	cx *Context, mode view.Mode, k command.KeyEvent,
) {
	next, step := e.keys.continuation(cx.Editor, k)
	switch step {
	case command.ContinuationStay:
		if next != nil {
			e.keys.continuation = next
		}
	case command.ContinuationPush:
		e.pushContinuation(cx, mode, k, next)
	case command.ContinuationPop:
		e.popContinuation(cx, mode)
	default:
		e.keys.continuation = nil
		e.keys.frames = nil
		e.keys.path = nil
		e.clearHints()
		e.clearInput(cx)
	}
}

func (e *EditorComponent) pushContinuation(
	cx *Context, mode view.Mode, k command.KeyEvent,
	next command.Continuation,
) {
	e.keys.frames = append(e.keys.frames, e.keys.continuation)
	e.keys.input = append(e.keys.input, keyInput{})
	e.keys.continuation = next
	e.keys.path = append(e.keys.path, k)
	e.setHints(cx, mode)
}

func (e *EditorComponent) popContinuation(cx *Context, mode view.Mode) {
	if n := len(e.keys.frames); n > 0 {
		frame := e.keys.frames[n-1]
		e.keys.frames = e.keys.frames[:n-1]
		e.keys.input = e.keys.input[:len(e.keys.input)-1]
		e.keys.path = e.keys.path[:len(e.keys.path)-1]
		e.keys.continuation = frame
		e.setHints(cx, mode)
		return
	}
	e.keys.continuation = nil
	e.clearHints()
	e.popPending(cx, mode)
}

func (e *EditorComponent) clearHints() {
	e.keys.infoTitle = ""
	e.keys.infoItems = nil
}

// setHints loads the hints for the node reached by the current key path
func (e *EditorComponent) setHints(cx *Context, mode view.Mode) {
	counting := e.keys.count > 0 && e.keys.continuation == nil
	title := e.keys.infoTitle
	e.keys.infoTitle, e.keys.infoItems = cx.Keymaps.PendingHints(
		cx.Editor, mode, e.keys.path, counting,
	)
	switch {
	case counting && len(e.keys.path) == 0:
		e.keys.infoTitle = i18n.Text(i18n.StatusCounted)
	case e.keys.infoTitle == "" && e.keys.continuation != nil:
		e.keys.infoTitle = title
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

// wordPrefixReady reports whether the limit characters before the cursor are
// all word characters, reading only those so a long line costs no more
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
