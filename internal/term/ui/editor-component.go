package ui

import (
	"fmt"
	"slices"
	"strings"
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
	}

	// renderPass bundles the state needed for a single render pass so
	// every render helper receives it without passing cx and ec separately
	renderPass struct {
		ec *EditorComponent
		cx *Context
		w  int
		h  int
	}
)

func newEditorComponent() *EditorComponent {
	return &EditorComponent{
		saveSlot:  &saveGenSlot{},
		cache:     newRenderCache(),
		macroSlot: &macroSlot{macros: map[rune][]command.KeyEvent{}},
		focused:   true,
	}
}

func newRenderCache() *renderCache {
	return &renderCache{
		docCaches:   map[view.DocumentId]*docRenderCache{},
		viewRowMaps: map[view.Id][]viewRowEntry{},
	}
}

func (e *EditorComponent) HandleEvent(
	msg tea.Msg, cx *Context,
) (EventResult, tea.Cmd) {
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
		return result, tea.Batch(cmd, e.autoSaveCmd(cx))

	case tea.FocusMsg:
		e.focused = true
		return ignored(), nil

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

	case tea.MouseClickMsg:
		e.cancelPending(cx)
		if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
			r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
			r.handleMouseClick(msg.X, msg.Y, msg.Mod)
		}
		return consumed(), nil

	case tea.MouseMotionMsg:
		if cx.Editor.Options().Mouse && msg.Button == tea.MouseLeft {
			r := &renderPass{ec: e, cx: cx, w: e.w, h: e.h}
			r.handleMouseDrag(msg.X, msg.Y)
		}
		return consumed(), nil

	case tea.MouseReleaseMsg:
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
		return consumed(), nil

	case tea.MouseWheelMsg:
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
		return consumed(), nil
	}
	return ignored(), nil
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
	r := &renderPass{ec: e, cx: cx, w: w, h: h}
	r.renderEditorContent(e.buf)
	return e.buf
}

func (e *EditorComponent) Cursor(w, h int, cx *Context) (tea.Cursor, bool) {
	r := &renderPass{ec: e, cx: cx, w: w, h: h}
	return r.editorCursor()
}

