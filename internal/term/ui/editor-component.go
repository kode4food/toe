package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	act "github.com/kode4food/toe/internal/view/action"
)

type (
	EditorComponent struct {
		w               int
		h               int
		buf             *tui.Buffer
		pending         []command.KeyEvent
		status          string
		hint            string
		continuation    command.Continuation
		cmdMsg          string
		saveSlot        *saveGenSlot
		cache           *renderCache
		macroSlot       *macroSlot
		nextLayer       layerFunc
		bufferlineShown bool
		focused         bool
		infoTitle       string
		infoItems       []command.KeyHint
		mouseDownRange  *core.Range
		mouseDownSep    *sepDrag
		mouseDownDrag   RawPane
		signatureHidden *signatureCall
		docHighlightGen int
		docHighlightPos docHighlightPosition
		completionGen   int
		fileWatcher     *editorFileWatcher
		autoScrollV     mouseAutoScrollAxis
		autoScrollH     mouseAutoScrollAxis
		spinFrame       int
	}

	saveGenSlot struct{ gen int }

	completionMsg struct {
		gen        int
		anchor     completionAnchor
		items      []view.CompletionItem
		incomplete bool
		err        error
	}

	docHighlightPosition struct {
		docID  view.DocumentId
		viewID view.Id
		rev    int
		pos    int
		ok     bool
	}

	sepDrag struct {
		containerID view.Id
		childIdx    int
		layout      view.Layout
	}

	autoSaveMsg struct{ gen int }

	docHighlightMsg struct{ gen int }

	vcsUpdatedMsg struct{}

	vcsRefreshMsg struct{}

	spinnerTickMsg struct{}
)

const (
	vcsRefreshInterval  = 5 * time.Second
	spinnerTickInterval = 80 * time.Millisecond
)

var _ BufferRenderer = (*EditorComponent)(nil)

func newEditorComponent() *EditorComponent {
	return &EditorComponent{
		saveSlot:    &saveGenSlot{},
		cache:       newRenderCache(),
		macroSlot:   &macroSlot{macros: map[rune][]command.KeyEvent{}},
		focused:     true,
		fileWatcher: newEditorFileWatcher(),
		autoScrollV: mouseAutoScrollAxis{
			scroll: func(e *view.Editor, v *view.View, toLo bool) {
				act.ScrollViewLines(e, v, 1, toLo)
			},
			pos: func(
				r *renderPass, doc *view.Document, v *view.View, fixed int,
				toLo bool,
			) (int, bool) {
				area := v.Area()
				edgeY := area.Y + max(area.Height-1, 0) - 1
				if toLo {
					edgeY = area.Y
				}
				return r.screenCharPos(doc, v, fixed, edgeY)
			},
		},
		autoScrollH: mouseAutoScrollAxis{
			scroll: func(e *view.Editor, v *view.View, toLo bool) {
				act.ScrollViewColumns(e, v, 1, toLo)
			},
			pos: func(
				r *renderPass, doc *view.Document, v *view.View, fixed int,
				toLo bool,
			) (int, bool) {
				area := v.Area()
				gutterW := gutterWidthFor(
					doc.Text(), r.cx.Editor.Options().Gutters,
				)
				edgeX := area.X + area.Width - 1
				if toLo {
					edgeX = area.X + gutterW
				}
				return r.screenCharPos(doc, v, edgeX, fixed)
			},
		},
	}
}

