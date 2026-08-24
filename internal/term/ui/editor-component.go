package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type (
	// EditorComponent renders the pane tree and drives the editor's key, mouse,
	// and animation handling for one frame at a time
	EditorComponent struct {
		keys       keyState
		mouse      mouseState
		language   languageState
		spinner    animationState
		macroBlink animationState
		autoSize   autoSizeState
		resizeHold resizeHoldState
		toasts     toastState

		size  geom.Size
		buf   *tui.Buffer
		cache *renderCache

		completion CompletionOptions
		saveSlot   *saveGenSlot
		macroSlot  *macroSlot

		bufferlineShown bool
		focused         bool
		animation       bool

		redraw chan struct{}
	}

	keyState struct {
		path         []command.KeyEvent
		input        []keyInput
		frames       []command.Continuation
		count        int
		continuation command.Continuation
		nextLayer    layerFunc
		infoTitle    string
		infoItems    []command.KeyHint
	}

	keyInput struct {
		countDigit bool
	}

	mouseState struct {
		downRange  *core.Range
		downSep    *sepDrag
		downDrag   Draggable
		vertical   mouseAutoScrollAxis
		horizontal mouseAutoScrollAxis
	}

	languageState struct {
		signatureHidden *signatureCall
		highlightGen    int
		highlightPos    docHighlightPosition
		completionGen   int
		autoGen         int
	}

	animationState struct {
		phase  int
		gen    int
		active bool
	}

	saveGenSlot struct{ gen int }

	completionMsg struct {
		gen        int
		anchor     completionAnchor
		items      []*view.CompletionItem
		incomplete bool
		err        error
	}

	docHighlightPosition struct {
		docID  view.DocumentId
		viewID view.Id
		rev    int
		pos    int
		valid  bool
	}

	sepDrag struct {
		containerID view.Id
		childIdx    int
		layout      view.Layout
	}

	autoSaveMsg struct{ gen int }

	autoCompletionMsg struct{ gen int }

	docHighlightMsg struct{ gen int }

	vcsUpdatedMsg struct{}

	spinnerTickMsg struct{ gen int }

	macroBlinkTickMsg struct{ gen int }

	toastTickMsg struct{ gen int }

	redrawMsg struct{}
)

const (
	spinnerTickInterval    = 80 * time.Millisecond
	macroBlinkTickInterval = 600 * time.Millisecond
)

var (
	_ BufferRenderer     = (*EditorComponent)(nil)
	_ highlightRefresher = (*EditorComponent)(nil)
)

func newEditorComponent() *EditorComponent {
	return &EditorComponent{
		saveSlot:   &saveGenSlot{},
		completion: DefaultCompletionOptions(),
		cache:      newRenderCache(),
		macroSlot:  &macroSlot{},
		focused:    true,
		animation:  true,
		redraw:     make(chan struct{}, 1),
		mouse: mouseState{
			vertical: mouseAutoScrollAxis{
				scroll: func(e *view.Editor, v *view.View, toLow bool) {
					action.ScrollViewLines(e, v, 1, toLow)
				},
				pos: func(
					r *renderPass, doc *view.Document, v *view.View, fixed int,
					toLow bool,
				) (int, bool) {
					area := v.Area()
					edgeY := area.Y + max(area.Height-1, 0) - 1
					if toLow {
						edgeY = area.Y
					}
					return r.screenCharPos(doc, v, geom.Point{
						X: fixed, Y: edgeY,
					})
				},
			},
			horizontal: mouseAutoScrollAxis{
				scroll: func(e *view.Editor, v *view.View, toLow bool) {
					action.ScrollViewColumns(e, v, 1, toLow)
				},
				pos: func(
					r *renderPass, doc *view.Document, v *view.View, fixed int,
					toLow bool,
				) (int, bool) {
					area := v.Area()
					gutterW := gutterWidthFor(
						doc.Text(), r.context.Editor.Options().Gutters,
					)
					edgeX := area.Right()
					if toLow {
						edgeX = area.X + gutterW
					}
					return r.screenCharPos(doc, v, geom.Point{
						X: edgeX, Y: fixed,
					})
				},
			},
		},
	}
}

