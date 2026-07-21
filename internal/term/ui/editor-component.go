package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type (
	EditorComponent struct {
		size            geom.Size
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
		mouseDownDrag   Draggable
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
				action.ScrollViewLines(e, v, 1, toLo)
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
				return r.screenCharPos(doc, v, geom.Point{X: fixed, Y: edgeY})
			},
		},
		autoScrollH: mouseAutoScrollAxis{
			scroll: func(e *view.Editor, v *view.View, toLo bool) {
				action.ScrollViewColumns(e, v, 1, toLo)
			},
			pos: func(
				r *renderPass, doc *view.Document, v *view.View, fixed int,
				toLo bool,
			) (int, bool) {
				area := v.Area()
				gutterW := gutterWidthFor(
					doc.Text(), r.cx.Editor.Options().Gutters,
				)
				edgeX := area.Right()
				if toLo {
					edgeX = area.X + gutterW
				}
				return r.screenCharPos(doc, v, geom.Point{X: edgeX, Y: fixed})
			},
		},
	}
}

func (e *EditorComponent) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	e.syncFileWatcher(cx)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return e.handleWindowSize(cx, msg)
	case tea.KeyPressMsg:
		return e.handleKeyPressEvent(cx, msg)
	case tea.FocusMsg:
		return e.handleFocus(cx)
	case tea.BlurMsg:
		return e.handleBlur(cx)
	case autoSaveMsg:
		return e.handleAutoSaveMsg(cx, msg)
	case docHighlightMsg:
		return e.handleDocHighlightMsg(cx, msg)
	case completionMsg:
		return e.handleCompletionMsg(cx, msg)
	case externalFileChangedMsg:
		return e.handleExternalFileChanged(cx, msg)
	case terminalPollMsg:
		return e.handleTerminalPoll(cx)
	case vcsUpdatedMsg:
		return e.handleVCSUpdated(cx)
	case vcsRefreshMsg:
		return e.handleVCSRefresh(cx)
	case spinnerTickMsg:
		return e.handleSpinnerTick(cx)
	case tea.MouseClickMsg:
		return e.handleMouseClick(cx, msg)
	case tea.MouseMotionMsg:
		return e.handleMouseMotion(cx, msg)
	case mouseAxisScrollMsg:
		return e.handleMouseAxisScroll(cx, msg)
	case terminalDragScrollMsg:
		return consumed(), msg.dc.DragTick(cx, msg.gen, msg.toTop)
	case tea.MouseReleaseMsg:
		return e.handleMouseRelease(cx, msg)
	case tea.MouseWheelMsg:
		return e.handleMouseWheel(cx, msg)
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
func (e *EditorComponent) Render(cx *Context, screen geom.Size) *tui.Buffer {
	if e.buf == nil || e.buf.Size != screen {
		e.buf = tui.NewBuffer(screen)
	}
	e.syncEditorMessages(cx)
	e.cache.evictClosed(cx.Editor)
	r := &renderPass{ec: e, cx: cx, size: screen}
	r.renderEditorContent(e.buf)
	return e.buf
}

func (e *EditorComponent) Cursor(
	cx *Context, screen geom.Size,
) (tea.Cursor, bool) {
	r := &renderPass{ec: e, cx: cx, size: screen}
	return r.editorCursor()
}

// reports the caret position regardless of cursor shape, so caret-anchored
// overlays still work under the normal-mode block cursor that Cursor hides
func (e *EditorComponent) caretScreenPos(cx *Context) (geom.Point, bool) {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return geom.Point{}, false
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return geom.Point{}, false
	}
	opts := cx.Editor.Options()
	text := doc.Text()
	cursor := doc.SelectionFor(v.ID()).Primary().Cursor(text)
	area := v.Area()
	yOff := area.Y
	if bufferlineVisible(cx) {
		yOff++
	}
	visual := cursorScreenPos(cursorScreenPosArgs{
		text:             text,
		cursor:           cursor,
		gutterWidth:      gutterWidthFor(text, opts.Gutters),
		rowMap:           e.cache.viewRowMaps[v.ID()],
		tabWidth:         doc.TabWidth(),
		horizontalOffset: v.Offset().HorizontalOffset,
	})
	return geom.Point{X: area.X + visual.X, Y: yOff + visual.Y}, true
}

func (e *EditorComponent) popupAnchorBelowCaret(
	cx *Context, screenH, fallbackRows int,
) geom.Point {
	if at, ok := e.caretScreenPos(cx); ok {
		at.Y++
		return at
	}
	return geom.Point{Y: max(screenH-fallbackRows-2, 0)}
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
	cx.Editor.SetViewHeight(e.size.Height - overhead)
	th := e.size.Height - 1
	if bufferlineVisible(cx) {
		th--
	}
	cx.Editor.ResizeTree(geom.Size{
		Width: e.size.Width, Height: max(th, 0),
	})
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