func (e *EditorComponent) HandleEvent(
	msg tea.Msg, cx *Context,
) (EventResult, tea.Cmd) {
	e.syncFileWatcher(cx)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.w = msg.Width
		e.h = msg.Height
		e.resize(cx)
		return consumed(), nil

	case tea.KeyPressMsg:
		result, cmd := e.handleKeyPress(msg, cx)
		if shown := bufferlineVisible(cx); shown != e.bufferlineShown {
			e.bufferlineShown = shown
			e.resize(cx)
		}
		return result, tea.Batch(
			cmd, e.autoSaveCmd(cx), e.documentHighlightCmd(cx),
		)

	case tea.FocusMsg:
		e.focused = true
		return ignored(), e.documentHighlightCmd(cx)

	case tea.BlurMsg:
		e.focused = false
		if cx.Editor.Options().AutoSaveFocusLost {
			cx.Editor.SaveAll(false)
		}
		return ignored(), nil

	case autoSaveMsg:
		if msg.gen == e.saveSlot.gen {
			cx.Editor.SaveAll(false)
		}
		return consumed(), nil

	case docHighlightMsg:
		if msg.gen != e.docHighlightGen {
			e.docHighlightPos = docHighlightPosition{}
			return consumed(), e.documentHighlightCmd(cx)
		}
		return consumed(), nil

	case completionMsg:
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
		return consumedWith(func(comp *Compositor, _ *Context) tea.Cmd {
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

	case externalFileChangedMsg:
		cx.Editor.ProcessExternalFileChange(msg.path)
		e.syncEditorMessages(cx)
		return consumed(), e.fileWatchCmd(cx)

	case terminalPollMsg:
		e.pollTerminals(cx)
		return consumed(), terminalPollCmd()

	case vcsUpdatedMsg:
		for _, doc := range cx.Editor.AllDocuments() {
			doc.MarkDirty()
		}
		return consumed(), vcsUpdateCmd(cx)

	case vcsRefreshMsg:
		if vc := cx.Editor.VersionControl(); vc != nil {
			vc.Refresh()
		}
		return consumed(), vcsRefreshCmd(cx)

	case spinnerTickMsg:
		if ls := cx.Editor.LanguageServerController(); ls != nil && ls.Busy() {
			e.spinFrame++
		} else {
			e.spinFrame = 0
		}
		return consumed(), spinnerTickCmd()

	case tea.MouseClickMsg:
		e.completionGen++
		e.cancelPending(cx)
		e.autoScrollV.stop()
		e.autoScrollH.stop()
		if dc := e.mouseDownDrag; dc != nil {
			dc.CancelDrag()
		}
		e.mouseDownDrag = nil
		if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
			r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
			r.handleMouseClick(msg.X, msg.Y, msg.Mod)
		}
		return consumed(), e.documentHighlightCmd(cx)

	case tea.MouseMotionMsg:
		e.completionGen++
		if dc := e.mouseDownDrag; dc != nil &&
			cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
			return consumed(), dc.ContinueDrag(cx, msg.X, msg.Y)
		}
		var dragCmd tea.Cmd
		if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
			if p, ok := paneAt(cx, msg.X, msg.Y); ok {
				if pi, ok := p.(RawPane); ok {
					if _, handled := pi.HandleMouse(msg, cx); handled {
						return consumed(), nil
					}
				}
			}
			r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
			dragCmd = r.handleMouseDrag(msg.X, msg.Y)
		}
		return consumed(), tea.Batch(dragCmd, e.documentHighlightCmd(cx))

	case mouseAxisScrollMsg:
		if msg.gen != msg.axis.gen {
			return consumed(), nil
		}
		return consumed(), e.continueAxisScroll(cx, msg.axis, msg.toLo)

	case terminalDragScrollMsg:
		return consumed(), msg.dc.DragTick(cx, msg.gen, msg.toTop)

	case tea.MouseReleaseMsg:
		e.completionGen++
		if dc := e.mouseDownDrag; dc != nil {
			e.mouseDownDrag = nil
			cmd := dc.EndDrag(cx, msg.X, msg.Y)
			e.syncEditorMessages(cx)
			return consumed(), tea.Batch(cmd, e.documentHighlightCmd(cx))
		}
		if !cx.Editor.Options().Mouse {
			return consumed(), nil
		}
		if p, ok := paneAt(cx, msg.X, msg.Y); ok {
			if pi, ok := p.(RawPane); ok {
				if _, handled := pi.HandleMouse(msg, cx); handled {
					return consumed(), e.documentHighlightCmd(cx)
				}
			}
		}
		switch msg.Button {
		case tea.MouseLeft:
			e.handleMouseLeftRelease(cx)
		case tea.MouseMiddle:
			if cx.Editor.Options().MiddleClickPaste {
				r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
				r.handleMouseMiddleRelease(msg.X, msg.Y, msg.Mod)
			}
		}
		return consumed(), e.documentHighlightCmd(cx)

	case tea.MouseWheelMsg:
		e.completionGen++
		e.cancelPending(cx)
		if !cx.Editor.Options().Mouse {
			return consumed(), nil
		}
		if p, ok := paneAt(cx, msg.X, msg.Y); ok {
			if pi, ok := p.(RawPane); ok {
				if _, handled := pi.HandleMouse(msg, cx); handled {
					return consumed(), nil
				}
			}
		}
		r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
		v, ok := r.contentViewAt(msg.X, msg.Y)
		if !ok {
			return consumed(), nil
		}
		n := cx.Editor.Options().ScrollLines
		switch msg.Button {
		case tea.MouseWheelLeft, tea.MouseWheelRight:
			left := msg.Button == tea.MouseWheelLeft
			act.ScrollViewColumns(cx.Editor, v, n, left)
		default:
			up := msg.Button == tea.MouseWheelUp
			act.ScrollViewLines(cx.Editor, v, n, up)
		}
		if doc, ok := cx.Editor.Document(v.DocID()); ok {
			v.BeginFreeScroll(doc.Revision(), doc.SelectionFor(v.ID()))
		}
		return consumed(), nil
	}
	return ignored(), nil
}

