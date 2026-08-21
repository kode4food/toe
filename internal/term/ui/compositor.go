package ui

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
)

type (
	// Compositor manages the editor's layered components
	Compositor struct {
		layers       []Component
		size         geom.Size
		startup      layerFunc
		lastOverlays []Component
	}

	bufferOverlayPlacement struct {
		overlay BufferOverlayComponent
		bounds  geom.Area
	}

	highlightRefresher interface {
		documentHighlightCmd(*Context) tea.Cmd
	}

	previewImager interface {
		previewImageCmd(*Context, geom.Size) tea.Cmd
		hasPreviewImage(*Context, geom.Size) bool
		markDirty()
	}

	layerFunc func(*Context) (Component, tea.Cmd)
)

// Push adds a layer above the current top
func (c *Compositor) Push(layer Component) {
	c.layers = append(c.layers, layer)
}

// Pop removes the topmost layer
func (c *Compositor) Pop() {
	if len(c.layers) > 1 {
		c.layers = c.layers[:len(c.layers)-1]
	}
}

// HandleEvent offers a message to each layer from the top down
func (c *Compositor) HandleEvent(cx *Context, msg tea.Msg) tea.Cmd {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		c.size = geom.Size{Width: ws.Width, Height: ws.Height}
	}

	var cmds []tea.Cmd
	var callbacks []Callback

	for _, v := range slices.Backward(c.layers) {
		result, cmd := c.handleLayerEvent(cx, v, msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if result.Callback != nil {
			callbacks = append(callbacks, result.Callback)
		}
		if result.Consumed {
			break
		}
	}

	for _, cb := range callbacks {
		if cmd := cb(cx, c); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if _, ok := msg.(tea.WindowSizeMsg); !ok {
		return tea.Batch(cmds...)
	}
	if c.size.IsEmpty() || c.startup == nil {
		return tea.Batch(cmds...)
	}
	fn := c.startup
	c.startup = nil
	layer, cmd := fn(cx)
	if layer == nil {
		return tea.Batch(cmds...)
	}
	c.Push(layer)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// Render paints every layer bottom up into one frame
func (c *Compositor) Render(cx *Context) string {
	if len(c.layers) == 0 {
		return ""
	}
	cx.composition.singleLayer = len(c.layers) == 1
	cx.composition.changed = !slices.Equal(c.lastOverlays, c.layers[1:])
	c.lastOverlays = slices.Clone(c.layers[1:])
	return c.renderViaBuffer(cx)
}

// Cursor returns the cursor of the topmost layer that wants one
func (c *Compositor) Cursor(cx *Context) (cur tea.Cursor, ok bool) {
	for i, v := range slices.Backward(c.layers) {
		if cur, ok = v.Cursor(cx, c.size); ok {
			if c.cursorCovered(cx, i+1, cur) {
				return tea.Cursor{}, false
			}
			return
		}
	}
	return tea.Cursor{}, false
}

func (c *Compositor) handleLayerEvent(
	cx *Context, layer Component, msg tea.Msg,
) (EventResult, tea.Cmd) {
	click, ok := msg.(tea.MouseClickMsg)
	if !ok {
		return layer.HandleEvent(cx, msg)
	}
	dismisser, ok := layer.(overlayDismisser)
	if !ok {
		return layer.HandleEvent(cx, msg)
	}
	bounds, active := dismisser.Layout(cx, c.size)
	if active && bounds.Contains(geom.Point{X: click.X, Y: click.Y}) {
		return layer.HandleEvent(cx, msg)
	}
	return dismisser.dismissOverlay()
}

func (c *Compositor) cursorCovered(cx *Context, from int, cur tea.Cursor) bool {
	at := geom.Point{X: cur.X, Y: cur.Y}
	for _, layer := range c.layers[from:] {
		ov, ok := layer.(BufferOverlayComponent)
		if !ok {
			continue
		}
		area, active := ov.Layout(cx, c.size)
		if active && area.Contains(at) {
			return true
		}
	}
	return false
}

func (c *Compositor) refreshEditorHighlight(cx *Context) tea.Cmd {
	if root, ok := c.layers[0].(highlightRefresher); ok {
		return root.documentHighlightCmd(cx)
	}
	return nil
}

func (c *Compositor) activePreviewImager() (previewImager, bool) {
	for _, v := range slices.Backward(c.layers) {
		if p, ok := v.(previewImager); ok {
			return p, true
		}
	}
	return nil, false
}

func (c *Compositor) renderViaBuffer(cx *Context) string {
	br := c.layers[0].(BufferRenderer)
	placements := make([]bufferOverlayPlacement, 0, len(c.layers)-1)
	for i := 1; i < len(c.layers); i++ {
		ov := c.layers[i].(BufferOverlayComponent)
		if pl, active := ov.Layout(cx, c.size); active {
			placements = append(placements, bufferOverlayPlacement{
				overlay: ov,
				bounds:  pl,
			})
		}
	}
	frame := br.Render(cx, c.size)
	regions := make([]geom.Area, 0, len(placements))
	for _, p := range placements {
		buf := p.overlay.PaintBuffer(cx, p.bounds)
		frame.Blit(buf, p.bounds.Point)
		regions = append(regions, p.bounds)
	}
	cx.composition.regions = regions
	cx.composition.precise = true
	return frame.RenderToANSI()
}
