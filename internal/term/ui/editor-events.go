package ui

import (
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func (e *EditorComponent) handleWindowSize(
	cx *Context, msg tea.WindowSizeMsg,
) (EventResult, tea.Cmd) {
	e.size = geom.Size{Width: msg.Width, Height: msg.Height}
	e.resize(cx)
	return consumed(), nil
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

func (e *EditorComponent) handleDocHighlightMsg(
	cx *Context, msg docHighlightMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != e.docHighlightGen {
		e.docHighlightPos = docHighlightPosition{}
		return consumed(), e.documentHighlightCmd(cx)
	}
	return consumed(), nil
}

func (e *EditorComponent) handleCompletionMsg(
	cx *Context, msg completionMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != e.completionGen {
		return consumed(), nil
	}
	if !completionRequestValid(cx, msg.anchor) {
		return consumed(), nil
	}
	if msg.err != nil {
		cx.Editor.SetStatusMsg(msg.err.Error())
		return consumed(), nil
	}
	if len(msg.items) == 0 {
		return consumed(), nil
	}
	return consumedWith(func(_ *Context, comp *Compositor) tea.Cmd {
		c := &completionComponent{
			ec:         e,
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
	e.syncEditorMessages(cx)
	return consumed(), e.fileWatchCmd(cx)
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

func (e *EditorComponent) handleTerminalPoll(
	cx *Context,
) (EventResult, tea.Cmd) {
	e.pollTerminals(cx)
	return consumed(), terminalPollCmd()
}

func (e *EditorComponent) handleVCSUpdated(cx *Context) (EventResult, tea.Cmd) {
	for _, doc := range cx.Editor.AllDocuments() {
		doc.MarkDirty()
	}
	return consumed(), vcsUpdateCmd(cx)
}

func (e *EditorComponent) handleVCSRefresh(cx *Context) (EventResult, tea.Cmd) {
	if vc := cx.Editor.VersionControl(); vc != nil {
		vc.Refresh()
	}
	return consumed(), vcsRefreshCmd(cx)
}

func (e *EditorComponent) handleSpinnerTick(
	cx *Context,
) (EventResult, tea.Cmd) {
	if ls := cx.Editor.LanguageServerController(); ls != nil && ls.Busy() {
		e.spinFrame++
	} else {
		e.spinFrame = 0
	}
	return consumed(), spinnerTickCmd()
}

func (e *EditorComponent) handleMouseClick(
	cx *Context, msg tea.MouseClickMsg,
) (EventResult, tea.Cmd) {
	e.completionGen++
	e.cancelPending(cx)
	e.autoScrollV.stop()
	e.autoScrollH.stop()
	if dc := e.mouseDownDrag; dc != nil {
		dc.CancelDrag()
	}
	e.mouseDownDrag = nil
	if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
		r := &renderPass{ec: e, cx: cx, size: e.size}
		r.handleMouseClick(geom.Point{X: msg.X, Y: msg.Y}, msg.Mod)
	}
	return consumed(), e.documentHighlightCmd(cx)
}

func (e *EditorComponent) handleMouseMotion(
	cx *Context, msg tea.MouseMotionMsg,
) (EventResult, tea.Cmd) {
	e.completionGen++
	at := geom.Point{X: msg.X, Y: msg.Y}
	if dc := e.mouseDownDrag; dc != nil &&
		cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
		return consumed(), dc.ContinueDrag(cx, at)
	}
	var dragCmd tea.Cmd
	if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
		if dispatchToPaneInput(cx, at, msg) {
			return consumed(), nil
		}
		r := &renderPass{ec: e, cx: cx, size: e.size}
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
	return consumed(), e.continueAxisScroll(cx, msg.axis, msg.toLo)
}

func (e *EditorComponent) handleMouseRelease(
	cx *Context, msg tea.MouseReleaseMsg,
) (EventResult, tea.Cmd) {
	e.completionGen++
	at := geom.Point{X: msg.X, Y: msg.Y}
	if dc := e.mouseDownDrag; dc != nil {
		e.mouseDownDrag = nil
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
			r := &renderPass{ec: e, cx: cx, size: e.size}
			r.handleMouseMiddleRelease(at, msg.Mod)
		}
	}
	return consumed(), e.documentHighlightCmd(cx)
}

func (e *EditorComponent) handleMouseWheel(
	cx *Context, msg tea.MouseWheelMsg,
) (EventResult, tea.Cmd) {
	e.completionGen++
	e.cancelPending(cx)
	if !cx.Editor.Options().Mouse {
		return consumed(), nil
	}
	at := geom.Point{X: msg.X, Y: msg.Y}
	if dispatchToPaneInput(cx, at, msg) {
		return consumed(), nil
	}
	r := &renderPass{ec: e, cx: cx, size: e.size}
	v, ok := r.contentViewAt(at)
	if !ok {
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
	if doc, ok := cx.Editor.Document(v.DocID()); ok {
		v.BeginFreeScroll(doc.Revision(), doc.SelectionFor(v.ID()))
	}
	return consumed(), nil
}

func (e *EditorComponent) handleMouseLeftRelease(cx *Context) {
	e.autoScrollV.stop()
	e.autoScrollH.stop()
	if e.mouseDownSep != nil {
		e.mouseDownSep = nil
		return
	}
	if e.mouseDownRange == nil {
		return
	}
	down := *e.mouseDownRange
	e.mouseDownRange = nil
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return
	}
	cur := doc.SelectionFor(v.ID()).Primary()
	if cur.Anchor != down.Anchor || cur.Head != down.Head {
		action.YankToPrimaryClipboard(cx.Editor)
	}
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

func vcsRefreshCmd(cx *Context) tea.Cmd {
	if cx.Editor.VersionControl() == nil {
		return nil
	}
	return tea.Tick(vcsRefreshInterval, func(time.Time) tea.Msg {
		return vcsRefreshMsg{}
	})
}

func spinnerTickCmd() tea.Cmd {
	return tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
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