func (e *EditorComponent) documentHighlightCmd(cx *Context) tea.Cmd {
	if e.mouseDownRange != nil {
		return nil
	}
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
		doc.ClearDocumentHighlights(v.ID())
		return nil
	}
	pos := documentHighlightPositionFor(doc, v)
	if e.docHighlightPos == pos {
		return nil
	}
	e.docHighlightPos = pos
	e.docHighlightGen++
	gen := e.docHighlightGen
	return func() tea.Msg {
		_, _ = ls.DocumentHighlights(doc, v.ID())
		return docHighlightMsg{gen: gen}
	}
}

// Render returns the editor's cell buffer for the compositor to blit overlays
// onto, skipping an ANSI round-trip
func (e *EditorComponent) Render(w, h int, cx *Context) *tui.Buffer {
	if e.buf == nil || e.buf.Width != w || e.buf.Height != h {
		e.buf = tui.NewBuffer(w, h)
	}
	e.syncEditorMessages(cx)
	e.cache.evictClosed(cx.Editor)
	r := &renderPass{ec: e, cx: cx, w: w, h: h}
	r.renderEditorContent(e.buf)
	return e.buf
}

func (e *EditorComponent) Cursor(w, h int, cx *Context) (tea.Cursor, bool) {
	r := &renderPass{ec: e, cx: cx, w: w, h: h}
	return r.editorCursor()
}

// reports the caret position regardless of cursor shape, so caret-anchored
// overlays still work under the normal-mode block cursor that Cursor hides
func (e *EditorComponent) caretScreenPos(cx *Context) (int, int, bool) {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return 0, 0, false
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return 0, 0, false
	}
	opts := cx.Editor.Options()
	text := doc.Text()
	cursor := doc.SelectionFor(v.ID()).Primary().Cursor(text)
	area := v.Area()
	yOff := area.Y
	if bufferlineVisible(cx) {
		yOff++
	}
	visualY, visualX := cursorScreenPos(cursorScreenPosArgs{
		text: text, cursor: cursor,
		gutterW: gutterWidthFor(text, opts.Gutters),
		rowMap:  e.cache.viewRowMaps[v.ID()],
		tabW:    doc.TabWidth(),
		hOff:    v.Offset().HorizontalOffset,
	})
	return area.X + visualX, yOff + visualY, true
}

func (e *EditorComponent) popupAnchorBelowCaret(
	screenH int, cx *Context, fallbackRows int,
) (int, int) {
	if x, y, ok := e.caretScreenPos(cx); ok {
		return x, y + 1
	}
	return 0, max(screenH-fallbackRows-2, 0)
}

func (e *EditorComponent) cancelPending(cx *Context) {
	e.pending = nil
	e.status = ""
	e.infoTitle = ""
	e.infoItems = nil
	e.continuation = nil
	e.hint = ""
	cx.Editor.ResetCount()
}

func (e *EditorComponent) syncEditorMessages(cx *Context) {
	if h := cx.Editor.TakeHint(); h != "" {
		e.hint = h
	}
	if m := cx.Editor.TakeStatusMsg(); m != "" {
		e.cmdMsg = m
	}
}

func (e *EditorComponent) resize(cx *Context) {
	overhead := 1
	if bufferlineVisible(cx) {
		overhead++
	}
	cx.Editor.SetViewHeight(e.h - overhead)
	th := e.h - 1
	if bufferlineVisible(cx) {
		th--
	}
	cx.Editor.ResizeTree(e.w, max(th, 0))
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
		act.YankToPrimaryClipboard(cx.Editor)
	}
}

func (e *EditorComponent) autoSaveCmd(cx *Context) tea.Cmd {
	opts := cx.Editor.Options()
	if !opts.AutoSaveAfterDelay {
		return nil
	}
	e.saveSlot.gen++
	gen := e.saveSlot.gen
	d := time.Duration(opts.AutoSaveDelayTimeout) * time.Millisecond
	return tea.Tick(d, func(time.Time) tea.Msg {
		return autoSaveMsg{gen: gen}
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

func documentHighlightPositionFor(
	doc *view.Document, v *view.View,
) docHighlightPosition {
	sel := doc.SelectionFor(v.ID())
	pos := sel.Primary().Cursor(doc.Text())
	return docHighlightPosition{
		docID:  doc.ID(),
		viewID: v.ID(),
		rev:    doc.Revision(),
		pos:    pos,
		ok:     true,
	}
}