// Animation reports whether UI animations (such as auto-size growth) play
func (m Model) Animation() bool {
	return m.component.animation
}

// SetAnimation controls whether UI animations play. When off they snap to
// their final state
func (m Model) SetAnimation(enabled bool) {
	m.component.animation = enabled
	if !enabled {
		m.component.toasts.snap()
	}
}

// HandleEvent routes keys, mouse, and editor messages to the panes
func (ec *EditorComponent) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	cx.fileWatcher.sync(cx.Editor)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return ec.handleWindowSize(cx, msg)
	case tea.KeyPressMsg:
		return ec.handleKeyPressEvent(cx, msg)
	case tea.PasteMsg:
		return ec.handlePaste(cx, msg)
	case tea.FocusMsg:
		return ec.handleFocus(cx)
	case tea.BlurMsg:
		return ec.handleBlur(cx)
	case autoSaveMsg:
		return ec.handleAutoSaveMsg(cx, msg)
	case autoCompletionMsg:
		return ec.handleAutoCompletionMsg(cx, msg)
	case docHighlightMsg:
		return ec.handleDocHighlightMsg(cx, msg)
	case completionMsg:
		return ec.handleCompletionMsg(cx, msg)
	case externalFileChangedMsg:
		return ec.handleExternalFileChanged(cx, msg)
	case redrawMsg:
		return ec.handleRedraw(cx)
	case vcsUpdatedMsg:
		return ec.handleVCSUpdated(cx)
	case spinnerTickMsg:
		return ec.handleSpinnerTick(cx, msg)
	case macroBlinkTickMsg:
		return ec.handleMacroBlinkTick(msg)
	case toastTickMsg:
		return ec.handleToastTick(msg)
	case autoSizeTickMsg:
		return ec.handleAutoSizeTick(cx, msg)
	case resizeSettleMsg:
		return ec.handleResizeSettle(msg)
	case tea.MouseClickMsg:
		return ec.handleMouseClick(cx, msg)
	case tea.MouseMotionMsg:
		return ec.handleMouseMotion(cx, msg)
	case mouseAxisScrollMsg:
		return ec.handleMouseAxisScroll(cx, msg)
	case terminalDragScrollMsg:
		return consumed(), msg.draggable.DragTick(cx, msg.gen, msg.toLow)
	case tea.MouseReleaseMsg:
		return ec.handleMouseRelease(cx, msg)
	case tea.MouseWheelMsg:
		return ec.handleMouseWheel(cx, msg)
	}
	return ignored(), nil
}

// Render returns the editor's cell buffer for the compositor to blit overlays
// onto, skipping an ANSI round-trip
func (ec *EditorComponent) Render(cx *Context, screen geom.Size) *tui.Buffer {
	if ec.buf == nil || ec.buf.Size != screen {
		ec.buf = tui.NewBuffer(screen)
	}
	ec.syncEditorMessages(cx)
	ec.cache.evictClosed(cx.Editor)
	r := &renderPass{editor: ec, context: cx, size: screen}
	r.renderEditorContent(ec.buf)
	return ec.buf
}

// Cursor returns the focused pane's cursor position and shape
func (ec *EditorComponent) Cursor(
	cx *Context, screen geom.Size,
) (tea.Cursor, bool) {
	r := &renderPass{editor: ec, context: cx, size: screen}
	return r.editorCursor()
}

func (ec *EditorComponent) queueNextLayer(next layerFunc) {
	ec.keys.nextLayer = next
}

func (ec *EditorComponent) takeNextLayer() layerFunc {
	next := ec.keys.nextLayer
	ec.keys.nextLayer = nil
	return next
}

