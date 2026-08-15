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

func (e *EditorComponent) handleWindowSize(
	cx *Context, msg tea.WindowSizeMsg,
) (EventResult, tea.Cmd) {
	// hold before resizing, so a drag of the window edge reaches the live panes
	// once, with the size it settles on
	cmd := e.settlePaneResizeCmd(cx)
	e.size = geom.Size{Width: msg.Width, Height: msg.Height}
	e.resize(cx)
	return consumed(), cmd
}

func (e *EditorComponent) handleKeyPressEvent(
	cx *Context, msg tea.KeyPressMsg,
) (EventResult, tea.Cmd) {
	result, cmd := e.handleKeyPress(cx, msg)
	if shown := bufferlineVisible(cx); shown != e.bufferlineShown {
		e.bufferlineShown = shown
		e.resize(cx)
	}
	return result, tea.Batch(
		cmd, e.autoSaveCmd(cx), e.documentHighlightCmd(cx),
		e.macroBlinkCmd(),
	)
}

func (e *EditorComponent) handleFocus(cx *Context) (EventResult, tea.Cmd) {
	e.focused = true
	return ignored(), e.documentHighlightCmd(cx)
}

func (e *EditorComponent) handleBlur(cx *Context) (EventResult, tea.Cmd) {
	e.focused = false
	if cx.Editor.Options().AutoSaveFocusLost {
		cx.Editor.SaveAll(false)
	}
	return ignored(), nil
}

func (e *EditorComponent) handleAutoSaveMsg(
	cx *Context, msg autoSaveMsg,
) (EventResult, tea.Cmd) {
	if msg.gen == e.saveSlot.gen {
		cx.Editor.SaveAll(false)
	}
	return consumed(), nil
}

func (e *EditorComponent) handleAutoCompletionMsg(
	cx *Context, msg autoCompletionMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != e.language.autoGen {
		return consumed(), nil
	}
	if cx.Editor.Mode() != view.ModeInsert {
		return consumed(), nil
	}
	return consumed(), e.completionCmd(cx, false)
}

func (e *EditorComponent) handleDocHighlightMsg(
	cx *Context, msg docHighlightMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != e.language.highlightGen {
		e.language.highlightPos = docHighlightPosition{}
		return consumed(), e.documentHighlightCmd(cx)
	}
	return consumed(), nil
}

func (e *EditorComponent) handleCompletionMsg(
	cx *Context, msg completionMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != e.language.completionGen {
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
			editor:     e,
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

func (e *EditorComponent) handleExternalFileChanged(
	cx *Context, msg externalFileChangedMsg,
) (EventResult, tea.Cmd) {
	cx.Editor.ProcessExternalFileChange(msg.path)
	reloadChangedImages(cx.Editor, msg.path)
	refreshVCS(cx)
	e.syncEditorMessages(cx)
	return consumed(), cx.fileWatcher.nextCmd(cx.Editor)
}

func (e *EditorComponent) handleRedraw(cx *Context) (EventResult, tea.Cmd) {
	// the frame repaints on its own; we only reap closed terminals, then re-arm
	e.pollTerminals(cx)
	cmd := e.redrawCmd()
	if ls := cx.Editor.LanguageServerController(); ls != nil && ls.Busy() {
		if !e.spinner.active {
			cmd = tea.Batch(cmd, spinnerTickCmd(e.spinner.start()))
		}
	} else if e.spinner.active {
		e.spinner.stop()
	}
	return consumed(), cmd
}

// the hook handed to panes that mutate off the event loop; a pane calls it to
// signal a change, leaving the frame policy to the render loop
func (e *EditorComponent) requestRedraw() {
	select {
	case e.redraw <- struct{}{}:
	default:
	}
}

func (e *EditorComponent) redrawCmd() tea.Cmd {
	return func() tea.Msg {
		<-e.redraw
		return redrawMsg{}
	}
}

func (e *EditorComponent) handleVCSUpdated(cx *Context) (EventResult, tea.Cmd) {
	for _, doc := range cx.Editor.AllDocuments() {
		doc.MarkDirty()
	}
	return consumed(), vcsUpdateCmd(cx)
}

func (e *EditorComponent) handleSpinnerTick(
	cx *Context, msg spinnerTickMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != e.spinner.gen || !e.spinner.active {
		return consumed(), nil
	}
	if ls := cx.Editor.LanguageServerController(); ls != nil && ls.Busy() {
		e.spinner.phase++
		return consumed(), spinnerTickCmd(msg.gen)
	}
	e.spinner.stop()
	return consumed(), nil
}

func (e *EditorComponent) handleMacroBlinkTick(
	msg macroBlinkTickMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != e.macroBlink.gen || !e.macroBlink.active {
		return consumed(), nil
	}
	if !e.macroSlot.recording || !e.animation {
		e.macroBlink.stop()
		return consumed(), nil
	}
	e.macroBlink.phase++
	return consumed(), macroBlinkTickCmd(msg.gen)
}

func (e *EditorComponent) macroBlinkCmd() tea.Cmd {
	active := e.macroSlot.recording && e.animation
	if active && !e.macroBlink.active {
		return macroBlinkTickCmd(e.macroBlink.start())
	}
	if !active && e.macroBlink.active {
		e.macroBlink.stop()
	}
	return nil
}

func (e *EditorComponent) handleMouseClick(
	cx *Context, msg tea.MouseClickMsg,
) (EventResult, tea.Cmd) {
	e.language.completionGen++
	at := geom.Point{X: msg.X, Y: msg.Y}
	if len(e.keys.input) > 0 {
		if e.cache.infoBounds.Contains(at) {
			return consumed(), nil
		}
		e.cancelPending(cx)
	}
	e.mouse.vertical.stop()
	e.mouse.horizontal.stop()
	if dc := e.mouse.downDrag; dc != nil {
		dc.CancelDrag()
	}
	e.mouse.downDrag = nil
	if cx.Editor.Options().Mouse {
		r := &renderPass{editor: e, context: cx, size: e.size}
		r.handleMouseClick(msg)
	}
	return consumed(), e.documentHighlightCmd(cx)
}

func (e *EditorComponent) handleMouseMotion(
	cx *Context, msg tea.MouseMotionMsg,
) (EventResult, tea.Cmd) {
	e.language.completionGen++
	at := geom.Point{X: msg.X, Y: msg.Y}
	if dc := e.mouse.downDrag; dc != nil &&
		cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
		return consumed(), dc.ContinueDrag(cx, at)
	}
	var dragCmd tea.Cmd
	if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
		if dispatchToPaneInput(cx, at, msg) {
			return consumed(), nil
		}
		r := &renderPass{editor: e, context: cx, size: e.size}
		dragCmd = r.handleMouseDrag(at)
	}
	return consumed(), tea.Batch(dragCmd, e.documentHighlightCmd(cx))
}

