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
	EditorComponent struct {
		keys     keyState
		mouse    mouseState
		language languageState
		spinner  spinnerState
		autoSize autoSizeState

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
		pending      []command.KeyEvent
		status       string
		hint         string
		continuation command.Continuation
		message      *commandMessage
		nextLayer    layerFunc
		infoTitle    string
		infoItems    []command.KeyHint
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

	spinnerState struct {
		frame  int
		gen    int
		active bool
	}

	commandMessage struct {
		value string
		error bool
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

	redrawMsg struct{}
)

const spinnerTickInterval = 80 * time.Millisecond

var (
	_ BufferRenderer     = (*EditorComponent)(nil)
	_ highlightRefresher = (*EditorComponent)(nil)
)

func newEditorComponent() *EditorComponent {
	return &EditorComponent{
		saveSlot:   &saveGenSlot{},
		completion: DefaultCompletionOptions(),
		cache:      newRenderCache(),
		macroSlot:  &macroSlot{macros: map[rune][]command.KeyEvent{}},
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

// SetAnimation controls whether UI animations play; when off they snap to
// their final state
func (m Model) SetAnimation(enabled bool) {
	m.component.animation = enabled
}

// HandleEvent routes keys, mouse, and editor messages to the panes
func (e *EditorComponent) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	cx.fileWatcher.sync(cx.Editor)
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
	case autoCompletionMsg:
		return e.handleAutoCompletionMsg(cx, msg)
	case docHighlightMsg:
		return e.handleDocHighlightMsg(cx, msg)
	case completionMsg:
		return e.handleCompletionMsg(cx, msg)
	case externalFileChangedMsg:
		return e.handleExternalFileChanged(cx, msg)
	case redrawMsg:
		return e.handleRedraw(cx)
	case vcsUpdatedMsg:
		return e.handleVCSUpdated(cx)
	case spinnerTickMsg:
		return e.handleSpinnerTick(cx, msg)
	case autoSizeTickMsg:
		return e.handleAutoSizeTick(cx, msg)
	case tea.MouseClickMsg:
		return e.handleMouseClick(cx, msg)
	case tea.MouseMotionMsg:
		return e.handleMouseMotion(cx, msg)
	case mouseAxisScrollMsg:
		return e.handleMouseAxisScroll(cx, msg)
	case terminalDragScrollMsg:
		return consumed(), msg.draggable.DragTick(cx, msg.gen, msg.toLow)
	case tea.MouseReleaseMsg:
		return e.handleMouseRelease(cx, msg)
	case tea.MouseWheelMsg:
		return e.handleMouseWheel(cx, msg)
	}
	return ignored(), nil
}

// Render returns the editor's cell buffer for the compositor to blit overlays
// onto, skipping an ANSI round-trip
func (e *EditorComponent) Render(cx *Context, screen geom.Size) *tui.Buffer {
	if e.buf == nil || e.buf.Size != screen {
		e.buf = tui.NewBuffer(screen)
	}
	e.syncEditorMessages(cx)
	e.cache.evictClosed(cx.Editor)
	r := &renderPass{editor: e, context: cx, size: screen}
	r.renderEditorContent(e.buf)
	return e.buf
}

// Cursor returns the focused pane's cursor position and shape
func (e *EditorComponent) Cursor(
	cx *Context, screen geom.Size,
) (tea.Cursor, bool) {
	r := &renderPass{editor: e, context: cx, size: screen}
	return r.editorCursor()
}

func (e *EditorComponent) documentHighlightCmd(cx *Context) tea.Cmd {
	if e.mouse.downRange != nil {
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
	if e.language.highlightPos == pos {
		return nil
	}
	e.language.highlightPos = pos
	e.language.highlightGen++
	gen := e.language.highlightGen
	return func() tea.Msg {
		_, _ = ls.DocumentHighlights(doc, v.ID())
		return docHighlightMsg{gen: gen}
	}
}

// reports the caret position regardless of cursor shape, so caret-anchored
// overlays still work under the normal-mode block cursor that Cursor hides
func (e *EditorComponent) caretScreenPos(cx *Context) (geom.Point, bool) {
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
		rowMap:      e.cache.viewRowMaps[v.ID()],
		tabWidth:    doc.TabWidth(),
		horzOff:     v.Offset().HorizontalOffset,
	})
	return geom.Point{X: area.X + visual.X, Y: yOff + visual.Y}, true
}

type popupAnchorArgs struct {
	screenHeight int
	fallbackRows int
}

func (e *EditorComponent) popupAnchorBelowCaret(
	cx *Context, args popupAnchorArgs,
) geom.Point {
	if at, ok := e.caretScreenPos(cx); ok {
		at.Y++
		return at
	}
	return geom.Point{Y: max(args.screenHeight-args.fallbackRows-2, 0)}
}

func (e *EditorComponent) cancelPending(cx *Context) {
	e.keys.pending = nil
	e.keys.status = ""
	e.keys.infoTitle = ""
	e.keys.infoItems = nil
	e.keys.continuation = nil
	e.keys.hint = ""
	cx.Editor.ResetCount()
}

func (e *EditorComponent) syncEditorMessages(cx *Context) {
	if h := cx.Editor.TakeHint(); h != "" {
		e.keys.hint = h
	}
	if m := cx.Editor.TakeStatusMsg(); m != "" {
		e.setCommandMessage(m)
	}
}

func (e *EditorComponent) setCommandResult(res command.Result) {
	if res.Error != nil {
		e.setCommandError(res.Error)
		return
	}
	if res.Message != "" {
		e.setCommandMessage(res.Message)
	}
}

func (e *EditorComponent) setCommandError(err error) {
	e.keys.message = &commandMessage{
		value: i18n.ErrorText(err),
		error: true,
	}
}

func (e *EditorComponent) setCommandMessage(msg string) {
	e.keys.message = &commandMessage{value: msg}
}

func (e *EditorComponent) clearCommandMessage() {
	e.keys.message = nil
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
		valid:  true,
	}
}
