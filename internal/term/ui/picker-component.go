package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
)

type (
	PickerComponent struct {
		state         *Picker
		bounds        bounds
		listBounds    bounds
		previewBounds bounds
	}

	bounds struct{ x, y, w, h int }
)

const pickerMinPreviewArea = 72

func newPickerComponent(p *Picker) *PickerComponent {
	return &PickerComponent{state: p}
}

func (p *PickerComponent) HandleEvent(
	msg tea.Msg, cx *Context,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case pickerFeedMsg:
		p.state.addItems(msg.items)
		if msg.feed != nil {
			return consumed(), drainPickerFeed(msg.feed, msg.done)
		}
		return consumed(), nil

	case pickerDynamicTriggerMsg:
		ps := p.state
		if msg.gen != ps.dynamicGen {
			return consumed(), nil
		}
		src, ok := ps.source.(DynamicPickerSource)
		if !ok {
			return consumed(), nil
		}
		src.Search(msg.query)
		items, ch, stop := src.Load(cx.Editor)
		ps.dynamicStop = stop
		ps.items = items
		ps.matched = make([]pickerMatch, len(items))
		for i := range items {
			ps.matched[i] = pickerMatch{item: &items[i]}
		}
		if ch != nil {
			return consumed(), drainDynamicFeed(msg.gen, ch)
		}
		return consumed(), nil

	case pickerDynamicFeedMsg:
		ps := p.state
		if msg.gen != ps.dynamicGen {
			return consumed(), nil
		}
		ps.addDynamicItems(msg.items)
		if msg.feed != nil {
			return consumed(), drainDynamicFeed(msg.gen, msg.feed)
		}
		return consumed(), nil

	case tea.WindowSizeMsg:
		// File span caches are geometry-independent; no action needed on resize
		return ignored(), nil

	case tea.KeyPressMsg:
		return p.handleKey(msg, cx)

	case tea.MouseClickMsg:
		if p.mouseOutside(msg.X, msg.Y) {
			return p.dismiss()
		}
		if p.listBounds.contains(msg.X, msg.Y) {
			idx := p.state.listScroll + (msg.Y - p.listBounds.y)
			if idx >= 0 && idx < len(p.state.matched) {
				p.state.cursor = idx
			}
		}
		return consumed(), nil

	case tea.MouseWheelMsg:
		step := cx.Editor.Options().ScrollLines
		switch {
		case p.listBounds.contains(msg.X, msg.Y):
			switch msg.Button {
			case tea.MouseWheelUp:
				p.state.scrollBy(-step)
			case tea.MouseWheelDown:
				p.state.scrollBy(step)
			}
		case p.previewBounds.contains(msg.X, msg.Y):
			// clamped by the renderer, which knows the document length
			switch msg.Button {
			case tea.MouseWheelUp:
				p.state.previewScroll -= step
			case tea.MouseWheelDown:
				p.state.previewScroll += step
			}
		}
		return consumed(), nil
	}
	return ignored(), nil
}

func (p *PickerComponent) Render(_, _ int, _ *Context) string {
	return ""
}

// RenderOverBuffer avoids the lipgloss Compositor round-trip that was the
// dominant per-frame cost in profiling
func (p *PickerComponent) RenderOverBuffer(buf *tui.Buffer, cx *Context) {
	ps := p.state
	width, height := buf.Width, buf.Height
	areaW, areaH := pickerOverlaySize(width, height)
	if areaW < 4 || areaH < 4 {
		return
	}
	left := (width - areaW) / 2
	top := max((height-2-areaH)/2, 0)
	p.bounds = bounds{x: left, y: top, w: areaW, h: areaH}

	if ps.preview && areaW > pickerMinPreviewArea {
		p.drawPickerBox(buf, left, top, areaW, areaH, cx)
	} else {
		p.drawPickerPane(buf, left, top, areaW, areaH, cx)
	}

	headerH := 0
	if len(ps.source.Columns()) > 1 {
		headerH = 1
	}
	listW := areaW - 2
	if ps.preview && areaW > pickerMinPreviewArea {
		listW = areaW/2 - 1
	}
	p.listBounds = bounds{
		x: left + 1, y: top + 3 + headerH, w: listW, h: ps.listHeight,
	}
}

func (p *PickerComponent) Cursor(
	_, _ int, _ *Context,
) (cur tea.Cursor, ok bool) {
	return tea.Cursor{}, false
}

