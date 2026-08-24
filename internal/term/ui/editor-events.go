package ui

import (
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func (ec *EditorComponent) handleWindowSize(
	cx *Context, msg tea.WindowSizeMsg,
) (EventResult, tea.Cmd) {
	// hold before resizing, so a drag of the window edge reaches the live panes
	// once, with the size it settles on
	cmd := ec.settlePaneResizeCmd(cx)
	ec.size = geom.Size{Width: msg.Width, Height: msg.Height}
	ec.resize(cx)
	return consumed(), cmd
}

func (ec *EditorComponent) handleKeyPressEvent(
	cx *Context, msg tea.KeyPressMsg,
) (EventResult, tea.Cmd) {
	result, cmd := ec.handleKeyPress(cx, msg)
	if shown := bufferlineVisible(cx); shown != ec.bufferlineShown {
		ec.bufferlineShown = shown
		ec.resize(cx)
	}
	return result, tea.Batch(
		cmd, ec.autoSaveCmd(cx), ec.documentHighlightCmd(cx),
		ec.macroBlinkCmd(),
	)
}

func (ec *EditorComponent) handlePaste(
	cx *Context, msg tea.PasteMsg,
) (EventResult, tea.Cmd) {
	p := cx.Editor.Tree().Get(cx.Editor.Tree().Focus())
	if pp, ok := p.(Pasteable); ok {
		pp.Paste(msg.Content)
		return consumed(), nil
	}
	if cx.Editor.Mode() == view.ModeInsert {
		action.InsertText(cx.Editor, msg.Content)
		return consumed(), nil
	}
	return ignored(), nil
}

func (ec *EditorComponent) handleFocus(cx *Context) (EventResult, tea.Cmd) {
	ec.focused = true
	refreshVCS(cx)
	return ignored(), ec.documentHighlightCmd(cx)
}

func (ec *EditorComponent) handleBlur(cx *Context) (EventResult, tea.Cmd) {
	ec.focused = false
	if cx.Editor.Options().AutoSaveFocusLost {
		cx.Editor.SaveAll(false)
	}
	return ignored(), nil
}

func (ec *EditorComponent) handleAutoSaveMsg(
	cx *Context, msg autoSaveMsg,
) (EventResult, tea.Cmd) {
	if msg.gen == ec.saveSlot.gen {
		cx.Editor.SaveAll(false)
	}
	return consumed(), nil
}

func (ec *EditorComponent) handleAutoCompletionMsg(
	cx *Context, msg autoCompletionMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != ec.language.autoGen {
		return consumed(), nil
	}
	if cx.Editor.Mode() != view.ModeInsert {
		return consumed(), nil
	}
	return consumed(), ec.completionCmd(cx, false)
}

func (ec *EditorComponent) handleDocHighlightMsg(
	cx *Context, msg docHighlightMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != ec.language.highlightGen {
		ec.language.highlightPos = docHighlightPosition{}
		return consumed(), ec.documentHighlightCmd(cx)
	}
	return consumed(), nil
}

func (ec *EditorComponent) handleCompletionMsg(
	cx *Context, msg completionMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != ec.language.completionGen {
		return consumed(), nil
	}
	if !completionRequestValid(cx, msg.anchor) {
		return consumed(), nil
	}
	if msg.err != nil {
		cx.Editor.SetStatusMsg(i18n.ErrorText(msg.err))
		return consumed(), nil
	}
	if len(msg.items) == 0 {
		return consumed(), nil
	}
	return consumedWith(func(_ *Context, comp *Compositor) tea.Cmd {
		c := &completionComponent{
			editor:     ec,
			all:        msg.items,
			items:      msg.items,
			anchor:     msg.anchor,
			incomplete: msg.incomplete,
		}
		c.resetCursor()
		comp.Push(c)
		return nil
	}), nil
}

func (ec *EditorComponent) handleExternalFileChanged(
	cx *Context, msg externalFileChangedMsg,
) (EventResult, tea.Cmd) {
	cx.Editor.ProcessExternalFileChange(msg.path)
	reloadChangedImages(cx.Editor, msg.path)
	refreshVCS(cx)
	ec.syncEditorMessages(cx)
	return consumed(), cx.fileWatcher.nextCmd(cx.Editor)
}

func (ec *EditorComponent) handleRedraw(cx *Context) (EventResult, tea.Cmd) {
	// the frame repaints on its own, so only reap closed terminals and re-arm
	ec.pollTerminals(cx)
	cmd := ec.redrawCmd()
	if ls := cx.Editor.LanguageServerController(); ls != nil && ls.Busy() {
		if !ec.spinner.active {
			cmd = tea.Batch(cmd, spinnerTickCmd(ec.spinner.start()))
		}
	} else if ec.spinner.active {
		ec.spinner.stop()
	}
	// a toast pushed off the UI goroutine arrives here, not through a key
	return consumed(), tea.Batch(cmd, ec.toastTickCmd())
}

func (ec *EditorComponent) requestRedraw() {
	select {
	case ec.redraw <- struct{}{}:
	default:
	}
}

func (ec *EditorComponent) redrawCmd() tea.Cmd {
	return func() tea.Msg {
		<-ec.redraw
		return redrawMsg{}
	}
}

func (ec *EditorComponent) handleVCSUpdated(cx *Context) (EventResult, tea.Cmd) {
	for _, doc := range cx.Editor.AllDocuments() {
		doc.MarkDirty()
	}
	return consumed(), vcsUpdateCmd(cx)
}

func (ec *EditorComponent) handleSpinnerTick(
	cx *Context, msg spinnerTickMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != ec.spinner.gen || !ec.spinner.active {
		return consumed(), nil
	}
	if ls := cx.Editor.LanguageServerController(); ls != nil && ls.Busy() {
		ec.spinner.phase++
		return consumed(), spinnerTickCmd(msg.gen)
	}
	ec.spinner.stop()
	return consumed(), nil
}

func (ec *EditorComponent) handleMacroBlinkTick(
	msg macroBlinkTickMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != ec.macroBlink.gen || !ec.macroBlink.active {
		return consumed(), nil
	}
	if !ec.macroSlot.recording || !ec.animation {
		ec.macroBlink.stop()
		return consumed(), nil
	}
	ec.macroBlink.phase++
	return consumed(), macroBlinkTickCmd(msg.gen)
}

func (ec *EditorComponent) macroBlinkCmd() tea.Cmd {
	active := ec.macroSlot.recording && ec.animation
	if active && !ec.macroBlink.active {
		return macroBlinkTickCmd(ec.macroBlink.start())
	}
	if !active && ec.macroBlink.active {
		ec.macroBlink.stop()
	}
	return nil
}

func (ec *EditorComponent) handleMouseClick(
	cx *Context, msg tea.MouseClickMsg,
) (EventResult, tea.Cmd) {
	ec.language.completionGen++
	at := geom.Point{X: msg.X, Y: msg.Y}
	if ec.toasts.dismissAt(at) {
		ec.requestRedraw()
		return consumed(), nil
	}
	if len(ec.keys.input) > 0 {
		if ec.cache.infoBounds.Contains(at) {
			return consumed(), nil
		}
		ec.cancelPending(cx)
	}
	ec.mouse.vertical.stop()
	ec.mouse.horizontal.stop()
	if dc := ec.mouse.downDrag; dc != nil {
		dc.CancelDrag()
	}
	ec.mouse.downDrag = nil
	if cx.Editor.Options().Mouse {
		r := &renderPass{editor: ec, context: cx, size: ec.size}
		r.handleMouseClick(msg)
	}
	return consumed(), ec.documentHighlightCmd(cx)
}

func (ec *EditorComponent) handleMouseMotion(
	cx *Context, msg tea.MouseMotionMsg,
) (EventResult, tea.Cmd) {
	ec.language.completionGen++
	at := geom.Point{X: msg.X, Y: msg.Y}
	if dc := ec.mouse.downDrag; dc != nil &&
		cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
		return consumed(), dc.ContinueDrag(cx, at)
	}
	var dragCmd tea.Cmd
	if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
		if dispatchToPaneInput(cx, at, msg) {
			return consumed(), nil
		}
		r := &renderPass{editor: ec, context: cx, size: ec.size}
		dragCmd = r.handleMouseDrag(at)
	}
	return consumed(), tea.Batch(dragCmd, ec.documentHighlightCmd(cx))
}

