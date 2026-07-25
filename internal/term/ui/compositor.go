package ui

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
)

type (
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

	// highlightRefresher re-issues its LSP symbol highlight on demand
	highlightRefresher interface {
		documentHighlightCmd(*Context) tea.Cmd
	}

	// previewImager renders an image preview and can redraw it each frame
	previewImager interface {
		previewImageCmd(*Context, geom.Size) tea.Cmd
		hasPreviewImage(*Context, geom.Size) bool
		markDirty()
	}

	layerFunc func(*Context) (Component, tea.Cmd)
)

func (c *Compositor) Push(layer Component) {
	c.layers = append(c.layers, layer)
}

func (c *Compositor) Pop() {
	if len(c.layers) > 1 {
		c.layers = c.layers[:len(c.layers)-1]
	}
}

func (c *Compositor) HandleEvent(cx *Context, msg tea.Msg) tea.Cmd {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		c.size = geom.Size{Width: ws.Width, Height: ws.Height}
	}

	var cmds []tea.Cmd
	var callbacks []Callback

	for i := len(c.layers) - 1; i >= 0; i-- {
		result, cmd := c.layers[i].HandleEvent(cx, msg)
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

	// After all layers have processed the resize (viewHeight is now set),
	// create and mount any deferred initial component
	if _, ok := msg.(tea.WindowSizeMsg); ok && !c.size.Empty() {
		if fn := c.startup; fn != nil {
			c.startup = nil
			if layer, cmd := fn(cx); layer != nil {
				c.Push(layer)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	return tea.Batch(cmds...)
}

func (c *Compositor) Render(cx *Context) string {
	if len(c.layers) == 0 {
		return ""
	}
	cx.composition.singleLayer = len(c.layers) == 1
	cx.composition.changed = !slices.Equal(c.lastOverlays, c.layers[1:])
	c.lastOverlays = slices.Clone(c.layers[1:])
	return c.renderViaBuffer(cx)
}

func (c *Compositor) Cursor(cx *Context) (cur tea.Cursor, ok bool) {
	for i := len(c.layers) - 1; i >= 0; i-- {
		if cur, ok = c.layers[i].Cursor(cx, c.size); ok {
			return
		}
	}
	return tea.Cursor{}, false
}

func (c *Compositor) refreshEditorHighlight(cx *Context) tea.Cmd {
	root, ok := c.layers[0].(highlightRefresher)
	if !ok {
		return nil
	}
	return root.documentHighlightCmd(cx)
}

func (c *Compositor) activePreviewImager() (previewImager, bool) {
	for i := len(c.layers) - 1; i >= 0; i-- {
		if p, ok := c.layers[i].(previewImager); ok {
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
