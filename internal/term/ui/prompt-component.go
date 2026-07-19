package ui

import (
	"slices"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type (
	PromptComponent struct {
		overlayBuf
		ec       *EditorComponent
		bounds   geom.Area
		kind     promptKind
		forward  bool
		prompt   string
		buf      string
		caret    int
		hOff     int
		comps    []promptCompletion
		compSel  *int
		compDone bool
		comp     geom.Size
		fn       promptHandler
		pickerFn pickerBuilder
	}

	promptCompletion struct {
		command.Completion
		score int
	}

	// pickerBuilder constructs a sub-picker from the submitted prompt text
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

// one column so the end-of-buffer caret cell stays on screen
const promptRightPad = 1

var _ BufferOverlayComponent = (*PromptComponent)(nil)

func (p *PromptComponent) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(cx, msg), nil

	case tea.MouseMsg, tea.MouseClickMsg:
		return consumed(), nil
	}
	return ignored(), nil
}

func (p *PromptComponent) Layout(
	cx *Context, screen geom.Size,
) (geom.Area, bool) {
	p.markDirty()
	if !p.compDone {
		p.recalculateCompletion(cx)
	}
	menuH := p.completionMenuHeight(screen)
	y := max(screen.Height-1-menuH, 0)
	p.bounds = geom.Area{
		Point: geom.Point{X: 0, Y: y},
		Size:  geom.Size{Width: screen.Width, Height: screen.Height - y},
	}
	return p.bounds, true
}

func (p *PromptComponent) PaintBuffer(cx *Context, pl geom.Area) *tui.Buffer {
	return p.maybePaint(cx, pl.Size, func(buf *tui.Buffer) {
		if p.comp.Height > 0 {
			p.paintCompletions(cx, buf, geom.Area{
				Size: geom.Size{
					Width:  pl.Width,
					Height: p.comp.Height + 2,
				},
			})
		}
		p.paintLine(cx, buf, geom.Area{
			Point: geom.Point{
				Y: pl.Height - 1,
			},
			Size: geom.Size{
				Width:  pl.Width,
				Height: 1},
		})
	})
}

func (p *PromptComponent) Cursor(
	cx *Context, _ geom.Size,
) (cur tea.Cursor, ok bool) {
	kind := cx.Editor.Options().CursorShapeForMode(view.ModeInsert.String())
	if kind == view.CursorKindHidden {
		return tea.Cursor{}, false
	}
	return tea.Cursor{
		Position: tea.Position{
			X: p.bounds.X + p.caretDisplayX(),
			Y: p.bounds.Y + p.bounds.Height - 1,
		},
		Shape: cursorKindToShape(kind),
		Blink: false,
	}, true
}

func (p *PromptComponent) textWidth() int {
	label := runewidth.StringWidth(p.promptLabel())
	return max(p.bounds.Width-label-promptRightPad, 1)
}

func (p *PromptComponent) syncScroll() {
	runes := []rune(p.buf)
	w := p.textWidth()
	switch {
	case runewidth.StringWidth(p.buf) < w:
		p.hOff = 0
	case p.caret <= p.hOff:
		p.hOff = max(p.caret-1, 0)
	case runewidth.StringWidth(string(runes[p.hOff:p.caret])) > w:
		width, i := 0, p.caret
		for i > 0 {
			width += runewidth.RuneWidth(runes[i-1])
			if width > w {
				break
			}
			i--
		}
		p.hOff = i
	}
	tail := runewidth.StringWidth(string(runes[p.hOff:]))
	caretCol := runewidth.StringWidth(string(runes[p.hOff:p.caret]))
	if tail > w && caretCol >= w {
		p.hOff++
	}
}

func (p *PromptComponent) caretDisplayX() int {
	p.syncScroll()
	label := runewidth.StringWidth(p.promptLabel())
	return label + runewidth.StringWidth(string([]rune(p.buf)[p.hOff:p.caret]))
}

func (p *PromptComponent) promptLabel() string {
	switch p.kind {
	case promptCmd:
		return i18n.Text(i18n.PromptCommand) + " "
	case promptSearch:
		if p.forward {
			return i18n.Text(i18n.PromptSearchForward) + ": "
		}
		return i18n.Text(i18n.PromptSearchBackward) + ": "
	default:
		return p.prompt + ": "
	}
}

type promptComponentArgs struct {
	ec       *EditorComponent
	kind     promptKind
	forward  bool
	prompt   string
	buf      string
	fn       promptHandler
	pickerFn pickerBuilder
}

func newPromptComponent(args promptComponentArgs) *PromptComponent {
	return &PromptComponent{
		ec:       args.ec,
		kind:     args.kind,
		forward:  args.forward,
		prompt:   args.prompt,
		buf:      args.buf,
		caret:    len([]rune(args.buf)),
		fn:       args.fn,
		pickerFn: args.pickerFn,
	}
}

