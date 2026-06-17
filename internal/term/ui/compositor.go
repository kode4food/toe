package ui

import tea "charm.land/bubbletea/v2"

type (
	Compositor struct {
		layers     []Component
		width      int
		height     int
		cachedView string
		startup    layerFunc
	}

	layerFunc func(*Context) (Component, tea.Cmd)
)

func newCompositor() *Compositor {
	return &Compositor{}
}

func (c *Compositor) Push(layer Component) {
	c.layers = append(c.layers, layer)
}

func (c *Compositor) Pop() {
	if len(c.layers) > 1 {
		c.layers = c.layers[:len(c.layers)-1]
	}
}

func (c *Compositor) HandleEvent(msg tea.Msg, cx *Context) tea.Cmd {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		c.width = ws.Width
		c.height = ws.Height
	}

	var cmds []tea.Cmd
	var callbacks []Callback

	for i := len(c.layers) - 1; i >= 0; i-- {
		result, cmd := c.layers[i].HandleEvent(msg, cx)
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
		if cmd := cb(c, cx); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// After all layers have processed the resize (viewHeight is now set),
	// create and mount any deferred initial component
	if _, ok := msg.(tea.WindowSizeMsg); ok && c.width > 0 && c.height > 0 {
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
	content, ok := c.renderViaBuffer(cx)
	if !ok {
		content = c.layers[0].Render(c.width, c.height, cx)
		for i := 1; i < len(c.layers); i++ {
			if ov, ok := c.layers[i].(OverlayComponent); ok {
				content = ov.RenderOver(c.width, c.height, content, cx)
			}
		}
	}
	if content == c.cachedView {
		return c.cachedView
	}
	c.cachedView = content
	return content
}

// falls back (!ok) when any layer doesn't implement the buffer interface,
// so the caller can use the per-layer ANSI compositing path instead
func (c *Compositor) renderViaBuffer(cx *Context) (string, bool) {
	br, ok := c.layers[0].(BufferRenderer)
	if !ok {
		return "", false
	}
	overlays := make([]BufferOverlayComponent, 0, len(c.layers)-1)
	for i := 1; i < len(c.layers); i++ {
		ov, ok := c.layers[i].(BufferOverlayComponent)
		if !ok {
			return "", false
		}
		overlays = append(overlays, ov)
	}
	buf := br.RenderBuffer(c.width, c.height, cx)
	for _, ov := range overlays {
		ov.RenderOverBuffer(buf, cx)
	}
	return buf.RenderToANSI(), true
}

func (c *Compositor) Cursor(cx *Context) (cur tea.Cursor, ok bool) {
	for i := len(c.layers) - 1; i >= 0; i-- {
		if cur, ok = c.layers[i].Cursor(c.width, c.height, cx); ok {
			return
		}
	}
	return tea.Cursor{}, false
}
