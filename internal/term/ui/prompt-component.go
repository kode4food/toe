package ui

import (
	"slices"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type (
	// PromptComponent renders and handles an interactive prompt
	PromptComponent struct {
		dismissibleOverlay

		completion completionState
		bg         tui.Color

		editor   *EditorComponent
		bounds   geom.Area
		screen   geom.Size
		dragEdge overlayDrag
		kind     promptKind
		forward  bool
		title    string
		head     string
		buf      string
		caret    int
		horzOff  int

		handler promptHandler
		builder pickerBuilder
	}

	completionState struct {
		items     []promptCompletion
		list      listScroll
		bounds    geom.Area
		done      bool
		nameWidth int
	}

	promptCompletion struct {
		command.Completion
		score int
	}

	pickerBuilder func(*view.Editor, string) (*Picker, error)

	promptKind uint8
)

const (
	promptCmd promptKind = iota + 1
	promptSearch
	promptRegex
	promptShell
	promptTerminalSearch
)

const (
	promptSearchForwardKey  i18n.Key = "prompt.searchForward"
	promptSearchBackwardKey i18n.Key = "prompt.searchBackward"
)

const (
	defaultPromptWidthScale = 0.55
	promptHeightScale       = 0.70

	promptChrome = 3 // top border, input row, bottom border
	promptRule   = 1 // divider between input and its completions
)

const promptEllipsis = "\u2026" // '…' - horizontal ellipsis

// layout ids keep each prompt's saved size separate from the pickers'
var promptLayoutIDs = [...]string{
	promptCmd:            "prompt.command",
	promptSearch:         "prompt.search",
	promptRegex:          "prompt.regex",
	promptShell:          "prompt.shell",
	promptTerminalSearch: "prompt.terminal-search",
}

var _ BufferOverlayComponent = (*PromptComponent)(nil)

type promptComponentArgs struct {
	cx       *Context
	editor   *EditorComponent
	kind     promptKind
	forward  bool
	titleKey i18n.Key
	head     string
	prefill  string
	handler  promptHandler
	builder  pickerBuilder
}

func newPromptComponent(args promptComponentArgs) *PromptComponent {
	th := args.cx.Theme()
	key := args.titleKey
	switch {
	case args.kind == promptSearch && args.forward:
		key = promptSearchForwardKey
	case args.kind == promptSearch:
		key = promptSearchBackwardKey
	case args.kind == promptCmd, key == "":
		key = i18n.PromptCommand
	}
	return &PromptComponent{
		bg:      promptBackground(th),
		editor:  args.editor,
		kind:    args.kind,
		forward: args.forward,
		title:   i18n.Text(key),
		head:    args.head,
		completion: completionState{
			list: listScroll{cursor: -1},
		},
		buf:     args.prefill,
		caret:   len([]rune(args.prefill)),
		handler: args.handler,
		builder: args.builder,
	}
}

// HandleEvent drives editing, history, and completion of the input line
func (p *PromptComponent) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(cx, msg), nil
	case tea.MouseClickMsg:
		return p.handleMouseClick(cx, msg), nil
	case tea.MouseMotionMsg:
		return p.handleMouseMotion(cx, msg), nil
	case tea.MouseReleaseMsg:
		return p.handleMouseRelease(msg), nil
	case tea.MouseWheelMsg:
		return p.handleMouseWheel(cx, msg), nil
	case tea.MouseMsg:
		return consumed(), nil
	}
	return ignored(), nil
}

// Layout claims a centered popup over the frame
func (p *PromptComponent) Layout(
	cx *Context, screen geom.Size,
) (geom.Area, bool) {
	p.markDirty()
	p.screen = screen
	if !p.completion.done {
		p.recalculateCompletion(cx)
	}
	within := geom.Size{
		Width:  screen.Width,
		Height: max(screen.Height-overlayKeepClear, 0),
	}
	frame := p.overlayFrame(cx, within)
	p.syncCompletionRows(frame.Size)
	rows := p.completion.list.rows
	size := geom.Size{Width: frame.Width, Height: promptChrome}
	if rows > 0 {
		size.Height += rows + promptRule
	}
	p.bounds = geom.Area{Point: frame.Point, Size: size}
	return p.bounds, true
}

