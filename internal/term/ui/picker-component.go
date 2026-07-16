package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/tui"
)

type PickerComponent struct {
	overlayBuf
	state         *Picker
	bounds        Bounds
	listBounds    Bounds
	previewBounds Bounds
	splitBounds   Bounds
	dragSplit     bool
}

const pickerMinPreviewArea = 72

var _ BufferOverlayComponent = (*PickerComponent)(nil)

func newPickerComponent(p *Picker) *PickerComponent {
	return &PickerComponent{state: p}
}

func (p *PickerComponent) HandleEvent(
	msg tea.Msg, cx *Context,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case pickerFeedMsg:
		return p.handleFeed(msg)
	case pickerDynamicTriggerMsg:
		return p.handleDynamicTrigger(msg, cx)
	case pickerDynamicFeedMsg:
		return p.handleDynamicFeed(msg)
	case tea.WindowSizeMsg:
		p.markDirty()
		p.state.clearPreviewCache()
		return ignored(), nil
	case tea.KeyPressMsg:
		return p.handleKey(msg, cx)
	case tea.MouseClickMsg:
		return p.handleMouseClick(msg, cx)
	case tea.MouseMotionMsg:
		return p.handleMouseMotion(msg, cx)
	case tea.MouseReleaseMsg:
		return p.handleMouseRelease(msg)
	case tea.MouseWheelMsg:
		return p.handleMouseWheel(msg, cx)
	}
	return ignored(), nil
}

func (p *PickerComponent) Layout(
	screenW, screenH int, _ *Context,
) (Bounds, bool) {
	areaW, areaH := pickerOverlaySize(screenW, screenH)
	if areaW < 4 || areaH < 4 {
		return Bounds{}, false
	}
	left := (screenW - areaW) / 2
	top := max((screenH-2-areaH)/2, 0)
	return Bounds{x: left, y: top, w: areaW, h: areaH}, true
}

func (p *PickerComponent) PaintBuffer(pl Bounds, cx *Context) *tui.Buffer {
	return p.maybePaint(pl.w, pl.h, cx, func(buf *tui.Buffer) {
		p.paint(buf, pl, cx)
	})
}

func (p *PickerComponent) paint(buf *tui.Buffer, pl Bounds, cx *Context) {
	ps := p.state
	areaW, areaH := pl.w, pl.h
	p.bounds = pl
	p.previewBounds = Bounds{}
	p.splitBounds = Bounds{}

	showPreview := areaW > pickerMinPreviewArea && previewEnabled(ps.source)
	splitW := 0
	if showPreview {
		ratio := cx.pickerLayout.SplitRatioFor(ps.source.ID())
		splitW = pickerSplitLeftWidth(areaW, ratio)
		p.splitBounds = Bounds{x: 1 + splitW, y: 0, w: 1, h: areaH}
		p.drawPickerBox(buf, 0, 0, areaW, areaH, splitW, cx)
	} else {
		p.drawPickerPane(buf, 0, 0, areaW, areaH, cx)
	}

	headerH := 0
	if len(ps.source.Columns()) > 1 {
		headerH = 1
	}
	listW := areaW - 2
	if showPreview {
		listW = splitW
	}
	p.listBounds = Bounds{x: 1, y: 3 + headerH, w: listW, h: ps.listHeight}

	p.splitBounds = p.splitBounds.translate(pl.x, pl.y)
	p.previewBounds = p.previewBounds.translate(pl.x, pl.y)
	p.listBounds = p.listBounds.translate(pl.x, pl.y)
}

func (p *PickerComponent) Cursor(
	_, _ int, _ *Context,
) (cur tea.Cursor, ok bool) {
	return tea.Cursor{}, false
}

func (p *PickerComponent) handleFeed(msg pickerFeedMsg) (EventResult, tea.Cmd) {
	p.markDirty()
	p.state.addItems(msg.items)
	if msg.feed != nil {
		return consumed(), drainPickerFeed(msg.feed, msg.done)
	}
	return consumed(), nil
}

func (p *PickerComponent) handleDynamicTrigger(
	msg pickerDynamicTriggerMsg, cx *Context,
) (EventResult, tea.Cmd) {
	ps := p.state
	if msg.gen != ps.dynamicGen {
		return consumed(), nil
	}
	src, ok := ps.source.(DynamicPickerSource)
	if !ok {
		return consumed(), nil
	}
	p.markDirty()
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
	ps.dynamicPending = false
	return consumed(), nil
}

