package ui

import (
	"cmp"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	// a danger capture tints its ground toward the accent theme scope
	optionCapture struct {
		dismissibleOverlay

		question string
		accent   string
		options  []captureOption
		lines    []popupLine
		cursor   int
		danger   bool
	}

	// a nil choose answers the question without doing anything
	captureOption struct {
		label  string
		choose func(*view.Editor) tea.Cmd
		key    rune
	}

	captureStyles struct {
		content tui.Style
		border  tui.Style
		option  tui.Style
	}
)

const (
	captureYesKey i18n.Key = "capture.yes"
	captureNoKey  i18n.Key = "capture.no"
)

const (
	captureMaxWidth = 60

	// buttons sit apart so their painted edges never touch
	captureButtonGap = 2

	// borders, the blank row under the question, and the button row
	captureChrome = 4
)

var _ BufferOverlayComponent = (*optionCapture)(nil)

// HandleEvent picks an answer, moves between the buttons, or dismisses the
// popup unanswered
func (o *optionCapture) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return o.handleKey(cx, FromTeaKey(msg))
	case tea.MouseClickMsg:
		return consumedWith(popLayer), nil
	case tea.MouseWheelMsg:
		// swallowed so the layer behind the popup does not scroll
		return consumed(), nil
	}
	return ignored(), nil
}

// Cursor leaves the cursor to the layer below
func (o *optionCapture) Cursor(*Context, geom.Size) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

// Layout centers the popup on the screen, wide enough for the question or the
// row of buttons, whichever needs more
func (o *optionCapture) Layout(
	_ *Context, screen geom.Size,
) (geom.Area, bool) {
	chrome := 2 + 2*overlayPadX
	o.lines = popupTextLines(
		o.question, min(screen.Width, captureMaxWidth)-chrome,
	)
	width := max(popupTextWidth(o.lines), o.buttonsWidth()) + chrome
	size := geom.Size{
		Width:  min(max(width, 2), screen.Width),
		Height: min(len(o.lines)+captureChrome, screen.Height),
	}
	return geom.Area{
		X:    max((screen.Width-size.Width)/2, 0),
		Y:    max((screen.Height-size.Height)/2, 0),
		Size: size,
	}, true
}

// PaintBuffer draws the centered question above its row of buttons
func (o *optionCapture) PaintBuffer(cx *Context, pl geom.Area) *tui.Buffer {
	return o.maybePaint(cx, pl.Size, func(buf *tui.Buffer) {
		st := o.styles(cx)
		pop := popup{
			borderStyle:  st.border,
			contentStyle: st.content,
			padX:         overlayPadX,
		}
		area := pop.drawInto(buf, geom.Area{Size: buf.Size})
		for i, line := range o.lines {
			width := runewidth.StringWidth(line.text)
			buf.SetString(geom.Point{
				X: area.X + centerOffset(area.Width, width),
				Y: area.Y + i,
			}, line.text, st.content)
		}
		o.paintButtons(buf, geom.Point{
			X: area.X,
			Y: area.Y + len(o.lines) + 1,
		}, area.Width, st)
	})
}

func (o *optionCapture) handleKey(
	cx *Context, k command.KeyEvent,
) (EventResult, tea.Cmd) {
	if k.Mods == 0 {
		for _, opt := range o.options {
			if k.Code.Char == opt.key {
				return o.answer(cx, opt)
			}
		}
	}
	switch {
	case k.Code.Special == command.Enter, k.Code.Char == ' ':
		return o.answer(cx, o.options[o.cursor])
	case k.Code.Special == command.Left, k.Code.Char == 'h',
		k.Code.Special == command.Tab && k.Mods == command.ModShift:
		o.moveBy(-1)
	case k.Code.Special == command.Right, k.Code.Char == 'l',
		k.Code.Special == command.Tab:
		o.moveBy(1)
	default:
		return consumedWith(popLayer), nil
	}
	return consumed(), nil
}

// every answer closes the popup, whether or not it does anything
func (o *optionCapture) answer(
	cx *Context, opt captureOption,
) (EventResult, tea.Cmd) {
	if opt.choose == nil {
		return consumedWith(popLayer), nil
	}
	return consumedWith(popLayer), opt.choose(cx.Editor)
}

func (o *optionCapture) moveBy(n int) {
	o.cursor = (o.cursor + n + len(o.options)) % len(o.options)
	o.markDirty()
}

// an offered danger tints the whole popup, so the ground itself says that
// something here cannot be undone
func (o *optionCapture) styles(cx *Context) captureStyles {
	th := cx.Theme()
	content := th.Get("ui.popup")
	accent := th.Get(cmp.Or(o.accent, "error")).FgColor()
	st := captureStyles{
		content: content,
		border:  content,
		option:  th.Get("ui.menu"),
	}
	if !o.danger {
		return st
	}
	st.content = content.Bg(tintToward(&tintColors{
		base:   content.BgColor(),
		accent: accent,
	}))
	st.border = st.content.Fg(accent)
	st.option = st.option.Bg(st.content.BgColor())
	return st
}

// the focused button wears its own colors inverted, which stands out against
// any theme's ground, and every button underlines the key that picks it
func (o *optionCapture) paintButtons(
	buf *tui.Buffer, at geom.Point, width int, st captureStyles,
) {
	at.X += centerOffset(width, o.buttonsWidth())
	for i, opt := range o.options {
		if i > 0 {
			at.X += captureButtonGap
		}
		base := st.option
		if i == o.cursor {
			base = base.Bg(base.FgColor()).
				Fg(st.content.BgColor()).
				Mod(tui.ModifierBold)
		}
		text := captureButtonText(opt)
		buf.SetString(at, text, base)
		buf.SetString(
			geom.Point{X: at.X + 1, Y: at.Y}, string(opt.key),
			base.UlStyle(tui.UnderlineLine).UlColor(base.FgColor()),
		)
		at.X += runewidth.StringWidth(text)
	}
}

func (o *optionCapture) buttonsWidth() int {
	w := captureButtonGap * max(len(o.options)-1, 0)
	for _, opt := range o.options {
		w += captureButtonWidth(opt)
	}
	return w
}

type optionCaptureArgs struct {
	question string
	accent   string
	options  []captureOption
	focus    int
	danger   bool
}

func newOptionCapture(args optionCaptureArgs) *optionCapture {
	return &optionCapture{
		question: args.question,
		accent:   args.accent,
		options:  args.options,
		cursor:   max(min(args.focus, len(args.options)-1), 0),
		danger:   args.danger,
	}
}

// a danger confirmation focuses no, the answer that changes nothing
func newConfirmation(
	question string, choose func(*view.Editor) tea.Cmd, danger bool,
) *optionCapture {
	focus := 0
	if danger {
		focus = 1
	}
	return newOptionCapture(optionCaptureArgs{
		question: question,
		focus:    focus,
		danger:   danger,
		options: []captureOption{
			{key: 'y', label: i18n.Text(captureYesKey), choose: choose},
			{key: 'n', label: i18n.Text(captureNoKey)},
		},
	})
}

func captureButtonText(opt captureOption) string {
	return " " + string(opt.key) + " " + opt.label + " "
}

func captureButtonWidth(opt captureOption) int {
	return runewidth.StringWidth(captureButtonText(opt))
}

func centerOffset(width, content int) int {
	return max((width-content)/2, 0)
}