// PaintBuffer draws the prompt popup, its input line and its completions
func (p *PromptComponent) PaintBuffer(cx *Context, pl geom.Area) *tui.Buffer {
	return p.maybePaint(cx, pl.Size, func(buf *tui.Buffer) {
		pop := p.popup(cx)
		box := geom.Area{Size: pl.Size}
		area := pop.drawInto(buf, box)
		drawPopupTitle(buf, box, p.title, pop.borderStyle)
		p.paintLine(cx, buf, area)
		rows := p.completion.list.rows
		if rows == 0 {
			p.completion.bounds = geom.Area{}
			return
		}
		drawPopupRule(drawPopupRuleArgs{
			buf:   buf,
			at:    geom.Point{X: area.X - 1 - pop.padX, Y: area.Y + 1},
			width: area.Width + 2*pop.padX + 2,
			style: pop.borderStyle,
		})
		list := geom.Area{
			Point: geom.Point{X: area.X, Y: area.Y + promptChrome - 1},
			Size:  geom.Size{Width: area.Width, Height: rows},
		}
		p.completion.bounds = list.Translate(pl.Point)
		p.paintCompletions(cx, buf, list)
	})
}

// Cursor returns the caret position within the input line
func (p *PromptComponent) Cursor(
	cx *Context, _ geom.Size,
) (cur tea.Cursor, ok bool) {
	return insertCursorAt(cx, geom.Point{
		X: p.bounds.X + 1 + overlayPadX + p.caretDisplayX(),
		Y: p.bounds.Y + 1,
	})
}

func (p *PromptComponent) popup(cx *Context) popup {
	st := p.rowStyle(cx)
	return popup{
		borderStyle:  st.Fg(pickerFrameStyle(cx).FgColor()),
		contentStyle: st,
		padX:         overlayPadX,
	}
}

func (p *PromptComponent) textWidth() int {
	label := runewidth.StringWidth(p.inputLabel())
	inner := p.bounds.Width - 2 - 2*overlayPadX
	return max(inner-label, 1)
}

func (p *PromptComponent) rowStyle(cx *Context) tui.Style {
	return tui.Style{}.Bg(p.bg).Fg(cx.Theme().Get("ui.text").FgColor())
}

func (p *PromptComponent) syncScroll() {
	runes := []rune(p.buf)
	w := p.textWidth()
	switch {
	case runewidth.StringWidth(p.buf) < w:
		p.horzOff = 0
	case p.caret <= p.horzOff:
		p.horzOff = max(p.caret-1, 0)
	case runewidth.StringWidth(string(runes[p.horzOff:p.caret])) > w:
		width := 0
		i := p.caret
		for i > 0 {
			width += runewidth.RuneWidth(runes[i-1])
			if width > w {
				break
			}
			i--
		}
		p.horzOff = i
	}
	tail := runewidth.StringWidth(string(runes[p.horzOff:]))
	caretCol := runewidth.StringWidth(string(runes[p.horzOff:p.caret]))
	if tail > w && caretCol >= w {
		p.horzOff++
	}
}

func (p *PromptComponent) caretDisplayX() int {
	p.syncScroll()
	label := runewidth.StringWidth(p.inputLabel())
	shown := []rune(p.buf)[p.horzOff:p.caret]
	return label + runewidth.StringWidth(string(shown))
}

func (p *PromptComponent) inputLabel() string {
	if p.head == "" {
		return ""
	}
	return p.head + " "
}