func (p *PickerComponent) handleDynamicFeed(
	msg pickerDynamicFeedMsg,
) (EventResult, tea.Cmd) {
	ps := p.state
	if msg.gen != ps.dynamicGen {
		return consumed(), nil
	}
	p.markDirty()
	ps.addDynamicItems(msg.items)
	if msg.feed != nil {
		return consumed(), drainDynamicFeed(msg.gen, msg.feed)
	}
	ps.dynamicPending = false
	return consumed(), nil
}

func (p *PickerComponent) handleMouseClick(
	msg tea.MouseClickMsg, cx *Context,
) (EventResult, tea.Cmd) {
	if p.mouseOutside(msg.X, msg.Y) {
		return p.dismiss()
	}
	p.markDirty()
	if msg.Button == tea.MouseLeft && p.splitBounds.contains(msg.X, msg.Y) {
		p.dragSplit = true
		p.updateSplitRatio(msg.X, cx)
		return consumed(), nil
	}
	if idx, ok := listIndexAt(
		p.listBounds, p.state.listScroll, msg.X, msg.Y,
	); ok {
		if idx >= 0 && idx < len(p.state.matched) {
			p.state.cursor = idx
		}
	}
	return consumed(), nil
}

func (p *PickerComponent) handleMouseMotion(
	msg tea.MouseMotionMsg, cx *Context,
) (EventResult, tea.Cmd) {
	if !p.dragSplit || msg.Button != tea.MouseLeft {
		return consumed(), nil
	}
	p.markDirty()
	p.updateSplitRatio(msg.X, cx)
	return consumed(), nil
}

func (p *PickerComponent) handleMouseRelease(
	msg tea.MouseReleaseMsg,
) (EventResult, tea.Cmd) {
	if msg.Button == tea.MouseLeft {
		p.dragSplit = false
	}
	return consumed(), nil
}

func (p *PickerComponent) updateSplitRatio(x int, cx *Context) {
	usable := p.bounds.w - pickerSplitFrameOverhead
	if usable <= 0 {
		return
	}
	left := x - (p.bounds.x + 1)
	ratio := float64(left) / float64(usable)
	ratio = clampPickerSplitRatio(ratio)
	opts := cx.pickerLayout.clone()
	if opts.SplitRatios == nil {
		opts.SplitRatios = map[string]float64{}
	}
	opts.SplitRatios[p.state.source.ID()] = ratio
	cx.pickerLayout = opts
}

func (p *PickerComponent) handleMouseWheel(
	msg tea.MouseWheelMsg, cx *Context,
) (EventResult, tea.Cmd) {
	step := cx.Editor.Options().ScrollLines
	p.markDirty()
	switch {
	case p.listBounds.contains(msg.X, msg.Y):
		p.scrollListByWheel(msg.Button, step)
	case p.previewBounds.contains(msg.X, msg.Y):
		p.scrollPreviewByWheel(msg.Button, step)
	}
	return consumed(), nil
}

func (p *PickerComponent) scrollListByWheel(button tea.MouseButton, step int) {
	switch button {
	case tea.MouseWheelUp:
		p.state.scrollBy(-step)
	case tea.MouseWheelDown:
		p.state.scrollBy(step)
	}
}

func (p *PickerComponent) scrollPreviewByWheel(
	button tea.MouseButton, step int,
) {
	// clamped by the renderer, which knows the document length
	switch button {
	case tea.MouseWheelUp:
		p.state.previewScroll -= step
	case tea.MouseWheelDown:
		p.state.previewScroll += step
	}
}

func (p *PickerComponent) mouseOutside(x, y int) bool {
	return !p.bounds.contains(x, y)
}

func (p *PickerComponent) dismiss() (EventResult, tea.Cmd) {
	ps := p.state
	p.dragSplit = false
	if ps.dynamicStop != nil {
		ps.dynamicStop()
	}
	ps.cancel()
	return ignoredWith(func(comp *Compositor, _ *Context) tea.Cmd {
		comp.Pop()
		return nil
	}), nil
}