func (p *PromptComponent) handleKey(
	cx *Context, msg tea.KeyPressMsg,
) EventResult {
	k := FromTeaKey(msg)
	pop := func(cmd tea.Cmd) EventResult {
		return consumedWith(func(_ *Context, comp *Compositor) tea.Cmd {
			comp.Pop()
			return cmd
		})
	}
	switch {
	case k.Code.Special == "esc":
		return pop(nil)

	case k.Code.Special == "ret":
		return p.accept(cx, pop)

	case k.Code.Special == "tab" && k.Mods.Has(command.ModShift):
		p.changeCompletion(-1)

	case k.Code.Special == "tab":
		p.changeCompletion(1)

	// word deletions come before char deletions so a modified backspace or
	// delete is not swallowed by the plain case
	case k.Code.Char == 'w' && k.Mods == command.ModCtrl,
		k.Code.Special == "backspace" && k.Mods.Has(command.ModCtrl),
		k.Code.Special == "backspace" && k.Mods.Has(command.ModAlt):
		runes := []rune(p.buf)
		start := promptWordLeft(runes, p.caret)
		p.buf = string(slices.Delete(runes, start, p.caret))
		p.caret = start
		p.recalculateCompletion(cx)

	case k.Code.Char == 'd' && k.Mods == command.ModAlt,
		k.Code.Special == "del" && k.Mods.Has(command.ModCtrl),
		k.Code.Special == "del" && k.Mods.Has(command.ModAlt):
		runes := []rune(p.buf)
		end := promptWordRight(runes, p.caret)
		p.buf = string(slices.Delete(runes, p.caret, end))
		p.recalculateCompletion(cx)

	case k.Code.Special == "backspace",
		k.Code.Char == 'h' && k.Mods == command.ModCtrl:
		if p.caret > 0 {
			runes := []rune(p.buf)
			p.buf = string(slices.Delete(runes, p.caret-1, p.caret))
			p.caret--
		}
		p.recalculateCompletion(cx)

	case k.Code.Special == "del",
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

	case k.Code.Special == "left" && k.Mods.Has(command.ModCtrl),
		k.Code.Char == 'b' && k.Mods == command.ModAlt:
		p.caret = promptWordLeft([]rune(p.buf), p.caret)

	case k.Code.Special == "right" && k.Mods.Has(command.ModCtrl),
		k.Code.Char == 'f' && k.Mods == command.ModAlt:
		p.caret = promptWordRight([]rune(p.buf), p.caret)

	case k.Code.Special == "left",
		k.Code.Char == 'b' && k.Mods == command.ModCtrl:
		p.caret = max(p.caret-1, 0)

	case k.Code.Special == "right",
		k.Code.Char == 'f' && k.Mods == command.ModCtrl:
		p.caret = min(p.caret+1, len([]rune(p.buf)))

	case k.Code.Special == "home",
		k.Code.Char == 'a' && k.Mods == command.ModCtrl:
		p.caret = 0

	case k.Code.Special == "end",
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
		sig, msg := execTypable(cx, strings.TrimSpace(p.buf))
		p.ec.cmdMsg = msg
		return pop(signalToCmd(sig))

	case promptSearch:
		pat := strings.TrimSpace(p.buf)
		if pat == "" {
			// empty search repeats the last pattern in the prompt's direction
			pat, _ = cx.Editor.FirstRegister(view.RegisterSearch)
		}
		if pat == "" {
			p.ec.cmdMsg = ""
			return pop(nil)
		}
		var err error
		if p.forward {
			err = action.SearchForward(cx.Editor, pat)
		} else {
			err = action.SearchBackward(cx.Editor, pat)
		}
		if err != nil {
			p.ec.cmdMsg = i18n.Text(i18n.ErrorMessage, i18n.Vars{
				"message": err.Error(),
			})
		} else {
			p.ec.cmdMsg = ""
		}
		return pop(nil)

	default:
		if p.buf == "" {
			return pop(nil)
		}
		if p.pickerFn != nil {
			picker, err := p.pickerFn(cx.Editor, p.buf)
			if err != nil {
				p.ec.cmdMsg = i18n.Text(i18n.ErrorMessage, i18n.Vars{
					"message": err.Error(),
				})
				return pop(nil)
			}
			if picker == nil {
				return pop(nil)
			}
			feedCmd := picker.feedCmd
			picker.feedCmd = nil
			return consumedWith(func(_ *Context, comp *Compositor) tea.Cmd {
				comp.Pop()
				comp.Push(newPickerComponent(picker))
				return feedCmd
			})
		}
		if p.fn != nil {
			if err := p.fn(cx.Editor, p.buf); err != nil {
				p.ec.cmdMsg = i18n.Text(i18n.ErrorMessage, i18n.Vars{
					"message": err.Error(),
				})
			} else {
				p.ec.cmdMsg = ""
			}
		}
		return pop(nil)
	}
}

func (p *PromptComponent) paintLine(
	cx *Context, buf *tui.Buffer, area geom.Area,
) {
	th := cx.Theme()
	rowBg := lipgloss.NewStyle().
		Background(th.Get("ui.cursorline.primary").GetBackground())
	labelSt := lipglossToTUIStyle(
		applyAccentStyle(rowBg, th.Get("ui.prompt")),
	)
	textSt := lipglossToTUIStyle(
		rowBg.Foreground(th.Get("ui.text").GetForeground()),
	)

	label := p.promptLabel()
	buf.FillRange(area.Point, area.Width, textSt)
	buf.SetString(area.Point, label, labelSt)
	x := runewidth.StringWidth(label)

	p.syncScroll()
	runes := []rune(p.buf)
	avail := p.textWidth()
	truncEnd := runewidth.StringWidth(string(runes[p.hOff:])) > avail

	limit := avail
	if truncEnd {
		limit--
	}
	col, i := 0, p.hOff
	for i < len(runes) && col+runewidth.RuneWidth(runes[i]) <= limit {
		buf.SetString(geom.Point{
			X: area.X + x + col,
			Y: area.Y,
		}, string(runes[i]), textSt)
		col += runewidth.RuneWidth(runes[i])
		i++
	}
	if p.hOff > 0 {
		buf.SetString(geom.Point{
			X: area.X + x,
			Y: area.Y,
		}, "…", textSt)
	}
	if truncEnd {
		buf.SetString(geom.Point{
			X: area.X + x + col,
			Y: area.Y,
		}, "…", textSt)
	}
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