func (p *PromptComponent) handleKey(
	cx *Context, msg tea.KeyPressMsg,
) EventResult {
	k := FromTeaKey(msg)
	pop := func(cmd tea.Cmd) EventResult {
		return consumedWith(func(cx *Context, comp *Compositor) tea.Cmd {
			comp.Pop()
			return tea.Batch(cmd, comp.refreshEditorHighlight(cx))
		})
	}
	switch {
	case k.Code.Special == command.Escape:
		return pop(nil)

	case k.Code.Special == command.Tab:
		p.acceptCompletion()
		return consumed()

	// Enter submits what is typed; only Tab reaches into the completions, so a
	// line the user finished themselves is never completed over
	case k.Code.Special == command.Enter:
		return p.accept(cx, pop)

	case k.Code.Special == command.Up,
		k.Code.Char == 'p' && k.Mods == command.ModCtrl:
		p.changeCompletion(-1)

	case k.Code.Special == command.Down,
		k.Code.Char == 'n' && k.Mods == command.ModCtrl:
		p.changeCompletion(1)

	case k.Code.Special == command.PageUp:
		p.changeCompletion(-max(p.completion.list.rows-1, 1))

	case k.Code.Special == command.PageDown:
		p.changeCompletion(max(p.completion.list.rows-1, 1))

	// word deletions come before char deletions so a modified backspace or
	// delete is not swallowed by the plain case
	case k.Code.Char == 'w' && k.Mods == command.ModCtrl,
		k.Code.Special == command.Backspace && k.Mods.Has(command.ModCtrl),
		k.Code.Special == command.Backspace && k.Mods.Has(command.ModAlt):
		runes := []rune(p.buf)
		start := promptWordLeft(runes, p.caret)
		p.buf = string(slices.Delete(runes, start, p.caret))
		p.caret = start
		p.recalculateCompletion(cx)

	case k.Code.Char == 'd' && k.Mods == command.ModAlt,
		k.Code.Special == command.Delete && k.Mods.Has(command.ModCtrl),
		k.Code.Special == command.Delete && k.Mods.Has(command.ModAlt):
		runes := []rune(p.buf)
		end := promptWordRight(runes, p.caret)
		p.buf = string(slices.Delete(runes, p.caret, end))
		p.recalculateCompletion(cx)

	case k.Code.Special == command.Backspace,
		k.Code.Char == 'h' && k.Mods == command.ModCtrl:
		if p.caret > 0 {
			runes := []rune(p.buf)
			p.buf = string(slices.Delete(runes, p.caret-1, p.caret))
			p.caret--
		}
		p.recalculateCompletion(cx)

	case k.Code.Special == command.Delete,
		k.Code.Char == 'd' && k.Mods == command.ModCtrl:
		runes := []rune(p.buf)
		if p.caret < len(runes) {
			p.buf = string(slices.Delete(runes, p.caret, p.caret+1))
		}
		p.recalculateCompletion(cx)

	case k.Code.Char == 'k' && k.Mods == command.ModCtrl:
		p.buf = string([]rune(p.buf)[:p.caret])
		p.recalculateCompletion(cx)

	case k.Code.Char == 'u' && k.Mods == command.ModCtrl:
		p.buf = string([]rune(p.buf)[p.caret:])
		p.caret = 0
		p.recalculateCompletion(cx)

	case k.Code.Special == command.Left && k.Mods.Has(command.ModCtrl),
		k.Code.Char == 'b' && k.Mods == command.ModAlt:
		p.caret = promptWordLeft([]rune(p.buf), p.caret)

	case k.Code.Special == command.Right && k.Mods.Has(command.ModCtrl),
		k.Code.Char == 'f' && k.Mods == command.ModAlt:
		p.caret = promptWordRight([]rune(p.buf), p.caret)

	case k.Code.Special == command.Left,
		k.Code.Char == 'b' && k.Mods == command.ModCtrl:
		p.caret = max(p.caret-1, 0)

	case k.Code.Special == command.Right,
		k.Code.Char == 'f' && k.Mods == command.ModCtrl:
		p.caret = min(p.caret+1, len([]rune(p.buf)))

	case k.Code.Special == command.Home,
		k.Code.Char == 'a' && k.Mods == command.ModCtrl:
		p.caret = 0

	case k.Code.Special == command.End,
		k.Code.Char == 'e' && k.Mods == command.ModCtrl:
		p.caret = len([]rune(p.buf))

	default:
		if k.IsTypable() {
			runes := []rune(p.buf)
			runes = slices.Insert(runes, p.caret, k.Code.Char)
			p.buf = string(runes)
			p.caret++
			p.recalculateCompletion(cx)
		}
	}
	return consumed()
}

func (p *PromptComponent) accept(
	cx *Context, pop func(tea.Cmd) EventResult,
) EventResult {
	switch p.kind {
	case promptCmd:
		res := execTypable(cx, strings.TrimSpace(p.buf))
		p.editor.setCommandResult(res)
		return pop(signalToCmd(res.Signal))

	case promptSearch:
		pat := strings.TrimSpace(p.buf)
		if pat == "" {
			// empty search repeats the last pattern in the prompt's direction
			pat, _ = cx.Editor.FirstRegister(view.RegisterSearch)
		}
		if pat == "" {
			p.editor.clearCommandMessage()
			return pop(nil)
		}
		var err error
		if p.forward {
			err = action.SearchForward(cx.Editor, pat)
		} else {
			err = action.SearchBackward(cx.Editor, pat)
		}
		if err != nil {
			p.editor.setCommandError(err)
		} else {
			p.editor.clearCommandMessage()
		}
		return pop(nil)

	default:
		if p.buf == "" {
			return pop(nil)
		}
		if p.builder != nil {
			picker, err := p.builder(cx.Editor, p.buf)
			if err != nil {
				p.editor.setCommandError(err)
				return pop(nil)
			}
			if picker == nil {
				return pop(nil)
			}
			feedCmd := picker.load.feedCmd
			picker.load.feedCmd = nil
			return consumedWith(func(_ *Context, comp *Compositor) tea.Cmd {
				comp.Pop()
				comp.Push(newPickerComponent(cx, picker))
				return feedCmd
			})
		}
		if p.handler != nil {
			if err := p.handler(cx.Editor, p.buf); err != nil {
				p.editor.setCommandError(err)
			} else {
				p.editor.clearCommandMessage()
			}
		}
		return pop(nil)
	}
}