func (e *EditorComponent) handleMouseAxisScroll(
	cx *Context, msg mouseAxisScrollMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != msg.axis.gen {
		return consumed(), nil
	}
	return consumed(), e.continueAxisScroll(cx, msg.axis, msg.toLow)
}

func (e *EditorComponent) handleMouseRelease(
	cx *Context, msg tea.MouseReleaseMsg,
) (EventResult, tea.Cmd) {
	e.language.completionGen++
	at := geom.Point{X: msg.X, Y: msg.Y}
	if dc := e.mouse.downDrag; dc != nil {
		e.mouse.downDrag = nil
		cmd := dc.EndDrag(cx, at)
		e.syncEditorMessages(cx)
		return consumed(), tea.Batch(cmd, e.documentHighlightCmd(cx))
	}
	if !cx.Editor.Options().Mouse {
		return consumed(), nil
	}
	if dispatchToPaneInput(cx, at, msg) {
		return consumed(), e.documentHighlightCmd(cx)
	}
	switch msg.Button {
	case tea.MouseLeft:
		e.handleMouseLeftRelease(cx)
	case tea.MouseMiddle:
		if cx.Editor.Options().MiddleClickPaste {
			r := &renderPass{editor: e, context: cx, size: e.size}
			r.handleMouseMiddleRelease(at, msg.Mod)
		}
	}
	return consumed(), e.documentHighlightCmd(cx)
}

func (e *EditorComponent) handleMouseWheel(
	cx *Context, msg tea.MouseWheelMsg,
) (EventResult, tea.Cmd) {
	e.language.completionGen++
	if len(e.keys.input) > 0 {
		return consumed(), nil
	}
	if !cx.Editor.Options().Mouse {
		return consumed(), nil
	}
	at := geom.Point{X: msg.X, Y: msg.Y}
	if dispatchToPaneInput(cx, at, msg) {
		return consumed(), nil
	}
	r := &renderPass{editor: e, context: cx, size: e.size}
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

func (e *EditorComponent) handleMouseLeftRelease(cx *Context) {
	e.mouse.vertical.stop()
	e.mouse.horizontal.stop()
	if e.mouse.downSep != nil {
		e.mouse.downSep = nil
		return
	}
	if e.mouse.downRange == nil {
		return
	}
	down := *e.mouse.downRange
	e.mouse.downRange = nil
	doc := cx.Editor.FocusedDocument()
	if doc == nil {
		return
	}
	v := cx.Editor.FocusedView()
	if v == nil {
		return
	}
	cur := doc.SelectionFor(v.ID()).Primary()
	if cur.IsSingleGrapheme(doc.Text()) || cur.Empty() {
		return
	}
	if cur.Anchor != down.Anchor || cur.Head != down.Head {
		action.YankToClipboard(cx.Editor)
		action.YankToPrimaryClipboard(cx.Editor)
	}
}

// reloadChangedImages re-decodes any image pane whose file matches path
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
	vc := cx.Editor.VersionControl()
	if vc == nil {
		return nil
	}
	updates := vc.Updates()
	return func() tea.Msg {
		<-updates
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

// flattens the pane-lookup + type-assert + dispatch chain shared by the
// mouse motion/release/wheel handlers
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
