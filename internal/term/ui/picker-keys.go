package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
)

type dismissFunc func() (EventResult, tea.Cmd)

func (p *PickerComponent) handleKey(
	cx *Context, msg tea.KeyPressMsg,
) (EventResult, tea.Cmd) {
	k := FromTeaKey(msg)
	ps := p.state
	p.markDirty()
	dismiss := func() (EventResult, tea.Cmd) {
		if ps.dynamicStop != nil {
			ps.dynamicStop()
		}
		ps.cancel()
		return consumedWith(func(cx *Context, comp *Compositor) tea.Cmd {
			comp.Pop()
			return nil
		}), nil
	}
	switch {
	case k.Code.Special == command.Escape ||
		(k.Code.Char == 'c' && k.Mods == command.ModCtrl):
		return dismiss()
	case k.Code.Special == command.Enter:
		return p.acceptItem(cx, ps, PickerAcceptReplace, dismiss)
	case k.Code.Char == 's' && k.Mods == command.ModCtrl:
		return p.acceptItem(cx, ps, PickerAcceptHorizontalSplit, dismiss)
	case k.Code.Char == 'v' && k.Mods == command.ModCtrl:
		return p.acceptItem(cx, ps, PickerAcceptVerticalSplit, dismiss)
	case k.Code.Special == command.Up ||
		(k.Code.Char == 'p' && k.Mods == command.ModCtrl) ||
		(k.Code.Special == command.Tab && k.Mods == command.ModShift):
		ps.moveBy(-1)
	case k.Code.Special == command.Down ||
		(k.Code.Char == 'n' && k.Mods == command.ModCtrl) ||
		k.Code.Special == command.Tab:
		ps.moveBy(1)
	case k.Code.Special == command.PageDown ||
		(k.Code.Char == 'd' && k.Mods == command.ModCtrl):
		ps.pageDown()
	case k.Code.Special == command.PageUp ||
		(k.Code.Char == 'u' && k.Mods == command.ModCtrl):
		ps.pageUp()
	case k.Code.Special == command.Home:
		ps.cursor = 0
	case k.Code.Special == command.End:
		if len(ps.matched) > 0 {
			ps.cursor = len(ps.matched) - 1
		}
	case k.Code.Special == command.Backspace ||
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

func (p *PickerComponent) acceptItem(
	cx *Context, ps *Picker, action PickerAcceptAction, dismiss dismissFunc,
) (EventResult, tea.Cmd) {
	item := ps.selection()
	if item == nil {
		return dismiss()
	}
	if action == PickerAcceptReplace {
		if r, cmd, ok := p.navigateItem(cx, ps, *item); ok {
			return r, cmd
		}
	}
	ps.source.Accept(cx.Editor, *item, action)
	return dismiss()
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
	return consumedWith(func(_ *Context, comp *Compositor) tea.Cmd {
		comp.Pop()
		comp.Push(newPickerComponent(next))
		return feedCmd
	}), nil, true
}