func (ec *EditorComponent) handleMouseAxisScroll(
	cx *Context, msg mouseAxisScrollMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != msg.axis.gen {
		return consumed(), nil
	}
	return consumed(), ec.continueAxisScroll(cx, msg.axis, msg.toLow)
}

func (ec *EditorComponent) handleMouseRelease(
	cx *Context, msg tea.MouseReleaseMsg,
) (EventResult, tea.Cmd) {
	ec.language.completionGen++
	at := geom.Point{X: msg.X, Y: msg.Y}
	if dc := ec.mouse.downDrag; dc != nil {
		ec.mouse.downDrag = nil
		cmd := dc.EndDrag(cx, at)
		ec.syncEditorMessages(cx)
		return consumed(), tea.Batch(cmd, ec.documentHighlightCmd(cx))
	}
	if !cx.Editor.Options().Mouse {
		return consumed(), nil
	}
	if dispatchToPaneInput(cx, at, msg) {
		return consumed(), ec.documentHighlightCmd(cx)
	}
	if msg.Button == tea.MouseLeft {
		ec.handleMouseLeftRelease(cx)
	}
	return consumed(), ec.documentHighlightCmd(cx)
}

func (ec *EditorComponent) handleMouseWheel(
	cx *Context, msg tea.MouseWheelMsg,
) (EventResult, tea.Cmd) {
	ec.language.completionGen++
	if len(ec.keys.input) > 0 {
		return consumed(), nil
	}
	if !cx.Editor.Options().Mouse {
		return consumed(), nil
	}
	at := geom.Point{X: msg.X, Y: msg.Y}
	if dispatchToPaneInput(cx, at, msg) {
		return consumed(), nil
	}
	r := &renderPass{editor: ec, context: cx, size: ec.size}
	v := r.contentViewAt(at)
	if v == nil {
		return consumed(), nil
	}
	n := cx.Editor.Options().ScrollLines
	switch msg.Button {
	case tea.MouseWheelLeft, tea.MouseWheelRight:
		left := msg.Button == tea.MouseWheelLeft
		action.ScrollViewColumns(cx.Editor, v, n, left)
	default:
		up := msg.Button == tea.MouseWheelUp
		action.ScrollViewLines(cx.Editor, v, n, up)
	}
	if doc := cx.Editor.Document(v.DocID()); doc != nil {
		v.BeginFreeScroll(doc.Revision(), doc.SelectionFor(v.ID()))
	}
	return consumed(), nil
}