func (ec *EditorComponent) documentHighlightCmd(cx *Context) tea.Cmd {
	if ec.mouse.downRange != nil {
		return nil
	}
	doc := cx.Editor.FocusedDocument()
	if doc == nil {
		return nil
	}
	v := cx.Editor.FocusedView()
	if v == nil {
		return nil
	}
	ls := cx.Editor.LanguageServerController()
	if ls == nil || cx.Editor.Mode() != view.ModeNormal {
		doc.ClearDocumentHighlights(v.ID())
		return nil
	}
	pos := documentHighlightPositionFor(doc, v)
	if ec.language.highlightPos == pos {
		return nil
	}
	ec.language.highlightPos = pos
	ec.language.highlightGen++
	gen := ec.language.highlightGen
	return func() tea.Msg {
		_, _ = ls.DocumentHighlights(doc, v.ID())
		return docHighlightMsg{gen: gen}
	}
}

func (ec *EditorComponent) caretScreenPos(cx *Context) (geom.Point, bool) {
	doc := cx.Editor.FocusedDocument()
	if doc == nil {
		return geom.Point{}, false
	}
	v := cx.Editor.FocusedView()
	if v == nil {
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
		text:        text,
		cursor:      cursor,
		gutterWidth: gutterWidthFor(text, opts.Gutters),
		rowMap:      ec.cache.viewRowMaps[v.ID()],
		tabWidth:    doc.TabWidth(),
		horzOff:     v.Offset().HorizontalOffset,
	})
	return geom.Point{X: area.X + visual.X, Y: yOff + visual.Y}, true
}

type popupAnchorArgs struct {
	screenHeight int
	fallbackRows int
}

func (ec *EditorComponent) popupAnchorBelowCaret(
	cx *Context, args popupAnchorArgs,
) geom.Point {
	if at, ok := ec.caretScreenPos(cx); ok {
		at.Y++
		return at
	}
	return geom.Point{Y: max(args.screenHeight-args.fallbackRows-2, 0)}
}

func (ec *EditorComponent) cancelPending(cx *Context) {
	ec.keys.path = nil
	ec.clearHints()
	ec.keys.continuation = nil
	ec.keys.frames = nil
	ec.clearInput(cx)
}

func (ec *EditorComponent) syncEditorMessages(cx *Context) {
	for _, m := range cx.Editor.TakeStatusMsgs() {
		if m != "" {
			ec.setStatusMessage(m)
		}
	}
	for _, m := range ec.toasts.takeLog() {
		cx.Editor.AppendMessage(m)
	}
}

func (ec *EditorComponent) setCommandResult(res command.Result) {
	if res.Error != nil {
		ec.setCommandError(res.Error)
		return
	}
	if res.Message != "" {
		ec.setCommandMessage(res.Message)
	}
}

func (ec *EditorComponent) setCommandError(err error) {
	ec.pushToast(i18n.ErrorText(err), toastError)
}

func (ec *EditorComponent) setCommandMessage(msg string) {
	ec.pushToast(msg, toastCommand)
}

func (ec *EditorComponent) setStatusMessage(msg string) {
	ec.pushToast(msg, toastInfo)
}

func (ec *EditorComponent) clearCommandMessage() {
	if ec.toasts.close(time.Now(), ec.animation) {
		ec.requestRedraw()
	}
}

func (ec *EditorComponent) resize(cx *Context) {
	overhead := 0
	if bufferlineVisible(cx) {
		overhead++
	}
	cx.Editor.SetViewHeight(ec.size.Height - overhead)
	cx.Editor.ResizeTree(geom.Size{
		Width:  ec.size.Width,
		Height: max(ec.size.Height-overhead, 0),
	})
}

func (ec *EditorComponent) autoSaveCmd(cx *Context) tea.Cmd {
	opts := cx.Editor.Options()
	if !opts.AutoSaveAfterDelay {
		return nil
	}
	ec.saveSlot.gen++
	gen := ec.saveSlot.gen
	d := time.Duration(opts.AutoSaveDelayTimeout) * time.Millisecond
	return tea.Tick(d, func(time.Time) tea.Msg {
		return autoSaveMsg{gen: gen}
	})
}

func (a *animationState) start() int {
	a.phase = 0
	a.active = true
	a.gen++
	return a.gen
}

func (a *animationState) stop() {
	a.phase = 0
	a.active = false
	a.gen++
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
		valid:  true,
	}
}
