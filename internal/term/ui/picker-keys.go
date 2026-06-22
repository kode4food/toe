package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
)

func (p *PickerComponent) handleKey(
	msg tea.KeyPressMsg, cx *Context,
) (EventResult, tea.Cmd) {
	k := FromTeaKey(msg)
	ps := p.state
	dismiss := func() (EventResult, tea.Cmd) {
		if ps.dynamicStop != nil {
			ps.dynamicStop()
		}
		ps.cancel()
		return consumedWith(func(comp *Compositor, cx *Context) tea.Cmd {
			comp.Pop()
			return nil
		}), nil
	}
	switch {
	case k.Code.Special == "esc" ||
		(k.Code.Char == 'c' && k.Mods == command.ModCtrl):
		return dismiss()
	case k.Code.Special == "ret":
		if item := ps.selection(); item != nil {
			if r, cmd, ok := p.navigateItem(cx, ps, *item); ok {
				return r, cmd
			}
			ps.source.Accept(cx.Editor, *item)
		}
		return dismiss()
	case k.Code.Special == "up" ||
		(k.Code.Char == 'p' && k.Mods == command.ModCtrl) ||
		(k.Code.Special == "tab" && k.Mods == command.ModShift):
		ps.moveBy(-1)
	case k.Code.Special == "down" ||
		(k.Code.Char == 'n' && k.Mods == command.ModCtrl) ||
		k.Code.Special == "tab":
		ps.moveBy(1)
	case k.Code.Special == "pagedown" ||
		(k.Code.Char == 'd' && k.Mods == command.ModCtrl):
		ps.pageDown()
	case k.Code.Special == "pageup" ||
		(k.Code.Char == 'u' && k.Mods == command.ModCtrl):
		ps.pageUp()
	case k.Code.Special == "home":
		ps.cursor = 0
	case k.Code.Special == "end":
		if len(ps.matched) > 0 {
			ps.cursor = len(ps.matched) - 1
		}
	case k.Code.Special == "backspace" ||
		(k.Code.Char == 'h' && k.Mods == command.ModCtrl):
		runes := []rune(ps.query)
		if len(runes) > 0 {
			return consumed(), ps.setQuery(string(runes[:len(runes)-1]))
		}
	default:
		if k.IsTypable() {
			return consumed(), ps.setQuery(ps.query + string(k.Code.Char))
		}
	}
	// keyboard navigation keeps the selection on screen, unlike the wheel
	ps.ensureCursorVisible()
	return consumed(), nil
}

func (p *PickerComponent) navigateItem(
	cx *Context, ps *Picker, item PickerItem,
) (EventResult, tea.Cmd, bool) {
	nav, ok := ps.source.(NavigablePickerSource)
	if !ok {
		return EventResult{}, nil, false
	}
	fn := nav.Navigate(cx.Editor, item)
	if fn == nil {
		return EventResult{}, nil, false
	}
	next := fn(cx.Editor)
	if next == nil {
		return EventResult{}, nil, false
	}
	feedCmd := next.feedCmd
	next.feedCmd = nil
	if ps.dynamicStop != nil {
		ps.dynamicStop()
	}
	ps.cancel()
	return consumedWith(func(comp *Compositor, _ *Context) tea.Cmd {
		comp.Pop()
		comp.Push(newPickerComponent(next))
		return feedCmd
	}), nil, true
}