func (ec *EditorComponent) handleMouseLeftRelease(cx *Context) {
	ec.mouse.vertical.stop()
	ec.mouse.horizontal.stop()
	if ec.mouse.downSep != nil {
		ec.mouse.downSep = nil
		return
	}
	if ec.mouse.downRange == nil {
		return
	}
	down := *ec.mouse.downRange
	ec.mouse.downRange = nil
	doc := cx.Editor.FocusedDocument()
	if doc == nil {
		return
	}
	v := cx.Editor.FocusedView()
	if v == nil {
		return
	}
	cur := doc.SelectionFor(v.ID()).Primary()
	if cur.IsSingleGrapheme(doc.Text()) || cur.IsEmpty() {
		return
	}
	if cur.Anchor != down.Anchor || cur.Head != down.Head {
		action.YankToClipboard(cx.Editor)
		action.YankToPrimaryClipboard(cx.Editor)
	}
}

func reloadChangedImages(e *view.Editor, path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	rangeImagePanes(e, func(img *ImagePane) {
		if img.Path() == abs {
			_ = img.Reload()
		}
	})
}

func vcsUpdateCmd(cx *Context) tea.Cmd {
	return func() tea.Msg {
		vc := cx.Editor.VersionControl()
		if vc == nil {
			return nil
		}
		<-vc.Updates()
		return vcsUpdatedMsg{}
	}
}

func refreshVCS(cx *Context) {
	if vc := cx.Editor.VersionControl(); vc != nil {
		vc.Refresh()
	}
}

func spinnerTickCmd(gen int) tea.Cmd {
	return tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg {
		return spinnerTickMsg{gen: gen}
	})
}

func macroBlinkTickCmd(gen int) tea.Cmd {
	return tea.Tick(macroBlinkTickInterval, func(time.Time) tea.Msg {
		return macroBlinkTickMsg{gen: gen}
	})
}

func dispatchToPaneInput(cx *Context, at geom.Point, msg tea.Msg) bool {
	p, ok := paneAt(cx, at)
	if !ok {
		return false
	}
	pi, ok := p.(PaneInput)
	if !ok {
		return false
	}
	_, handled := pi.HandleEvent(cx, msg)
	return handled
}