func (p *PickerComponent) mouseOutside(x, y int) bool {
	return !p.bounds.contains(x, y)
}

func (p *PickerComponent) dismiss() (EventResult, tea.Cmd) {
	ps := p.state
	if ps.dynamicStop != nil {
		ps.dynamicStop()
	}
	ps.cancel()
	return ignoredWith(func(comp *Compositor, _ *Context) tea.Cmd {
		comp.Pop()
		return nil
	}), nil
}

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

func (p *PickerComponent) drawPickerBox(
	buf *tui.Buffer, x, y, w, h int, cx *Context,
) {
	ps := p.state
	lw := w/2 - 1
	innerH := h - 2

	cols := ps.source.Columns()
	headerH := 0
	if len(cols) > 1 {
		headerH = 1
	}
	ps.listHeight = max(innerH-2-headerH, 1)

	frame := pickerBoxFrame{
		border:       lipgloss.RoundedBorder(),
		borderStyle:  lipglossToTUIStyle(pickerFrameStyle(cx)),
		contentStyle: lipglossToTUIStyle(pickerContentStyle(cx)),
	}
	areas := frame.drawSplit(buf, x, y, w, h, lw, 2)

	writePickerPromptRow(
		buf, areas.left.x, areas.left.y, areas.left.w, ps, cx,
	)
	itemY := areas.left.y + 2 // row 1 is the cut-separator, skip it
	if len(cols) > 1 {
		writePickerHeader(buf, areas.left.x, itemY, areas.left.w, ps, cx)
		itemY++
	}
	ps.clampScroll()
	for i := range ps.listHeight {
		idx := ps.listScroll + i
		if idx >= len(ps.matched) {
			break
		}
		writePickerItem(buf, areas.left.x, itemY+i, areas.left.w,
			&pickerItemRender{
				p: ps, match: ps.matched[idx], w: areas.left.w,
				selected: idx == ps.cursor, cx: cx,
			},
		)
	}

	p.drawPreviewInto(buf, areas.right.x, areas.right.y, areas.right.w,
		areas.right.h, cx)
}

func (p *PickerComponent) drawPickerPane(
	buf *tui.Buffer, x, y, w, h int, cx *Context,
) {
	ps := p.state
	innerH := h - 2

	cols := ps.source.Columns()
	headerH := 0
	if len(cols) > 1 {
		headerH = 1
	}
	ps.listHeight = max(innerH-2-headerH, 1)

	frame := pickerBoxFrame{
		border:       lipgloss.RoundedBorder(),
		borderStyle:  lipglossToTUIStyle(pickerFrameStyle(cx)),
		contentStyle: lipglossToTUIStyle(pickerContentStyle(cx)),
	}
	area := frame.drawSingle(buf, x, y, w, h, 2)

	writePickerPromptRow(buf, area.x, area.y, area.w, ps, cx)
	itemY := area.y + 2 // row 1 is the cut-separator, skip it
	if len(cols) > 1 {
		writePickerHeader(buf, area.x, itemY, area.w, ps, cx)
		itemY++
	}
	ps.clampScroll()
	for i := 0; ps.listScroll+i < len(ps.matched) && i < ps.listHeight; i++ {
		idx := ps.listScroll + i
		writePickerItem(buf, area.x, itemY+i, area.w, &pickerItemRender{
			p: ps, match: ps.matched[idx], w: area.w,
			selected: idx == ps.cursor, cx: cx,
		})
	}
}

func (p *PickerComponent) drawPreviewInto(
	buf *tui.Buffer, x0, y0, w, h int, cx *Context,
) {
	ps := p.state
	p.previewBounds = bounds{x: x0, y: y0, w: w, h: h}
	if ps.cursor != ps.previewScrollFor {
		ps.previewScroll = 0
		ps.previewScrollFor = ps.cursor
	}
	item := ps.selection()
	if item == nil {
		return
	}
	innerW := max(w-2*pickerPadX, 1)
	from, to, ok := item.Location.lineRange()
	ctx := previewCtx{
		picker: ps,
		item:   item,
		editor: cx.Editor,
		w:      innerW,
		h:      h,
		th:     cx.Theme(),
		hlFrom: -1,
	}
	if ok {
		ctx.hlFrom = from
		ctx.hlTo = to
	}
	ctx.renderInto(buf, x0+pickerPadX, y0)
}

func (b bounds) contains(x, y int) bool {
	return x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h
}
