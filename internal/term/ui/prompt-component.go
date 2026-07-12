package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type (
	PromptComponent struct {
		overlayBuf
		ec       *EditorComponent
		kind     promptKind
		forward  bool
		prompt   string
		buf      string
		comps    []promptCompletion
		compSel  *int
		compDone bool
		compCols int
		compRows int
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

const (
	promptCompletionBaseWidth = 30
	promptCompletionMaxRows   = 10
	promptCompletionPadX      = 1
)

var _ BufferOverlayComponent = (*PromptComponent)(nil)

func (p *PromptComponent) HandleEvent(
	msg tea.Msg, cx *Context,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(msg, cx), nil

	case tea.MouseMsg, tea.MouseClickMsg:
		return consumed(), nil
	}
	return ignored(), nil
}

func (p *PromptComponent) Layout(
	screenW, screenH int, cx *Context,
) (Bounds, bool) {
	p.markDirty()
	if !p.compDone {
		p.recalculateCompletion(cx)
	}
	menuH := p.completionMenuHeight(screenW, screenH)
	y := max(screenH-1-menuH, 0)
	return Bounds{x: 0, y: y, w: screenW, h: screenH - y}, true
}

func (p *PromptComponent) PaintBuffer(pl Bounds, cx *Context) *tui.Buffer {
	return p.maybePaint(pl.w, pl.h, cx, func(buf *tui.Buffer) {
		if p.compRows > 0 {
			p.paintCompletions(buf, 0, pl.w, cx)
		}
		p.paintLine(buf, pl.h-1, pl.w, cx)
	})
}

func (p *PromptComponent) Cursor(
	_, _ int, _ *Context,
) (cur tea.Cursor, ok bool) {
	return tea.Cursor{}, false
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
		fn:       args.fn,
		pickerFn: args.pickerFn,
	}
}

func (p *PromptComponent) handleKey(
	msg tea.KeyPressMsg, cx *Context,
) EventResult {
	k := FromTeaKey(msg)
	pop := func(cmd tea.Cmd) EventResult {
		return consumedWith(func(comp *Compositor, _ *Context) tea.Cmd {
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

	case k.Code.Special == "backspace" ||
		(k.Code.Char == 'h' && k.Mods == command.ModCtrl):
		runes := []rune(p.buf)
		if len(runes) > 0 {
			p.buf = string(runes[:len(runes)-1])
		}
		p.recalculateCompletion(cx)

	default:
		if k.IsTypable() {
			p.buf += string(k.Code.Char)
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
			pat, _ = cx.Editor.FirstRegister('/')
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
			p.ec.cmdMsg = "error: " + err.Error()
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
				p.ec.cmdMsg = "error: " + err.Error()
				return pop(nil)
			}
			if picker == nil {
				return pop(nil)
			}
			feedCmd := picker.feedCmd
			picker.feedCmd = nil
			return consumedWith(func(comp *Compositor, _ *Context) tea.Cmd {
				comp.Pop()
				comp.Push(newPickerComponent(picker))
				return feedCmd
			})
		}
		if p.fn != nil {
			if err := p.fn(cx.Editor, p.buf); err != nil {
				p.ec.cmdMsg = "error: " + err.Error()
			} else {
				p.ec.cmdMsg = ""
			}
		}
		return pop(nil)
	}
}

func (p *PromptComponent) paintLine(buf *tui.Buffer, y, w int, cx *Context) {
	r := &renderPass{ec: p.ec, cx: cx, w: w}
	st := lipglossToTUIStyle(r.cmdlineStyle(false))
	var content string
	switch p.kind {
	case promptCmd:
		content = ":" + p.buf
	case promptSearch:
		if p.forward {
			content = "/" + p.buf
		} else {
			content = "?" + p.buf
		}
	default:
		content = p.prompt + " " + p.buf
	}
	buf.SetString(0, y, strings.Repeat(" ", w), st)
	buf.SetString(0, y, content, st)
}
