package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type PickerComponent struct {
	overlayBuf
	state         *Picker
	bounds        geom.Area
	listBounds    geom.Area
	previewBounds geom.Area
	splitBounds   geom.Area
	dragSplit     bool
}

const pickerMinPreviewArea = 72

var _ BufferOverlayComponent = (*PickerComponent)(nil)

func newPickerComponent(p *Picker) *PickerComponent {
	return &PickerComponent{state: p}
}

func (p *PickerComponent) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case pickerFeedMsg:
		return p.handleFeed(msg)
	case pickerDynamicTriggerMsg:
		return p.handleDynamicTrigger(cx, msg)
	case pickerDynamicFeedMsg:
		return p.handleDynamicFeed(msg)
	case tea.WindowSizeMsg:
		p.markDirty()
		p.state.clearPreviewCache()
		return ignored(), nil
	case tea.KeyPressMsg:
		return p.handleKey(cx, msg)
	case tea.MouseClickMsg:
		return p.handleMouseClick(cx, msg)
	case tea.MouseMotionMsg:
		return p.handleMouseMotion(cx, msg)
	case tea.MouseReleaseMsg:
		return p.handleMouseRelease(msg)
	case tea.MouseWheelMsg:
		return p.handleMouseWheel(cx, msg)
	}
	return ignored(), nil
}

func (p *PickerComponent) Layout(
	_ *Context, screen geom.Size,
) (geom.Area, bool) {
	size := pickerOverlaySize(screen)
	if size.Width < 4 || size.Height < 4 {
		return geom.Area{}, false
	}
	left := (screen.Width - size.Width) / 2
	top := max((screen.Height-2-size.Height)/2, 0)
	return geom.Area{
		Point: geom.Point{X: left, Y: top},
		Size:  size,
	}, true
}

func (p *PickerComponent) PaintBuffer(cx *Context, pl geom.Area) *tui.Buffer {
	return p.maybePaint(cx, pl.Size, func(buf *tui.Buffer) {
		p.paint(cx, buf, pl)
	})
}

func (p *PickerComponent) paint(cx *Context, buf *tui.Buffer, pl geom.Area) {
	ps := p.state
	areaW, areaH := pl.Width, pl.Height
	p.bounds = pl
	p.previewBounds = geom.Area{}
	p.splitBounds = geom.Area{}

	showPreview := areaW > pickerMinPreviewArea && previewEnabled(ps.source)
	splitW := 0
	if showPreview {
		ratio := cx.pickerLayout.SplitRatioFor(ps.source.ID())
		splitW = pickerSplitLeftWidth(areaW, ratio)
		p.splitBounds = geom.Area{
			Point: geom.Point{X: 1 + splitW, Y: 0},
			Size:  geom.Size{Width: 1, Height: areaH},
		}
		p.drawPickerBox(cx, buf, geom.Area{
			Size: geom.Size{Width: areaW, Height: areaH},
		}, splitW)
	} else {
		p.drawPickerPane(cx, buf, geom.Area{
			Size: geom.Size{Width: areaW, Height: areaH},
		})
	}

	headerH := 0
	if len(ps.source.Columns()) > 1 {
		headerH = 1
	}
	listW := areaW - 2
	if showPreview {
		listW = splitW
	}
	p.listBounds = geom.Area{
		Point: geom.Point{X: 1, Y: 3 + headerH},
		Size:  geom.Size{Width: listW, Height: ps.listHeight},
	}

	p.splitBounds = p.splitBounds.Translate(pl.Point)
	p.previewBounds = p.previewBounds.Translate(pl.Point)
	p.listBounds = p.listBounds.Translate(pl.Point)
}

func (p *PickerComponent) Cursor(
	*Context, geom.Size,
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
	cx *Context, msg pickerDynamicTriggerMsg,
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
	cx *Context, msg tea.MouseClickMsg,
) (EventResult, tea.Cmd) {
	if p.mouseOutside(geom.Point{X: msg.X, Y: msg.Y}) {
		return p.dismiss()
	}
	p.markDirty()
	clickPt := geom.Point{X: msg.X, Y: msg.Y}
	if msg.Button == tea.MouseLeft && p.splitBounds.Contains(clickPt) {
		p.dragSplit = true
		p.updateSplitRatio(cx, msg.X)
		return consumed(), nil
	}
	if idx, ok := listIndexAt(p.listBounds, p.state.listScroll, clickPt); ok {
		if idx >= 0 && idx < len(p.state.matched) {
			p.state.cursor = idx
		}
	}
	return consumed(), nil
}

func (p *PickerComponent) handleMouseMotion(
	cx *Context, msg tea.MouseMotionMsg,
) (EventResult, tea.Cmd) {
	if !p.dragSplit || msg.Button != tea.MouseLeft {
		return consumed(), nil
	}
	p.markDirty()
	p.updateSplitRatio(cx, msg.X)
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

func (p *PickerComponent) updateSplitRatio(cx *Context, x int) {
	usable := p.bounds.Width - pickerSplitFrameOverhead
	if usable <= 0 {
		return
	}
	left := x - (p.bounds.X + 1)
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
	cx *Context, msg tea.MouseWheelMsg,
) (EventResult, tea.Cmd) {
	step := cx.Editor.Options().ScrollLines
	p.markDirty()
	switch {
	case p.listBounds.Contains(geom.Point{X: msg.X, Y: msg.Y}):
		p.scrollListByWheel(msg.Button, step)
	case p.previewBounds.Contains(geom.Point{X: msg.X, Y: msg.Y}):
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

func (p *PickerComponent) mouseOutside(at geom.Point) bool {
	return !p.bounds.Contains(at)
}

func (p *PickerComponent) dismiss() (EventResult, tea.Cmd) {
	ps := p.state
	p.dragSplit = false
	if ps.dynamicStop != nil {
		ps.dynamicStop()
	}
	ps.cancel()
	return ignoredWith(func(_ *Context, comp *Compositor) tea.Cmd {
		comp.Pop()
		return nil
	}), nil
}