func (e *EditorComponent) handleKeyPress(
	msg tea.KeyPressMsg, cx *Context,
) (EventResult, tea.Cmd) {
	k := FromTeaKey(msg)

	if len(e.pending) == 0 && k.Code.Char == 'z' && k.Mods == command.ModCtrl {
		return consumed(), tea.Suspend
	}

	if e.macroSlot.recording {
		e.macroSlot.keys = append(e.macroSlot.keys, k)
	}

	if e.continuation != nil {
		cont := e.continuation(cx.Editor, k)
		e.continuation = cont
		if cont == nil {
			e.hint = ""
			cx.Editor.ResetCount()
		}
		e.syncEditorMessages(cx)
		e.handleReplay(cx)
		return consumed(), nil
	}

	mode := cx.Editor.Mode()
	modeStr := mode.String()

	noPending := len(e.pending) == 0 && k.Mods == command.ModNone &&
		k.Code.Special == ""
	if noPending && mode == view.ModeSelect {
		ch := k.Code.Char
		cur := cx.Editor.Count()
		if ch >= '1' && ch <= '9' || (ch == '0' && cur > 0) {
			cx.Editor.SetCount(cur*10 + int(ch-'0'))
			e.status = fmt.Sprintf("%d", cx.Editor.Count())
			return consumed(), nil
		}
	}

	e.pending = append(e.pending, k)
	action, found, prefix := cx.Keymaps.Lookup(modeStr, e.pending)
	switch {
	case found:
		e.pending = nil
		e.status = ""
		e.cmdMsg = ""
		e.infoTitle = ""
		e.infoItems = nil
		cont := action(cx.Editor)
		e.continuation = cont
		if cont == nil {
			e.hint = ""
			cx.Editor.ResetCount()
		}
		e.syncEditorMessages(cx)
		e.handleReplay(cx)

		if ov := e.nextLayer; ov != nil {
			e.nextLayer = nil
			return consumedWith(func(comp *Compositor, cx *Context) tea.Cmd {
				layer, cmd := ov(cx)
				if layer != nil {
					comp.Push(layer)
				}
				return cmd
			}), nil
		}
		return consumed(), nil

	case prefix:
		var sb strings.Builder
		if c := cx.Editor.Count(); c > 0 {
			_, _ = fmt.Fprintf(&sb, "%d", c)
		}
		for _, pk := range e.pending {
			sb.WriteString(pk.String())
		}
		e.status = sb.String()
		title, items := cx.Keymaps.PendingHints(modeStr, e.pending)
		e.infoTitle = title
		e.infoItems = items
		return consumed(), nil

	default:
		if mode == view.ModeInsert && len(e.pending) == 1 {
			if e.pending[0].IsTypable() {
				act.InsertChar(cx.Editor, e.pending[0].Code.Char)
			}
		}
		e.pending = nil
		e.status = ""
		e.infoTitle = ""
		e.infoItems = nil
		cx.Editor.ResetCount()
		return consumed(), nil
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

func (e *EditorComponent) handleReplay(cx *Context) {
	ms := e.macroSlot
	if !ms.hasReplay {
		return
	}
	ms.hasReplay = false
	e.replayMacro(cx, ms.macros[ms.replayReg], ms.replayN)
}

func (e *EditorComponent) replayMacro(
	cx *Context, keys []command.KeyEvent, n int,
) {
	for range n {
		i := 0
		for i < len(keys) {
			k := keys[i]
			i++
			mode := cx.Editor.Mode()
			modeStr := mode.String()
			if replaySkip(k, mode) {
				continue
			}
			action, found, _ := cx.Keymaps.Lookup(
				modeStr, []command.KeyEvent{k},
			)
			if found {
				cont := action(cx.Editor)
				for cont != nil && i < len(keys) {
					k = keys[i]
					i++
					cont = cont(cx.Editor, k)
				}
				cx.Editor.ResetCount()
			} else if mode == view.ModeInsert && k.IsTypable() {
				act.InsertChar(cx.Editor, k.Code.Char)
			}
		}
	}
}

func (e *EditorComponent) MacroRecordAction(
	ed *view.Editor,
) command.Continuation {
	ms := e.macroSlot
	if ms.recording {
		if len(ms.keys) > 0 {
			ms.keys = ms.keys[:len(ms.keys)-1]
		}
		ms.macros[ms.reg] = slices.Clone(ms.keys)
		ms.recording = false
		ms.keys = nil
		ms.reg = 0
		return nil
	}
	ed.SetHint("Q ...")
	return func(ed *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char == 0 || k.Mods != command.ModNone {
			ed.SetHint("")
			return nil
		}
		ms.recording = true
		ms.reg = k.Code.Char
		ms.keys = nil
		return nil
	}
}

func (e *EditorComponent) MacroReplayAction(
	ed *view.Editor,
) command.Continuation {
	ms := e.macroSlot
	if ms.recording {
		return nil
	}
	ed.SetHint("q ...")
	return func(ed *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char == 0 || k.Mods != command.ModNone {
			ed.SetHint("")
			return nil
		}
		n := ed.Count()
		if n == 0 {
			n = 1
		}
		ms.replayReg = k.Code.Char
		ms.replayN = n
		ms.hasReplay = true
		return nil
	}
}

func bufferlineVisible(cx *Context) bool {
	switch cx.Editor.Options().BufferLine {
	case view.BufferLineAlways:
		return true
	case view.BufferLineMultiple:
		return len(cx.Editor.AllDocuments()) > 1
	default:
		return false
	}
}

func replaySkip(k command.KeyEvent, mode view.Mode) bool {
	if mode != view.ModeNormal && mode != view.ModeSelect {
		return false
	}
	if k.Mods == command.ModNone {
		switch k.Code.Char {
		case ':', '/', '?', 's', 'S', 'K', '|', '!', '$', 'Q', 'q':
			return true
		}
	}
	if k.Mods == command.ModAlt {
		switch k.Code.Char {
		case '|', '!', 'K':
			return true
		}
	}
	return false
}