func (p *PromptComponent) paintLine(
	cx *Context, buf *tui.Buffer, area geom.Area,
) {
	th := cx.Theme()
	rowBg := tui.Style{}.Bg(p.bg)
	labelSt := applyAccentStyle(styleOverlay{
		base:    rowBg,
		overlay: th.Get("ui.prompt"),
	})
	textSt := p.rowStyle(cx)

	label := p.inputLabel()
	buf.FillRange(area.Point, area.Width, textSt)
	buf.SetString(area.Point, label, labelSt)
	x := runewidth.StringWidth(label)

	p.syncScroll()
	runes := []rune(p.buf)
	avail := p.textWidth()
	tailWidth := runewidth.StringWidth(string(runes[p.horzOff:]))
	truncEnd := tailWidth > avail

	limit := avail
	if truncEnd {
		limit--
	}
	col := 0
	i := p.horzOff
	for i < len(runes) && col+runewidth.RuneWidth(runes[i]) <= limit {
		buf.SetString(geom.Point{
			X: area.X + x + col,
			Y: area.Y,
		}, string(runes[i]), textSt)
		col += runewidth.RuneWidth(runes[i])
		i++
	}
	if p.horzOff > 0 {
		buf.SetString(geom.Point{
			X: area.X + x,
			Y: area.Y,
		}, promptEllipsis, textSt)
	}
	if truncEnd {
		buf.SetString(geom.Point{
			X: area.X + x + col,
			Y: area.Y,
		}, promptEllipsis, textSt)
	}
}

func (p *PromptComponent) overlayFrame(
	cx *Context, within geom.Size,
) geom.Area {
	width := cx.pickerLayout.widthScale(p.layoutID(), defaultPromptWidthScale)
	size := geom.Size{
		Width: max(scaleExtent(within.Width, width), 1),
		Height: max(
			scaleExtent(within.Height, promptHeightScale), promptChrome,
		),
	}
	return geom.Area{
		Point: geom.Area{Size: within}.Center(size),
		Size:  size,
	}
}

func (p *PromptComponent) handleMouseMotion(
	cx *Context, msg tea.MouseMotionMsg,
) EventResult {
	if msg.Button == tea.MouseLeft && p.dragEdge.active() {
		p.resizeToEdge(cx, geom.Point{X: msg.X, Y: msg.Y})
	}
	return consumed()
}

func (p *PromptComponent) handleMouseRelease(
	msg tea.MouseReleaseMsg,
) EventResult {
	if msg.Button == tea.MouseLeft {
		p.dragEdge = overlayDrag{}
	}
	return consumed()
}

func (p *PromptComponent) beginEdgeDrag(cx *Context, at geom.Point) bool {
	drag := overlayDrag{
		startWidth: cx.pickerLayout.widthScale(
			p.layoutID(), defaultPromptWidthScale,
		),
	}
	if !drag.begin(p.bounds, at) {
		return false
	}
	// the box grows downward from a fixed input row, so a vertical drag would
	// move it rather than resize it
	drag.signY = 0
	if !drag.active() {
		return false
	}
	p.dragEdge = drag
	return true
}

func (p *PromptComponent) resizeToEdge(cx *Context, at geom.Point) {
	cx.pickerLayout = p.dragEdge.applyTo(
		cx.pickerLayout, p.layoutID(), at,
		geom.Size{Width: p.screen.Width},
	)
	p.markDirty()
}

func (p *PromptComponent) layoutID() string {
	if int(p.kind) >= len(promptLayoutIDs) || promptLayoutIDs[p.kind] == "" {
		return "prompt"
	}
	return promptLayoutIDs[p.kind]
}

func promptWordLeft(runes []rune, caret int) int {
	i := caret
	for i > 0 && unicode.IsSpace(runes[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(runes[i-1]) {
		i--
	}
	return i
}

func promptWordRight(runes []rune, caret int) int {
	i := caret
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	for i < len(runes) && !unicode.IsSpace(runes[i]) {
		i++
	}
	return i
}
