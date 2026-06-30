package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type (
	PromptComponent struct {
		ec       *EditorComponent
		kind     promptKind
		forward  bool
		prompt   string
		buf      string
		comps    []promptCompletion
		compSel  *int
		compDone bool
		fn       func(*view.Editor, string) error
		pickerFn func(*view.Editor, string) (*Picker, error)
	}

	promptCompletion struct {
		command.Completion
		score int
	}

	promptKind uint8
)

const (
	promptCmd promptKind = iota + 1
	promptSearch
	promptRegex
	promptShell
)

const (
	promptCompletionBaseWidth = 30
	promptCompletionMaxRows   = 10
)

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

func (p *PromptComponent) Render(_, _ int, _ *Context) string {
	return ""
}

func (p *PromptComponent) RenderOver(
	width, height int, base string, cx *Context,
) string {
	if !p.compDone {
		p.recalculateCompletion(cx)
	}
	line := p.renderLine(width, cx)
	baseLayer := lipgloss.NewLayer(base)
	promptLayer := lipgloss.NewLayer(line).X(0).Y(height - 1).Z(1)
	layers := []*lipgloss.Layer{baseLayer}
	if menu := p.renderCompletions(width, height, cx); menu != "" {
		y := max(height-1-strings.Count(menu, "\n")-1, 0)
		layers = append(layers, lipgloss.NewLayer(menu).X(0).Y(y).Z(1))
	}
	layers = append(layers, promptLayer)
	return lipgloss.NewCompositor(layers...).Render()
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
	fn       func(*view.Editor, string) error
	pickerFn func(*view.Editor, string) (*Picker, error)
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
		if pat != "" {
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

func (p *PromptComponent) renderLine(w int, cx *Context) string {
	r := &renderPass{ec: p.ec, cx: cx, w: w}
	st := r.cmdlineStyle(false)
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
	return st.Width(w).Render(content)
}
