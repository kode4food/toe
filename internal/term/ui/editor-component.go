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
		signatureHidden *signatureCall
		docHighlightGen int
		docHighlightPos docHighlightPosition
		completionGen   int
		completionOpts  CompletionOptions
		fileWatcher     *editorFileWatcher
	}

	saveGenSlot struct{ gen int }

	autoSaveMsg struct{ gen int }

	docHighlightMsg struct{ gen int }

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
)

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
			cx.Editor.SaveAll()
		}
		return ignored(), nil

	case autoSaveMsg:
		if msg.gen == e.saveSlot.gen {
			cx.Editor.SaveAll()
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
				opts:       e.completionOpts.WithDefaults(),
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

	case tea.MouseClickMsg:
		e.completionGen++
		e.cancelPending(cx)
		if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
			r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
			r.handleMouseClick(msg.X, msg.Y, msg.Mod)
		}
		return consumed(), e.documentHighlightCmd(cx)

	case tea.MouseMotionMsg:
		e.completionGen++
		if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
			r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
			r.handleMouseDrag(msg.X, msg.Y)
		}
		return consumed(), e.documentHighlightCmd(cx)

	case tea.MouseReleaseMsg:
		e.completionGen++
		if !cx.Editor.Options().Mouse {
			return consumed(), nil
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
		up := msg.Button == tea.MouseWheelUp
		r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
		v, ok := r.contentViewAt(msg.X, msg.Y)
		if !ok {
			return consumed(), nil
		}
		act.ScrollViewLines(cx.Editor, v, cx.Editor.Options().ScrollLines, up)
		if doc, ok := cx.Editor.Document(v.DocID()); ok {
			v.BeginFreeScroll(doc.Revision(), doc.SelectionFor(v.ID()))
		}
		return consumed(), nil
	}
	return ignored(), nil
}

func (e *EditorComponent) documentHighlightCmd(cx *Context) tea.Cmd {
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

func (e *EditorComponent) Render(w, h int, cx *Context) string {
	return e.RenderBuffer(w, h, cx).RenderToANSI()
}

// RenderBuffer lets an overlay layer draw on top of the editor buffer
// without an ANSI round-trip
func (e *EditorComponent) RenderBuffer(w, h int, cx *Context) *tui.Buffer {
	if e.buf == nil || e.buf.Width != w || e.buf.Height != h {
		e.buf = tui.NewBuffer(w, h)
	} else {
		e.buf.Clear()
	}
	e.cache.evictClosed(cx.Editor)
	r := &renderPass{ec: e, cx: cx, w: w, h: h}
	r.renderEditorContent(e.buf)
	return e.buf
}

func (e *EditorComponent) Cursor(w, h int, cx *Context) (tea.Cursor, bool) {
	r := &renderPass{ec: e, cx: cx, w: w, h: h}
	return r.editorCursor()
}

func newEditorComponent() *EditorComponent {
	return &EditorComponent{
		saveSlot:       &saveGenSlot{},
		cache:          newRenderCache(),
		macroSlot:      &macroSlot{macros: map[rune][]command.KeyEvent{}},
		focused:        true,
		completionOpts: DefaultCompletionOptions(),
		fileWatcher:    newEditorFileWatcher(),
	}
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
